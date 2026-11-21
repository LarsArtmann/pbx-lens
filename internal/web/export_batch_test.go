package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestExportBatchSizeIsPerServer pins that the export page size is a
// property of the SERVER, not shared state: a two-row server and a
// default-page server stream the same corpus at the same time, and both
// come out complete and identical. The option replaced a mutable
// package-level default that tests shrank and restored; under -race this
// fails the moment such shared state returns.
func TestExportBatchSizeIsPerServer(t *testing.T) {
	seeded := seedCalls(7, time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC))

	paged := newTestServerWith(t, Options{ExportBatchSize: 2}, seeded...)
	defaults := newTestServerWith(t, Options{}, seeded...)

	export := func(mux *http.ServeMux) string {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/export.csv", nil))

		return recorder.Body.String()
	}

	// Two slots, written by one goroutine each and read only after Wait.
	var artifacts [2]string

	var streams sync.WaitGroup

	for slot, mux := range []*http.ServeMux{paged, defaults} {
		streams.Go(func() {
			artifacts[slot] = export(mux)
		})
	}

	streams.Wait()

	if artifacts[0] != artifacts[1] {
		t.Errorf("the two servers disagree on the same corpus:\npaged:\n%s\ndefault:\n%s", artifacts[0], artifacts[1])
	}

	// header + one line per call
	for name, artifact := range map[string]string{"paged": artifacts[0], "default": artifacts[1]} {
		if got := strings.Count(artifact, "\n"); got != 8 {
			t.Errorf("%s export line count = %d, want 8 (header + 7 calls)", name, got)
		}
	}
}
