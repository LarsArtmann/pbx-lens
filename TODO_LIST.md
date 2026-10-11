<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Refreshed 2026-10-11 against the shipped UI era: the frontend plan
> (`docs/planning/2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md`)
> F1–F24 are DONE and recorded in FEATURES.md/CHANGELOG.md; only the data-half
> gaps, e2e polish, and sibling-repo handoffs remain.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## High impact

| Task                                                                                                             | Status    | Impact | Effort | Evidence                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------- |
| T15 Detail-Records poller: fetch cost micros per conversation into Enrichment (display already renders it)       | 🔴 `TODO` | High   | 60min  | `internal/web/detail.templ:67` renders `view.CostMicros`; nothing populates it from Telnyx Detail Records yet |
| T13 Telnyx conversation-insights fetch: store summary into Enrichment at ingest (detail card already renders it) | 🔴 `TODO` | High   | 60min  | `internal/web/detail.templ:73` renders `view.InsightSummary`; no API fetch populates it                       |
| T11 dialplan-path AI-vs-human badge for FreeSWITCH CDRs (Telnyx side already badges)                             | 🔴 `TODO` | Med    | 30min  | `cdr.go` never sets `IsAI`; dashboard AI share undercounts CDR calls                                          |

## Medium impact

| Task                                                                                                                                      | Status    | Impact | Effort | Evidence                                                                                        |
| ----------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------- |
| One-flow UI e2e test: seed → list → filter → detail → search → auth gate → stale banner (routes are tested individually; the loop is not) | 🔴 `TODO` | Med    | 45min  | per-route tests in `internal/web/*_test.go`; no single flow test                                |
| Poller test with an httptest-fake Telnyx source (webhook/CDR/transcript sources tested; Telnyx is not)                                    | 🔴 `TODO` | Med    | 45min  | `internal/cli/poller_test.go` covers webhook + missing-source; Telnyx client still fixture-only |
| T14 wire `pbx-lens backup` into the nightly backup staging (timer is host/sibling-side)                                                   | 🔴 `TODO` | Med    | 15min  | command shipped (`internal/backup`); no timer config anywhere here                              |
| Retention/PII purge (T23): retention command + PII mode; `E164`/`ParticipantHash` become load-bearing                                     | 🔴 `TODO` | Med    | 90min  | `domain.go` primitives exist, unused; plan T23                                                  |

## Low impact

| Task                                                                                                        | Status    | Impact | Effort | Evidence                                                                                            |
| ----------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------- |
| Hand `deploy/nixos/module-draft.nix` to nix-international-telephony as `services.pbx-lens` (T7 half 2)      | 🔴 `TODO` | Low    | 30min  | draft shipped here 2026-10-11; sibling repo owns the module                                         |
| Swap the hand-rolled AI/human proportion bar for `display.SegmentBar` when templ-components publishes it    | 🔴 `TODO` | Low    | 10min  | `internal/web/dashboard.templ` `aiHumanCard`; SegmentBar exists unreleased upstream, not in v1.21.0 |
| FTS prefix queries (or a search-box hint) for partial phone numbers                                         | 🔴 `TODO` | Low    | 30min  | FTS5 tokenizes `+17287289311` as one token; substring queries match nothing                         |
| Verify at source level + file upstream: cqrs-lint C009 exemption excludes bare `must` (len 4)               | 🔴 `TODO` | Low    | 30min  | go-cqrs-lite `c009.go` `len(name) > 4` guard; report §b.3                                           |
| Upstream filings (D6): sqliteengine time binding, memory-engine field-name resolution, treefmt templ go pin | 🔴 `TODO` | Low    | 60min  | all three verified in-repo with repros; AGENTS.md gotchas document them                             |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
