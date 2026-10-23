<!-- docs/DOMAIN_LANGUAGE.md — the ubiquitous language of pbx-lens.
     Domain terms and definitions, not implementation details or API docs. -->

# Domain Language

> The ubiquitous language of pbx-lens. Every term used in code, conversations,
> and docs is defined here unambiguously. If a term is not in this table, it
> does not belong in a pbx-lens conversation.

## Core terms

| Term                            | Definition                                                                                                                                                                                                                                                                          | Where used in code                                     |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| Observation                     | One row as seen by ONE source: the atomic unit of ingest. Immutable in practice; re-observing the same row must produce the same content.                                                                                                                                           | `calls.Observation`, `internal/calls/domain.go`        |
| SourceKind                      | Which family a source row came from: `freeswitch-cdr`, `telnyx-conversation`, `webhook-log`, `agent-transcript`.                                                                                                                                                                    | `calls.SourceKind`, `domain.go`                        |
| Direction                       | Call leg orientation: `inbound` or `outbound`. Caller-declared for Telnyx (its metadata exposes none); derived from dialplan context for FreeSWITCH.                                                                                                                                | `calls.Direction`, `domain.go`                         |
| NativeID                        | The source's own identifier for the row (CDR `uuid`, Telnyx conversation id). Never invented by the lens.                                                                                                                                                                           | `calls.NativeID`, `domain.go`                          |
| StreamID                        | `<kind>:<native-id>`. Deterministic, so re-ingesting the same source row targets the same event stream.                                                                                                                                                                             | `Observation.StreamID`, `domain.go`                    |
| ContentHash                     | sha256 over the canonical Observation fields. Equality of hashes IS equality of observations: same hash on a live stream replays to zero events; a different hash supersedes via `call.corrected` (sources grow between polls — a Telnyx conversation gains messages).              | `Observation.ContentHash`, `domain.go`                 |
| Call stream                     | The event stream behind one StreamID; type `Call`. Holds the current observation (`call.observed`, superseded by `call.corrected` when the snapshot changes), its enrichment, and optionally a `call.retired` tombstone.                                                            | `calls.StreamType`, `domain.go`                        |
| `call.record` / `call.observed` | The command (record this observation) and the event emitted when a stream is new; a changed snapshot on a live stream emits `call.corrected` instead.                                                                                                                               | `domain.go`                                            |
| Enrichment                      | Derived per-call data fetched from Telnyx — `CostMicros`/`CostCurrency` (Detail Records) and `InsightSummary` (conversations-insights, newest completed row). Hash-idempotent by value: an unchanged enrichment replays to zero events, and corrections carry it forward.           | `calls.Enrichment`, `internal/calls/domain.go`         |
| EnrichmentSeed                  | The poller's per-pass work list: streams that still lack cost/insight, so enrichment fetches stay lazy (≤1 detail-list request per pass) and already-final fields skip their fetch.                                                                                                 | `cli.SeedsFor`, `internal/cli/telnyx.go`               |
| MOS                             | The call's Mean Opinion Score (1.0–5.0 voice quality) when a source reports one. Zero means "not reported" — it is the domain's "no data", not a score, so the dashboard keeps its honest placeholder until the first score lands and averages only over scored calls (`MOSKnown`). | `Observation.MOS`, `internal/calls/domain.go`          |
| CostMicros / CostCurrency       | The call's billed cost from Telnyx Detail Records: an integer micro-unit amount (1 micro = 10⁻⁶ of the currency unit) plus its ISO currency. CostMicros zero means "not reported yet"; a later value arrives as `call.enriched` and is hash-idempotent thereafter.                  | `calls.Enrichment`, `internal/calls/domain.go`         |
| CallView                        | The read-model row the UI will render; one per stream, folded from `call.observed` into the `call_views` collection.                                                                                                                                                                | `calls.CallView`, `internal/calls/projection.go`       |
| Index (the lens)                | The disposable sqlite database of CallViews. Always re-derivable from sources; delete-and-rebuild is a supported operation, not a failure mode.                                                                                                                                     | `internal/app/app.go` (composition root)               |
| Master.csv                      | FreeSWITCH's CDR export file; ships in 13- and 18-column templates the parser recognizes.                                                                                                                                                                                           | `internal/ingest/cdr/cdr.go` (`ParseWithOptions`)      |
| FlexTime                        | Wire-level timestamp decoder for Telnyx: RFC3339Nano, then offset-less `2006-01-02T15:04:05`. Exists because Telnyx timestamps are precision-unstable across responses.                                                                                                             | `wire.FlexTime`, `internal/wire/flextime.go`           |
| Excerpt                         | Short transcript preview stored on the Observation: the first user turn (what did they want?), falling back to the first turn of any role, capped at `DefaultExcerptMax` (200 runes). The selection rule is domain-owned (`calls.Excerpt`); mappers feed turns in.                  | `calls.Excerpt`, `internal/calls/domain.go`            |
| TurnRole                        | The transcript speaker role — `user` or `assistant`; every source's vocabulary normalizes onto it (the agent's caller/agent maps via `normalizeTurn`). Constants are byte-identical to the pre-enum strings, so ContentHash is unchanged.                                           | `calls.TurnRole`, `internal/calls/domain.go`           |
| Age gate                        | Ingest-side retention: observations older than `-max-age` are refused with a `retention:age-cutoff` label instead of recorded (the ingest half of retention; the purge half is the tombstone). Shared by the serve poller and the one-shot ingest commands.                         | `cli.AgeGate`, `internal/cli/ingest.go`                |
| Recording correlation           | Client-side join matching a Telnyx recording to its conversation via the shared `call_control_id` metadata field.                                                                                                                                                                   | `RecordingByCallControl`, `map.go`                     |
| E164 / ParticipantHash          | PII-policy primitives: normalized number and its sha256-truncated hash. Load-bearing in the `?pii=hash` export mode.                                                                                                                                                                | `domain.go`, `web/export.go`                           |
| Tombstone (call.retired)        | The retention event that removes a call from every read surface while its stream stays append-only; re-retiring is a no-op, so an interrupted purge reruns safely.                                                                                                                  | `calls.CallRetired`, `internal/retention/retention.go` |
| Voice log                       | The transcript's presentation form: a terminal-style, top-down, tone-tagged line log (library Scrollback; the `?q=` search path hand-rolls the same look with per-turn anchors). A transcript is READING content, not a replay — no stagger animation.                              | `web.transcriptPane`, `internal/web/detail.templ`      |
| Signal palette                  | The teal theming contract: `app.css` overrides the library's semantic tokens (`--color-tc-primary/-hover/-info`) after the theme import, so every library blue re-skins from one block. Raw `*-blue-*` classes in pbx-lens templates are banned (gate: `TestNoRawPaletteClasses`).  | `internal/web/static/app.css`, `palette_gate_test.go`  |

## Transcript truth (2026-10-23)

**The lens performs NO speech-to-text.** `Transcript` is a verbatim mapping of
the source's own turns (Telnyx `user`/`assistant`; the FreeSWITCH agent's
`caller`/`agent`), so a transcript with no user turn means THE SOURCE wrote
none — never that a page filtered one out.

- **`SourceEvents`** — the source's non-speech rows (Telnyx `system` idle
  markers such as `[long silence]`, tool calls, blank text). They render as a
  muted "Source log" and are deliberately OUTSIDE `ContentHash`: hashing them
  would fork every stored view.
- **Idle markers** — match on role and shape only; the wording is explicitly
  not a contract. `[long silence]` means "no user turn arrived", NOT "the
  caller was silent". The 2026-10-23 forensic call had a human greeting and a
  closing line in the audio and zero user turns in the log (a greeting
  collision plus a turn that never closed before teardown).
- **Call-fact signals** on that shape: `no user turn`, `self-dial` (both
  parties the same number), and `Recording tail` (the recording runs well past
  the last turn).
- **Turn times** come from Telnyx's `sent_at`, never `created_at` — the latter
  is persistence time and can be a full minute late. Measured 2026-10-23 over
  the account's whole history: all 25 of 25 conversations carried wrong turn
  times under `created_at`, so this correction rewrote every Telnyx stream
  once (247 spoken turns, 37 markers across 6 streams).
- **A second ASR layer** is an owner-gated, clearly labelled variant that must
  never overwrite a source transcript.

## Signal thresholds — the single home for the NUMBERS (2026-10-23)

The vocabularies live above; the numbers live here, so a reader never has to
reverse-engineer `templ` or a scorer to learn what a bar actually is.

| Signal                    | Bar                                           | Where it fires                                        | What it does NOT mean                                                                                                                                                                                                                                                                  |
| ------------------------- | --------------------------------------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `no user turn` badge      | `len(Transcript) > 0 && CountRole(user) == 0` | call detail page (`internal/web/detail.templ`)        | not "the caller said nothing" — the SOURCE closed no turn                                                                                                                                                                                                                              |
| `source markers` badge    | `len(SourceEvents) > 0`                       | call detail page                                      | markers are not speech; they are the source's own notes                                                                                                                                                                                                                                |
| `self-dial` badge         | `FromNumber != "" && FromNumber == ToNumber`  | call detail page                                      | not proof of a misconfiguration — a test dial looks identical                                                                                                                                                                                                                          |
| `Recording tail` badge    | recording end − last turn START ≥ **20 s**    | call detail page (`transcriptTailThreshold`)          | not "nobody spoke"; it means the source stopped annotating long before the line dropped. 20 s, not 30: the forensic call's tail was 29.2 s while the far end was still speaking at 44.3 s                                                                                              |
| bench `no-user-turn` rule | `expectations.expectsUserTurns` tri-state     | `inkasso-bench` (pbx-artmann `internal/inkassobench`) | absent ⇒ **WARN** (fact, not verdict); `true` ⇒ **FAIL** (a real conversation must have a user turn); `false` ⇒ rule off (voicemail/answering-machine fixtures). The offline modes (`--transcript`, `--conversation`) always WARN — an archived call cannot prove the collector failed |

Two rules of interpretation, worth more than the table:

- **A WARN is a fact about the source; a FAIL is a claim about the agent.** Never
  promote the first to the second without a live call that could have had a
  user turn.
- **Span vs tail are different questions.** The forensic report asked for the
  speech span (first to last speech in the AUDIO); the page renders the
  transcript tail (last recorded TURN to line end). Both are honest; they
  disagree whenever the recognizer lost the audio, which is exactly the case
  the badge exists to expose.

## Additive fields and the schema version (2026-10-23)

`ContentHash` is the observation's identity, and the decider answers an equal
hash with a no-op. That makes a field which is persisted but NOT hashed
(`SourceEvents`) invisible to history: it lands only when something else moves
the hash.

- **The mechanism:** `calls.ObservationSchemaVersion` (and
  `EnrichmentSchemaVersion`) rides the hash pre-image. Bumping it forces
  exactly ONE correction per re-ingested stream — the deliberate, sized
  alternative to silent drift. Bump it only when an unhashed field must reach
  history, and never for a cosmetic edit.
- **The backfill:** `pbx-lens refresh -db <dsn>` fetches from the source and
  classifies every stream (new / stale / additive-only / unchanged /
  unbuildable) WITHOUT writing; `-apply` re-ingests and rebuilds the search
  index. It defaults to an unbounded look-back, because the 48 h poll window
  is exactly what leaves history behind.
- **The detector:** `additive-only` in that report is the alarm for "the hashes
  match but the stored row lacks data the source has" — the class that would
  otherwise never heal.
- **Delivery is never automatic for Telnyx history.** Local-file sources
  (CDR, webhook log, agent transcripts) re-parse on every poller pass and heal
  for free; Telnyx conversations cost API budget, so the backfill stays an
  explicit operator command.

## Read-time call join (2026-10-23)

The per-call detail page is a READ-TIME join (`CallDetailPage`): the requested
source row is completed from its sibling row that observed the same call.

- **What joins:** only the FreeSWITCH uuid is shared (CDR ↔ agent-transcript);
  Telnyx and webhook ids live in other key spaces, so they never participate.
- **Precedence:** the requested row's facts and identity always win;
  `Also indexed as` names what was borrowed.
- **Local WAVs** resolve by the dialplan's stems plus a `{uuid}_*` sweep,
  because a FreeSWITCH row indexes no `RecordingRef`.
- **The list and dashboard** still show one row per source (an AI call appears
  twice); the open decision lives in TODO_LIST.

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

| Surface                                            | Zone                           | Why                                                                               | Code                                    |
| -------------------------------------------------- | ------------------------------ | --------------------------------------------------------------------------------- | --------------------------------------- |
| Dashboard day buckets + heatmap hour buckets       | Europe/Berlin                  | "Calls today" must match the operator's calendar, incl. DST (`dashboardLocation`) | `internal/web/dashboard.templ`          |
| Bare date filter inputs (`dateValue`)              | Berlin wall date               | The operator types their own today/yesterday                                      | `internal/web/calls.templ`              |
| Detail "Started", search-hit `<time>` display text | Server-local (`Local()`)       | Deployment-tz wall clock; server runs Europe/Berlin in prod                       | `detail.templ:135`, `search.templ:63`   |
| Calls-table "When" column                          | Relative text (`RelativeTime`) | Scanning recency, not reading clocks                                              | `internal/web/calls.templ`              |
| Walk cursors (`before`/`after` query params)       | UTC RFC3339                    | Keyset pagination must be zone-unambiguous and sortable                           | `internal/web/calls.templ` (`listHref`) |
| Wire timestamps (`FlexTime`)                       | UTC at decode                  | Telnyx precision-unstable; normalized once at the boundary                        | `internal/wire/flextime.go`             |

Known mismatch (deliberate): detail/search render `Local()` while buckets
render Berlin — these coincide in prod (server tz = Berlin) and the split
keeps buckets DST-correct even when the server is not. Revisit only if the
deploy tz ever diverges (see owner decision pack D1 for the FORMAT
question, which is orthogonal to the ZONE rule above).

## Deprecated terms

| Old term   | Current term | Why changed                                                                                                                                   |
| ---------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| CallRecord | Observation  | Plan-era name (SUPERB plan T2); renamed because rows are per-source observations, not merged call records — merging happens in the read model |
