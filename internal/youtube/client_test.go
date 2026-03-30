package youtube

import "testing"

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "watch URL with extra params",
			input: "https://www.youtube.com/watch?v=tG1CSRaJhKQ&list=WL&t=957s",
			want:  "tG1CSRaJhKQ",
		},
		{
			name:  "short URL",
			input: "https://youtu.be/tG1CSRaJhKQ?t=957",
			want:  "tG1CSRaJhKQ",
		},
		{
			name:  "shorts URL",
			input: "https://www.youtube.com/shorts/tG1CSRaJhKQ",
			want:  "tG1CSRaJhKQ",
		},
		{
			name:  "embed URL",
			input: "https://www.youtube.com/embed/tG1CSRaJhKQ",
			want:  "tG1CSRaJhKQ",
		},
		{
			name:  "raw video ID",
			input: "tG1CSRaJhKQ",
			want:  "tG1CSRaJhKQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVideoID(tt.input)
			if err != nil {
				t.Fatalf("ParseVideoID(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseVideoID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseVideoIDRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "missing video id", input: "https://www.youtube.com/watch?list=WL"},
		{name: "wrong host", input: "https://example.com/watch?v=tG1CSRaJhKQ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVideoID(tt.input)
			if err == nil {
				t.Fatalf("ParseVideoID(%q) error = nil, want error", tt.input)
			}
		})
	}
}

func TestChooseCaptionTrackPrefersManualTracks(t *testing.T) {
	tracks := []captionTrack{
		{BaseURL: "https://example.com/asr", Kind: "asr", LanguageCode: "en"},
		{BaseURL: "https://example.com/manual", LanguageCode: "en"},
	}

	got, err := chooseCaptionTrack(tracks)
	if err != nil {
		t.Fatalf("chooseCaptionTrack() error = %v", err)
	}

	if got.BaseURL != "https://example.com/manual" {
		t.Fatalf("chooseCaptionTrack() base URL = %q, want %q", got.BaseURL, "https://example.com/manual")
	}
}

func TestParseTranscriptPlainText(t *testing.T) {
	transcript := []byte(`<transcript>
		<text start="0" dur="0.5">Hello &amp; welcome</text>
		<text start="0.6" dur="0.5">This is &lt;i&gt;formatted&lt;/i&gt; text</text>
		<text start="2.2" dur="0.5">&amp;gt;&amp;gt; Speaker line one.</text>
		<text start="2.8" dur="0.4">Still the same speaker.</text>
		<text start="3.2" dur="0.4">[applause]</text>
		<text start="3.6" dur="0.4">[cheering]</text>
		<text start="4.0" dur="0.4">Back to normal text.</text>
	</transcript>`)

	got, err := parseTranscriptPlainText(transcript)
	if err != nil {
		t.Fatalf("parseTranscriptPlainText() error = %v", err)
	}

	want := "Hello & welcome This is formatted text\n" +
		">> Speaker line one. Still the same speaker.\n" +
		"[applause]\n[cheering]\n" +
		"Back to normal text."
	if got != want {
		t.Fatalf("parseTranscriptPlainText() = %q, want %q", got, want)
	}
}

func TestFormatCaptionLines(t *testing.T) {
	lines := []string{
		">> I thought you were going to make me grab",
		"one of the chairs, too, so I was",
		"preparing. Um,",
		"[applause]",
	}

	got := formatCaptionLines(lines, 60)
	want := ">> I thought you were going to make me grab one of the\n" +
		"chairs, too, so I was preparing. Um,\n" +
		"[applause]"
	if got != want {
		t.Fatalf("formatCaptionLines() = %q, want %q", got, want)
	}
}

func TestWrapLine(t *testing.T) {
	line := ">> I thought you were going to make me grab one of the chairs, too, so I was preparing."

	got := wrapLine(line, 40)
	want := ">> I thought you were going to make me\ngrab one of the chairs, too, so I was\npreparing."
	if got != want {
		t.Fatalf("wrapLine() = %q, want %q", got, want)
	}
}

func TestExtractVideoTitle(t *testing.T) {
	tests := []struct {
		name      string
		watchBody string
		want      string
	}{
		{
			name:      "og title",
			watchBody: `<html><head><meta property="og:title" content="Talk &amp; Demo"></head></html>`,
			want:      "Talk & Demo",
		},
		{
			name:      "title tag fallback",
			watchBody: `<html><head><title>My Video - YouTube</title></head></html>`,
			want:      "My Video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractVideoTitle([]byte(tt.watchBody))
			if err != nil {
				t.Fatalf("extractVideoTitle() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("extractVideoTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPlainTextOutput(t *testing.T) {
	got := formatPlainTextOutput("Video Title", "line one\nline two")
	want := "Video Title\n---\nline one\nline two"
	if got != want {
		t.Fatalf("formatPlainTextOutput() = %q, want %q", got, want)
	}
}

func TestTranscriptURLRemovesFmtParameter(t *testing.T) {
	got, err := transcriptURL("https://example.com/api?lang=en&fmt=srv3&foo=bar")
	if err != nil {
		t.Fatalf("transcriptURL() error = %v", err)
	}

	want := "https://example.com/api?foo=bar&lang=en"
	if got != want {
		t.Fatalf("transcriptURL() = %q, want %q", got, want)
	}
}
