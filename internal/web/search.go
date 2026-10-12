package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// searchPage serves the FTS content-search page (B3/m28-m29). The page size
// is search.DefaultLimit — one source of truth for what one result page is.
func (s *server) searchPage(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		render(w, r, searchRoute("", nil, ""))

		return
	}

	if s.search == nil {
		http.Error(
			w,
			"search needs a sqlite index: restart serve with -db <file> (memory deployments have no FTS)",
			http.StatusServiceUnavailable,
		)

		return
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "fts")
	hits, err := s.search.Query(r.Context(), query, search.DefaultLimit)

	stop()

	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusBadRequest)

		return
	}

	render(w, r, searchRoute(query, hits, ""))
}

// detailURL is the canonical detail deep link for one indexed call — the
// same path list rows, search hits, and the search-redirect (T21) build on.
func detailURL(kind calls.SourceKind, id calls.NativeID) string {
	return "/calls/" + url.PathEscape(string(kind)) + "/" + url.PathEscape(string(id))
}

// hitHref deep-links a search hit to its call page, carrying the query
// along (plan T21): the detail page re-runs the match server-side, marks
// the matching turns, and redirects once to the first match's anchor, so
// the click lands at the evidence — no JS required.
func hitHref(hit search.Hit, query string) string {
	path := detailURL(hit.Kind, hit.NativeID)
	if query == "" {
		return path
	}

	return path + "?q=" + url.QueryEscape(query)
}

// searchJumpTarget is the one redirect the search deep link produces: the
// detail page re-served with the query plus the first match's turn anchor.
// Same-origin by construction — fixed /calls/ prefix, path-escaped identity,
// query-escaped search text — and unit-tested on exactly that property.
func searchJumpTarget(view calls.CallView, query string, firstTurn int) string {
	return detailURL(view.Kind, view.NativeID) +
		"?q=" + url.QueryEscape(query) +
		"&at=" + strconv.Itoa(firstTurn) +
		"#turn-" + strconv.Itoa(firstTurn)
}
