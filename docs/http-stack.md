# HTTP stack — serve wiring and cache doctrine

<!-- Living doc extracted from AGENTS.md 2026-10-12 (220-line budget).
     The WHY behind the middleware chain, the ETag skip policy, and the
     render-buffering fix. Adoption history and considered-but-not-adopted
     alternatives: docs/planning/2026-10-12_12-47_SUPERB-*.md. -->

## Middleware chain

appkit defaults: Recovery → RequestID → Logging → Timeout(30s) →
SecurityHeaders; `ExtraMiddlewares` are appended inner of the defaults.
`WriteTimeout: appkit.NoTimeout` drops the Timeout middleware — exports and
the recording WAV stream are unbounded by contract, and the former 30 s
deadline truncated both (`StreamExportCalls` honors ctx; the WAV proxy
builds its upstream request on `r.Context()`).

ExtraMiddlewares, in strict order:

1. **auth** (htpasswd middleware when `-htpasswd` is set) — innermost of the
   three additions but outer of everything web-local.
2. **`web.ServerTimingMiddleware`** — W3C Server-Timing via
   `servertiming.WrapServerTiming(w, r)`. NO middleware constructor exists in
   any released `httputil/server_timing` version — always wrap manually.
   Handlers add named metrics with `MeasureServerTiming(r.Context(), name)`
   and MUST call `stop()` BEFORE any `if err` branch and before the first
   body write: the header flushes at the first write, and metrics recorded
   after it are silently dropped. Streaming routes therefore time only their
   pre-first-write hops (metadata fetch, response headers).
3. **`httputil.Compression`** (`DefaultCompressionConfig`) — q-value
   negotiation. MUST sit OUTER of ETag so tags hash identity bodies
   (go-etag's ordering doctrine: compress-inner would hash the compressed
   bytes per encoding).
4. **`web.ETagMiddleware`** (go-etag/server, `DefaultETagConfig` +
   `SkipIfPresent: true`).

## ETag skip policy

`skipBodyCaching` in `internal/web/caching.go` excludes:

- `/frag/staleness` + `/api/freshness` — the embedded age changes every
  response; a tag would never match.
- prefix `/api/export.` + suffix `/recording` — unbounded streams must never
  be buffered for hashing.

`ETagConfig` hooks `OnETagGenerated` / `On304` / `OnBufferOverflow` exist in
the pinned v0.6.1 (etag.go:67-88); wiring them into `/api/ops` counters is a
TODO_LIST row.

## Render buffering (the flush trap)

`web.render` renders templ components into a `bytes.Buffer` and performs ONE
atomic write. templ's streaming writer flushes per section; go-etag bails to
untagged streaming on any Flush — without the buffer, HTML pages got ZERO
ETags. Found by live probe, not by tests. A future gotchas.md entry tracks
this (TODO_LIST).

## Cache-Control: no-cache (2026-10-12)

Every HTML response (pages AND fragments — all funnel through `render`)
carries `Cache-Control: no-cache`: browsers and htmx polls must revalidate
(ETag → 304 when unchanged) instead of silently serving stale HTML. The
header survives into go-etag's 304 because it is set before the first write;
`assertConditionalGET(..., wantNoCache=true)` pins the contract on the 200
AND the 304. Streams (export/recording) and the static asset opt out.

## Version and metrics

`ServiceConfig.Version` → `GET /version` + `appkit_build_info`; `Metrics`
with the explicit `AllowUnauthenticated` opt-in → `GET /metrics` (Prometheus
text). Both exist beside the appkit `/health{,/live,/ready}` routes; auth
posture is the htpasswd middleware or loopback-only dev.
