<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Refreshed 2026-10-11 (second pass): the Telnyx data half (T15 cost, T13
> insights) and its poller coverage, T11's dialplan badge, T14's backup
> staging, the one-flow UI e2e test, and the FTS partial-number hint are DONE
> and recorded in FEATURES.md/CHANGELOG.md. What remains: T23 (in flight),
> sibling-repo handoffs, and upstream filings.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## Medium impact

| Task                                                                                                  | Status    | Impact | Effort | Evidence                                                                                                                                                                                         |
| ----------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Retention/PII purge (T23): retention command + PII mode; `E164`/`ParticipantHash` become load-bearing | 🟢 `DONE` | Med    | 90min  | `internal/retention/` + `purge -max-age [-dry-run]` + `call.retired` tombstone + `?pii=hash` export; e2e verified. Open enhancement: wire `Policy.Admits` as an ingest-side cutoff in the poller |

## Low impact

| Task                                                                                                                                                     | Status       | Impact | Effort | Evidence                                                                                                                           |
| -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| Hand `deploy/nixos/module-draft.nix` to nix-international-telephony as `services.pbx-lens` (T7 half 2)                                                   | 🔴 `TODO`    | Low    | 30min  | draft shipped here 2026-10-11 (now incl. `backup.*` options); sibling repo owns the module                                         |
| Swap the hand-rolled AI/human proportion bar for `display.SegmentBar` when templ-components publishes it                                                 | 🔵 `BLOCKED` | Low    | 10min  | `internal/web/dashboard.templ` `aiHumanCard`; SegmentBar exists unreleased upstream, not in v1.21.0                                |
| Verify at source level + file upstream: cqrs-lint C009 exemption excludes bare `must` (len 4)                                                            | 🔴 `TODO`    | Low    | 30min  | go-cqrs-lite `c009.go` `len(name) > 4` guard; report §b.3                                                                          |
| Upstream filings (D6): sqliteengine time binding, memory-engine field-name resolution, treefmt templ go pin                                              | 🔴 `TODO`    | Low    | 60min  | all three verified in-repo with repros; AGENTS.md gotchas document them                                                            |
| Rebuild/reinstall the BuildFlow binary (200ab99 → HEAD) so license-check's sandbox go meets the 1.27.1 floor; then drop the AGENTS.md license-check note | 🔴 `TODO`    | Low    | 15min  | the dev gate's only red step; direct `go-licenses` is clean (AGENTS.md license-check note)                                         |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec)       | 🔴 `TODO`    | Low    | 30min  | `internal/ingest/telnyx/telnyx.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                                   |
| Recording-proxy upstream timeout knob (metadata fetch + WAV stream ride `http.DefaultClient`, no timeout)                                                | 🔴 `TODO`    | Low    | 20min  | `web/recording.go` `proxyTelnyxRecording`; bound deferred until real traffic data (owner decision 2026-10-11)                      |
| Optional: consumer-side `RecordingFetcher` interface in `web.Options` (mirrors `Searcher`; decouples the UI from `*telnyx.Client`)                       | 🔴 `TODO`    | Low    | 15min  | `web/recording.go` `RecordingOptions`; no second implementation yet, pure pattern symmetry                                         |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)            | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate |
| Shrink AGENTS.md from 469 to ≤220 lines (extract gotchas/residuals into docs/) — go-structure-linter ERROR, gate red until fixed                         | 🔴 `TODO`    | Med    | 60min  | naming-review session 2026-10-11 hit it as the gate's only remaining error; docs-health ANNOTATE-style extraction, keep invariants |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
