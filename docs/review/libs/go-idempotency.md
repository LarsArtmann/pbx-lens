# go-idempotency — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-idempotency` v0.3.2 (checkout last commit 2026-11-04; single module, dep: go-error-family) · **Status in pbx-lens:** **IN GRAPH (INDIRECT only, zero imports)** — `go-idempotency v0.3.2 // indirect`; was NOT USED at the 2026-11-04 snapshot
**Verdict:** **REJECT** — pbx-lens' idempotency problem is solved by construction, in a shape this library does not target.
> **Refreshed 2026-11-12:** status delta — now **IN GRAPH (INDIRECT only)**: `go-idempotency v0.3.2 // indirect` rode in via `cqrs-htmx/loginpage/v4 → cqrs-htmx/v4 → go-idempotency` (`go mod why`); zero direct imports in `internal`/`cmd`. v0.3.2 == latest tag (tags fetched). Ride-along ≠ adoption: REJECT-direct verdict stands.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- `Store` interface (`Seen`/`Record`/atomic `CheckAndRecord`) with TTL-windowed claim semantics for **client-supplied keys**.
- A 13-invariant contract test suite for backend implementors.
- error-family sentinels (`ErrDuplicate` → 409).
- Only a deprecated in-memory backend ships — the library is a contract, consumers write the backend.

## Why pbx-lens does not need it

1. **Idempotency here is natural-key, not client-key.** Stream ids are deterministic (`<kind>:<native-id>`), so re-ingesting the same source row IS the same stream, and `call.record` replays a matching ContentHash to zero events (`internal/calls/commands.go:30`; AGENTS.md domain rules). There is no client-supplied idempotency key anywhere in the pipeline to dedupe on.
2. **The cqrs-lite dedup module is already in the graph** (`go-cqrs-lite/dedup/v4 v4.3.0`, go.mod:83 — indirect via projectionhost), and go-idempotency's own README routes cqrs-lite users to `metaengine.DedupStore` rather than writing a Store backend.
3. **Adopting it means writing a SQLite Store backend** (a new module + new code) to solve a problem the event log already solves for free.

## When this verdict flips

If pbx-lens ever grows an externally-triggered ingest API (a client POSTing observations with client-side retries), client-key idempotency becomes real — then revisit `Store` + `metaengine`-backed dedup as one unit.

## How to verify

```console
rg -n "go-idempotency" internal cmd --type go   # zero direct imports (indirect cqrs-htmx rider in go.mod since the v4.20 wave)
rg -n "ContentHash" internal/calls/decider.go internal/calls/events.go | head -3   # the natural key lives in the decider (commands.go split since the audit)
rg -n "dedup/v4" go.mod
```
