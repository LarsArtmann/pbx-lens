package web

import (
	"net/http"

	servertiming "github.com/larsartmann/httputil/server_timing"
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
	if number == "" || len(number) > maxNumberFilterLen {
		http.Error(w, "no such segment: the number is missing or implausible", http.StatusNotFound)

		return
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "stats")

	stats, err := DashboardStatsQuery(r.Context(), s.sys, s.now(), ListCallsQuery{Number: number}, dashboardWindowDays)

	stop()

	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	render(w, r, segmentRoute(nonce(r), number, stats, s.now()))
}
