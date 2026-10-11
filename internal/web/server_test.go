package web

import (
	"errors"
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
	// The observation starts 22:30 on the 9th; a bare ?to=2026-10-09 must
	// cover the whole day, not just its midnight (the old start-of-day read
	// silently excluded the day the parameter names).
	lateEvening := time.Date(2026, 10, 9, 22, 30, 0, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-morning", time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)),
		webObservation("conv-evening", lateEvening),
	)

	status, body := getBody(t, mux, "/?to=2026-10-09")
	if status != http.StatusOK {
		t.Fatalf("GET /?to=2026-10-09 status = %d, want 200", status)
	}

	if !strings.Contains(body, "conv-evening") {
		t.Error("?to=<date> should include calls later that same day")
	}

	status, body = getBody(t, mux, "/?to=2026-10-08")
	if status != http.StatusOK {
		t.Fatalf("GET /?to=2026-10-08 status = %d, want 200", status)
	}

	if strings.Contains(body, "conv-morning") {
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
