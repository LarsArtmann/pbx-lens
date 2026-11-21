package web

import (
	"encoding/json/v2"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/query"
)

// freshness serves the freshness probe as JSON.
func (s *server) freshness(w http.ResponseWriter, r *http.Request) {
	stop := servertiming.MeasureServerTiming(r.Context(), "query")
	page, err := query.ListCalls(r.Context(), s.sys, query.ListCallsQuery{})

	stop()

	if err != nil {
		s.writeIndexError(w, r, err, http.StatusInternalServerError, indexReadFailureMsg, true)

		return
	}

	countStop := servertiming.MeasureServerTiming(r.Context(), "count")
	total, err := metaengine.NewReader[calls.CallView](s.sys.MetaEngine(), calls.Collection).
		Count(r.Context(), metaengine.WithLimit(0))

	countStop()

	if err != nil {
		s.writeIndexError(w, r, err, http.StatusInternalServerError,
			"counting calls failed; the detail is in the server log", true)

		return
	}

	fresh := query.Freshness{CallsIndexed: total}

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
