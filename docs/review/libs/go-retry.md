# go-retry — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-retry` v0.9.0 (checkout last commit 2026-11-04; deps: go-error-family only) · **Status in pbx-lens:** **IN GRAPH (INDIRECT only, zero imports)** — `go-retry v0.9.0 // indirect` (go.mod:117); was NOT USED at the 2026-11-04 snapshot
**Verdict:** **DEFER** — nothing in pbx-lens retries today, by decision; adopt when the Telnyx client grows per-request handling.
> **Refreshed 2026-11-12:** status delta — now **IN GRAPH (INDIRECT only)**: `go-retry v0.9.0 // indirect` (go.mod:117) rode in via `internal/web → cqrs-htmx/usermgmt/v4 → go-cqrs-lite/middleware/v4 → go-retry` (`GOEXPERIMENT=jsonv2 go mod why`); `rg "go-retry" --type go internal cmd` returns zero hits. v0.9.0 == latest tag (tags fetched). A ride-along dependency is not adoption: **DEFER verdict and the Telnyx-client adopt trigger stand unchanged.**

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- `Do`/`DoWithValue`: exponential backoff + additive jitter.
- error-family `IsRetryable` predicate integration (pairs with pbx-lens' existing family classification).
- `DelayFunc` honoring HTTP `Retry-After`; `OnRetry`/`OnExhausted` hooks; `ErrExhausted`/`ErrCanceled`/`ErrDeadlineExceeded` sentinels; fail-closed nesting; per-call options. 100% coverage + fuzz/stress CI upstream.

## Why pbx-lens defers

1. **There is no retry loop to replace.** The ingest poller is a fixed 15-minute ticker with log-and-skip per source (`internal/cli/poller.go:146-170` PollLoop) — a documented cqrs-lint F009 decision (docs/cqrs-lint-policy.md:70-83). A grep finds no `time.Sleep`/backoff loops in `internal/`; the only `Retry-After` in the repo is _server-side_, sent by the recording proxy (`internal/web/recording.go:61-62`).
2. **The poll cadence already acts as the retry.** Sources are re-polled wholesale every 15 min; a transient failure costs one cycle, and the ops page surfaces it (`internal/cli/poller.go:93-113`).
3. **It is NOT a cenkalti/backoff wrapper** (verified: no code import; backoff appears only in upstream ROADMAP comparisons) — so "it's already in the graph indirectly (cenkalti v4+v5 via watermill/go-appkit)" is _not_ a promotion argument for this lib; adopting it adds a new module.
4. The error-family pairing is half-wired already: the poller's fail-open Transient semantics are pinned by tests (`internal/cli/poller_test.go:458-480`) — a future `IsRetryable` gate must respect that documented reasoning (see the [go-error-family report](go-error-family.md) risks).

## When this verdict flips

The Telnyx conversations/recordings client (per-request API calls, unlike the JSONL scans) gaining 429/5xx handling is the natural trigger — `Do` + `IsRetryable` + `Retry-After` would land in one small function there. Re-evaluate on any such client work.

## How to verify

```console
rg -n "go-retry" internal cmd --type go           # zero direct imports (indirect rider in go.mod since the cqrs-htmx wave)
rg -n "time.Sleep|backoff" internal/ --type go    # no retry loops
rg -n "Retry-After" internal/web/recording.go
```
