package web

import (
	"net/http"
	"strings"
	"sync/atomic"

	errorfamily "github.com/larsartmann/go-error-family"
	etagmetrics "github.com/larsartmann/go-etag/metrics"
	etag "github.com/larsartmann/go-etag/server"
	"github.com/larsartmann/httputil"
)

// CacheCounters is the ops-facing façade over the library's counting
// module (go-etag/metrics, plan T12): generated/304/overflow counting is
// the library's maintained path (Attach wraps any local hooks), and the
// one counter the module does NOT cover — post-commit write failures via
// OnError — stays a local atomic. Every method is safe for concurrent
// use; the middleware increments from request goroutines while the ops
// handler snapshots.
type CacheCounters struct {
	inner       *etagmetrics.Counters
	writeErrors atomic.Uint64
}

// NewCacheCounters builds an empty counter set.
func NewCacheCounters() *CacheCounters {
	return &CacheCounters{inner: &etagmetrics.Counters{}}
}

// CacheStats is the /api/ops "cache" section: a point-in-time read of the
// middleware counters. ETagGenerated counts every tag-resolved response —
// go-etag resolves the tag on 304 replays too, so it doubles as the
// conditional-capable request count. HitRatio is NotModified over
// ETagGenerated (the share of responses that skipped their body); 0 until
// the first tag is resolved. (The library's HitRatio documents the >1
// caveat for adopted handler tags — pbx-lens handlers set none today.)
type CacheStats struct {
	ETagGenerated   uint64  `json:"etag_generated"`
	NotModified     uint64  `json:"not_modified"`
	HitRatio        float64 `json:"hit_ratio"`
	BufferOverflows uint64  `json:"buffer_overflows"`
	WriteErrors     uint64  `json:"write_errors"`
}

// Snapshot reads the counters atomically. Individual fields are consistent;
// the set as a whole may interleave with in-flight requests, which is fine
// for an ops display.
func (c *CacheCounters) Snapshot() CacheStats {
	inner := c.inner.Snapshot()

	return CacheStats{
		ETagGenerated:   uint64(inner.Generated),
		NotModified:     uint64(inner.NotModified),
		HitRatio:        c.inner.HitRatio(),
		BufferOverflows: uint64(inner.BufferOverflows),
		WriteErrors:     c.writeErrors.Load(),
	}
}

// ETagMiddleware answers conditional GETs with 304 Not Modified: response
// bodies are hashed and revalidated against If-None-Match, so the htmx poll
// regions, the static assets (no-cache with no validator used to mean a
// full re-download every navigation), and the read-model pages cost zero
// body bytes while unchanged. SkipIfPresent keeps the door open for
// handlers that compute data-derived tags instead of body hashes.
//
// Stats, when non-nil, receives one counter per middleware outcome —
// generated tag, 304, buffer overflow (the three the metrics module
// attaches), plus post-commit write errors via OnError, which the module
// does not cover; a client gone mid-flush drops the buffered body after
// the header already committed, unreportable through net/http, so the
// counter is the only witness. Nil disables the hooks. Log, when non-nil,
// receives the overflow warning — a response larger than the buffer
// streams without an ETag, and operators should know that happened rather
// than infer it from a missing hit — and the write-error warning.
func ETagMiddleware(stats *CacheCounters, log func(msg string, args ...any)) httputil.Middleware {
	cfg := etag.DefaultETagConfig()
	cfg.SkipIfPresent = true
	cfg.Skip = skipBodyCaching

	if stats == nil {
		return etag.New(cfg)
	}

	// Attach wraps the local hooks preserve-and-append (library contract):
	// counting runs first, the overflow warning rides after.
	if log != nil {
		cfg.OnBufferOverflow = func(limit int) {
			log(
				"etag buffer overflow: response body exceeded the buffer and streams without an ETag",
				"limit_bytes", limit,
			)
		}
	}

	cfg.OnError = func(e *errorfamily.Error) {
		stats.writeErrors.Add(1)

		if log != nil {
			log("etag post-commit write failed", "family", e.Family().String(), "code", e.Code(), "detail", e.Message())
		}
	}

	wired, counters := etagmetrics.Attach(cfg)
	stats.inner = counters

	return etag.New(wired)
}

// skipBodyCaching excludes the routes where a body-hash ETag is wrong or
// pointless: the two exports and the recording proxy stream unbounded (the
// buffer must never engage), and the staleness fragment plus freshness
// probe embed an age that changes on every response, so they could never
// revalidate. Everything else is a stable-until-the-index-moves query.
func skipBodyCaching(r *http.Request) bool {
	path := r.URL.Path

	return path == "/frag/staleness" ||
		path == "/api/freshness" ||
		strings.HasPrefix(path, "/api/export.") ||
		strings.HasSuffix(path, "/recording")
}
