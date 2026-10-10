<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Canonical backlog for pbx-lens work. Supersedes the interim rows harvested
> into `pbx-artmann/TODO_LIST.md` §30 (2026-10-10); owner-gated questions from
> the same report live in ROADMAP.md "Open questions".

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## High impact

| Task                                                                                                                                                    | Status    | Impact | Effort | Evidence                                                                                                                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------- |
| Fix the transcript-excerpt order-dependence: define semantics (sort messages asc by created_at = call opening, or pin newest-first) and pin with a test | 🔴 `TODO` | High   | 15min  | `internal/ingest/telnyx/map.go:70` `excerpt()` takes returned order; API serves newest-first (report 2026-10-10_06-57 §d.1) |
| T6 list page: one table, all calls, AI badge, excerpt, direction/date filters (M18-M20) — completes the 51% slice                                       | 🔴 `TODO` | High   | 90min  | No `serve` command: `cmd/pbx-lens/main.go:26`; plan T6                                                                      |
| T5 webhook JSONL tailer + ingest command (M15-M17); extract FlexTime into a shared internal wire package when the second consumer lands                 | 🔴 `TODO` | High   | 45min  | `webhook-log` enum only: `internal/calls/domain.go:30`; plan T5                                                             |
| T12 poller timers + freshness self-check with stale-index UI banner (M33-M34) — makes the index autonomous                                              | 🔴 `TODO` | High   | 45min  | No timer/scheduler code anywhere; ingest is CLI-invoked only; plan T12                                                      |
| httptest end-to-end test for the ingest-telnyx command loop (wiring test; the pieces are tested, the loop is not)                                       | 🔴 `TODO` | High   | 30min  | `cmd/pbx-lens/main.go:172-206` has no test; report 2026-10-10_06-57 §e.8                                                    |

## Medium impact

| Task                                                                                                                         | Status    | Impact | Effort | Evidence                                                       |
| ---------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------- |
| T8 per-call detail page + recording streaming proxy, signed-URL refetch server-side (M23-M25)                                | 🔴 `TODO` | Med    | 90min  | No detail route; plan T8                                       |
| T10 FTS5 index over transcripts + search box (M29-M30)                                                                       | 🔴 `TODO` | Med    | 60min  | No FTS code; plan T10                                          |
| T9 join agent JSONL transcripts into the same model (M26-M28); revisit the E003 flat-`internal/calls` decision when it lands | 🔴 `TODO` | Med    | 60min  | `agent-transcript` enum only: `domain.go:31`; plan T9          |
| Fix the smoke command's real-looking fake number (+14084757593 → obviously-fake 555 number)                                  | 🔴 `TODO` | Med    | 5min   | `cmd/pbx-lens/main.go:283`; real handset differs (report §d.6) |
| Move `resolveKey`/`parseSince` out of main.go into a tested internal package                                                 | 🔴 `TODO` | Med    | 30min  | `cmd/pbx-lens/main.go:215-255`, untested                       |
| Add cqrs-lint to the flake (devShell tool or flake app) — kills the /tmp binary dependency at reboot                         | 🔴 `TODO` | Med    | 15min  | `flake.nix` has no cqrs-lint; build command lives in AGENTS.md |
| T7 systemd unit draft + thin NixOS module option for the sibling repo (M21-M22)                                              | 🔴 `TODO` | Med    | 60min  | No unit/module code here; plan T7                              |
| T14 put the lens index DB into the existing nightly backup-staging                                                           | 🔴 `TODO` | Med    | 30min  | No backup hook; plan T14                                       |
| Decide the CI story (GitHub Actions lane like pbx-artmann, or buildflow-only) and wire it                                    | 🔴 `TODO` | Med    | 30min  | `.github/` holds only `dependabot.yml`; report §b.4            |

## Low impact

| Task                                                                                          | Status    | Impact | Effort | Evidence                                                  |
| --------------------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------- |
| T11 dialplan-path AI-vs-human badge for FreeSWITCH CDRs (M31-M32; Telnyx side already badges) | 🔴 `TODO` | Low    | 30min  | `cdr.go:146-155` never sets `IsAI`; plan T11              |
| T13 fetch + store Telnyx conversation insights per AI call (M35)                              | 🔴 `TODO` | Low    | 60min  | No insights code; plan T13                                |
| T15 cost column via Detail Records poller (lag-tolerant, nullable) (M39-M40)                  | 🔴 `TODO` | Low    | 60min  | No cost code; plan T15                                    |
| Verify at source level + file upstream: cqrs-lint C009 exemption excludes bare `must` (len 4) | 🔴 `TODO` | Low    | 30min  | go-cqrs-lite `c009.go` `len(name) > 4` guard; report §b.3 |

<!-- Harvest ledger (2026-10-10 audit pass):
  - report 06-57 §f.1 excerpt ordering      → row 1 (new)
  - §f.2 T6 list page                       → row 2 (new)
  - §f.3 T5 webhook                         → row 3 (new; §f.30 FlexTime-DRY folded in)
  - §f.4 smoke fake number                  → Medium row (new)
  - §f.5 httptest e2e                       → row 5 (new)
  - §f.6 resolveKey/parseSince              → Medium row (new)
  - §f.7 T16 repo docs                      → done in this pass (FEATURES/ROADMAP/DOMAIN_LANGUAGE/README built) → CHANGELOG
  - §f.8 cqrs-lint into flake               → Medium row (new)
  - §f.9 C009 upstream                      → Low row (new)
  - §f.10 T12 timers                        → High row (new)
  - §f.11 T7, §f.12 T8, §f.13 T10, §f.14 T9, §f.15 T11, §f.16 T13, §f.17 T14, §f.18 T15 → rows (new)
  - §f.19 CI story                          → Medium row (new)
  - §f.20 Woolly batch, §f.23 gemini key, §f.26 IPv6, §f.27 firewall → declined for this repo (pbx-artmann/deployment lane, not pbx-lens)
  - §f.21 visibility, §f.22 E003 split      → ROADMAP "Open questions" (owner decisions, not tasks)
  - §f.24 T20, §f.25 T21                    → declined (work lives in sibling repos; plan wording)
  - §f.28 retracted-in-report               → dropped (author retracted)
  - §f.29 docs-freshness upstream tooling   → declined (crush-config lane, not this repo)
-->
