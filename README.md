# pbx-lens

The insights and analytics platform for every call your PBX ever saw: who
called whom, when, how long, AI or human, what it cost, how good it was, and
where the transcript and recording live.

One sqlite index turns four disagreeing sources — FreeSWITCH CDRs, Telnyx
conversations/recordings, the receiver webhook log, and agent transcripts —
into one queryable history, then answers the operator's questions over it:
volume and seasonality, AI-vs-human mix, duration and MOS quality, cost per
call, hangup causes, transcript search, and machine-readable exports for
whatever comes next.

pbx-lens is READ-ONLY over every source: it parses FreeSWITCH CDR exports and
calls Telnyx read APIs (conversations, messages, recordings). It never writes
back to the PBX runtime and never mutates source logs. Its own database is
disposable: every row is re-derivable, so it can be deleted and rebuilt from
the sources at any time.

Built on [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) (system
composition root + metaengine sqlite engine).

## Status

Core ingest, enrichment (cost + AI insight summaries), and the read-only
analytics surface (list, detail, FTS search, dashboard, `/ops`, streaming
export) all work; Telnyx ingest is live-proven (8 real conversations
double-ingested with zero duplicate events). The dashboard is the current
analytics front door: volume, duration distribution, weekday×hour density,
AI/human mix, and MOS, all filterable by window. Deeper analytics
(segmentation by DID, cost trends, quality/anomaly views, saved views) is the
active direction — see [FEATURES.md](FEATURES.md) for the honest inventory,
including the analytics gaps, and [ROADMAP.md](ROADMAP.md) for the path.

## Installation

Requires Go 1.27+ (or Nix):

```console
go build -o pbx-lens ./cmd/pbx-lens                              # from a checkout
nix build                                                         # from a checkout (flake)
```

v0.2.0 is the latest tag; `go install github.com/larsartmann/pbx-lens/cmd/pbx-lens@v0.2.0`
works from a module proxy (v0.1.0 remains available).

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
`-dry-run` parses and age-gates the file and prints the would-record report
without touching any index (no `-db` required) — the zero-risk rehearsal
step before any production re-ingest.

### Index Telnyx AI calls

```console
export TELNYX_API_KEY="$(cat ~/.pbx-prod-secrets/telnyx_api_key)"
pbx-lens ingest-telnyx -db /var/lib/pbx-lens/calls.db -since 48h
```

Flags: `-key-file` (default reads `$TELNYX_API_KEY`), `-since` (Go duration
or RFC3339 timestamp, default `48h`), `-direction` (`outbound` default —
Telnyx conversation metadata exposes no direction field), `-limit N`
(max conversations, 0 = all), `-ai-did` (conversations touching these DIDs
index as inbound regardless of the declared direction), `-max-age` (refuse
older conversations; `0` keeps everything). Conversations are correlated
with their recordings via `call_control_id`; each call's Observation carries
a transcript excerpt and the recording reference. Like `ingest-cdr`,
`-dry-run` fetches the window and prints the would-record report without
touching any index (no `-db` required); `-api-base` points the fetch at a
stub server for rehearsals.

### Backfill history after a schema change (`refresh`)

```console
pbx-lens refresh -db /var/lib/pbx-lens/calls.db            # classify only, writes nothing
pbx-lens refresh -db /var/lib/pbx-lens/calls.db -apply     # re-ingest what is stale
```

`ContentHash` decides identity, and the decider answers an equal hash with a
no-op — so a field that is persisted but deliberately unhashed (the source's
non-speech markers) lands only when something else moves the hash. A
`calls.ObservationSchemaVersion` bump is how that is requested; `refresh` is
how it is delivered. It classifies every fetched stream (new, stale,
additive-only, unchanged, unbuildable) and prints the counts first, because
the counts ARE the measurement: how many streams will one deploy rewrite?
`-apply` then re-ingests and rebuilds the search index. Unlike
`ingest-telnyx` it looks back without a window by default — the 48 h poll
window is exactly what leaves history behind. Local-file sources (CDR,
webhook log, agent transcripts) need none of this: every poll pass re-parses
them for free.

### Serve the call-history UI

```console
pbx-lens serve -db /var/lib/pbx-lens/calls.db -addr 127.0.0.1:8300
```

![Dashboard: stat cards, 30-day volume, duration histogram, weekday×hour
density heatmap, AI/human donut, MOS card](docs/assets/dashboard-light.png)

The UI re-skins the component library's blues to the signal teal through one
design-token override — including the density heatmap, which is pinned to
`rgb(15, 118, 110)` by a built-CSS gate test and a browser-level style
contract (`scripts/verify-style-contract.js`), so the brand can't silently
drift back to the library violet.

![Call detail on mobile: copy-to-clipboard on the id and phone numbers, the
amber search-jump match log](docs/assets/detail-match-log-mobile.png)

A read-only web UI over the index: the calls list with direction/AI/date
filters, per-call detail with transcript and recording playback (the proxy
bounds each upstream hop to 30 s by default, `-recording-timeout`; the audio
body itself streams unbounded), FTS5
search (partial phone numbers match too: any run of 4+ digits finds the
call), a freshness banner, and a quality dashboard whose day buckets follow
the operator's wall clock (Europe/Berlin — "Calls today" means today, not
UTC's version of it; owner decision R2). Every chart shares the one teal
signal theme — the heatmap included — pinned by browser-level style-contract
probes, so accent colors never drift between pages. Dashboard cards poll for updates
every 30 s and the staleness banner re-checks itself every minute (htmx,
self-hosted, so a JS-less browser gets the same page rendered server-side).

The dashboard's analytics cards are denominator-honest — every rate names
what it divides: Spend averages priced calls only (calls Telnyx never priced
contribute nothing, and a mixed-currency window refuses to sum); the segment
tables (top PBX-side numbers and outside parties) show each segment's AI
share, average duration, and cost rate, and every row deep-links into the
filtered call list via `?number=`; the quality trend moves the no-user-turn
share week over week in percentage points, silently absent until the prior
week has a real baseline; and the density heatmap carries a weekday baseline
note — today's call count against that weekday's usual ±2σ band, quiet prose
rather than a new chart, and honest about "not enough history" until two
full weeks exist. Counterpart history on the detail page (the last meetings
with that outside party) rounds out the who-calls-whom picture.

Ingest keeps working
from the same CLI; `serve` also embeds an optional poll loop
(`-poll-every 15m -cdr-file … -webhook-log … -transcripts-dir …`) so the
index stays fresh without external timers. `-max-age 720h` refuses polled
observations older than the window at ingest (`0`, the default, keeps
everything) — the ingest-side half of retention.

Every row is also exportable without the UI: `/api/export.jsonl` and
`/api/export.csv` accept the same filters as the list page and stream the
whole index (keyset-paginated, bounded memory — no row cap). The response's
`X-PII-Mode` header names what left: `?pii=hash` swaps participant numbers
for their salted hashes before anything leaves the server.

For machine checks, `GET /api/ops` (same basic auth) serves the latest poll
pass's per-source offered/recorded/corrected/rejected counters plus the
response-cache section (`etag_generated`, `not_modified`, `hit_ratio`,
`buffer_overflows`) — the numbers the serve log prints, without grepping
the journal. Without a poll loop the poller section is simply absent (the
cache section always answers); a 503 with an explanation means neither
counter source is wired.

Other operational surfaces, all behind the same auth unless noted:
`GET /version` reports the build; `GET /metrics` serves Prometheus text
exposition (without basic auth by deliberate opt-in — the perimeter is the
bind address or firewall, see SECURITY.md); `/health`, `/health/live`, and
`/health/ready` are appkit-owned readiness probes. Every surface answers
conditional GETs with `ETag` → `304 Not Modified`, so htmx polls and
revisits cost zero body bytes while unchanged, and reports handler spans
as `Server-Timing`; HTML responses carry `Cache-Control: no-cache` so
browsers revalidate instead of silently serving stale pages.

**PII disclosure (R3):** `pii=hash` bounds participant identity ONLY.
Transcripts, excerpts, and insight summaries stay raw in every mode — see
the field-coverage matrix in FEATURES.md. If you need free text scrubbed
too, that is a separate feature; do not assume hash mode covers it.

Protect it with bcrypt basic auth for anything beyond loopback:

```console
htpasswd -B -c /etc/pbx-lens/htpasswd operator
pbx-lens serve -db calls.db -htpasswd /etc/pbx-lens/htpasswd
```

`/health` style readiness is appkit-owned; under systemd (`Type=notify`,
see `deploy/systemd/pbx-lens.service`) the binary sends `READY=1` after the
listener binds and feeds the watchdog.

### Purge old calls

```console
pbx-lens purge -db /var/lib/pbx-lens/calls.db -max-age 2160h [-dry-run] [-json]
```

Retention is tombstone-based: over-age calls get a `call.retired` event and
disappear from every read surface (list, search, export, dashboard), while
the event log stays append-only. Re-polling a purged call is a no-op, so
the view stays purged. Full erasure remains the disposable-DB move: drop
the file and re-ingest. `-json` turns both the dry run and the applied
purge into machine-readable reports for automation.

**Alerting recipe (ops/retention visibility):** two health facts live in
different places, and both belong in your alerter.

- _Is the poller alive?_ `GET /api/ops` (same auth as the UI) answers
  `last_pass_age_seconds`; alert when it exceeds a few poll intervals:

  ```console
  curl -fsS -u ops:… https://lens.internal/api/ops \
    | jq -e '.last_pass_age_seconds >= 0 and .last_pass_age_seconds < 600'
  ```

  (The age is `-1` when no pass ever ran — the `>= 0` guard makes "never
  ran" alert instead of silently passing.)

  `/ops` renders the same verdict chips for a human glance. What the card
  deliberately does NOT show: the purge timer's state and the last purge's
  age. Those are host-side facts (the timer runs out-of-band), so the
  process honestly cannot see them — alert on them at the host instead:

  ```console
  systemctl list-timers --no-pager | grep pbx-lens-purge   # timer armed + next run
  journalctl -u pbx-lens-purge.service --since -7d         # last pass + its report
  ```

### Back up the index

```console
pbx-lens backup -db /var/lib/pbx-lens/calls.db -out /var/backups/pbx-lens/$(date +%F).db
```

The snapshot is a consistent copy (`VACUUM INTO`) taken while `serve` keeps
writing, fsynced before it is renamed into place, and verified readable
afterwards. `deploy/systemd/pbx-lens-backup.{service,timer}` stage it
nightly; retention and off-host shipping stay with the host backup tool.

**Restore drill (R16):** restoring = placing a verified snapshot at a fresh
DSN path and pointing `serve` at it — the serve startup rebuild recreates
the search index. The full chain (snapshot → verify → restore → rows and
FTS hits match the source) is a checked-in test,
`internal/backup/restore_drill_test.go`, so "backups work" stays a proven
claim instead of a hope.

### Deep-link contract (for webphone and friends)

`/calls/{kind}/{id}` is a **stable URL**: ids are deterministic stream
identities (`<source-kind>:<native-id>`), so the same real-world call keeps
the same URL across re-ingests. Webphone (sibling repo) links INTO these
URLs read-only; pbx-lens never links back into webphone and holds no
webphone data. Bookmark and share freely — an unknown id renders a 404,
never a wrong call.

### Export the event/command catalog

```console
pbx-lens catalog -o docs/catalog       # DB-free: renders an EventCatalog project tree
```

Message schemas are derived from the command/event structs by reflection, so
the catalog cannot drift from the domain. `.eventcatalog-hub.json` points the
architecture-catalog hub at the exported tree (`-plain` emits plain
frontmatter-ID refs for the hub's governance lint).

## Project layout

- `cmd/pbx-lens/` — CLI (`version`, `smoke`, `ingest-cdr`, `ingest-telnyx`,
  `refresh`, `ingest-webhook`, `ingest-agent`, `serve`, `backup`, `purge`,
  `catalog`)
- `internal/calls/` — domain (Observation, invariants, idempotency) + read model
- `internal/cli/` — tested command internals: flags, dispatch loop, projection drain, poller, search rebuild
- `internal/ingest/` — source parsers/mappers (`cdr`, `telnyx`, `webhook`, `agent`)
- `internal/web/` — templ UI: list/detail/search/dashboard, recording proxy, auth, freshness, export
- `internal/search/` — FTS5 index over transcripts
- `internal/app/` — the ADR-0123 composition root (`system.New` wiring)
- `internal/retention/` — retention policy + tombstone purge
- `internal/backup/` — live-safe index DB backup command
- `internal/wire/` — shared wire-decoding helpers (`FlexTime`)
- `deploy/` — systemd unit + NixOS module drafts (the sibling repo owns the module)
- `docs/DOMAIN_LANGUAGE.md` — the vocabulary the code speaks
