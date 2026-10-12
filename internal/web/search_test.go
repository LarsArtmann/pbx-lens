package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// sqliteSearcher builds a real FTS index over the given observations — the
// same wiring serve uses when -db points at a sqlite file.
func sqliteSearcher(t *testing.T, observations ...calls.Observation) *search.Index {
	t.Helper()

	index, err := search.Open(context.Background(), filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open search index: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	views := make([]calls.CallView, 0, len(observations))
	for _, obs := range observations {
		views = append(views, calls.ViewFrom(obs, calls.Enrichment{}))
	}

	if err := index.Rebuild(context.Background(), views); err != nil {
		t.Fatalf("rebuild search index: %v", err)
	}

	return index
}

// newTestServerWithSearch is newTestServerWith with a real FTS index rebuilt
// over the same observations — the one-flow serve pattern, where the search
// db is a separate file from the server's index db.
func newTestServerWithSearch(t *testing.T, observations ...calls.Observation) *http.ServeMux {
	t.Helper()

	return newTestServerWith(t, Options{Search: sqliteSearcher(t, observations...)}, observations...)
}

func TestSearchAnswersFromSqliteIndex(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	observation := webObservation("conv-a", started)

	mux := newTestServerWith(t,
		Options{Search: sqliteSearcher(t, observation)},
		observation,
	)

	status, body := getBody(t, mux, "/search?q=hello")
	if status != http.StatusOK {
		t.Fatalf("GET /search status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{
		"conv-a",
		">hello</mark>",
		"/calls/telnyx-conversation/conv-a",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("search page missing %q", want)
		}
	}
}

func TestSearchEmptyQueryShowsForm(t *testing.T) {
	status, body := getBody(t, newTestServerWith(t, Options{Search: sqliteSearcher(t)}), "/search")
	if status != http.StatusOK {
		t.Fatalf("GET /search status = %d, want 200", status)
	}

	if !strings.Contains(body, `name="q"`) {
		t.Error("search page should render the query input")
	}

	// Partial numbers match since R6 (digits shadow table): the hint must
	// say so, with the 4-digit minimum spelled out.
	if !strings.Contains(body, "Partial numbers match") {
		t.Error("search page should hint that partial numbers now match")
	}
}

func TestSearchWithoutIndexExplainsItself(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/search?q=hello")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("GET /search without an index status = %d, want 503", status)
	}

	if !strings.Contains(body, "-db") {
		t.Error("503 should name the -db fix")
	}
}

func TestSearchPartialNumberEndToEnd(t *testing.T) {
	// R6 through the UI: a mid-number digit run finds the call whose party
	// number carries it, and the snippet highlights the number itself.
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	observation := webObservation("conv-a", started)
	observation.FromNumber = "+17287289311"

	mux := newTestServerWithSearch(t, observation)

	status, body := getBody(t, mux, "/search?q=8931")
	if status != http.StatusOK {
		t.Fatalf("GET /search?q=8931 status = %d, want 200 (body: %s)", status, body)
	}

	if !strings.Contains(body, "/calls/telnyx-conversation/conv-a") {
		t.Errorf("partial number search must find the call (body: %s)", body)
	}

	if !strings.Contains(body, "+17287289311") {
		t.Error("the matching number should appear highlighted in the snippet")
	}
}
