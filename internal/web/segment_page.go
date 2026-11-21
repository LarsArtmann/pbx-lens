package web

import (
	"net/http"

	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/query"
)

// segmentPage serves one number's trend card (/segments/{number}, plan P16):
// the dashboard aggregates narrowed to every call that has the number on
// either side — the same representative-row scan and denominators the
// dashboard cards use, so a segment row's click-through can never disagree
// with the table it came from. The flat filtered list stays one click away
// (the page links it); the page itself answers volume, AI share, and how
// the calls ended.
func (s *server) segmentPage(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	if number == "" || len(number) > query.MaxNumberFilterLen {
		http.Error(w, "no such segment: the number is missing or implausible", http.StatusNotFound)

		return
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "stats")

	stats, err := query.DashboardStatsQuery(
		r.Context(),
		s.sys,
		s.now(),
		query.ListCallsQuery{Number: number},
		query.DashboardWindowDays,
	)

	stop()

	if err != nil {
		s.writeIndexError(w, r, err, http.StatusInternalServerError, indexReadFailureMsg, false)

		return
	}

	render(w, r, segmentRoute(nonce(r), csrfToken(r), s.authEnabled, number, stats, s.now()))
}
