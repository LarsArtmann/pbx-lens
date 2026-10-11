# cqrs-lint findings policy (pbx-lens)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit).

Run: `nix develop -c cqrs-lint .` (cqrs-lint ships in the flake devShell since
2026-10-11 — no more /tmp binary; the `./...` pattern crashes the package
loader — use `.`). Gate: ERROR severity fails; warnings/infos below are each a
DECISION.

- **C003 (ERROR, fixed)**: the fold is TOTAL — unknown event types on a
  Call stream return `errorfamily.NewCorruption("call.fold.unknown_event")`
  instead of silently ignoring (the original `if evt.Type() != X` shape was
  the bug class the rule exists for). `decider.StrictApply` was considered
  and REJECTED: it wraps with a family-less `fmt.Errorf`, losing the
  Corruption classification this repo's error doctrine requires.
- **C009 (fixed via `mustRegister`)**: the panic-in-production rule exempts
  functions named `must*`/`Must*` with `len(name) > 4` — a helper literally
  named `must` is NOT exempt. The composition root uses `mustRegister`.
- **E005 (false positive)**: "RecordCallCmd has no registered handler" —
  the handler IS registered, in `internal/app` via
  `system.RegisterCommand` (ADR-0123 composition root); the lint analyzes
  one package at a time and cannot see the wiring. Non-fix. Same class
  covers the 2026-10-11 `EnrichCallCmd` warning.
- **E014 (false positive, same package-blindness class)**: "uses
  projectionhost but has no projection drain/sync/flush call" — the drain
  lives in `internal/cli.DrainProjections` and EVERY dispatcher (one-shot
  commands, serve startup, poller passes) calls it explicitly; the lint
  sees `internal/calls` alone. Non-fix.
- **P-class Close() ignored in `cli.OpenSystem` cleanup (deliberate)**:
  one-shot CLI processes close the system at exit; a close error there has
  no recovery path and the event log is already durable. The serve process
  owns a stricter shutdown.
- **E003 (flat domain, DECIDED 2026-10-11: HOLD, seam pre-agreed)**: `internal/calls`
  mixes command/event/fold by design — the domain is one aggregate
  (Observation) with one projection; a split today would be ceremony, not
  boundary. Owner decision: keep flat. IF a second aggregate or an external
  Go consumer ever lands, the pre-agreed seam is write/read:
  `internal/calls` (aggregate: Observation, commands, events, CallState,
  Apply) vs `internal/callviews` (CallView, ViewFrom, Projection());
  ingest+cli depend on the write half, web+search on the read half, and
  retention legitimately spans both (scan reads → dispatch writes).
- **F007 (superseded by domain-level idempotency)**: the middleware rule
  wants `middleware.CommandIdempotency` with a durable claim store. This
  app's idempotency is STRONGER at the domain layer: ContentHash replay
  emits zero events (tested), and handlers have no side effects beyond the
  deduplicating append — a claim store would be redundant state. Non-fix;
  revisit only if a handler grows an external side effect.
- **F005 (expected at v1)**: schema version 1 has no historical payloads to
  upcast. Adding an upcaster is mandatory work at v2, not now.
- **C013 (deliberate)**: `CallView.StartedAt time.Time` — the view is
  serialized as JSON (not CBOR epoch) and `ViewFrom` forces `.UTC()`, so no
  timezone is lost in the sqlite round-trip.
- **P006 (deliberate)**: the smoke command polls `reader.Get` with a 20 ms
  sleep under a 5 s context deadline — a one-shot CLI convergence wait, no
  hot path. `Watcher.Watch` is change-only (no current-value replay), so
  switching would trade a bounded poll for a subscribe race.
- **F009 (deliberate, poller)**: `internal/cli/poller.go` uses a hand-rolled
  `time.NewTicker`; the scheduling module's durable TimerStore buys nothing
  for a stateless re-ingest loop whose only failure mode is "poll again next
  tick". Same class of one-shot/periodic CLI wait as P006.
- **D014 (deliberate)**: `CallCorrected.Enrichment` carries no json tag
  because the field is not persisted as a flat column; the whole event
  payload is stored as the stream record. Adding a tag would be noise.
- **F026 (deliberate at v1)**: the readers in `internal/cli/agent.go` scan
  bounded result sets (transcripts, recent recordings); `WithPrefetch` only
  pays off on large cursor pages. Revisit when a source grows past a few
  hundred rows/pass.
- **B004/B005 (informational)**: `cqrs-gen` constructors and
  `decider.StrictApply` were both considered; StrictApply rejected above
  (loses the Corruption family), and handwritten constructors stay because
  the flat domain has only two commands.
