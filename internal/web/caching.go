package web

import (
	"net/http"
	"strings"
	"sync/atomic"

	etag "github.com/larsartmann/go-etag/server"
	"github.com/larsartmann/httputil"
)

// CacheCounters accumulates what the ETag middleware did, so /api/ops can
// answer "is the response cache earning its keep?" without log spelunking.
// Every method is safe for concurrent use; the middleware increments from
// request goroutines while the ops handler snapshots.
type CacheCounters struct {
	generated   atomic.Uint64
	notModified atomic.Uint64
	overflows   atomic.Uint64
}

// NewCacheCounters builds an empty counter set.
func NewCacheCounters() *CacheCounters {
	return &CacheCounters{}
}

// CacheStats is the /api/ops "cache" section: a point-in-time read of the
// middleware counters. HitRatio is NotModified over all tag-resolved
// conditional-capable responses (the share of responses that skipped their
// body); 0 until the first such response.
type CacheStats struct {
	ETagGenerated   uint64  `json:"etag_generated"`
	NotModified     uint64  `json:"not_modified"`
	HitRatio        float64 `json:"hit_ratio"`
	BufferOverflows uint64  `json:"buffer_overflows"`
}

// Snapshot reads the counters atomically. Individual fields are consistent;
// the set as a whole may interleave with in-flight requests, which is fine
// for an ops display.
func (c *CacheCounters) Snapshot() CacheStats {
	generated := c.generated.Load()
	notModified := c.notModified.Load()

	ratio := 0.0
	if total := generated + notModified; total > 0 {
		ratio = float64(notModified) / float64(total)
	}

	return CacheStats{
		ETagGenerated:   generated,
		NotModified:     notModified,
		HitRatio:        ratio,
		BufferOverflows: c.overflows.Load(),
	}
}

// ETagMiddleware answers conditional GETs with 304 Not Modified: response
// bodies are hashed and revalidated against If-None-Match, so the htmx poll
// regions, the static assets (no-cache with no validator used to mean a
// full re-download every navigation), and the read-model pages cost zero
// body bytes while unchanged. SkipIfPresent keeps the door open for
// handlers that compute data-derived tags instead of body hashes.
//
// Stats, when non-nil, receives one counter per middleware outcome
// (generated tag, 304, buffer overflow); nil disables the hooks. Log, when
// non-nil, receives the overflow warning — a response larger than the
// buffer streams without an ETag, and operators should know that happened
// rather than infer it from a missing hit.
func ETagMiddleware(stats *CacheCounters, log func(msg string, args ...any)) httputil.Middleware {
	cfg := etag.DefaultETagConfig()
	cfg.SkipIfPresent = true
	cfg.Skip = skipBodyCaching

	if stats != nil {
		cfg.OnETagGenerated = func(etag.ETag) { stats.generated.Add(1) }
		cfg.On304 = func(etag.ETag) { stats.notModified.Add(1) }
		cfg.OnBufferOverflow = func(limit int) {
			stats.overflows.Add(1)

			if log != nil {
				log(
					"etag buffer overflow: response body exceeded the buffer and streams without an ETag",
					"limit_bytes", limit,
				)
			}
		}
	}

	return etag.New(cfg)
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
