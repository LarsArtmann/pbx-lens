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
  Verified at source (`cmd/cqrs-lint/pkg/rules/correctness/c009.go:95`, same
  shape for `New` at :99) and covered by the open upstream filing
  [go-cqrs-lite#64](https://github.com/LarsArtmann/go-cqrs-lite/issues/64)
  (2026-10-12); if that lands, `mustRegister` can return to bare `must`.
- **E005 (false positive)**: "RecordCallCmd has no registered handler" —
  the handler IS registered, in `internal/app` via
  `system.RegisterCommand` (ADR-0123 composition root); the lint analyzes
  one package at a time and cannot see the wiring. Non-fix. Same class
  covers the 2026-10-11 `EnrichCallCmd` warning; also tracked in
  [go-cqrs-lite#64](https://github.com/LarsArtmann/go-cqrs-lite/issues/64).
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
  timezone is lost in the sqlite round-trip. The same reasoning covers the
  2026-10-25 F025-shaped warning on the same field (rule id F025, message
  "timezone may be lost in SQL storage"): the stored form is the RFC3339Nano
  TEXT the queries bind, not a driver-native time.
- **C013-adjacent (deliberate): `SourceEvent.At time.Time`** — markers are
  display-only (a source's own non-speech notes, truncated to whole seconds
  at the mapper) and never participate in ordering or filtering; the JSON
  payload keeps the zone. A CBOR epoch round-trip on a display field is not a
  correctness risk here.
- **P006 (deliberate)**: the smoke command polls `reader.Get` with a 20 ms
  sleep under a 5 s context deadline — a one-shot CLI convergence wait, no
  hot path. `Watcher.Watch` is change-only (no current-value replay), so
  switching would trade a bounded poll for a subscribe race.
- **F009 (deliberate, poller)**: `internal/cli/poller.go` uses a hand-rolled
  `time.NewTicker`; the scheduling module's durable TimerStore buys nothing
  for a stateless re-ingest loop whose only failure mode is "poll again next
  tick". Same class of one-shot/periodic CLI wait as P006.
  Suppression-channel note (2026-10-18, T10): cqrs-lint v4.17.0 has NO
  inline suppression (it does not consume `//nolint:...` or any
  `//cqrs-lint:ignore` comment — verified against `pkg/rules/adoption/f009.go`
  and the analyzer source) and no per-path exclude in the config loader; the
  only channels are the preset's `rules.disable` list (repo-wide) and preset
  choice. For one deliberate finding like this F009, repo-wide disable would
  over-suppress, so the surviving channel is this documented-decision file.
  If an inline-suppression mechanism lands upstream, migrate the surviving
  warning-level findings to it (an upstream suggestion is drafted in
  docs/drafts/ only if the owner wants one filed — none is queued).
- **D014 (deliberate)**: `CallCorrected.Enrichment` carries no json tag
  because the field is not persisted as a flat column; the whole event
  payload is stored as the stream record. Adding a tag would be noise.
- **F026 (inapplicable, verified 2026-10-25)**: `WithPrefetch` is not a
  ScanOption — it attaches a `PrefetchCache` to the READER and only caches
  results for CURSOR-based pagination (`typed_reader_scan.go:59`), while
  every reader in this repo pages by a `started_at` filter. The rule fires on
  `internal/retention.OldViews` (one bounded scan, `maxScan`) and the
  agent-source readers (bounded result sets); prefetch would cache nothing.
  Non-fix; the rule's own suggestion text is inaccurate for filter-paged
  scans (an upstream-filing candidate, not filed).
- **F031 (false positive at helper-built scans)**: "Scan without WithLimit"
  fires on `internal/web/queries.go` at the list walk and the NextNewer
  probe. Both DO bound the result: the walk passes `WithLimit(listScanBatch+1)`
  at the call site (spelled out 2026-10-25) and the probe carries
  `WithLimit(1)` inside its option slice. The rule reads the Scan arguments
  literally and cannot see a limit inside a slice built by a helper. The
  absence of a limit is a real bug class, so the limit stays VISIBLE at the
  walk call site; the probe slice keeps the helper shape. Non-fix.
- **F021 (accepted: four folds are the contract)**: `call_views` needs four
  create folds (`call.observed`, `call.corrected`, `call.enriched`) plus the
  retirement remove — one per event type, each a FULL recompute of the row
  (the pairing rule in AGENTS: a create fold with an update fold stores a
  zero view). Batching them would mean one handler per event type anyway;
  write amplification is one row write per event, which is the model, not an
  accident.
- **B004/B005 (informational)**: `cqrs-gen` constructors and
  `decider.StrictApply` were both considered; StrictApply rejected above
  (loses the Corruption family), and handwritten constructors stay because
  the flat domain has only two commands.
- **B010 (informational, declined — re-verified 2026-10-26)**: the four
  explicit `catalog.Event[...]` registrations in `BuildCatalog` are the
  whole catalog; `cqrs-gen`'s struct-tag codegen would buy nothing at this
  size and cost a generated-code layer the repo deliberately avoids
  (AGENTS: generated vs handwritten — right tool for the job). Revisit on
  the E003 trigger (a second aggregate), where registration count stops
  being trivial. The 2026-10-26 pass found the census UNCHANGED at 19
  findings (this B010 row was the only code missing a disposition).
