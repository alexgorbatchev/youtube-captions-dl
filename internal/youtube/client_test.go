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
		<text start="0" dur="1.2">Hello &amp; welcome</text>
		<text start="1.2" dur="1.1">This is &lt;i&gt;formatted&lt;/i&gt; text</text>
		<text start="2.3" dur="0.8">   </text>
	</transcript>`)

	got, err := parseTranscriptPlainText(transcript)
	if err != nil {
		t.Fatalf("parseTranscriptPlainText() error = %v", err)
	}

	want := "Hello & welcome\nThis is formatted text"
	if got != want {
		t.Fatalf("parseTranscriptPlainText() = %q, want %q", got, want)
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
