<!-- docs/DOMAIN_LANGUAGE.md — the ubiquitous language of pbx-lens.
     Domain terms and definitions, not implementation details or API docs. -->

# Domain Language

> The ubiquitous language of pbx-lens. Every term used in code, conversations,
> and docs is defined here unambiguously. If a term is not in this table, it
> does not belong in a pbx-lens conversation.

## Core terms

| Term                          | Definition                                                                                          | Where used in code                            |
| ----------------------------- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Observation                   | One row as seen by ONE source: the atomic unit of ingest. Immutable in practice; re-observing the same row must produce the same content. | `calls.Observation`, `internal/calls/domain.go:77` |
| SourceKind                    | Which family a source row came from: `freeswitch-cdr`, `telnyx-conversation`, `webhook-log`, `agent-transcript`. | `calls.SourceKind`, `domain.go:25`            |
| Direction                     | Call leg orientation: `inbound` or `outbound`. Caller-declared for Telnyx (its metadata exposes none); derived from dialplan context for FreeSWITCH. | `calls.Direction`, `domain.go:43`             |
| NativeID                      | The source's own identifier for the row (CDR `uuid`, Telnyx conversation id). Never invented by the lens. | `calls.NativeID`, `domain.go:75`              |
| StreamID                      | `<kind>:<native-id>`. Deterministic, so re-ingesting the same source row targets the same event stream. | `Observation.StreamID`, `domain.go:137`       |
| ContentHash                   | sha256 over the canonical Observation fields. Equality of hashes IS equality of observations: same hash on a live stream replays to zero events; a different hash is a conflict (sources are immutable, so mutation means a parsing bug). | `Observation.ContentHash`, `domain.go:128`    |
| Call stream                   | The event stream behind one StreamID; type `Call`. Holds at most one `call.observed` event today.   | `calls.StreamType`, `domain.go:23`            |
| `call.record` / `call.observed` | The command (record this observation) and its single event type (observation accepted).           | `domain.go:18-21`                             |
| CallView                      | The read-model row the UI will render; one per stream, folded from `call.observed` into the `call_views` collection. | `calls.CallView`, `internal/calls/projection.go:16` |
| Index (the lens)              | The disposable sqlite database of CallViews. Always re-derivable from sources; delete-and-rebuild is a supported operation, not a failure mode. | `internal/app/app.go` (composition root)      |
| Master.csv                    | FreeSWITCH's CDR export file; ships in 13- and 18-column templates the parser recognizes.           | `internal/ingest/cdr/cdr.go:15-50`            |
| FlexTime                      | Wire-level timestamp decoder for Telnyx: RFC3339Nano, then offset-less `2006-01-02T15:04:05`. Exists because Telnyx timestamps are precision-unstable across responses. | `telnyx.FlexTime`, `internal/ingest/telnyx/telnyx.go:46` |
| Excerpt                       | Short transcript preview stored on the Observation (first non-empty user/assistant message, capped at 200 runes). Ordering semantics pending a domain decision. | `excerpt()`, `internal/ingest/telnyx/map.go:70` |
| Recording correlation         | Client-side join matching a Telnyx recording to its conversation via the shared `call_control_id` metadata field. | `RecordingByCallControl`, `map.go:45`         |
| E164 / ParticipantHash        | PII-policy primitives: normalized number and its sha256-truncated hash. Defined and tested, deliberately unused until the retention/PII policy lands. | `domain.go:54-74`                             |

## Bounded contexts

| Term          | Telnyx context                                            | FreeSWITCH context                                     |
| ------------- | --------------------------------------------------------- | ------------------------------------------------------ |
| Conversation  | An AI assistant's message thread (`/ai/conversations`) — the unit ingested | Not a concept; a call is the CDR row identified by `uuid` |
| Duration      | Recording `duration_millis` when a recording matched, else `last_message_at − created_at` | `billsec` from the CDR row                             |
| Direction     | Caller-declared at ingest (`-direction` flag); no source field | Derived: dialplan `context == "public"` means inbound   |
| AI-ness       | `assistant_id` present in conversation metadata           | Not yet derivable (dialplan-path badge is planned)      |

## Deprecated terms

| Old term     | Current term | Why changed                                                          |
| ------------ | ------------ | -------------------------------------------------------------------- |
| CallRecord   | Observation  | Plan-era name (SUPERB plan T2); renamed because rows are per-source observations, not merged call records — merging happens in the read model |
