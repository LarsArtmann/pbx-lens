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

	index, err := search.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open search index: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	views := make([]calls.CallView, len(observations))
	for i, obs := range observations {
		views[i] = calls.ViewFrom(obs, calls.Enrichment{})
	}

	if err := index.Rebuild(context.Background(), views); err != nil {
		t.Fatalf("rebuild search index: %v", err)
	}

	return index
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
