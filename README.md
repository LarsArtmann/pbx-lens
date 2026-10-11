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

### Serve the call-history UI

```console
pbx-lens serve -db /var/lib/pbx-lens/calls.db -addr 127.0.0.1:8300
```

A read-only web UI over the index: the calls list with direction/AI/date
filters, per-call detail with transcript and recording playback, FTS5
search, a freshness banner, and a quality dashboard. Ingest keeps working
from the same CLI; `serve` also embeds an optional poll loop
(`-poll-every 15m -cdr-file … -webhook-log … -transcripts-dir …`) so the
index stays fresh without external timers.

Protect it with bcrypt basic auth for anything beyond loopback:

```console
htpasswd -B -c /etc/pbx-lens/htpasswd operator
pbx-lens serve -db calls.db -htpasswd /etc/pbx-lens/htpasswd
```

`/health` style readiness is appkit-owned; under systemd (`Type=notify`,
see `deploy/systemd/pbx-lens.service`) the binary sends `READY=1` after the
listener binds and feeds the watchdog.

### Deep-link contract (for webphone and friends)

`/calls/{kind}/{id}` is a **stable URL**: ids are deterministic stream
identities (`<source-kind>:<native-id>`), so the same real-world call keeps
the same URL across re-ingests. Webphone (sibling repo) links INTO these
URLs read-only; pbx-lens never links back into webphone and holds no
webphone data. Bookmark and share freely — an unknown id renders a 404,
never a wrong call.

## Project layout

- `cmd/pbx-lens/` — CLI (`version`, `smoke`, `ingest-cdr`, `ingest-telnyx`,
  `ingest-webhook`, `ingest-agent`, `serve`, `backup`)
- `internal/calls/` — domain (Observation, invariants, idempotency) + read model
- `internal/cli/` — tested command internals: flags, dispatch loop, projection drain, poller, search rebuild
- `internal/ingest/` — source parsers/mappers (`cdr`, `telnyx`, `webhook`, `agent`)
- `internal/web/` — templ UI: list/detail/search/dashboard, recording proxy, auth, freshness, export
- `internal/search/` — FTS5 index over transcripts
- `internal/backup/` — live-safe index DB backup command
- `internal/wire/` — shared wire-decoding helpers (`FlexTime`)
- `deploy/` — systemd unit + NixOS module drafts (the sibling repo owns the module)
- `docs/DOMAIN_LANGUAGE.md` — the vocabulary the code speaks
