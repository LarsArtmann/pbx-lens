package web

import (
	"net/http"

	"github.com/a-h/templ"
)

// render writes a templ component as the HTML response. Handlers needing a
// non-200 status set status AND Content-Type before calling render — after
// WriteHeader the header map is ignored.
func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "rendering the page failed: "+err.Error(), http.StatusInternalServerError)
	}
}
