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

var auditMP3 = []byte("ID3-fake-mp3-bytes-0123456789")

// fakeTelnyxState records what the fake API saw, for the audit assertions.
type fakeTelnyxState struct {
	authOnMetadata string
	rangeHeader    string
}

// fakeTelnyx stands in for the Telnyx v2 API: recording metadata at
// /v2/recordings/{id}, the signed WAV at /v2/audio.wav (Range-aware).
// rec-mp3 mirrors the live-verified AI-assistant shape (empty wav, mp3
// only) and serves its audio WITHOUT a Content-Type so the proxy's
// rendition default is exercised.
func fakeTelnyx(t *testing.T, state *fakeTelnyxState) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/recordings/rec-1":
			state.authOnMetadata = r.Header.Get("Authorization")
			wavURL := "http://" + r.Host + "/v2/audio.wav"

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":{"id":"rec-1","download_urls":{"wav":%q}}}`, wavURL)
		case "/v2/recordings/rec-mp3":
			state.authOnMetadata = r.Header.Get("Authorization")
			mp3URL := "http://" + r.Host + "/v2/audio.mp3"

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":{"id":"rec-mp3","download_urls":{"wav":"","mp3":%q}}}`, mp3URL)
		case "/v2/recordings/rec-500":
			state.authOnMetadata = r.Header.Get("Authorization")

			w.WriteHeader(http.StatusInternalServerError)
		case "/v2/recordings/rec-expired":
			state.authOnMetadata = r.Header.Get("Authorization")
			expiredURL := "http://" + r.Host + "/v2/audio-expired.wav"

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":{"id":"rec-expired","download_urls":{"wav":%q}}}`, expiredURL)
		case "/v2/audio-expired.wav":
			// The signed URL expired between metadata fetch and audio GET —
			// the live-API failure class the negative path must answer
			// honestly instead of hanging.
			w.WriteHeader(http.StatusForbidden)
		case "/v2/recordings/rec-none":
			state.authOnMetadata = r.Header.Get("Authorization")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"rec-none","download_urls":{"wav":"","mp3":""}}}`))
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
		case "/v2/audio.mp3":
			_, _ = w.Write(auditMP3)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(server.Close)

	return server
}

func telnyxRecordingObservation() calls.Observation {
	return telnyxRecordingObservationWithRef("rec-1")
}

// telnyxRecordingObservationWithRef builds a telnyx-conversation view
// pointing at one of the fake's recording ids; every test registers it in
// its own mux, so the native id stays the canonical conv-rec.
func telnyxRecordingObservationWithRef(recordingRef string) calls.Observation {
	return calls.Observation{
		NativeID:     "conv-rec",
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		DurationSecs: 5,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
		RecordingRef: recordingRef,
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

// TestRecordingProxyFallsBackToMP3 is the live-verified shape (2026-10-12):
// AI-assistant recordings carry an empty download_urls.wav and MP3-only
// audio, so the proxy must stream the MP3 rendition instead of 404ing.
func TestRecordingProxyFallsBackToMP3(t *testing.T) {
	api := fakeTelnyx(t, &fakeTelnyxState{})

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
	}, telnyxRecordingObservationWithRef("rec-mp3"))

	status, body, header := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusOK {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, body)
	}

	if body != string(auditMP3) {
		t.Errorf("body = %q, want the proxied MP3 bytes", body)
	}

	if got := header.Get("Content-Type"); got != "audio/mpeg" {
		t.Errorf("Content-Type = %q, want the audio/mpeg rendition default", got)
	}

	if got := header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store (signed URLs expire)", got)
	}
}

func TestRecordingProxyWithoutAnyRenditionAnswers404(t *testing.T) {
	api := fakeTelnyx(t, &fakeTelnyxState{})

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
	}, telnyxRecordingObservationWithRef("rec-none"))

	status, body, _ := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusNotFound {
		t.Fatalf("recording status = %d, want 404 (body: %s)", status, body)
	}

	if !strings.Contains(body, "WAV or MP3") {
		t.Errorf("404 body should name both renditions: %q", body)
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

// TestRecordingNegativePathsAnswerHonestly pins the failure classes of
// the proxy (2026-10-17 quality pass): an upstream metadata 500 and a
// signed-URL-expired audio 403 must both surface as an honest 502 with a
// descriptive body — never a hang, never a leaked upstream URL.
func TestRecordingNegativePathsAnswerHonestly(t *testing.T) {
	state := &fakeTelnyxState{}
	api := fakeTelnyx(t, state)

	cases := []struct {
		name     string
		ref      string
		wantCode int
		wantBody string
	}{
		{"metadata 500", "rec-500", http.StatusBadGateway, "fetching the recording metadata failed"},
		{"audio 403 (signed-URL expiry)", "rec-expired", http.StatusBadGateway, "recording source answered 403"},
	}

	for _, tc := range cases {
		mux := newTestServerWith(t, Options{
			Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
		}, telnyxRecordingObservationWithRef(tc.ref))

		status, body, header := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
		if status != tc.wantCode {
			t.Errorf("%s: status = %d, want %d (body: %s)", tc.name, status, tc.wantCode, body)
		}

		if !strings.Contains(body, tc.wantBody) {
			t.Errorf("%s: body %q does not name the failure (%q)", tc.name, body, tc.wantBody)
		}

		if strings.Contains(body, api.URL) {
			t.Errorf("%s: body leaks the upstream URL: %q", tc.name, body)
		}

		if ct := header.Get("Content-Type"); strings.HasPrefix(ct, "audio/") {
			t.Errorf("%s: error answered as %q, want a text error", tc.name, ct)
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
			// Block until the client's UpstreamTimeout cancels the request:
			// headers never arrive inside the budget, so only the audio hop
			// can trip the 502, while the metadata fetch (instant handler)
			// still succeeds. No wall-clock sleep, so the test cannot race a
			// loaded machine.
			<-r.Context().Done()

			_, _ = w.Write(auditWav)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{
			Telnyx:          telnyx.NewClient(auditAPIKey, api.URL+"/v2"),
			UpstreamTimeout: 50 * time.Millisecond,
		},
	}, telnyxRecordingObservation())

	status, body, _ := getWithHeaders(t, mux, "/calls/telnyx-conversation/conv-rec/recording")
	if status != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body: %s)", status, body)
	}
}

// TestRecordingProxyTimingHopBudget pins the two-hop Server-Timing contract
// docs/http-stack.md promises: through the real middleware wrapper, a
// successful WAV proxy names both upstream hops (metadata, headers) in the
// Server-Timing header — both recorded pre-first-write, per the doctrine —
// and names NO body metric, because the unbounded stream itself is never
// timed.
func TestRecordingProxyTimingHopBudget(t *testing.T) {
	api := fakeTelnyx(t, &fakeTelnyxState{})

	mux := newTestServerWith(t, Options{
		Recordings: RecordingOptions{Telnyx: telnyx.NewClient(auditAPIKey, api.URL+"/v2")},
	}, telnyxRecordingObservation())

	timed := ServerTimingMiddleware()(mux)

	rec := httptest.NewRecorder()
	timed.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/calls/telnyx-conversation/conv-rec/recording", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("recording status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	if rec.Body.String() != string(auditWav) {
		t.Fatalf("body = %q, want the proxied WAV bytes", rec.Body.String())
	}

	timing := rec.Header().Get("Server-Timing")

	for _, hop := range []string{"metadata", "headers"} {
		if !strings.Contains(timing, hop) {
			t.Errorf("Server-Timing = %q, want the %q hop", timing, hop)
		}
	}

	if strings.Contains(timing, "body") {
		t.Errorf("Server-Timing = %q — the unbounded body must never be timed", timing)
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
