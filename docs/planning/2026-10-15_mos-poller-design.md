# MOS poller — execution design (2027-01 window prep)

- **Created:** 2026-10-15 (plan 05-40 M7, F18–F20). Design only; zero live
  calls consumed. Everything here is pre-verified against the pinned code
  so the January window is EXECUTION-ONLY (live DTO verification + wire-in).
- **Goal:** fill the dashboard MOS card (`mosCard`,
  internal/web/dashboard.templ:350) with real scores from Telnyx Detail
  Records, via a lag-tolerant nullable join — never blocking ingest.

## What already exists (verified 2026-10-15)

- `internal/ingest/telnyx/dto.go:62` — `DetailRecord` with a tolerant
  `UnmarshalJSON`; `DetailRecordByCallControl` (:141) does the client-side
  correlation the live API forces (no server-side join).
- The dashboard renders mean + per-day trend once ANY call reports a score;
  zero-score indexes stay the honest placeholder — no UI work needed.
- CDR rows may carry their own quality fields (T25/M55 class); the poller
  only covers Telnyx-side scores.

## Types (new package `internal/ingest/telnyx/mos` or a `DetailRecords` method set)

- `ScoreRecord { CallControlID string; MOS float64; CapturedAt time.Time; Raw json.RawMessage }`
  — parse tolerantly (the union shape is spec-unpublished; decode what is
  there, keep Raw for re-verification).
- `Fetcher interface { DetailRecords(ctx, since time.Time) ([]ScoreRecord, cursor, error) }`
  — paginated like the conversations pager; cursor persisted per source.
- Wire shape into the domain: an `Observation` enrichment path, NOT a new
  SourceKind — scores annotate an EXISTING call stream.

## Fetch + idempotence plan

1. Poller pass fetches Detail Records `since` last cursor (lag-tolerant:
   records may appear minutes-to-hours after call.ended; re-fetch window
   overlaps the cursor by one interval, dedupe by ContentHash semantics).
2. Correlate client-side via `DetailRecordByCallControl` to indexed calls.
3. Emit as `call.corrected` on the EXISTING stream (append-only audit, same
   ContentHash idempotence — a re-fetch with the same score replays to zero
   events). A score arriving for an UNKNOWN stream logs and skips (the call
   may index later; the overlap window re-offers it).
4. Drain-before-enrich (docs/gotchas.md): the serve-side poller drain must
   complete before the score projection folds, or the enrichment lands on
   a not-yet-projected view. Reuse `cli.DrainProjections` semantics — no
   new drain path.

## Fixture shapes (checked in before the window)

- `testdata/mos/detail_record_full.json` — every field the union might
  carry (from the Jan live capture, then frozen).
- `testdata/mos/detail_record_minimal.json` — only `call_control_id` + a
  score field; the tolerant decode must accept it.
- `testdata/mos/detail_record_negative.json` — unknown shape → skipped,
  counted, never fatal (mirror the malformed-JSONL property test).

## Wire points (when the window opens)

- `internal/cli/poller.go`: add the detail-records source to the pass loop
  (per-source counters ride `/api/ops` for free).
- `internal/calls`: score field on the observation/correction path +
  nullable `MOS` on `CallView` (the card already reads it).
- Flags: `-mos-poll-every` (default off) mirroring `-poll-every`; NixOS
  module option lands sibling-side in the same window.

## Non-goals

- No retroactive re-ingest of old conversations for scores.
- No new SourceKind, no schema v2 (the nullable fold needs no migration).
- No live verification before 2027-01 (docs/telnyx-live-api.md quarterly
  cluster; calendar entry exists, m27.9).
