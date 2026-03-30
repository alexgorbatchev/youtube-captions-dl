package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexgorbatchev/youtube-captions-dl/internal/cache"
	"github.com/alexgorbatchev/youtube-captions-dl/internal/youtube"
)

const (
	requestTimeout = 30 * time.Second
	description    = "Download YouTube captions and print plain text to stdout."
)

var (
	errUsage   = errors.New("usage")
	errHelp    = errors.New("help")
	errVersion = errors.New("version")
	version    = "dev"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := runMain(args, stdout, stderr); err != nil {
		if errors.Is(err, errHelp) {
			writeHelp(stdout)
			return 0
		}
		if errors.Is(err, errVersion) {
			writeVersion(stdout)
			return 0
		}
		if errors.Is(err, errUsage) {
			writeHelp(stderr)
			return 2
		}

		writeDiagnosticf(stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func runMain(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 1 {
		if isHelpFlag(args[0]) {
			return errHelp
		}
		if isVersionFlag(args[0]) {
			return errVersion
		}
	}
	if len(args) != 1 {
		return errUsage
	}

	videoID, err := youtube.ParseVideoID(args[0])
	if err != nil {
		return fmt.Errorf("parsing video URL: %w", err)
	}

	store, err := cache.NewStore()
	if err != nil {
		return fmt.Errorf("creating cache store: %w", err)
	}

	cachedText, ok, err := store.Load(videoID)
	if err != nil {
		writeDiagnosticf(stderr, "warning: reading cache failed: %v\n", err)
	} else if ok {
		return writePlainText(stdout, cachedText)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	client := youtube.NewClient(nil)
	plainText, err := client.FetchPlainText(ctx, videoID)
	if err != nil {
		return err
	}

	if err := store.Save(videoID, plainText); err != nil {
		writeDiagnosticf(stderr, "warning: writing cache failed: %v\n", err)
	}

	return writePlainText(stdout, plainText)
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func isVersionFlag(arg string) bool {
	switch arg {
	case "--version", "-version", "version":
		return true
	default:
		return false
	}
}

func writeHelp(w io.Writer) {
	writeDiagnosticf(w, "%s\n\n", description)
	writeDiagnosticf(w, "usage:\n")
	writeDiagnosticf(w, "  %s <youtube-url>\n\n", programName())
	writeDiagnosticf(w, "flags:\n")
	writeDiagnosticf(w, "  -h, --help              Show help\n")
	writeDiagnosticf(w, "  --version, -version     Show version\n")
	writeDiagnosticf(w, "  version                 Show version\n")
}

func writeVersion(w io.Writer) {
	writeDiagnosticf(w, "%s %s\n", programName(), version)
}

func programName() string {
	return filepath.Base(os.Args[0])
}

func writeDiagnosticf(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return
	}
}

func writePlainText(w io.Writer, text string) error {
	if _, err := io.WriteString(w, text); err != nil {
		return fmt.Errorf("writing stdout: %w", err)
	}

	if strings.HasSuffix(text, "\n") {
		return nil
	}

	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}

	return nil
}
