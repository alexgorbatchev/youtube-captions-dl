package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	watchURLFormat     = "https://www.youtube.com/watch?v=%s"
	innertubeURLFormat = "https://www.youtube.com/youtubei/v1/player?key=%s"
)

var (
	videoIDPattern      = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	consentURLPattern   = regexp.MustCompile(`https://consent\.youtube\.com/s`)
	consentValuePattern = regexp.MustCompile(`name="v" value="(.*?)"`)
	apiKeyPattern       = regexp.MustCompile(`"INNERTUBE_API_KEY":\s*"([A-Za-z0-9_-]+)"`)
	stripTagsPattern    = regexp.MustCompile(`(?i)<[^>]*>`)
)

type Client struct {
	httpClient *http.Client
}

type playerResponse struct {
	Captions captionsContainer `json:"captions"`
}

type captionsContainer struct {
	TrackList *captionTrackList `json:"playerCaptionsTracklistRenderer"`
}

type captionTrackList struct {
	CaptionTracks []captionTrack `json:"captionTracks"`
}

type captionTrack struct {
	BaseURL      string    `json:"baseUrl"`
	LanguageCode string    `json:"languageCode"`
	Kind         string    `json:"kind,omitempty"`
	Name         trackName `json:"name"`
}

type trackName struct {
	SimpleText string `json:"simpleText"`
}

type transcriptDocument struct {
	Entries []transcriptEntry `xml:"text"`
}

type transcriptEntry struct {
	Text string `xml:",chardata"`
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Client{httpClient: httpClient}
}

func ParseVideoID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("empty input")
	}

	if videoIDPattern.MatchString(trimmed) {
		return trimmed, nil
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parsing URL %q: %w", trimmed, err)
	}

	host := strings.ToLower(strings.TrimPrefix(parsedURL.Host, "www."))
	path := strings.Trim(parsedURL.EscapedPath(), "/")

	switch host {
	case "youtu.be":
		id := strings.Split(path, "/")[0]
		if videoIDPattern.MatchString(id) {
			return id, nil
		}
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		if parsedURL.Path == "/watch" {
			id := parsedURL.Query().Get("v")
			if videoIDPattern.MatchString(id) {
				return id, nil
			}
		}

		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			switch parts[0] {
			case "shorts", "live", "embed":
				if videoIDPattern.MatchString(parts[1]) {
					return parts[1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract a YouTube video ID from %q", trimmed)
}

func (c *Client) FetchPlainText(ctx context.Context, videoID string) (string, error) {
	watchBody, consentCookie, err := c.fetchWatchPage(ctx, videoID)
	if err != nil {
		return "", fmt.Errorf("fetching watch page for %s: %w", videoID, err)
	}

	apiKey, err := extractInnertubeAPIKey(watchBody)
	if err != nil {
		return "", fmt.Errorf("extracting Innertube API key for %s: %w", videoID, err)
	}

	trackList, err := c.fetchCaptionTracks(ctx, videoID, apiKey, consentCookie)
	if err != nil {
		return "", fmt.Errorf("fetching caption tracks for %s: %w", videoID, err)
	}

	track, err := chooseCaptionTrack(trackList)
	if err != nil {
		return "", fmt.Errorf("selecting caption track for %s: %w", videoID, err)
	}

	transcriptBody, err := c.fetchTranscript(ctx, track, consentCookie)
	if err != nil {
		return "", fmt.Errorf("fetching transcript for %s: %w", videoID, err)
	}

	plainText, err := parseTranscriptPlainText(transcriptBody)
	if err != nil {
		return "", fmt.Errorf("parsing transcript for %s: %w", videoID, err)
	}

	if plainText == "" {
		return "", fmt.Errorf("captions were available for %s but the transcript body was empty", videoID)
	}

	return plainText, nil
}

func (c *Client) fetchWatchPage(ctx context.Context, videoID string) ([]byte, *http.Cookie, error) {
	watchURL := fmt.Sprintf(watchURLFormat, videoID)

	body, err := c.get(ctx, watchURL, nil)
	if err != nil {
		return nil, nil, err
	}

	if !requiresConsent(body) {
		return body, nil, nil
	}

	consentCookie, err := c.createConsentCookie(ctx, watchURL)
	if err != nil {
		return nil, nil, err
	}

	body, err = c.get(ctx, watchURL, consentCookie)
	if err != nil {
		return nil, nil, err
	}

	return body, consentCookie, nil
}

func (c *Client) fetchCaptionTracks(ctx context.Context, videoID string, apiKey string, consentCookie *http.Cookie) ([]captionTrack, error) {
	requestURL := fmt.Sprintf(innertubeURLFormat, apiKey)
	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "ANDROID",
				"clientVersion": "20.10.38",
			},
		},
		"videoId": videoID,
	}

	responseBody, err := c.postJSON(ctx, requestURL, payload, consentCookie)
	if err != nil {
		return nil, err
	}

	var response playerResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decoding player response: %w", err)
	}

	if response.Captions.TrackList == nil || len(response.Captions.TrackList.CaptionTracks) == 0 {
		return nil, fmt.Errorf("captions are not available for this video")
	}

	return response.Captions.TrackList.CaptionTracks, nil
}

func chooseCaptionTrack(tracks []captionTrack) (captionTrack, error) {
	for _, track := range tracks {
		if track.BaseURL == "" {
			continue
		}
		if track.Kind == "" {
			return track, nil
		}
	}

	for _, track := range tracks {
		if track.BaseURL != "" {
			return track, nil
		}
	}

	return captionTrack{}, fmt.Errorf("no usable caption tracks found")
}

func (c *Client) fetchTranscript(ctx context.Context, track captionTrack, consentCookie *http.Cookie) ([]byte, error) {
	requestURL, err := transcriptURL(track.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("building transcript URL: %w", err)
	}

	return c.get(ctx, requestURL, consentCookie)
}

func transcriptURL(baseURL string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parsing transcript URL: %w", err)
	}

	query := parsedURL.Query()
	query.Del("fmt")
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func parseTranscriptPlainText(transcriptBody []byte) (string, error) {
	var document transcriptDocument
	if err := xml.Unmarshal(transcriptBody, &document); err != nil {
		return "", fmt.Errorf("decoding transcript XML: %w", err)
	}

	lines := make([]string, 0, len(document.Entries))
	for _, entry := range document.Entries {
		line := cleanCaptionText(entry.Text)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

func cleanCaptionText(raw string) string {
	cleaned := html.UnescapeString(raw)
	cleaned = stripTagsPattern.ReplaceAllString(cleaned, "")
	cleaned = html.UnescapeString(cleaned)
	fields := strings.Fields(cleaned)
	return strings.Join(fields, " ")
}

func extractInnertubeAPIKey(watchBody []byte) (string, error) {
	match := apiKeyPattern.FindSubmatch(watchBody)
	if len(match) != 2 {
		return "", fmt.Errorf("INNERTUBE_API_KEY was not present in the watch page")
	}

	return string(match[1]), nil
}

func requiresConsent(body []byte) bool {
	return consentURLPattern.Match(body)
}

func (c *Client) createConsentCookie(ctx context.Context, watchURL string) (*http.Cookie, error) {
	body, err := c.get(ctx, watchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching consent page: %w", err)
	}

	match := consentValuePattern.FindSubmatch(body)
	if len(match) != 2 {
		return nil, fmt.Errorf("could not find consent token in response")
	}

	return &http.Cookie{
		Name:   "CONSENT",
		Value:  "YES+" + string(match[1]),
		Domain: ".youtube.com",
	}, nil
}

func (c *Client) get(ctx context.Context, requestURL string, cookie *http.Cookie) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}

	setCommonHeaders(req)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	return c.do(req)
}

func (c *Client) postJSON(ctx context.Context, requestURL string, payload any, cookie *http.Cookie) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding JSON payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}

	setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}

	return c.do(req)
}

func (c *Client) do(req *http.Request) (body []byte, err error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing HTTP request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing HTTP response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, req.URL.Host)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response body: %w", err)
	}

	return body, nil
}

func setCommonHeaders(req *http.Request) {
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
}
