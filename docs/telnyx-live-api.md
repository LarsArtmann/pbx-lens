# Telnyx live-API truths (pbx-lens)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit). The
core cluster was verified against the live API 2026-10-10 — unit fixtures
alone missed all three findings. **Re-verify this whole cluster quarterly**
(the enrichment endpoints were only spec-verified 2026-10-11; live
verification still pending — no key at implementation time — so they fold
into the next quarterly pass). Upstream filing status lives in
[upstream-filings.md](upstream-filings.md).

## Live-verified 2026-10-10

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

## FlexTime receiver-only method promotion

`FlexTime` promotes time.Time methods on the RECEIVER only —
`conv.CreatedAt.UTC()` compiles, but `last.After(first)` does NOT accept a
FlexTime argument; write `.Time.After(x.Time)` explicitly.

## Direction is caller-declared

The conversation→Observation direction is caller-declared (MapOptions.
Direction, CLI `-direction`): Telnyx conversation metadata exposes no
direction field (docs read 2026-10-10), so this deployment's assistant
calls default to outbound. Revisit when inbound AI calls land (the DID
interception path).

## Recording correlation is client-side

Conversations carry `call_control_id` in metadata, recordings carry the
same field, so ingest-telnyx lists recent recordings and matches
in-process (`RecordingByCallControl`) — the recordings list filter's
supported keys are undocumented, so no server-side filter is trusted.
Detail Records correlation works the same way
(`DetailRecordByCallControl`).

## Enrichment endpoints (spec-verified 2026-10-11; LIVE verification pending)

1. `GET /ai/conversations/{id}/conversations-insights` — the suffix is
   `conversations-insights`, NOT `insights` (both 404-magnet guesses
   exist in the docs). Response: `data[]{id, status:
   pending|in_progress|completed|failed, created_at,
   conversation_insights[]{insight_id, result}}`. Only `completed` rows
   are summary sources; the newest row wins; its non-empty results join
   with `\n` (`telnyx.InsightSummary`).
2. `GET /detail_records` — deepObject filter with REQUIRED
   `filter[record_type]` (we send `call-control`), optional
   `filter[started_at][gte]`; `page[size]` caps at 50 (vs 100
   elsewhere). `cost` is a decimal STRING in currency units ("0.002")
   with `currency` — NOT micros; convert via `telnyx.MicrosFromCost`
   (round to `calls.MicrosPerUnit`).
3. The call-control record shape is NOT in the public spec (its
   DetailRecord union lists only messaging/conference/amd/verify/sim/
   media), so `DetailRecord.UnmarshalJSON` is deliberately
   alias-tolerant: `id`/`uuid`, `call_control_id`/
   `telnyx_call_control_id`, cost as string or number. If live calls
   show a third spelling, extend the aliases there.
