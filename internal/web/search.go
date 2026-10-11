package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/larsartmann/pbx-lens/internal/search"
)

// searchResultLimit bounds one result page.
const searchResultLimit = 50

// searchPage serves the FTS content-search page (B3/m28-m29).
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

	hits, err := s.search.Query(r.Context(), query, searchResultLimit)
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
