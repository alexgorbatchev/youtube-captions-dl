package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRunHelpFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"--help"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run(--help) exit code = %d, want 0", exitCode)
	}

	if stderr.Len() != 0 {
		t.Fatalf("run(--help) stderr = %q, want empty", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Download YouTube captions and print plain text to stdout.") {
		t.Fatalf("run(--help) output missing description: %q", output)
	}
	if !strings.Contains(output, "usage:") {
		t.Fatalf("run(--help) output missing usage header: %q", output)
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
	if !strings.Contains(output, "Download YouTube captions and print plain text to stdout.") {
		t.Fatalf("run(nil) stderr missing description: %q", output)
	}
	if !strings.Contains(output, "usage:") {
		t.Fatalf("run(nil) stderr missing usage header: %q", output)
	}
}

func TestRunVersionFlags(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{name: "double dash", arg: "--version"},
		{name: "single dash", arg: "-version"},
		{name: "command", arg: "version"},
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

			want := fmt.Sprintf("%s %s", programName(), version)
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("run(%q) stdout = %q, want substring %q", tt.arg, stdout.String(), want)
			}
		})
	}
}
