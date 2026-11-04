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
// The library ships its own constructor since v1.0.1
// (ServerTimingMiddleware / ServerTimingMiddlewareWhen); this wrapper
// predates it and delegates to the lower-level WrapServerTiming primitive
// — the wiring swaps onto the library constructor (deleting this
// duplicate) rather than maintaining two wrappers.
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
// construction: mark records once even though Measure's stop would record
// again on every call.
type firstPageMetric struct {
	stop func()
	done bool
}

func newFirstPageMetric(ctx context.Context) *firstPageMetric {
	return &firstPageMetric{stop: servertiming.MeasureServerTiming(ctx, "firstpage")}
}

func (f *firstPageMetric) mark() {
	if f.done {
		return
	}

	f.done = true
	f.stop()
}
