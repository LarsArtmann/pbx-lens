<!-- docs/DOMAIN_LANGUAGE.md — the ubiquitous language of pbx-lens.
     Domain terms and definitions, not implementation details or API docs. -->

# Domain Language

> The ubiquitous language of pbx-lens. Every term used in code, conversations,
> and docs is defined here unambiguously. If a term is not in this table, it
> does not belong in a pbx-lens conversation.

## Core terms

| Term                            | Definition                                                                                                                                                                                                                                                                         | Where used in code                                     |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| Observation                     | One row as seen by ONE source: the atomic unit of ingest. Immutable in practice; re-observing the same row must produce the same content.                                                                                                                                          | `calls.Observation`, `internal/calls/domain.go`        |
| SourceKind                      | Which family a source row came from: `freeswitch-cdr`, `telnyx-conversation`, `webhook-log`, `agent-transcript`.                                                                                                                                                                   | `calls.SourceKind`, `domain.go`                        |
| Direction                       | Call leg orientation: `inbound` or `outbound`. Caller-declared for Telnyx (its metadata exposes none); derived from dialplan context for FreeSWITCH.                                                                                                                               | `calls.Direction`, `domain.go`                         |
| NativeID                        | The source's own identifier for the row (CDR `uuid`, Telnyx conversation id). Never invented by the lens.                                                                                                                                                                          | `calls.NativeID`, `domain.go`                          |
| StreamID                        | `<kind>:<native-id>`. Deterministic, so re-ingesting the same source row targets the same event stream.                                                                                                                                                                            | `Observation.StreamID`, `domain.go`                    |
| ContentHash                     | sha256 over the canonical Observation fields. Equality of hashes IS equality of observations: same hash on a live stream replays to zero events; a different hash supersedes via `call.corrected` (sources grow between polls — a Telnyx conversation gains messages).             | `Observation.ContentHash`, `domain.go`                 |
| Call stream                     | The event stream behind one StreamID; type `Call`. Holds the current observation (`call.observed`, superseded by `call.corrected` when the snapshot changes), its enrichment, and optionally a `call.retired` tombstone.                                                           | `calls.StreamType`, `domain.go`                        |
| `call.record` / `call.observed` | The command (record this observation) and the event emitted when a stream is new; a changed snapshot on a live stream emits `call.corrected` instead.                                                                                                                              | `domain.go`                                            |
| Enrichment                      | Derived per-call data fetched from Telnyx — `CostMicros`/`CostCurrency` (Detail Records) and `InsightSummary` (conversations-insights, newest completed row). Hash-idempotent by value: an unchanged enrichment replays to zero events, and corrections carry it forward.          | `calls.Enrichment`, `internal/calls/domain.go`         |
| EnrichmentSeed                  | The poller's per-pass work list: streams that still lack cost/insight, so enrichment fetches stay lazy (≤1 detail-list request per pass) and already-final fields skip their fetch.                                                                                                | `cli.SeedsFor`, `internal/cli/telnyx.go`               |
| CallView                        | The read-model row the UI will render; one per stream, folded from `call.observed` into the `call_views` collection.                                                                                                                                                               | `calls.CallView`, `internal/calls/projection.go`       |
| Index (the lens)                | The disposable sqlite database of CallViews. Always re-derivable from sources; delete-and-rebuild is a supported operation, not a failure mode.                                                                                                                                    | `internal/app/app.go` (composition root)               |
| Master.csv                      | FreeSWITCH's CDR export file; ships in 13- and 18-column templates the parser recognizes.                                                                                                                                                                                          | `internal/ingest/cdr/cdr.go` (`ParseWithOptions`)      |
| FlexTime                        | Wire-level timestamp decoder for Telnyx: RFC3339Nano, then offset-less `2006-01-02T15:04:05`. Exists because Telnyx timestamps are precision-unstable across responses.                                                                                                            | `wire.FlexTime`, `internal/wire/flextime.go`           |
| Excerpt                         | Short transcript preview stored on the Observation: the first user turn (what did they want?), falling back to the first turn of any role, capped at `DefaultExcerptMax` (200 runes). The selection rule is domain-owned (`calls.Excerpt`); mappers feed turns in.                 | `calls.Excerpt`, `internal/calls/domain.go`            |
| TurnRole                        | The transcript speaker role — `user` or `assistant`; every source's vocabulary normalizes onto it (the agent's caller/agent maps via `normalizeTurn`). Constants are byte-identical to the pre-enum strings, so ContentHash is unchanged.                                          | `calls.TurnRole`, `internal/calls/domain.go`           |
| Age gate                        | Ingest-side retention: observations older than `-max-age` are refused with a `retention:age-cutoff` label instead of recorded (the ingest half of retention; the purge half is the tombstone). Shared by the serve poller and the one-shot ingest commands.                        | `cli.AgeGate`, `internal/cli/ingest.go`                |
| Recording correlation           | Client-side join matching a Telnyx recording to its conversation via the shared `call_control_id` metadata field.                                                                                                                                                                  | `RecordingByCallControl`, `map.go`                     |
| E164 / ParticipantHash          | PII-policy primitives: normalized number and its sha256-truncated hash. Load-bearing in the `?pii=hash` export mode.                                                                                                                                                               | `domain.go`, `web/export.go`                           |
| Tombstone (call.retired)        | The retention event that removes a call from every read surface while its stream stays append-only; re-retiring is a no-op, so an interrupted purge reruns safely.                                                                                                                 | `calls.CallRetired`, `internal/retention/retention.go` |
| Voice log                       | The transcript's presentation form: a terminal-style, top-down, tone-tagged line log (library Scrollback; the `?q=` search path hand-rolls the same look with per-turn anchors). A transcript is READING content, not a replay — no stagger animation.                             | `web.transcriptPane`, `internal/web/detail.templ`      |
| Signal palette                  | The teal theming contract: `app.css` overrides the library's semantic tokens (`--color-tc-primary/-hover/-info`) after the theme import, so every library blue re-skins from one block. Raw `*-blue-*` classes in pbx-lens templates are banned (gate: `TestNoRawPaletteClasses`). | `internal/web/static/app.css`, `palette_gate_test.go`  |

## Bounded contexts

| Term         | Telnyx context                                                                            | FreeSWITCH context                                                                                   |
| ------------ | ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Conversation | An AI assistant's message thread (`/ai/conversations`) — the unit ingested                | Not a concept; a call is the CDR row identified by `uuid`                                            |
| Duration     | Recording `duration_millis` when a recording matched, else `last_message_at − created_at` | `billsec` from the CDR row                                                                           |
| Direction    | Caller-declared at ingest (`-direction` flag); no source field                            | Derived: dialplan `context == "public"` means inbound                                                |
| AI-ness      | `assistant_id` present in conversation metadata                                           | Destination badge: `destination_number` matching an AI extension (`cdr.ParseOptions.AIDestinations`) |

## Time vocabulary (owner R2, 2026-10-12; audited 2026-10-15)

One rule: **the operator's wall clock is Europe/Berlin; the wire and the
query string are UTC.**

| Surface                                             | Zone          | Why                                                                                    | Code                                                            |
| --------------------------------------------------- | ------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| Dashboard day buckets + heatmap hour buckets        | Europe/Berlin | "Calls today" must match the operator's calendar, incl. DST (`dashboardLocation`)       | `internal/web/dashboard.templ`                                  |
| Bare date filter inputs (`dateValue`)               | Berlin wall date | The operator types their own today/yesterday                                        | `internal/web/calls.templ`                                      |
| Detail "Started", search-hit `<time>` display text  | Server-local (`Local()`) | Deployment-tz wall clock; server runs Europe/Berlin in prod                  | `detail.templ:135`, `search.templ:63`                           |
| Calls-table "When" column                           | Relative text (`RelativeTime`) | Scanning recency, not reading clocks                                  | `internal/web/calls.templ`                                      |
| Walk cursors (`before`/`after` query params)        | UTC RFC3339   | Keyset pagination must be zone-unambiguous and sortable                                | `internal/web/calls.templ` (`listHref`)                         |
| Wire timestamps (`FlexTime`)                        | UTC at decode | Telnyx precision-unstable; normalized once at the boundary                             | `internal/wire/flextime.go`                                     |

Known mismatch (deliberate): detail/search render `Local()` while buckets
render Berlin — these coincide in prod (server tz = Berlin) and the split
keeps buckets DST-correct even when the server is not. Revisit only if the
deploy tz ever diverges (see owner decision pack D1 for the FORMAT
question, which is orthogonal to the ZONE rule above).

## Deprecated terms

| Old term   | Current term | Why changed                                                                                                                                   |
| ---------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| CallRecord | Observation  | Plan-era name (SUPERB plan T2); renamed because rows are per-source observations, not merged call records — merging happens in the read model |
