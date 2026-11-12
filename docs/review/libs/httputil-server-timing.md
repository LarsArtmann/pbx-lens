# httputil/server_timing — utilization audit (pbx-lens)

**Date:** 2026-11-04 (refreshed **2026-11-11** — pin unchanged, still proxy latest) · **Pin:** `github.com/larsartmann/httputil/server_timing v1.0.3` (go.mod:25) · **Status:** USED
**Refreshed 2026-11-12:** pin still v1.0.3 (go.mod:30) == latest `server_timing/v1.0.3` tag — server_timing versions live in their OWN tag namespace inside the httputil repo (not shared with the root v1.x tags), verified via `git fetch --tags` + `git tag --sort=-v:refname` plus an empty commit log on `server_timing/` after the tag. Zero drift; verdict and both scores unchanged (**67/100 direct / 67/100 effective**).
**Adoption score:** **67/100 direct (moderate)** — 2 of 3 applicable primitives called directly (`MeasureServerTiming`, `ServerTimingMiddleware`); **67/100 effective** — every used capability rides the maintained path since the 2026-11-04 swap deleted the divergent hand-wrap (`ServerTimingMiddlewareWhen` declined by recorded decision: single-operator gated surface — htpasswd then, the cqrs-htmx identity session gate now; escape named in docs/http-stack.md).
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## Verdict

pbx-lens times its queries thoroughly — 11 `MeasureServerTiming` sites across every query path, the stop-before-first-write rule honored everywhere — but it wraps the library with a **hand-rolled middleware that duplicates a constructor the library has shipped since v1.0.1**, and three documents assert the opposite. The justifying comment is factually wrong at every released version. This is the audit's clearest miss: a one-line swap deletes a divergent copy and its stale doctrine.

## The headline finding — a duplicate built on a false premise

**What the library offers:** `ServerTimingMiddleware()` and `ServerTimingMiddlewareWhen(pred)` — verified present in the module cache at v1.0.1 (server_timing.go:405), v1.0.2, and the pin v1.0.3 (server_timing.go:400, 416). `When` is zero-overhead passthrough for untagged routes and propagates `r.Pattern` back to the outer request (v1.0.3 server_timing.go:429-431).

**What pbx-lens does instead:** `internal/web/timing.go:18-25` hand-builds the same middleware on `WrapServerTiming`.

**Where the false claim lives (verified 2026-11-04):**

- `internal/web/timing.go:16-17` — "The released module surface has no middleware constructor"
- `docs/http-stack.md:32-34` — same claim
- AGENTS.md (stack decisions) — "server_timing has NO middleware constructor in any release (wrap manually)" — **corrected in AGENTS.md on 2026-11-04** by this audit
- Prior audit finding #2 (2026-10-12 HTML) — was already wrong at its own v1.0.2 pin

**Behavioral delta of the hand-roll:** it does not propagate `r.Pattern` (inert today — appkit v0.9.0 imports no otelhttp — but the library version is the maintained path). The hand-roll also stamps the header on _every_ route including unauthenticated runs; `ServerTimingMiddlewareWhen` is the upstream-recommended gate (godoc).

## Capability inventory vs use

Ground truth: v1.0.3 module cache — 15 exported symbols, stdlib-only.

| Symbol                       | Used?                                                                                                                                                                                                                                     |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `WrapServerTiming`           | ❌ no longer called (was the hand-wrap's plumbing; wrapper deleted 2026-11-04)                                                                                                                                                            |
| `MeasureServerTiming`        | ✅ 11 call sites: `timing.go:40` (firstpage); `server.go:187,348,494`; `freshness.go:25,36`; `recording.go:120,160`; `search.go:60`; `segment_page.go:24`; `ops.go:147` (12 rg lines incl. one comment)                                   |
| `ServerTimingMiddleware`     | ✅ `cmd/pbx-lens/serve.go` (the 2026-11-04 swap)                                                                                                                                                                                          |
| `ServerTimingMiddlewareWhen` | ❌ declined by recorded decision (unconditional OK behind the single-operator gate — htpasswd at the 2026-11-04 decision, the identity session gate since 2026-11-11; escape named in docs/http-stack.md)                                                                                           |
| `RecordServerTiming`         | ❌ N/A: it records an ALREADY-MEASURED duration (`(ctx, name, desc, dur)`, `server_timing.go:237`); no producer here hands us one — every hop is timed inline with `Measure`, which brackets the work itself                              |
| `MeasureWithDesc`            | ❌ N/A: descriptions ride the same header line as the name, and the header is a per-route budget (a proxy hop, a query, a first page) — a prose description per metric buys nothing a reader of this repo cannot get from the metric name |

Types/aliases (`ServerTiming`, `WithServerTiming`, `ServerTimingFromContext`, `HeaderValue`, `Enabled`, `HeaderServerTiming`) are implementation plumbing — Not Applicable to a consumer.

## Usage census

Every named metric stops before the first body write (the recording proxy times only its two hops — `recording.go:120,160`; `ops.go:147-148` `defer stop()` is correct only while the ops JSON is written after the snapshot — a reorder would silently drop the metric; docs/http-stack.md:35-39 owns the rule).

## Gap analysis

- **Missed opportunity (high, trivial ease):** swap `timing.go:18-25` for `servertiming.ServerTimingMiddleware()`; consider `...When(pred)` to stop leaking route internals on unauthenticated runs; fix the three stale documents in the same commit.
- Not applicable, with the reason recorded (plan M27.1): `RecordServerTiming` takes a duration someone else measured — the library's escape for a producer that cannot bracket its own work — and every metric here IS the bracketed work (11 `Measure` sites); `MeasureWithDesc` spends header bytes on prose the metric name already carries (the budget is one line per route hop). Neither is a gap, and neither flips without a producer that reports its own timing.

## Version currency

Pin v1.0.3 == latest release ("zero drift since v1.0.3" per httputil v1.7.0 CHANGELOG). The constructor predates the pin — the pin is current; the _doctrine_ is stale, not the dependency.

## Risks

1. Agent-behavior risk: any future session reading the stale comment will "restore" or preserve the duplicate. The comment is the bug.
2. `ops.go:147` ordering caveat documented above — one refactor away from a silently dropped metric.

## Recommendations

1. Swap to the library middleware + correct `timing.go:16-17`, `docs/http-stack.md:32-34` (one commit).
2. Evaluate `ServerTimingMiddlewareWhen` gating in the same change.

## Delta appendix — 2026-11-04 closeout (plan T01/T02/T03)

All three recommendations executed the day of this audit:

1. `serve.go` wires `servertiming.ServerTimingMiddleware()`; the hand-wrap
   (`timing.go:18-25`) and its false justifying comment are deleted. A/B
   bench: allocs/op identical (1216–1218 both trees), ns/op inside box
   noise — performance-neutral.
2. Gating decision recorded: UNCONDITIONAL (single-operator htpasswd;
   `ServerTimingMiddlewareWhen` is the named escape for any future public
   route) — docs/http-stack.md middleware item 3.
3. All false-claim carriers corrected (timing.go comment deleted with the
   wrapper, http-stack.md rewritten, AGENTS.md shrunk to a pointer, the
   2026-10-12 HTML carries an errata on finding #2). A guard test pins
   header presence on a plain route (`TestServerTimingHeaderOnPlainRoute`).

Score movement: 33/50 → 67/67 (the wrapper-divergence penalty is gone;
the remaining gap is the two never-applicable plumbing primitives and the
recorded When-decline).

## How to verify

```console
rg -n "func ServerTimingMiddleware" ~/go/pkg/mod/github.com/larsartmann/httputil/server_timing@v1.0.3/server_timing.go
rg -n "ServerTimingMiddleware" cmd/pbx-lens/serve.go
rg -n "MeasureServerTiming" internal/web -g '!*_templ.go' | wc -l   # 11
rg -in "middleware constructor" internal/web/timing.go docs/http-stack.md   # zero hits
GOEXPERIMENT=jsonv2 go test ./internal/web/ -run TestServerTimingHeaderOnPlainRoute
```
