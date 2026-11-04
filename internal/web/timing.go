package web

import (
	"context"

	servertiming "github.com/larsartmann/httputil/server_timing"
)

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
