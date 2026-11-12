# go-health-dashboard — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-health-dashboard` v0.12.2 (checkout last commit 2026-11-04; pulls go-health + templ-components + go-sse/datastar) · **Status in pbx-lens:** **NOT USED** (not in the graph)
**Verdict:** **REJECT** — doubly inapplicable: needs go-health (which itself is rejected, see [go-health.md](go-health.md)) and its real-time transport is SSE, excluded by the R9 polling decision.

> **Refreshed 2026-11-12:** re-verified: still not imported (absent from go.mod/go.sum); latest tag now **v0.13.2** (tags fetched; report's v0.12.2 snapshot predates the bumps). REJECT verdict stands.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- HTML status page over go-health via **Datastar SSE**: severity-grouped badges, trend sparklines, content-negotiated `/health` (HTML vs JSON), SSE push modes, rate limiting, CSP nonce support, federation hub binaries, Prometheus/trend/introspection routes.

## Why pbx-lens rejects it

1. **Transport clash:** its live updates are SSE/Datastar; pbx-lens' UI doctrine is htmx polling (R9: `/frag/dashboard` 30 s, `/frag/staleness` 60 s — AGENTS.md). Adopting this would drag a competing JS runtime into an htmx page.
2. **Dependency clash:** it builds on go-health's samber/do-injector model — rejected above (composition root is `system.New`, ADR-0123).
3. **The niche is already served better-fit:** the hand-rolled `/api/ops` page ships the same honesty discipline (unknown-vs-false pointers, "nothing wired" states, per-source staleness verdicts — `internal/web/ops.go:24-38,69-99,110-134`, rendered via templ-components at `ops.templ`), and the dashboard's freshness UI already exists (`dashboard.templ` PolledRegion). A generic health-dashboard would replace bespoke domain truth with generic probe truth.

## When this verdict flips

If pbx-lens ever operates as a fleet member inside a go-health federation hub (multiple services reporting to one status page), this becomes the natural viewer — until then it is dead weight.

## How to verify

```console
rg -n "go-health-dashboard" go.mod go.sum   # absent
rg -n "PolledRegion" internal/web/dashboard.templ internal/web/ops.templ | head -4
rg -n "unknown|never" internal/web/ops.go | head -6
```
