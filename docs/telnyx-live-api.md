# Telnyx live-API truths (pbx-lens)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit). The
core cluster was verified against the live API 2026-10-10 — unit fixtures
alone missed all three findings. The recordings cluster joined it live
2026-10-12 (first key-in-hand pass). **Re-verify this whole cluster
quarterly** (the enrichment endpoints were only spec-verified 2026-10-11;
live verification still pending — no key at implementation time — so they
fold into the next quarterly pass). Upstream filing status lives in
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

The correlation window is a BOUND, not a guarantee (audit 2026-10-17):
one pass pulls the `DefaultRecordingsLookback` (300) most recent
recordings; a conversation ingested after its recording fell out of that
window maps with no `RecordingRef` and the detail page 404s "no recording
reference indexed" permanently (re-ingest replays the same ContentHash, so
the miss never self-heals). Deliberate trade: paging deeper costs
unbounded API budget per pass against a rare miss. Remedy if it bites: a
one-off pass with a raised lookback constant.

## Recordings are MP3-only on the AI assistant connection (live 2026-10-12)

Verified against the live API with the real key: EVERY recording on the
assistant connection (9/9, all `status=completed`) carries an EMPTY
`download_urls.wav` and a populated `download_urls.mp3` — the WAV
rendition does not exist for these calls, so a WAV-only proxy 404s every
playback. The REST object's `download_urls` is confirmed `{mp3, wav}`
(the `recording_urls` spelling belongs to the `call.recording.saved`
webhook, not the REST resource). Other live-verified recording fields:
`source=call`, `channels=dual`, `duration_millis`, offset-less
`recording_started_at`/`recording_ended_at`, and `initiated_by`
(distinguishes OutboundAPI dials). The recording proxy streams WAV when
present and falls back to MP3 (`proxyTelnyxRecording`); neither means a
real 404. A `processing` recording can also serve empty URLs — media
transcoding is asynchronous — so empty-on-both is not necessarily
permanent (not yet observed live; the fallback order handles it).

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
