# pbx-lens

One sqlite database answering, for every call the PBX ever saw: who, when,
how long, AI or human, and where the transcript lives.

pbx-lens is READ-ONLY over every source: it parses FreeSWITCH CDR exports and
calls Telnyx read APIs (conversations, messages, recordings). It never writes
back to the PBX runtime and never mutates source logs. Its own database is
disposable: every row is re-derivable, so it can be deleted and rebuilt from
the sources at any time.

Built on [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) (system
composition root + metaengine sqlite engine).

## Status

Ingest core works and is live-proven (Telnyx: 8 real conversations
double-ingested with zero duplicate events). The web UI is not built yet —
see [FEATURES.md](FEATURES.md) for the honest inventory and
[ROADMAP.md](ROADMAP.md) for direction.

## Installation

Requires Go 1.27+ (or Nix):

```console
go build -o pbx-lens ./cmd/pbx-lens                              # from a checkout
nix build                                                         # from a checkout (flake)
```

No version has been tagged yet; `go install ...@latest` will work once a
release exists.

## Quick start

```console
go run ./cmd/pbx-lens smoke                # in-memory pipeline, prints one CallView as JSON
go test ./...
```

## Usage

### Index FreeSWITCH CDRs

```console
pbx-lens ingest-cdr -db /var/lib/pbx-lens/calls.db /path/to/Master.csv
# ingest-cdr: 412 rows parsed, 3 parse-skipped, 412 recorded, 0 corrected, 0 rejected
```

Re-running the same file is safe: identical rows replay to zero events
(content-hash idempotency), and a changed snapshot of an existing stream is
superseded by a `call.corrected` event — the log keeps the full audit trail.

### Index Telnyx AI calls

```console
export TELNYX_API_KEY="$(cat ~/.pbx-prod-secrets/telnyx_api_key)"
pbx-lens ingest-telnyx -db /var/lib/pbx-lens/calls.db -since 48h
```

Flags: `-key-file` (default reads `$TELNYX_API_KEY`), `-since` (Go duration
or RFC3339 timestamp, default `48h`), `-direction` (`outbound` default —
Telnyx conversation metadata exposes no direction field), `-limit N`
(max conversations, 0 = all). Conversations are correlated with their
recordings via `call_control_id`; each call's Observation carries a transcript
excerpt and the recording reference.

## Project layout

- `cmd/pbx-lens/` — CLI (`version`, `smoke`, `ingest-cdr`, `ingest-telnyx`)
- `internal/calls/` — domain (Observation, invariants, idempotency) + read model
- `internal/cli/` — tested command internals: flags, dispatch loop, projection drain
- `internal/ingest/` — source parsers/mappers (`cdr`, `telnyx`)
- `internal/wire/` — shared wire-decoding helpers (`FlexTime`)
- `docs/DOMAIN_LANGUAGE.md` — the vocabulary the code speaks
