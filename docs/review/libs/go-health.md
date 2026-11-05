# go-health — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-health` v0.9.0 (checkout last commit 2026-11-04; deps: samber/do v2 + go-branded-id) · **Status in pbx-lens:** **NOT USED** (not in the graph)
**Verdict:** **REJECT** — hard prerequisite mismatch (samber/do injector) and the job is already done in-house with the right semantics.
**Fleet report:** [index.html](index.html)

## What the library offers

- k8s three-probe SDK (liveness/readiness/startup) over a **samber/do v2** DI injector; critical/non-critical check classification (503-vs-warn semantics); background probe caching; shutdown awareness; `RegisterRoutes`; federation; metrics; DB/redis check batteries.

## Why pbx-lens rejects it

1. **Prerequisite mismatch:** it requires a samber/do injector as its composition root. pbx-lens' composition root is go-cqrs-lite `system.New` — ADR-0123 makes that the _only_ one (AGENTS.md stack decisions; verified: `internal/app/app.go:87` is the sole construction site). Two competing DI/story roots in one binary is exactly the split-brain the house rules forbid.
2. **The job is already done, with honest semantics:** appkit serves `/health,/live,/ready` (docs/http-stack.md), and pbx-lens adds per-source counters + staleness verdicts with unknown-vs-false honesty (`internal/web/ops.go:110-134`, poller freshness `internal/cli/poller.go:93-113`) — including nil-aware "nothing wired" states (`ops.go:24-38,69-99`). A k8s probe SDK targets an orchestration surface pbx-lens does not have (single NixOS systemd unit).
3. The genuinely missing readiness bit — _projection workers live_ — is better addressed with the cqrs-htmx health-bridge **idea** (see [cqrs-htmx.md](cqrs-htmx.md)) than by importing a DI-bound SDK.

## When this verdict flips

If pbx-lens is ever deployed into a probe-driven orchestrator _and_ migrates to a samber/do composition — jointly. Neither is plausible under the current architecture decisions.

## How to verify

```console
rg -n "go-health" go.mod go.sum                        # absent
rg -n "samber/do" go.mod                               # prerequisite absent
rg -n "/health|/live|/ready" docs/http-stack.md | head -4
rg -n "unknown" internal/web/ops.go | head -4
```
