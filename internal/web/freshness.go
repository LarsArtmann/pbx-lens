package web

import (
	"encoding/json/v2"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// Freshness is the /api/freshness payload: one honest glance at how
// current the index is (C2). The machine-readable twin of the staleness
// banner — monitors and the dashboard read this, humans read the banner.
type Freshness struct {
	NewestStartedAt time.Time `json:"newest_started_at"`
	AgeSeconds      float64   `json:"age_seconds"`
	Stale           bool      `json:"stale"`
	CallsIndexed    int       `json:"calls_indexed"`
}

// freshness serves the freshness probe as JSON.
func (s *server) freshness(w http.ResponseWriter, r *http.Request) {
	page, err := ListCalls(r.Context(), s.sys, ListCallsQuery{})
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	total, err := metaengine.NewReader[calls.CallView](s.sys.MetaEngine(), calls.Collection).
		Count(r.Context(), metaengine.WithLimit(0))
	if err != nil {
		http.Error(w, "counting calls failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	fresh := Freshness{CallsIndexed: total}

	if len(page.Rows) > 0 {
		fresh.NewestStartedAt = page.Rows[0].When.UTC()
		fresh.AgeSeconds = time.Since(page.Rows[0].When).Seconds()
		fresh.Stale = fresh.AgeSeconds > staleAfter.Seconds()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.MarshalWrite(w, fresh); err != nil {
		s.logf("freshness encode: %v", err)
	}
}
