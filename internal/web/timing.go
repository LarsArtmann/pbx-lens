package web

import (
	"context"
	"net/http"

	"github.com/larsartmann/httputil"
	servertiming "github.com/larsartmann/httputil/server_timing"
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

// firstPageMetric times a streaming route's only pre-first-write hop: the
// request parse plus the FIRST keyset page fetch, i.e. the time-to-first-row
// a downstream consumer actually feels. The stop fires on the first row (or
// at stream end when a filter matched nothing) — never around the body
// loop, whose cost is the download itself, not the server's. Single-shot by
// construction: the mark is a guarded bool, because Measure's stop records
// again on every call.
func firstPageMetric(ctx context.Context) func() {
	stop := servertiming.MeasureServerTiming(ctx, "firstpage")

	var done bool

	return func() {
		if done {
			return
		}

		done = true
		stop()
	}
}
