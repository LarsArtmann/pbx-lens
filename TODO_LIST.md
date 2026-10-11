<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Canonical backlog for pbx-lens work. Supersedes the pbx-lens rows of the
> interim block harvested into `pbx-artmann/TODO_LIST.md` §30 (2026-10-10);
> owner-gated questions from the same report live in ROADMAP.md "Open
> questions". Harvest ledger for that pass:
> `docs/status/2026-10-10_23-56_docs-health-audit-first-doc-set.md`.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## High impact

| Task                                                                                                                                                                                  | Status    | Impact | Effort   | Evidence                                                                                                                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | -------- | --------------------------------------------------------------------------------------------------------------------------- |
| Fix the transcript-excerpt order-dependence: define semantics (sort messages asc by created_at = call opening, or pin newest-first) and pin with a test                               | 🔴 `TODO` | High   | 15min    | `internal/ingest/telnyx/map.go:70` `excerpt()` takes returned order; API serves newest-first (report 2026-10-10_06-57 §d.1) |
| T6 list page: one table, all calls, AI badge, excerpt, direction/date filters (M18-M20) — completes the 51% slice                                                                     | 🔴 `TODO` | High   | 90min    | No `serve` command: `cmd/pbx-lens/main.go:26`; plan T6                                                                      |
| Adopt go-appkit core + templ-components and wire the templ/Tailwind BuildFlow pipeline (go directive → 1.27.1; AGENTS.md dependency-policy edit) — the frontend stack decision        | 🔴 `TODO` | High   | 90min    | No UI deps in `go.mod`; plan: `docs/planning/2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md` §2–§4 (F3–F4)       |
| T5 webhook JSONL tailer + ingest command (M15-M17); extract FlexTime into a shared internal wire package when the second consumer lands                                               | 🔴 `TODO` | High   | 45min    | `webhook-log` enum only: `internal/calls/domain.go:30`; plan T5                                                             |
| T12 poller timers + freshness self-check with stale-index UI banner (M33-M34) — makes the index autonomous                                                                            | 🔴 `TODO` | High   | 45min    | No timer/scheduler code anywhere; ingest is CLI-invoked only; plan T12                                                      |
| httptest end-to-end test for the ingest-telnyx command loop (wiring test; the pieces are tested, the loop is not)                                                                     | 🔴 `TODO` | High   | 30min    | `cmd/pbx-lens/main.go:172-206` has no test; report 2026-10-10_06-57 §e.8                                                    |

## Medium impact

| Task                                                                                                                         | Status    | Impact | Effort | Evidence                                                                                                   |
| ---------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------------- |
| T8 per-call detail page + recording streaming proxy, signed-URL refetch server-side (M23-M25)                                | 🔴 `TODO` | Med    | 90min  | No detail route; plan T8                                                                                   |
| T10 FTS5 index over transcripts + search box (M29-M30)                                                                       | 🔴 `TODO` | Med    | 60min  | No FTS code; plan T10                                                                                      |
| T9 join agent JSONL transcripts into the same model (M26-M28); revisit the E003 flat-`internal/calls` decision when it lands | 🔴 `TODO` | Med    | 60min  | `agent-transcript` enum only: `domain.go:31`; plan T9                                                      |
| Fix the smoke command's real-looking fake number (+14084757593 → obviously-fake 555 number)                                  | 🔴 `TODO` | Med    | 5min   | `cmd/pbx-lens/main.go:283`; real handset differs (report §d.6)                                             |
| Move `resolveKey`/`parseSince` out of main.go into a tested internal package                                                 | 🔴 `TODO` | Med    | 30min  | `cmd/pbx-lens/main.go:215-255`, untested                                                                   |
| T7 systemd unit draft + thin NixOS module option for the sibling repo (M21-M22)                                              | 🔴 `TODO` | Med    | 60min  | No unit/module code here; plan T7                                                                          |
| T14 put the lens index DB into the existing nightly backup-staging                                                           | 🔴 `TODO` | Med    | 30min  | No backup hook; plan T14                                                                                   |
| T25 quality dashboard page: volume-by-day, AI share, duration trend, MOS placeholder (M54-M55 UI)                            | 🔴 `TODO` | Med    | 90min  | No `/dashboard` route; plan: `docs/planning/2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md` F20 |
| T24 htpasswd basic-auth for the lens UI (bcrypt, operator-pattern reuse; public `/health` only)                              | 🔴 `TODO` | Med    | 60min  | No auth middleware anywhere; plan F21 (same doc, §2 stack matrix)                                          |
| httptest e2e suite for the UI routes: seed→list→filter→detail→search→auth→stale-banner                                       | 🔴 `TODO` | Med    | 60min  | No serve/UI code yet; plan F24 (same doc)                                                                  |

## Low impact

| Task                                                                                          | Status    | Impact | Effort | Evidence                                                  |
| --------------------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------- |
| T11 dialplan-path AI-vs-human badge for FreeSWITCH CDRs (M31-M32; Telnyx side already badges) | 🔴 `TODO` | Low    | 30min  | `cdr.go:146-155` never sets `IsAI`; plan T11              |
| T13 fetch + store Telnyx conversation insights per AI call (M35)                              | 🔴 `TODO` | Low    | 60min  | No insights code; plan T13                                |
| T15 cost column via Detail Records poller (lag-tolerant, nullable) (M39-M40)                  | 🔴 `TODO` | Low    | 60min  | No cost code; plan T15                                    |
| Verify at source level + file upstream: cqrs-lint C009 exemption excludes bare `must` (len 4) | 🔴 `TODO` | Low    | 30min  | go-cqrs-lite `c009.go` `len(name) > 4` guard; report §b.3 |
| Export API: JSONL/CSV dump of the index (graduated from ROADMAP; plan D5)                     | 🔴 `TODO` | Low    | 60min  | No export code; plan 2026-10-11_00-09 (docs/planning/)    |
| README install line: drop the `go install @latest` apology comment until a tag exists         | 🔴 `TODO` | Low    | 5min   | `README.md` Installation; session finding 2026-10-10      |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
