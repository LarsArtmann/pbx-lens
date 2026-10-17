<!-- docs/telnyx-quarterly-checklist.md — the 2027-01 re-verify bundle.
     Run this top to bottom when the window opens; append verdicts inline. -->

# Telnyx quarterly re-verify checklist

Next window: **2027-01** (previous passes: core cluster 2026-10-10,
recordings 2026-10-12). The truths under test live in
[telnyx-live-api.md](telnyx-live-api.md); this file is the run order.

- [ ] **Owner: confirm the 2027-01 calendar entry exists** (AGENTS.md
      quarterly cluster says "calendar entry, m27.9"). Without the reminder
      this file never runs.

## 0. Setup

```console
export TELNYX_API_KEY=...            # or -key-file
GOEXPERIMENT=jsonv2 go test ./internal/ingest/telnyx/ ./internal/wire/
```

All repo regression tests must be green BEFORE probing live — they pin
everything already learned.

## 1. Conversations list (verified 2026-10-10)

Live-probe the field-as-param filters and re-confirm the 400 shape:

- `last_message_at=gte.<RFC3339>` and
  `metadata->assistant_id=eq.<id>` still work as BARE params (a
  `filter[...]=` wrapper must still 400 with error code `10015`).
- Fetch one conversation twice a few minutes apart; timestamps may change
  PRECISION (nano vs micro) but the mapper must stay stable —
  `TestMapConversationTimestampPrecisionStable` is the regression tripwire.
- Metadata values still not uniformly strings; decode stays
  `map[string]any` via `Conversation.MetadataString`.

## 2. Recordings (verified 2026-10-12)

- List recent recordings on the assistant connection: `status=completed`
  rows should still carry EMPTY `download_urls.wav` + populated `.mp3`.
  If a WAV rendition has appeared, the proxy already prefers WAV — no
  code change needed, just update the doctrine paragraph.
- A `processing` row may serve empty on both — confirm the MP3 fallback
  order handles it without a hard 404.
- **Range support on the MP3 endpoint** (first live test this window):
  `Range: bytes=0-1023` → expect `206` with `Content-Range`; if the
  endpoint ignores Range (`200` full-body), record that here — the proxy
  degrades gracefully either way, but the doc should stop saying
  "pending".

## 3. Enrichment endpoints (SPEC-VERIFIED ONLY — the main event)

These have never been live-verified (no key at implementation time,
2026-10-11). Everything below is expected shape; record any deviation.

- `GET /ai/conversations/{id}/conversations-insights` — suffix is
  `conversations-insights`, NOT `insights`. Response:
  `data[]{id, status: pending|in_progress|completed|failed, created_at,
  conversation_insights[]{insight_id, result}}`. Newest `completed` row
  wins; non-empty results join with `\n`.
- `GET /detail_records` — deepObject filter with REQUIRED
  `filter[record_type]=call-control`, optional
  `filter[started_at][gte]`; `page[size]` caps at 50. `cost` is a decimal
  STRING in currency units ("0.002") + `currency` — convert via
  `telnyx.MicrosFromCost`.
- The call-control record is NOT in the public spec; aliases
  `id`/`uuid`, `call_control_id`/`telnyx_call_control_id`, cost as
  string or number are tolerated. A THIRD spelling found live → extend
  `DetailRecord.UnmarshalJSON` aliases.
- Live smoke end to end: `go run ./cmd/pbx-lens ingest-telnyx -db
  /tmp/lens-verify.db -since 48h -limit 5` then check the ingest report
  for enrichment rejections before trusting the values.

## 4. Close-out

- Update [telnyx-live-api.md](telnyx-live-api.md) verdict lines (spec →
  live + date), the AGENTS.md quarterly cluster line, and the TODO_LIST
  row.
- Schedule the next window (+1 quarter) and refresh this checklist's
  "Next window" line.
