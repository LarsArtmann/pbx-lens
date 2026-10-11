package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

const auditAPIKey = "KEYsuper-secret-never-in-browser"

var auditWav = []byte("RIFF-fake-wav-bytes-0123456789")

// fakeTelnyxState records what the fake API saw, for the audit assertions.
type fakeTelnyxState struct {
	authOnMetadata string
	rangeHeader    string
}

// fakeTelnyx stands in for the Telnyx v2 API: recording metadata at
// /v2/recordings/{id}, the signed WAV at /v2/audio.wav (Range-aware).
func fakeTelnyx(t *testing.T, state *fakeTelnyxState) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/recordings/rec-1":
			state.authOnMetadata = r.Header.Get("Authorization")
			wavURL := "http://" + r.Host + "/v2/audio.wav"

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":{"id":"rec-1","download_urls":{"wav":%q}}}`, wavURL)
		case "/v2/audio.wav":
			state.rangeHeader = r.Header.Get("Range")
			w.Header().Set("Content-Type", "audio/wav")

			if state.rangeHeader != "" {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-1/%d", len(auditWav)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(auditWav[:2])

				return
			}

			_, _ = w.Write(auditWav)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(server.Close)

	return server
}

func telnyxRecordingObservation() calls.Observation {
	return calls.Observation{
		NativeID:     "conv-rec",
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		DurationSecs: 5,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
		RecordingRef: "rec-1",
	}
}

// TestRecordingProxyStreamsTelnyxWav is the B2 audit: the WAV streams
// through the server, the API key rides only the server-to-Telnyx hop, and
// neither the key nor the signed URL ever appears in the browser response.
func TestRecordingProxyStreamsTelnyxWav(t *testing.T) {
	state := &fakeTelnyxState{}
	api := fakeTelnyx(t, state)

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
	}, telnyxRecordingObservation())

	status, body, header := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusOK {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, body)
	}

	if body != string(auditWav) {
		t.Errorf("body = %q, want the proxied WAV bytes", body)
	}

	if got := header.Get("Content-Type"); got != "audio/wav" {
		t.Errorf("Content-Type = %q, want audio/wav", got)
	}

	if got := header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store (signed URLs expire)", got)
	}

	if loc := header.Get("Location"); loc != "" {
		t.Errorf("Location = %q — a redirect would leak the signed URL to the browser", loc)
	}

	// The audit proper: key and upstream URL must not reach the response.
	for key, values := range header {
		for _, value := range values {
			if strings.Contains(value, auditAPIKey) || strings.Contains(value, "/v2/audio.wav") {
				t.Errorf("response header %s leaks the upstream: %q", key, value)
			}
		}
	}

	if strings.Contains(body, auditAPIKey) {
		t.Error("response body leaks the API key")
	}

	if state.authOnMetadata != "Bearer "+auditAPIKey {
		t.Errorf("upstream Authorization = %q, want the server-side Bearer key", state.authOnMetadata)
	}
}

func TestRecordingProxyPassesRangeThrough(t *testing.T) {
	state := &fakeTelnyxState{}
	api := fakeTelnyx(t, state)

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
	}, telnyxRecordingObservation())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/calls/telnyx-conversation/conv-rec/recording", nil)
	request.Header.Set("Range", "bytes=0-1")
	mux.ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusPartialContent {
		t.Fatalf("range status = %d, want 206", response.StatusCode)
	}

	if state.rangeHeader != "bytes=0-1" {
		t.Errorf("upstream Range = %q, want pass-through of bytes=0-1", state.rangeHeader)
	}

	if got := response.Header.Get("Content-Range"); got == "" {
		t.Error("Content-Range missing on the 206 response")
	}
}

func TestRecordingProxyWithoutKeyExplainsItself(t *testing.T) {
	mux := newTestServer(t, telnyxRecordingObservation())

	status, body, _ := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", status)
	}

	if !strings.Contains(body, "telnyx-key-file") {
		t.Errorf("503 body should name the fix flag: %q", body)
	}
}

func TestRecordingUnknownCallAndNoRef(t *testing.T) {
	started := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

	withRef := telnyxRecordingObservation()
	withoutRef := webObservation("conv-noref", started)
	withoutRef.Kind = calls.SourceWebhookLog

	mux := newTestServer(t, withRef, withoutRef)

	tests := []struct {
		path string
	}{
		{path: "/calls/telnyx-conversation/missing/recording"},
		{path: "/calls/webhook-log/conv-noref/recording"},
	}

	for _, test := range tests {
		status, _, _ := getWithHeaders(t, mux, test.path)
		if status != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", test.path, status)
		}
	}
}

func TestRecordingLocalServesFileWithRange(t *testing.T) {
	dir := t.TempDir()

	observation := calls.Observation{
		NativeID:   "uuid-1",
		Kind:       calls.SourceFreeSwitchCDR,
		Direction:  calls.DirectionOutbound,
		StartedAt:  time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		FromNumber: "+17287289311",
		ToNumber:   "+15555550123",
	}

	wav := []byte("RIFF-local-wav-contents")
	if err := os.WriteFile(filepath.Join(dir, "uuid-1_+15555550123.wav"), wav, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	mux := newTestServerWith(t, Options{Recordings: RecordingOptions{RecordingsDir: dir}}, observation)

	status, body, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/uuid-1/recording")
	if status != http.StatusOK {
		t.Fatalf("local recording status = %d, want 200", status)
	}

	if body != string(wav) {
		t.Errorf("body = %q, want the local file bytes", body)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/calls/freeswitch-cdr/uuid-1/recording", nil)
	request.Header.Set("Range", "bytes=0-1")
	mux.ServeHTTP(recorder, request)

	if recorder.Result().StatusCode != http.StatusPartialContent {
		t.Errorf("range status = %d, want 206 (http.ServeFile Range support)", recorder.Result().StatusCode)
	}
}

func TestRecordingLocalMissingFileIs404(t *testing.T) {
	observation := calls.Observation{
		NativeID:   "uuid-2",
		Kind:       calls.SourceFreeSwitchCDR,
		Direction:  calls.DirectionOutbound,
		StartedAt:  time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		FromNumber: "+17287289311",
		ToNumber:   "+15555550123",
	}

	mux := newTestServerWith(t, Options{Recordings: RecordingOptions{RecordingsDir: t.TempDir()}}, observation)

	status, _, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/uuid-2/recording")
	if status != http.StatusNotFound {
		t.Fatalf("missing local file status = %d, want 404", status)
	}
}

// newTestServerWith is newTestServer with explicit options.
func newTestServerWith(t *testing.T, opts Options, observations ...calls.Observation) *http.ServeMux {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	// Real sqlite, not the memory engine: the memory engine's closure
	// fallback resolves filter columns by Go field name, so JSON-tag columns
	// (started_at, is_ai) misfilter there — production always serves from
	// sqlite, and the tests must share those semantics.
	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "web-tests.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	if len(observations) > 0 {
		if stats := cli.Ingest(ctx, sys, observations); len(stats.Rejected) > 0 {
			t.Fatalf("ingest rejected: %v", stats.Rejected)
		}

		if err := cli.DrainProjections(ctx, sys); err != nil {
			t.Fatalf("drain: %v", err)
		}
	}

	mux := http.NewServeMux()
	Register(mux, sys, opts)

	return mux
}

func getWithHeaders(t *testing.T, mux *http.ServeMux, path string) (int, string, http.Header) {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	response := recorder.Result()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return response.StatusCode, string(body), response.Header
}

// TestRecordingUpstreamTimeoutBoundsMetadata proves the UpstreamTimeout
// knob on the metadata hop: an upstream that never answers inside the
// deadline fails fast with 504 instead of pinning the browser request.
func TestRecordingUpstreamTimeoutBoundsMetadata(t *testing.T) {
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(blocked.Close)

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{
			Telnyx:          telnyx.NewClient(auditAPIKey, blocked.URL+"/v2"),
			UpstreamTimeout: time.Millisecond,
		},
	}, telnyxRecordingObservation())

	status, body, _ := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504 (body: %s)", status, body)
	}
}

// TestRecordingUpstreamTimeoutBoundsWavHeaders proves the same knob on the
// audio hop: WAV response headers must arrive within the budget, so a
// stalled upstream becomes a 502 rather than a hung stream.
func TestRecordingUpstreamTimeoutBoundsWavHeaders(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/recordings/rec-1":
			fmt.Fprintf(w, `{"data":{"id":"rec-1","download_urls":{"wav":"http://%s/v2/audio.wav"}}}`, r.Host)
		case "/v2/audio.wav":
			time.Sleep(50 * time.Millisecond)

			_, _ = w.Write(auditWav)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{
			Telnyx:          telnyx.NewClient(auditAPIKey, api.URL+"/v2"),
			UpstreamTimeout: time.Millisecond,
		},
	}, telnyxRecordingObservation())

	status, body, _ := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body: %s)", status, body)
	}
}

func TestResolveUpstreamTimeout(t *testing.T) {
	tests := []struct {
		name       string
		configured time.Duration
		want       time.Duration
	}{
		{"zero selects the default", 0, DefaultUpstreamTimeout},
		{"negative selects the default", -time.Second, DefaultUpstreamTimeout},
		{"explicit wins", 7 * time.Second, 7 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveUpstreamTimeout(tt.configured); got != tt.want {
				t.Errorf("resolveUpstreamTimeout(%v) = %v, want %v", tt.configured, got, tt.want)
			}
		})
	}
}
