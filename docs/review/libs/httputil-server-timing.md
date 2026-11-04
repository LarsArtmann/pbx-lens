# httputil/server_timing — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pin:** `github.com/larsartmann/httputil/server_timing v1.0.3` (go.mod:25) · **Status:** USED
**Adoption score:** **33/100 direct (low)** — 2 of 6 applicable primitives called directly; **50/100 effective (moderate)** — a third primitive is covered by a hand-rolled wrapper that diverges from the maintained path.
**Fleet report:** [index.html](index.html)

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

**Behavioral delta of the hand-roll:** it does not propagate `r.Pattern` (inert today — appkit v0.9.0 imports no otelhttp — but the library version is the maintained path). The hand-roll also stamps the header on *every* route including unauthenticated runs; `ServerTimingMiddlewareWhen` is the upstream-recommended gate (godoc).

## Capability inventory vs use

Ground truth: v1.0.3 module cache — 15 exported symbols, stdlib-only.

| Symbol | Used? |
|---|---|
| `WrapServerTiming` | ✅ `internal/web/timing.go:21` |
| `MeasureServerTiming` | ✅ 11 sites: `server.go:187,348,494`; `freshness.go:25,36`; `recording.go:120,160`; `search.go:60`; `segment_page.go:24`; `ops.go:147`; (tests) |
| `ServerTimingMiddleware` | ❌ hand-rolled duplicate instead |
| `ServerTimingMiddlewareWhen` | ❌ (the gating variant, unused) |
| `RecordServerTiming` | ❌ (plain `Measure` suffices) |
| `MeasureWithDesc` | ❌ (descriptions unused) |

Types/aliases (`ServerTiming`, `WithServerTiming`, `ServerTimingFromContext`, `HeaderValue`, `Enabled`, `HeaderServerTiming`) are implementation plumbing — Not Applicable to a consumer.

## Usage census

Every named metric stops before the first body write (the recording proxy times only its two hops — `recording.go:120,160`; `ops.go:147-148` `defer stop()` is correct only while the ops JSON is written after the snapshot — a reorder would silently drop the metric; docs/http-stack.md:35-39 owns the rule).

## Gap analysis

- **Missed opportunity (high, trivial ease):** swap `timing.go:18-25` for `servertiming.ServerTimingMiddleware()`; consider `...When(pred)` to stop leaking route internals on unauthenticated runs; fix the three stale documents in the same commit.
- Not applicable: per-record context plumbing (`RecordServerTiming`) — 11 plain `Measure` sites cover the budget; descriptions — headers stay compact.

## Version currency

Pin v1.0.3 == latest release ("zero drift since v1.0.3" per httputil v1.7.0 CHANGELOG). The constructor predates the pin — the pin is current; the *doctrine* is stale, not the dependency.

## Risks

1. Agent-behavior risk: any future session reading the stale comment will "restore" or preserve the duplicate. The comment is the bug.
2. `ops.go:147` ordering caveat documented above — one refactor away from a silently dropped metric.

## Recommendations

1. Swap to the library middleware + correct `timing.go:16-17`, `docs/http-stack.md:32-34` (one commit).
2. Evaluate `ServerTimingMiddlewareWhen` gating in the same change.

## How to verify

```console
rg -n "func ServerTimingMiddleware" ~/go/pkg/mod/github.com/larsartmann/httputil/server_timing@v1.0.3/server_timing.go
sed -n '14,26p' internal/web/timing.go
rg -n "MeasureServerTiming" internal/web -g '!*_templ.go' | wc -l   # 11
rg -n "no middleware constructor" internal/web/timing.go docs/http-stack.md
```
