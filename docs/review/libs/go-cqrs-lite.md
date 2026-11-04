# go-cqrs-lite — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pins:** 13 direct modules (go.mod:9-21), 12 indirect — **all 25 exactly match the checkout's `versions.json`** · **Status:** USED (the backbone)
**Adoption score:** **92/100 (very high) for the runtime core** — 23 of 25 applicable capabilities (the missed row: `projectionhost.Reset`; partials: schema/upcaster declarations, the `record.Record` envelope). Pooled naively with the prior catalog/testing audit: **≈81/100 library-wide (high)** — and that understates today, because the prior audit's four missed findings have all since closed.
**Prior report:** [2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html](../../research/2026-10-24_go-cqrs-lite-catalog-testing-modules-deep-dive.html) (58/100 — scoped ONLY catalog + scenario/systemscenario/benchkit, mid-adoption; its errata footer + `internal/cli/catalog.go:24,43` record the closures). This report covers everything that report did not: the runtime core.

## Verdict

The load-bearing half of the dependency is used at reference quality. `system.New` is the sole composition root (ADR-0123 verified clean: no `command.NewDispatcher`, no direct engine construction, no `decider.Repository`, no `stack.*` anywhere; the sqliteengine blank import lives exactly once, `app.go:8`). Every hard-won library contract is fenced by a named guard or test: bare record keys (`internal/calls/viewkey.go:10-22`), the one-shot drain choreography (`internal/cli/system.go:34-40`), create-fold-only pairing (`internal/calls/projection.go:89-93`), `WithSort` desc semantics, RFC3339Nano time filters. The domain deliberately stops short of ~14 subsystems it doesn't need — each refusal carries a reason — and the one genuinely missed capability (`Reset`) maps exactly onto a TODO already planned.

## Capability map (per module, at pin)

| Module | Key surface | pbx-lens use |
|---|---|---|
| `system/v4@v4.16.0` | `New(ctx, DomainConfig, DeploymentConfig)` (ADR-0123), coeffect/safety gates, role wiring, checkpoint store, accessors (`MetaEngine`, `CommandDispatcher`, `ProjectionHost`, `EventStore`, …) | ✅ composition root: Events/Commands/Projections/ProjectionTypeDecoder + 2 Instances (`internal/app/app.go:24-87`); accessors: `MetaEngine`, `CommandDispatcher`, `ProjectionHost` (+`EventStore` in systemscenario tests only) |
| `metaengine/v4@v4.20.0` | `Query[Q,R]` ADT, folds (`OnRecordTyped`, `Remove`), `Store`, `TypedReader`, ScanOptions, SSE/temporal/graph/vector/aggregations, failover, Explain | ✅ `Query` decl + `Volume(100_000)` + 3 create-folds + tombstone (`projection.go:94-126`); `NewReader().Get` (`cmd/pbx-lens/ops.go:40-78`); `Scan`+`WithSort`/`WithLimit` (`internal/retention/retention.go:84-121`); keyset walk + `WithFilter` (`internal/web/queries.go:98-117,225-227`); `Count(WithLimit(0))` (`internal/web/freshness.go:37-38`) |
| `metaengine/sqliteengine/v4@v4.6.0` | `"sqlite"` driver: full ADT on JSON columns, FTS ADT, typed WAL, StreamLogBackend, keyset pagination | ✅ blank import registers the driver (`app.go:8`); EngineConfig{Driver:"sqlite",DSN} / memory (`app.go:72-74`) |
| `metaengine/projectionadapter/v4@v4.6.2` | projectionhost bridge, bare-ID `EventWithID[P]`, `RegisterWithHost`, `NewTypeDecoder`, `Resettable` | ✅ (`projection.go:128-135`); `Resettable` never invoked (see missed) |
| `projectionhost/v4@v4.7.1` | managed workers, batch-drainer default, WorkerState machine, checkpoints, DLQ, `WithMetrics`, `WithOnFailed`, `Reset` | ✅ start/stop/status via `DrainProjections` (`system.go:41-74`); `WorkerFailed` → Transient (`system.go:79-100`) |
| `event/v4@v4.15.2` | typed CBOR events, `DecodePayloadAuto`, schema version, ISP store split | ✅ `DecodePayloadAuto` in the Apply fold (`decider.go:22-82`); `event.New(..., WithSchemaVersion(1))` (`decider.go:84-183`) |
| `command/v4@v4.14.1` | `BasicCommand`, Dispatcher, audit store | ✅ `command.New` ×3 handlers (`commands.go:30,55,95`; `app.go:32-60`) |
| `decider/v4@v4.8.0` | `Decider[State]{Initial,Apply}`, Repository, snapshots, time-travel | ✅ `Decider[calls.CallState]` via `system.RegisterDecider` (`app.go:27-30`); Repository deliberately unused (system-owned path) |
| `record/v4@v4.6.2` | journal envelope + metadata | Partial — only the ignored first fold parameter (`projection.go:99,106,113`) |
| `id/v4@v4.7.4` | branded ULID ids | ✅ `ParseStreamID`, `sid.Get()` record keys (`events.go:105-114`, `viewkey.go:16-22`) |
| `catalog/v4@v4.12.0` | eventcatalog export + architecture | ✅ both renderers incl. the v4.10.0 exporter (`internal/cli/catalog.go:6,24,40-45` — prior findings F5/F7 closed) |
| `scenario/v4@v4.4.3`, `systemscenario/v4@v4.2.1` | BDD harness | ✅ all tests boot via `app.DomainConfig()`/`app.DeploymentConfig(dsn)` (prior F1/F4 closed) |

Never touched (with reasons — N/A): QueryDispatcher/Publisher/Bus/watermill (in-process dispatch is the architecture; no distributed bus); otel (no collector); claiming/dedup (live-tail/queue modes never enabled — batch-drainer only); commandlifecycle (one aggregate, three commands); snapshot module (streams are a handful of events; the projection row IS the read snapshot); timers/scheduling (stateless poll loop — F009 decision, docs/cqrs-lint-policy.md:70-83); query module direct use (reads go through the sanctioned `metaengine.NewReader` path); `TypedReader.Exists` (correct avoidance — paid-for trap, docs/gotchas.md); event correlation/causation ids + command audit store (single-writer CLI; ContentHash idempotency + event log are the audit trail, F007); metaengine SSE/temporal `AsOf`/graph/vector/Counter/Multimap ADTs (no feature needs them; search is the repo-owned FTS5 shadow index, deliberate); distributed engines (disposable single-process sqlite/memory — stack decision).

## Top findings

1. **`projectionhost.Reset` (+ projectionadapter `Resettable`) is the one missed capability — and it fits a planned TODO exactly.** The planned one-time read-model rebuild that retires the legacy marker-prefixed keys (docs/gotchas.md record-key entry) is precisely Reset's job: drop checkpoint + clear read model → next `Start` replays from zero, no source re-fetch. Today's equivalent ("drop the file, re-poll") re-pays every API fetch. Small, high-fit.
2. **`WithOnFailed` unwired:** `DrainProjections` learns of worker death only by polling `Status()`; a callback would alert mid-drain. Cosmetic (the CLI surfaces Transient either way).
3. **`DomainConfig.Schema`/upcasters unused:** events written with `WithSchemaVersion(1)`, no `schema.EventSchema` declarations — deliberate at v1 (F005: mandatory at v2; distinct from the domain-level `ObservationSchemaVersion` refresh, `events.go:24-43`).
4. **`EngineConfig.Pragmas` unset:** sqlite runs on driver defaults; WAL/synchronous tuning is available if backup windows ever need it.

## Version currency

All 13 direct pins and all 12 indirect pins equal `versions.json` (= system/v4@v4.16.0's and projectionhost's own closures) — **maximally coherent, one release wave ahead of the prior audit** (which ran against catalog v4.9.0). Cache pseudo-versions (`snapshot@v4.7.1-0.2026…`, `decider@v4.6.1-0.2026…`, `catalog@v4.7.2-0.2026…`) show untagged master brewing — re-run the `git tag --contains` verification on the next bump. **v5 watchlist** (docs/gotchas.md 365-371 + AGENTS.md): no v5 tags upstream yet (verified 2026-10-26); the v5 cut will touch `system.New` constructor names — **pbx-lens has exactly one call site to fix (`app.go:87`)**. Never trust versions.json alone (the "versions.json lies" rule).

## Risks

- D6 upstream divergence (sqliteengine time-filter binding vs memory engine) — worked around in-repo (`queries.go:225-227`), issue pending; re-verify on bump.
- Pre-1.0-style minor churn in v4.x — the fence tests above are the insurance; keep them green on every bump.

## Recommendations

1. Use `projectionhost.Reset` for the planned legacy-key rebuild instead of drop-and-repoll.
2. On the next upstream tag wave: re-verify pins via `git tag --contains`, then consider wiring `WithOnFailed`.

## How to verify

```console
rg -n "system.New" internal cmd --type go | head          # one composition root
rg -n "OnRecordTyped|Remove\[" internal/calls/projection.go
go doc github.com/larsartmann/go-cqrs-lite/projectionhost/v4 Host.Reset
cat /home/lars/projects/go-cqrs-lite/versions.json        # pin parity
git -C /home/lars/projects/go-cqrs-lite tag --contains v4.16.0 -- 2>/dev/null | head -2  # the verification habit
```
