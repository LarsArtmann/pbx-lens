# AGENTS.md — pbx-lens

Unified call-history index for pbx.artmann.tech (the SUPERB plan:
`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).
Read-only over every source: FreeSWITCH CDRs, Telnyx conversations/recordings
APIs, the receiver webhook JSONL. The index DB is disposable and always
re-derivable; never add a write-back path to any source. The sibling repo
(nix-international-telephony) later gets only a thin NixOS deploy module;
webphone later gets a read-only deep link, never a data dependency.

## Stack decisions

- **Modules:** `go-cqrs-lite` `system/v4` + `metaengine/v4` + projectionadapter
  - sqliteengine (blank import registers the "sqlite" driver), pinned to the
    vetted set from go-cqrs-lite `example/getting-started/go.mod` (exact
    versions live in go.mod — never re-enumerate them here). ADR-0123: system.New is
    the only composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27 + `GOEXPERIMENT=jsonv2` (fleet house style, see
  go-output's flake). Nothing imports encoding/json/v2 yet (plain
  `go test ./...` passes without the env var); it becomes load-bearing when a
  surface needs it.
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family ONLY. No testify
  (banned; stdlib table-driven tests), no cobra, no ORM.
- **Flake:** go-standard flakeModule from go-nix-helpers (copy of go-output's
  pattern); `vendorHash.nix` carries the landed vendor hash. If a go.mod/
  go.sum change breaks the nix build, repair with
  `buildflow -s nix-hash-fix --fix` — never paste hashes by hand.

## Domain

- `Observation` = one source row. `SourceKind`:
  freeswitch-cdr | telnyx-conversation | webhook-log | agent-transcript.
  `Direction`: inbound | outbound. Numbers are stored raw as observed (real
  CDRs carry unknown/empty parties); `E164` + `ParticipantHash` exist for the
  planned PII policy (see TODO_LIST.md) and are NOT enforced on stored fields
  yet.
- Stream id = `<kind>:<native-id>` — deterministic, so re-ingest is the same
  stream and `call.record` is idempotent (same ContentHash replays to zero
  events; a different hash on a live stream is a conflict — CDR rows are
  immutable, so mutation means a parsing bug).
- Read model: `call_views` collection, one `CallView` per stream, folded from
  `call.observed` via `metaengine.OnRecordTyped`.

## Commands

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
buildflow --build-mode dev            # the quality gate (lint+format+test)
nix build                             # BROKEN: exits 0 but ships an empty
                                      # result (no bin/pbx-lens) — see TODO_LIST
./scripts/...                         # (none yet)
```

## Gotchas (paid for)

- **metaengine fold selection:** for a full-recompute view ship
  ONLY the 2-arg create fold (`OnRecordTyped` returning `(key, view)`).
  Pairing it with a 3-arg update fold lets metaengine route the FIRST event
  through the update fold with a zero `prev`, storing a zero view. Two-fold
  patterns exist for INCREMENTAL folds (`prev.Field += ...`), not recompute.
- **The coeffect gate's consumed set ignores `system.RawQuery`**
  (projection_builder passthrough): declaring `DomainConfig.Events` with a
  RawQuery projection logs an advisory "unconsumed event type" WARN at boot.
  Harmless; keep the declaration (the hard-error direction still guards
  deciders). The fluent `Evolve` + explicit-fold path feeds the gate but
  generates ONLY an update fold and keys by the view's `ID` field, which the
  closure cannot populate from the stream id — it panicked in poisoned-
  collection here. Revisit only if the library grows an explicit-create fold.
- **`errorfamily.NewRejection/NewConflict` take exactly (code, message)** —
  no format args; wrap with fmt.Sprintf. StreamID is a struct (cbid), not a
  string: use .String() / id.ParseStreamID, never a cast.
- **Telnyx live-API truths (verified against the live API 2026-10-10; unit
  fixtures alone missed all three — re-verify this cluster quarterly):
  1. Conversations filters are FIELD-AS-PARAM (`last_message_at=gte.TS`,
     `metadata->assistant_id=eq.x`) — NOT a `filter=` wrapper; the endpoint
     400s (10015) listing its available params when you guess wrong.
  2. Conversation metadata values are NOT all strings (`called_tools` is an
     array despite the docs typing metadata.* as string) — decode as
     `map[string]any` and read through `Conversation.MetadataString`.
  3. Timestamps are precision-UNSTABLE across responses (the same
     conversation served nanoseconds then microseconds minutes apart) and
     recordings carry OFFSET-LESS stamps (`2026-10-09T19:51:35`). Wire DTOs
     decode through `FlexTime` (RFC3339Nano, then bare
     `2006-01-02T15:04:05`, UTC); the MAPPER truncates to whole seconds
     before building the Observation — sub-second noise must never reach
     `ContentHash`, or every re-poll rejects the stream as `call.record.
     mutated`. Regression test:
     `TestMapConversationTimestampPrecisionStable`.
- **`FlexTime` promotes time.Time methods on the RECEIVER only** —
  `conv.CreatedAt.UTC()` compiles, but `last.After(first)` does NOT accept a
  FlexTime argument; write `.Time.After(x.Time)` explicitly.
- **The conversation→Observation direction is caller-declared** (MapOptions.
  Direction, CLI `-direction`): Telnyx conversation metadata exposes no
  direction field (docs read 2026-10-10), so this deployment's assistant
  calls default to outbound. Revisit when inbound AI calls land (the DID
  interception path).
- **Recording correlation is client-side**: conversations carry
  `call_control_id` in metadata, recordings carry the same field, so
  ingest-telnyx lists recent recordings and matches in-process
  (`RecordingByCallControl`) — the recordings list filter's supported keys
  are undocumented, so no server-side filter is trusted.

## cqrs-lint findings policy (0 errors; every surviving warning/info below is a documented decision)

Run: `go build -o /tmp/cqrs-lint ~/projects/go-cqrs-lite/cmd/cqrs-lint` then
`/tmp/cqrs-lint .` (the `./...` pattern crashes the package loader — use
`.`). Gate: ERROR severity fails; warnings/infos below are each a DECISION.

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
  one package at a time and cannot see the wiring. Non-fix.
- **E003 (deliberate flat domain, OWNER QUESTION OPEN)**: `internal/calls`
  mixes command/event/fold by design — the domain is one aggregate
  (Observation) with one projection; a split today would be ceremony, not
  boundary. Revisit when a second aggregate or T9 (agent transcripts) lands;
  the split-vs-policy decision stays open pending the owner's answer.
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
