package web

import (
	"net/http"
	"net/url"
	"strings"

	servertiming "github.com/larsartmann/httputil/server_timing"
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

// hitHref deep-links a search hit to its call page. The hit's identity
// fields are the domain's own SourceKind/NativeID, so the path is the same
// deep link the list rows render.
func hitHref(hit search.Hit) string {
	return "/calls/" + url.PathEscape(string(hit.Kind)) + "/" + url.PathEscape(string(hit.NativeID))
}
