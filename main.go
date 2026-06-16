package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/alexgorbatchev/youtube-captions-dl/internal/cache"
	"github.com/alexgorbatchev/youtube-captions-dl/internal/youtube"
	"github.com/spf13/cobra"
)

const (
	requestTimeout = 30 * time.Second
	description    = "Download YouTube captions and print plain text to stdout."
)

var (
	version = "dev"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	var runECalled bool

	cmd := newRootCmd(stdout, stderr, &runECalled)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if runECalled {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		cmd.SetOut(stderr)
		_ = cmd.Help()
		return 2
	}
	return 0
}

func newRootCmd(stdout, stderr io.Writer, runECalled *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "youtube-captions-dl <youtube-url>",
		Short:         description,
		Args:          cobra.ExactArgs(1),
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			*runECalled = true

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
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: reading cache failed: %v\n", err)
			} else if ok {
				return writePlainText(cmd.OutOrStdout(), cachedText)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), requestTimeout)
			defer cancel()

			client := youtube.NewClient(nil)
			plainText, err := client.FetchPlainText(ctx, videoID)
			if err != nil {
				return err
			}

			if err := store.Save(videoID, plainText); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: writing cache failed: %v\n", err)
			}

			return writePlainText(cmd.OutOrStdout(), plainText)
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	return cmd
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
