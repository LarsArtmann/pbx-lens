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
  + sqliteengine (blank import registers the "sqlite" driver), pinned to the
  vetted set from go-cqrs-lite `example/getting-started/go.mod` (system
  v4.11.0, metaengine v4.17.0, event v4.13.1, ...). ADR-0123: system.New is
  the only composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27 + `GOEXPERIMENT=jsonv2` (fleet house style, see
  go-output's flake). encoding/json/v2 is the JSON API once T4 needs it.
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family ONLY. No testify
  (banned; stdlib table-driven tests), no cobra, no ORM.
- **Flake:** go-standard flakeModule from go-nix-helpers (copy of go-output's
  pattern); `vendorHash.nix` starts as the fake hash — first `nix build` then
  `buildflow -s nix-hash-fix --fix` lands the real one.

## Domain

- `Observation` = one source row. `SourceKind`:
  freeswitch-cdr | telnyx-conversation | webhook-log | agent-transcript.
  `Direction`: inbound | outbound. Numbers are stored raw as observed (real
  CDRs carry unknown/empty parties); `E164` + `ParticipantHash` exist for the
  PII policy (TODO T23) and are NOT enforced on stored fields yet.
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
nix build                             # needs vendorHash.nix landed first
./scripts/...                         # (none yet)
```

## Gotchas (paid for)

- **metaengine fold selection (2026-10-10):** for a full-recompute view ship
  ONLY the 2-arg create fold (`OnRecordTyped` returning `(key, view)`).
  Pairing it with a 3-arg update fold lets metaengine route the FIRST event
  through the update fold with a zero `prev`, storing a zero view. Two-fold
  patterns exist for INCREMENTAL folds (`prev.Field += ...`), not recompute.
- **The coeffect gate's consumed set ignores `system.RawQuery`** (v4.11.0
  projection_builder passthrough): declaring `DomainConfig.Events` with a
  RawQuery projection logs an advisory "unconsumed event type" WARN at boot.
  Harmless; keep the declaration (the hard-error direction still guards
  deciders). The fluent `Evolve` + explicit-fold path feeds the gate but
  generates ONLY an update fold and keys by the view's `ID` field, which the
  closure cannot populate from the stream id — it panicked in poisoned-
  collection here. Revisit only if the library grows an explicit-create fold.
- **`errorfamily.NewRejection/NewConflict` take exactly (code, message)** —
  no format args; wrap with fmt.Sprintf. StreamID is a struct (cbid), not a
  string: use .String() / id.ParseStreamID, never a cast.
