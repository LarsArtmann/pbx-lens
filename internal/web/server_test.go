package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

func webObservation(nativeID string, started time.Time) calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    started,
		DurationSecs: 42,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
		Transcript: []calls.TranscriptTurn{
			{Role: "user", Text: "hello there"},
			{Role: "assistant", Text: "how can I help"},
		},
	}
}

// newTestServer builds a registered mux over a fresh in-memory index and
// ingests the given observations (drained, so projections are readable).
func newTestServer(t *testing.T, observations ...calls.Observation) *http.ServeMux {
	t.Helper()

	return newTestServerWith(t, Options{}, observations...)
}

func getBody(t *testing.T, mux *http.ServeMux, path string) (int, string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	response := recorder.Result()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return response.StatusCode, string(body)
}

func TestListEmptyExplainsHowToFill(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	if !strings.Contains(body, "No calls match") {
		t.Error("empty list should show the empty state")
	}

	if !strings.Contains(body, "ingest-cdr") {
		t.Error("empty state should name the ingest commands")
	}
}

func TestListShowsIngestedCalls(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-a", started),
		webObservation("conv-b", started.Add(time.Minute)),
	)

	status, body := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	for _, want := range []string{"conv-a", "conv-b", "+17287289311", "+15555550123"} {
		if !strings.Contains(body, want) {
			t.Errorf("list page missing %q", want)
		}
	}
}

func TestListFilterRejectsGarbage(t *testing.T) {
	mux := newTestServer(t)

	tests := []struct {
		path string
	}{
		{path: "/?direction=sideways"},
		{path: "/?ai=maybe"},
		{path: "/?from=not-a-date"},
	}

	for _, test := range tests {
		status, body := getBody(t, mux, test.path)
		if status != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want 400 (body: %s)", test.path, status, body)
		}
	}
}

func TestListFilterHonorsDirection(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	outbound := webObservation("conv-out", started)

	inbound := webObservation("conv-in", started.Add(time.Minute))
	inbound.Direction = calls.DirectionInbound

	mux := newTestServer(t, outbound, inbound)

	status, body := getBody(t, mux, "/?direction=inbound")
	if status != http.StatusOK {
		t.Fatalf("GET /?direction=inbound status = %d, want 200", status)
	}

	if !strings.Contains(body, "conv-in") {
		t.Error("inbound filter should show the inbound call")
	}

	if strings.Contains(body, "conv-out") {
		t.Error("inbound filter should hide the outbound call")
	}
}

func TestDetailShowsFactsAndTranscript(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServer(t, webObservation("conv-a", started))

	status, body := getBody(t, mux, "/calls/telnyx-conversation/conv-a")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	for _, want := range []string{"conv-a", "+17287289311", "+15555550123", "Transcript", "hello there"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}
}

func TestDetailNotFound(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/calls/telnyx-conversation/missing")
	if status != http.StatusNotFound {
		t.Fatalf("GET missing detail status = %d, want 404", status)
	}

	if !strings.Contains(body, "Not found") {
		t.Error("404 page should say Not found")
	}
}

func TestStylesUnbuiltReturns404(t *testing.T) {
	// styles.css is a build artifact (tailwind), absent in a clean checkout.
	status, body := getBody(t, newTestServer(t), "/app.css")
	if status != http.StatusNotFound {
		t.Fatalf("GET /app.css status = %d, want 404 (body: %s)", status, body)
	}
}

func TestListToDateIncludesThatDay(t *testing.T) {
	// Bare dates mean the operator's day (Europe/Berlin, R2): ?to=2026-10-09
	// covers up to Berlin midnight. 20:30 UTC is 22:30 CEST — still the 9th
	// in Berlin; 22:30 UTC is already 00:30 CEST on the 10th and must NOT
	// appear (the old UTC-day read silently lumped it in).
	mux := newTestServer(t,
		webObservation("conv-evening", time.Date(2026, 10, 9, 20, 30, 0, 0, time.UTC)),
		webObservation("conv-past-midnight", time.Date(2026, 10, 9, 22, 30, 0, 0, time.UTC)),
	)

	status, body := getBody(t, mux, "/?to=2026-10-09")
	if status != http.StatusOK {
		t.Fatalf("GET /?to=2026-10-09 status = %d, want 200", status)
	}

	if !strings.Contains(body, "conv-evening") {
		t.Error("?to=<date> should include calls later that Berlin day")
	}

	if strings.Contains(body, "conv-past-midnight") {
		t.Error("?to=<date> must exclude calls that crossed Berlin midnight")
	}

	status, body = getBody(t, mux, "/?to=2026-10-08")
	if status != http.StatusOK {
		t.Fatalf("GET /?to=2026-10-08 status = %d, want 200", status)
	}

	if strings.Contains(body, "conv-evening") {
		t.Error("?to= before the calls should exclude them")
	}
}

func TestStalenessBannerOnlyOnUnfilteredPage(t *testing.T) {
	// 2026-10-09 is far past the 6h threshold; the banner belongs on the
	// unfiltered first page, never on a deliberate filtered slice.
	mux := newTestServer(t, webObservation("conv-old", time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)))

	status, body := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	if !strings.Contains(body, "Index may be stale") {
		t.Error("unfiltered page should carry the staleness banner")
	}

	status, body = getBody(t, mux, "/?direction=outbound")
	if status != http.StatusOK {
		t.Fatalf("GET filtered status = %d, want 200", status)
	}

	if strings.Contains(body, "Index may be stale") {
		t.Error("filtered page should not carry the staleness banner")
	}
}

func TestEveryRegisteredRouteAnswers(t *testing.T) {
	mux := newTestServer(t, webObservation("conv-a", time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)))

	routes := []string{
		"/",
		"/dashboard",
		"/search",
		"/api/freshness",
		"/api/export.jsonl",
		"/api/export.csv",
		"/calls/telnyx-conversation/conv-a",
		"/calls/telnyx-conversation/conv-a/recording",
		"/app.css",
	}

	for _, path := range routes {
		status, body := getBody(t, mux, path)

		// The route table must stay wired: a missing registration answers
		// with ServeMux's bare "404 page not found", and resource-level
		// misses (an unenriched recording) answer with a real message.
		if status == http.StatusNotImplemented {
			t.Errorf("GET %s = 501", path)
		}

		if status == http.StatusNotFound && strings.TrimSpace(body) == "404 page not found" {
			t.Errorf("GET %s hit no registered handler", path)
		}

		if status == http.StatusMethodNotAllowed {
			t.Errorf("GET %s = 405 — route registered under the wrong method", path)
		}
	}
}

// failingWriter answers every write with an error, emulating a client that
// vanished mid-response, and counts the attempts.
type failingWriter struct {
	header http.Header
	writes int
}

var errClientVanished = errors.New("client vanished")

func (f *failingWriter) Header() http.Header { return f.header }

func (f *failingWriter) Write(p []byte) (int, error) {
	f.writes++

	return 0, errClientVanished
}

func (f *failingWriter) WriteHeader(int) {}

func TestFreshnessMidWriteFailureLogsInsteadOfCorrupting(t *testing.T) {
	var logged []string

	mux := newTestServerWith(t, Options{
		Log: func(msg string, args ...any) { logged = append(logged, msg) },
	})

	writer := &failingWriter{header: http.Header{}}
	request := httptest.NewRequest(http.MethodGet, "/api/freshness", nil)
	mux.ServeHTTP(writer, request)

	if len(logged) == 0 {
		t.Fatal("a mid-write encode failure must reach the log sink")
	}

	if writer.writes != 1 {
		t.Errorf(
			"write attempts = %d, want 1 — an error body must not be appended after the response started",
			writer.writes,
		)
	}
}

// TestLiveServerEndToEnd is the checked-in version of the manual UI probe
// the 2026-10-11 dispatch ran by hand: a real listener (httptest.NewServer
// over the registered mux, real HTTP over the loopback stack) serving a
// real sqlite index, probed on all four surfaces. Index wiring rides the
// shared newTestServerWithSearch helper (the one-flow pattern).
func TestLiveServerEndToEnd(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	seeded := []calls.Observation{
		webObservation("conv-e2e", started),
		webObservation("conv-e2e-b", started.Add(time.Minute)),
	}

	mux := newTestServerWithSearch(t, seeded...)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	checks := []struct {
		path     string
		wantText []string
	}{
		{"/", []string{"conv-e2e", "+17287289311"}},
		{"/dashboard", []string{"Call quality (MOS)"}},
		{"/search?q=17287289311", []string{"conv-e2e"}},
		{"/api/freshness", []string{`"calls_indexed":2`}},
	}

	for _, c := range checks {
		resp, err := http.Get(server.URL + c.path)
		if err != nil {
			t.Fatalf("GET %s: %v", c.path, err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200 (body: %.200s)", c.path, resp.StatusCode, body)

			continue
		}

		for _, want := range c.wantText {
			if !strings.Contains(string(body), want) {
				t.Errorf("GET %s body missing %q", c.path, want)
			}
		}
	}
}

func TestOpsEndpointServesCounters(t *testing.T) {
	// R15: /api/ops answers with the injected counters; without a poller
	// (nil Ops) it explains itself with a 503 instead of lying empty.
	mux := newTestServerWith(t, Options{
		Ops: func() any {
			return map[string]any{
				"sources": map[string]any{
					"webhook-log": map[string]any{"offered": 2, "recorded": 2, "rejected": []string{}},
				},
				"last_pass": "2026-10-12T04:00:00Z",
			}
		},
	})

	status, body := getBody(t, mux, "/api/ops")
	if status != 200 {
		t.Fatalf("GET /api/ops status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{`"webhook-log"`, `"offered":2`, `"last_pass":"2026-10-12T04:00:00Z"`} {
		if !strings.Contains(body, want) {
			t.Errorf("/api/ops missing %q (body: %s)", want, body)
		}
	}

	status, body = getBody(t, newTestServer(t), "/api/ops")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/ops without counters status = %d, want 503", status)
	}

	if !strings.Contains(body, "poller") {
		t.Error("the 503 should name the poller as the fix")
	}
}

func TestListKeysetWalksBeyondPageOne(t *testing.T) {
	// m27.7: the older-calls cursor must walk the whole index — 55 calls
	// (pageSize+5) arrive complete across two pages, newest first, with no
	// duplicates and a clean stop at the end.
	started := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	observations := make([]calls.Observation, 0, 55)

	for i := range 55 {
		observations = append(observations, webObservation(
			fmt.Sprintf("conv-%02d", i),
			started.Add(-time.Duration(i)*time.Minute),
		))
	}

	mux := newTestServer(t, observations...)

	seen := make(map[string]bool, 55)
	path := "/"

	for page := 0; ; page++ {
		status, body := getBody(t, mux, path)
		if status != 200 {
			t.Fatalf("GET %s status = %d, want 200", path, status)
		}

		for i := range 55 {
			id := fmt.Sprintf("conv-%02d", i)
			if strings.Contains(body, id) {
				if seen[id] {
					t.Errorf("call %s rendered on two pages", id)
				}

				seen[id] = true
			}
		}

		hrefStart := strings.Index(body, `href="/?before=`)
		if hrefStart < 0 {
			break
		}

		rest := body[hrefStart+len(`href="`):]
		end := strings.Index(rest, `"`)
		if end < 0 {
			t.Fatalf("malformed cursor href on page %d", page)
		}

		if page > 5 {
			t.Fatal("keyset walk exceeded two pages: the cursor is looping")
		}

		path = rest[:end]
	}

	if len(seen) != 55 {
		t.Errorf("keyset walk delivered %d of 55 calls", len(seen))
	}
}
