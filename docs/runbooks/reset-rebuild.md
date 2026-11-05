# Runbook: one-shot read-model rebuild (legacy-key retirement, TODO_LIST row 113)

**Status:** DESIGNED + memory-engine-proven (`TestHostResetRebuildsReadModel` and
`TestResetRebuildsReadModelFromJournal`, both in `internal/app`) — the EXECUTION
is owner-gated: it needs the prod deploy path (row 113 stays BLOCKED until Lars
runs it). Rehearse it against a copy of the production DB with
`pbx-lens serve -db <copy>` before the first real run.

## Why

Rows written before the projection adapter's bare-key flip stay keyed with
the marker-prefixed form. They stay READABLE through `calls.LookupView`'s
legacy fallback (docs/gotchas.md owns the incident), but the fallback can
never be retired while those rows exist. A full rebuild re-keys every view
bare and lets the fallback die.

## The procedure (per deploy)

The whole rebuild is three calls; the first two live in
`cli.ResetReadModel`, so the code path and the runbook cannot drift.

1. **Stop the writer.** `systemctl stop pbx-lens` (or pause the poller) —
   a mid-reset poll would re-fold onto cleared engines and double-apply.
2. **Reset the read model and its checkpoint in one call.**
   `cli.ResetReadModel(ctx, sys)`:

   ```go
   sys.ProjectionHost().Stop()                        // Host.Reset refuses while running
   sys.ResetProjection(ctx, app.ProjectionName)       // "projections"
   ```

   `projectionhost.Host.Reset(ctx, name)` (`host_reset.go:76` at v4.7.1)
   drops the projection's checkpoint FIRST, then calls the projection's
   `Resettable.Reset` — and the metaengine projection adapter is
   `Resettable` (`projectionadapter/reset.go`), delegating to
   `metaengine.Store.Reset`. Checkpoint-then-state is the safe order: dying
   between the two replays from zero ON TOP of stale state, which idempotent
   handlers absorb; the reverse order would silently skip pre-checkpoint
   events. The JOURNAL IS NEVER TOUCHED (ADR-0143: the journal is the replay
   source and a fact on the invertibility ladder).

   `Host.Reset` rejects two cases loudly: `projectionhost.reset_while_running`
   (hence the Stop) and `projectionhost.unknown_projection` (a name other than
   the system constructor's `"projections"`).
3. **Watch for the partial-reset warning.** The adapter logs at WARN when the
   underlying engine cannot be bulk-cleared (`ResetResult.Partial()`); every
   first-party engine implements `metaengine.EngineResetter`, so this repo's
   sqlite engine is clearable and the warning means a wiring change. A partial
   reset is the loud-stop signal: stale state would survive UNDER the replay
   and double-apply.
4. **Replay from the journal.** `cli.DrainProjections(ctx, sys)` — the
   production drain (host.Stop → host.Start → poll until every worker is
   stopped/failed) replays the whole journal through the projection
   adapter. Every view re-keys bare.
5. **Verify count parity.** Views after == views before (the memory-engine
   dry run pins exactly this: 2 → 0 → 2). A shortfall means the journal lost
   a stream — stop and investigate, do not re-run blindly.
6. **Restart** and let the poller re-drain; the FTS index rebuild rides the
   next poll pass (serve.go's PollLoop drains + rebuilds every pass).

## Rollback

The read model is disposable and always re-derivable — there is nothing to
restore. If the rebuild goes wrong:

1. Re-run `ResetReadModel` (back to empty; the journal never moved).
2. Re-drain (`DrainProjections`). The result is deterministic: same journal,
   same views. The checkpoint is re-created implicitly by the replay — workers
   rebuild their positions from zero (verified by the dry run: post-reset
   count 0 → replay → full parity).
3. If the journal itself is suspect: `pbx-lens backup -db <dsn> -out
   <snapshot.db>` (VACUUM INTO + integrity verify) BEFORE the rebuild gives
   the point-in-time copy to compare against. The index DB stays
   disposable; the SOURCES (CDR CSVs, Telnyx, webhook JSONL) stay
   authoritative — worst case is a full re-ingest, never data loss.

## Owner gates (unchanged, row 113)

- Prod deploy path (where does `go run ./cmd/pbx-lens` become a service?
  the NixOS `modules/telephony/lens.nix` deploy).
- The one-shot timing: right after a deploy, before the poller's first pass.
