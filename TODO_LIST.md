<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Refreshed 2026-10-12 (third pass): the recording-proxy upstream timeout
> knob, the file-size split of the three >350-line files (calls domain, CLI
> main, telnyx client), the flake app `meta.description`, and the upstream
> loop (five D6 filings re-verified OPEN; the C009 bare-`must` gap confirmed
> covered by go-cqrs-lite#64; the BuildFlow compile-gate misattribution filed
> as BuildFlow#45) are DONE and recorded in FEATURES.md/CHANGELOG.md. What
> remains is blocked work only (below).

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## Low impact

| Task                                                                                                                                               | Status       | Impact | Effort | Evidence                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Swap the hand-rolled AI/human proportion bar for `display.SegmentBar` when templ-components publishes it                                           | 🔵 `BLOCKED` | Low    | 10min  | `internal/web/dashboard.templ` `aiHumanCard`; SegmentBar exists unreleased upstream, not in v1.21.0                                       |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec) | 🔵 `BLOCKED` | Low    | 30min  | needs a real API key; `internal/ingest/telnyx/dto.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                       |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)      | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
