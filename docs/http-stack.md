# HTTP stack — serve wiring and cache doctrine

<!-- Living doc extracted from AGENTS.md 2026-10-12 (220-line budget).
     The WHY behind the middleware chain, the ETag skip policy, and the
     render-buffering fix. Adoption history and considered-but-not-adopted
     alternatives: docs/planning/archived/2026-10-12_12-47_SUPERB-*.md. -->

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
2. **`httputil.SecurityHeaders`** (Pareto P1-T5/T6, 2026-10-17) — tightens the
   appkit default for an internal single-operator tool: `Referrer-Policy:
   no-referrer` (the internal host + query strings must not leak to the
   recording/audio origin), `X-Content-Type-Options: nosniff`, and
   `X-Robots-Tag: noindex` via the config's `Custom` map (the HTML pages
   carry the meta robots tag already; the header covers exports and
   `/api/*` uniformly). The appkit default SecurityHeaders still runs OUTER
   in the default stack; the inner `Set` calls win, so the effective values
   are ours while the default's `X-Frame-Options: DENY` survives untouched.
   Header-only and set before the first write, so every value rides into
   304 revalidations. Pinned by `TestSecurityHeadersContract`.
3. **`web.ServerTimingMiddleware`** — W3C Server-Timing via
   `servertiming.WrapServerTiming(w, r)`. NO middleware constructor exists in
   any released `httputil/server_timing` version — always wrap manually.
   Handlers add named metrics with `MeasureServerTiming(r.Context(), name)`
   and MUST call `stop()` BEFORE any `if err` branch and before the first
   body write: the header flushes at the first write, and metrics recorded
   after it are silently dropped. Streaming routes therefore time only their
   pre-first-write hops (metadata fetch, response headers).
4. **`httputil.Compression`** (`DefaultCompressionConfig`) — q-value
   negotiation. MUST sit OUTER of ETag so tags hash identity bodies
   (go-etag's ordering doctrine: compress-inner would hash the compressed
   bytes per encoding).
5. **`web.ETagMiddleware`** (go-etag/server, `DefaultETagConfig` +
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

## Recording proxy: two bounded hops, unbounded body (T19 extraction, 2026-10-18)

The recording proxy (`/recording` route, WAV with MP3 fallback) makes TWO
upstream hops, both bounded by `web.DefaultUpstreamTimeout` (30 s; the serve
flag `-recording-timeout` overrides): (1) the recording-metadata fetch
deadline and (2) the WAV response-headers deadline. The BODY then streams
unbounded (WriteTimeout is NoTimeout by contract) — a slow download never
trips the hop budget, and the Server-Timing metrics name each hop
(pre-first-write only, per the Server-Timing rule above). Total request
lifetime is therefore bounded by hops + body size, never by a global
deadline that would truncate large recordings (the former 30 s WriteTimeout
did exactly that).

## Render buffering (the flush trap)

`web.render` renders templ components into a `bytes.Buffer` and performs ONE
atomic write. templ's streaming writer flushes per section; go-etag bails to
untagged streaming on any Flush — without the buffer, HTML pages got ZERO
ETags. Found by live probe, not by tests. A future gotchas.md entry tracks
this (TODO_LIST).

Exact `Content-Length` on buffered identity renders (2026-10-15, plan
M11/F29): the buffer alone still shipped chunked at 25k+ bytes — Go's
implicit 2 KiB sniffing buffer overshoots on big bodies — so `render` states
the length explicitly before writing. The compression middleware then owns
it: `httputil` v1.5.0 DELETES the handler-set length when it actually
gzip-encodes and PRESERVES it on identity responses (verified in the module
source before wiring). `TestCacheSemanticsHardening` pins both directions.

## Cache-Control: no-cache (2026-10-12)

Every HTML response (pages AND fragments — all funnel through `render`)
carries `Cache-Control: no-cache`: browsers and htmx polls must revalidate
(ETag → 304 when unchanged) instead of silently serving stale HTML. The
header survives into go-etag's 304 because it is set before the first write;
`assertConditionalGET(..., wantNoCache=true)` pins the contract on the 200
AND the 304. Streams (export/recording) and the static asset opt out.

### The nav-id volatility trap (found live, 2026-10-15 — plan M8/F28)

The no-cache contract above was DOCUMENTED but never exercised live after a
real change. The live probe found it silently broken: the shell's
`SimpleNav` carried no `BaseProps.ID`, so the library's
`utils.EnsureID("mobile-menu", "")` minted a RANDOM id (crypto/rand, 8
bytes) into the mobile-menu markup on EVERY render. Page bodies were unique
per request → ETags never matched → every navigation and fragment poll was a
full 200 (fragments dodge it only because they embed no nav). Measured:
three consecutive `/` fetches carried three different ETags.

Fix: the shell pins `BaseProps.ID: "pbx-lens-nav"` (EnsureID returns a
non-empty id unchanged; it flows SimpleNav → Nav → mobile-menu). Verified
live after the fix: `/`, `/app.css`, `/frag/dashboard` all 304 on
If-None-Match, `no-cache` survives into the 304, and the skip policy still
keeps `/frag/staleness` untagged. Pinned by
`TestNavIDIsDeterministic` (random `tc-mobile-menu-<16-hex>` ids fail the
build). UPSTREAM FILING CANDIDATE: the EnsureID-by-default random id makes
EVERY consumer's page bodies volatile and defeats HTTP caching out of the
box — deterministic-by-default (or a documented caching note) belongs
upstream; evidence above.

Fragment Server-Timing budget (2026-10-15, empty-ish index, M8/F29):
`/frag/dashboard` total ≈ 1.2 ms, `/frag/staleness` ≈ 0.5 ms — both orders
of magnitude under the 30 s/60 s poll intervals; polling cost is negligible
per pass.

## CSP report-only sweep (2026-10-17, Pareto P1-T7)

`scripts/csp-sweep.js` boots the chain with `CSP_NONCE=1 CSP_REPORT_ONLY=1`
(both flags: scripts carry no nonce attribute in non-nonce mode, so
report-only alone would report every inline script against the empty
`'nonce-'`), drives six surfaces (/ , /dashboard, /ops, /search,
`/search?q=…`, a detail page) plus the theme-toggle and copy clicks, and
collects `securitypolicyviolation` events. First run: REAL violations — two
un-nonced library singleton scripts (the calls Table row-href script and the
CopyButton scripts, once per button). Fix: thread the per-request nonce into
`tableProps` and both copy affordances (the library already nonce-threads;
we just never passed it). Re-run: ZERO violations — the nonce policy is
**enforce-clean**. The enforce call: flip = enabling nonce mode
(`CSP_NONCE=1`) in the deploy config; kept report-only as the standing
opt-in until the next host deploy (owner gate, TODO_LIST).

Concurrent-poll ceiling (2026-10-15, M11/F31, `BenchmarkConcurrentPoll`
through the real middleware chain, parallel, dev box): 304 revalidation
≈ 82 µs/op, full 200 ≈ 101 µs/op → ~12k polls/s per box. At the 30 s
dashboard poll a viewer costs ~0.033 req/s; the cache contract, not CPU, is
the scaling story. (The raw FNV-64a tag hash alone is µs-scale, sized by
body: ≈22 µs @ 25 KB, ≈104 µs @ 122 KB — measured 2026-10-15.)

## Version and metrics

`ServiceConfig.Version` → `GET /version` + `appkit_build_info`; `Metrics`
with the explicit `AllowUnauthenticated` opt-in → `GET /metrics` (Prometheus
text). Both exist beside the appkit `/health{,/live,/ready}` routes; auth
posture is the htpasswd middleware or loopback-only dev.
