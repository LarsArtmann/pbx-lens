package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, "")
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
	Register(mux, sys, RecordingOptions{})

	return mux
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
