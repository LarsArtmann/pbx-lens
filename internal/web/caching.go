package web

import (
	"net/http"
	"strings"

	etag "github.com/larsartmann/go-etag/server"
	"github.com/larsartmann/httputil"
)

// ETagMiddleware answers conditional GETs with 304 Not Modified: response
// bodies are hashed and revalidated against If-None-Match, so the htmx poll
// regions, the static assets (no-cache with no validator used to mean a
// full re-download every navigation), and the read-model pages cost zero
// body bytes while unchanged. SkipIfPresent keeps the door open for
// handlers that compute data-derived tags instead of body hashes.
func ETagMiddleware() httputil.Middleware {
	cfg := etag.DefaultETagConfig()
	cfg.SkipIfPresent = true
	cfg.Skip = skipBodyCaching

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
