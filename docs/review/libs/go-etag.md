# go-etag — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pin:** `github.com/larsartmann/go-etag/server v0.6.1` (go.mod:23; `entitytag` v0.6.1 indirect — re-exported by server, `go mod why` verified) · **Status:** USED
**Adoption score:** **95/100 (very high)** — every applicable config knob/area is now wired (5 knobs explicit, 3 defaulted, plus `OnError` and the `go-etag/metrics` module adopted 2026-11-04; `Validate` N/A, entitytag parse/match re-exports N/A). The residual 5 is the standing `go-etag/client` Hold, not an unadopted capability.
**Fleet report:** [index.html](index.html)

## Verdict

The ETag middleware is configured with above-default care: five `ETagConfig` knobs are explicitly wired (SkipIfPresent, Skip predicate, and three observability hooks feeding `/api/ops`), the GET/HEAD-only + streaming-skip semantics match the export/recording contract, and there is **no hand-rolled ETag hashing anywhere in prod**. Two gaps: `OnError` (post-commit write-failure telemetry) is unwired, and the hand-built `CacheCounters` (`internal/web/caching.go:16-58`) duplicates `go-etag/metrics`, a sub-module that exists upstream but is not in the graph.

## Capability inventory vs use

Ground truth: module cache `~/go/pkg/mod/github.com/larsartmann/go-etag/server@v0.6.1` (`etag.go:28-89` for the config struct) + checkout `/home/lars/projects/go-etag` for the metrics/client modules.

| Capability                                                                             | Used?                                                                                                         |
| -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `etag.New(ETagConfig)` body-hash + If-None-Match 304                                   | ✅ `internal/web/caching.go:92`                                                                               |
| `MaxBufferSize` (default 1 MiB)                                                        | ✅ defaulted; overflow observable                                                                             |
| `HashFunc`                                                                             | ✅ defaulted (FNV-64a); perf rail mirrors it (`internal/web/perf_test.go:171-176`)                            |
| `SkipIfPresent`                                                                        | ✅ `caching.go:74`                                                                                            |
| `Skip` (streaming pages opt out)                                                       | ✅ `caching.go:75` (`skipBodyCaching`)                                                                        |
| `OnETagGenerated` / `On304` / `OnBufferOverflow`                                       | ✅ `caching.go:78-88` → `CacheCounters` + `/api/ops` "streamed without an ETag" card (`ops_templ.go:539-540`) |
| `OnError`                                                                              | ❌ **Missed** — post-commit write failures are invisible (TODO_LIST row per docs/http-stack.md:56-58)         |
| `Validate`                                                                             | N/A — config from defaults, not user input                                                                    |
| `DefaultETagConfig`                                                                    | ✅ `caching.go:73`                                                                                            |
| entitytag re-exports `ParseETag`/`ParseETagList`/`MatchesIfNoneMatch`/`MatchesIfMatch` | N/A — no handler-set tags, no If-Match preconditions (GET-only UI)                                            |
| `go-etag/client` module                                                                | N/A — standing Hold (Telnyx ETag emission unverified; window 2027-01)                                         |
| `go-etag/metrics` module                                                               | ❌ **Missed** — see below                                                                                     |

## The metrics duplication

`internal/web/caching.go:16-58` hand-builds `CacheCounters`/`CacheStats`/`Snapshot`/hit-ratio. Upstream `metrics/` offers exactly this: `Attach(cfg)` (preserve-and-append hooks), `Counters` (metrics.go:11), `Snapshot` (metrics.go:72), `HitRatio` (metrics.go:87) — verified in the checkout `/home/lars/projects/go-etag/metrics/metrics.go`; available since v0.5.0 (it was moved out of httputil to live next to the hooks it attaches to). pbx-lens's extra overflow-log hook composes cleanly per `Attach`'s contract. **Tradeoff:** adopting it adds a module to the graph (hermetic-nix cost, `vendorHash` repair) for ~40 lines deleted — reasonable at the next go-etag bump, not urgent alone.

## Usage census

`internal/web/caching.go:73-92` is the entire integration; tests pin semantics (`caching_test.go`, `server_test.go`, `ops_page_test.go` — hook counters, `If-None-Match` round-trips). The `no-cache` revalidation contract is doctrine (docs/http-stack.md).

## Version currency

Pin v0.6.1. Upstream has since tagged **v0.7.0** — commit range v0.6.1..v0.7.0 is dependency chores + the doc-snippet gate + config convergence: **no API change** to server/entitytag. Checkout `[Unreleased]` adds a metrics glossary (documents the hit-ratio >1 case below) — worth reading, not worth a bump alone.

## Risks

1. **Hit-ratio can exceed 1** once any handler adopts its own tag (`SkipIfPresent` respects it; its 304s fire `On304` without `OnETagGenerated`). Latent — no handler sets tags today — but the hand-rolled ratio (`caching.go:31-32,48-50`) has no guard; the metrics glossary documents exactly this trap.
2. **1 MiB buffer cliff:** any cacheable page growing past 1 MiB silently loses its 304s — mitigated by `OnBufferOverflow` → counter → ops card.
3. `OnError` gap: a client disconnect mid-write is uncounted today.

## Recommendations

1. Wire `OnError` → counter + log (completes the observability story; small).
2. At the next go-etag bump, adopt `go-etag/metrics.Attach` and delete `CacheCounters`.
3. If a handler ever sets its own ETag, guard the ratio display first.

## How to verify

```console
sed -n '70,95p' internal/web/caching.go
go doc github.com/larsartmann/go-etag/server ETagConfig
git -C /home/lars/projects/go-etag log --oneline v0.6.1..v0.7.0 | head
ls /home/lars/projects/go-etag/metrics   # module exists, not in pbx-lens graph
```

## Delta appendix — 2026-11-04 closeout (plan T11/T12/T13)

All three findings closed the day of the audit, in this order:

1. **`OnError` wired (T11).** `ETagConfig.OnError` now counts post-commit
   write failures into a local `writeErrors` atomic and logs them. It is
   deliberately NOT routed through `metrics.Attach`: upstream's `Attach`
   covers the generated/304/overflow hooks but not `OnError`, so the
   counter lives beside the façade (re-checked at the pin,
   `metrics@v0.6.1/metrics.go`). Surfaced on `/ops` as the "Write errors"
   row plus an alert; pinned by `TestETagMiddlewareWriteErrorCounts`.
2. **`go-etag/metrics` adopted (T12).** `CacheCounters` is now a façade
   over `metrics.Counters`, attached with `Attach`'s preserve-and-append
   contract (our overflow-log hook composes after the counter hook). The
   hand-rolled `Snapshot`/ratio arithmetic is gone; `+v0.6.1` of the
   `metrics` module is in the graph (promotion precedent: an already-
   vendored family member rides free) and the vendor hashes were re-cut
   and verified against BOTH the package build and the devShell.
3. **Ratio guard (T13).** `cacheRatioText` refuses to print a lie: it
   shows `n/a` until a handler sets tags and raises an explicit
   `>100% — adopted handler tags` banner in the documented >1 case instead
   of a clamped fake percentage. Pinned by `TestCacheRatioTextHonestStates`.

Score movement: **83/100 → 95/100**. Every applicable knob/area is now
wired (`OnError`, `metrics.Attach`); the residual 5 is the standing
`go-etag/client` Hold (Telnyx ETag emission unverified, window 2027-01)
and the display-guard caveat rather than an unadopted capability.
