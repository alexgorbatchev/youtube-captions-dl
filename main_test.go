package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRunHelpFlag(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{name: "long help", arg: "--help"},
		{name: "short help", arg: "-h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run([]string{tt.arg}, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("run(%q) exit code = %d, want 0", tt.arg, exitCode)
			}

			if stderr.Len() != 0 {
				t.Fatalf("run(%q) stderr = %q, want empty", tt.arg, stderr.String())
			}

			output := stdout.String()
			if !strings.Contains(output, "Download YouTube captions and print plain text to stdout.") {
				t.Fatalf("run(%q) output missing description: %q", tt.arg, output)
			}
			if !strings.Contains(output, "Usage:") {
				t.Fatalf("run(%q) output missing Usage header: %q", tt.arg, output)
			}
		})
	}
}

func TestRunWithoutArgumentsPrintsHelpToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("run(nil) exit code = %d, want 2", exitCode)
	}

	if stdout.Len() != 0 {
		t.Fatalf("run(nil) stdout = %q, want empty", stdout.String())
	}

	output := stderr.String()
	// Cobra prints the error, followed by the usage screen on stderr
	if !strings.Contains(output, "Error: accepts 1 arg(s), received 0") {
		t.Fatalf("run(nil) stderr missing error message: %q", output)
	}
	if !strings.Contains(output, "Download YouTube captions and print plain text to stdout.") {
		t.Fatalf("run(nil) stderr missing description: %q", output)
	}
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("run(nil) stderr missing Usage header: %q", output)
	}
}

func TestRunVersionFlags(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{name: "double dash version", arg: "--version"},
		{name: "short version flag", arg: "-v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run([]string{tt.arg}, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("run(%q) exit code = %d, want 0", tt.arg, exitCode)
			}
			if stderr.Len() != 0 {
				t.Fatalf("run(%q) stderr = %q, want empty", tt.arg, stderr.String())
			}

			want := fmt.Sprintf("youtube-captions-dl version %s", version)
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("run(%q) stdout = %q, want substring %q", tt.arg, stdout.String(), want)
			}
		})
	}
}
