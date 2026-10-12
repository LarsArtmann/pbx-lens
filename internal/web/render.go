package web

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
)

// render writes a templ component as the HTML response, fully rendered into
// memory first. The atomic body matters twice: a render failure surfaces as a
// real 500 instead of plaintext appended to a half-written page, and the
// ETag middleware sees a single unflushed write — templ's streaming writer
// flushes per section, and a flushed response can never be hashed and
// answered with 304. Handlers needing a non-200 status set status AND
// Content-Type before calling render — after WriteHeader the header map is
// ignored. Every HTML response carries `Cache-Control: no-cache`: browsers
// and htmx polls must revalidate (ETag → 304 when unchanged) instead of
// silently serving a stale page or fragment; the header survives into
// go-etag's 304 because it is set before the first write.
func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	var page bytes.Buffer

	if err := component.Render(r.Context(), &page); err != nil {
		http.Error(w, "rendering the page failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	_, _ = w.Write(page.Bytes())
}
