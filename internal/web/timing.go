package web

import (
	"net/http"

	"github.com/larsartmann/httputil"
	"github.com/larsartmann/httputil/server_timing"
)

// ServerTimingMiddleware instruments every route with W3C Server-Timing
// headers (httputil/server_timing): the wrapper owns the total and the
// header flush; handlers add named metrics with
// servertiming.MeasureServerTiming before their first body write —
// anything recorded after the response commits never reaches the header.
// The released module surface has no middleware constructor, so the wiring
// rides the library's own framework-adapter primitive, WrapServerTiming.
func ServerTimingMiddleware() httputil.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timed, tracked := servertiming.WrapServerTiming(w, r)
			next.ServeHTTP(timed, tracked)
		})
	}
}
