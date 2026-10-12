<!-- docs/status/2026-10-12_14-30_superb-followups-harvest-wrap.md -->
<!-- Point-in-time wrap of the 2026-10-12 ~14:00→ session (follow-up closure
     of the 13-49 report: harvest, Cache-Control, AGENTS budget, annotations,
     crush-config routing). Snapshot only — bring current via ANNOTATE. -->

# Status (wrap): 13-49 follow-ups — harvest, no-cache, budget, routing

**Session:** 2026-10-12 ~14:00 → · **Report:** 14:30 · **Prior reports:** 13-28, 13-49 (both stay as-is)

**One-line verdict:** All nine actionable 13-49 loose ends are closed or routed with evidence; the three §g owner questions resolved themselves into evidence-backed answers (Q1 by events, Q2 half-shipped, Q3 documented); nothing was pushed unverified this time.

## a) DONE this session

| #   | Work                                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                                |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| A1  | **HARVEST:** all 50 items of 13-49 §f routed — 7 TODO + 5 BLOCKED rows into TODO_LIST.md (7th pass header), 8 deferred-trigger ideas + Q3 into ROADMAP.md, 2 lessons into crush-config, done/decided/duplicate items dropped with citations                                                                                              | TODO_LIST.md, ROADMAP.md, ledger below                                                                  |
| A2  | **Cache-Control: no-cache** on every HTML response (pages + fragments) via `web.render`; asserted on the 200 AND the 304 of `/frag/dashboard`, and on the ETag-less `/frag/staleness` (deterministic revalidation)                                                                                                                       | `internal/web/render.go`; `cmd/pbx-lens/serve_test.go` `assertConditionalGET(wantNoCache)`; tests green |
| A3  | **AGENTS.md back to its 220-line budget** (was 273): templ-components adoption map → `docs/templ-components-adoption.md`, HTTP-stack doctrine → `docs/http-stack.md`, six bullets condensed without fact loss (both docs listed in Deep context)                                                                                         | `wc -l AGENTS.md` = 220                                                                                 |
| A4  | **12-47 plan annotated**: EXECUTING → EXECUTED + `fd0e082` + heal chain (inline strikethrough, no rewrite)                                                                                                                                                                                                                               | plan line 3                                                                                             |
| A5  | **go-etag v0.6.1 hook truth verified**: `OnETagGenerated`/`On304`/`OnBufferOverflow` exist in the PINNED version (etag.go:67-88) — the /api/ops hit-ratio row is actionable, NOT upstream-blocked (13-49 §c implied unknown)                                                                                                             | module cache grep                                                                                       |
| A6  | **Upstream watch:** go-etag#2 + go-appkit#26 both still OPEN, 0 comments — nothing to fold in; ledger unchanged                                                                                                                                                                                                                          | `gh issue view`                                                                                         |
| A7  | **Markdown hygiene:** table rows column-aligned via the repo's dprint (`nix develop -c dprint fmt`); dprint's touches to docs/status/ files REVERTED (historical snapshots + concurrent session's file); `buildflow -s markdown-lint` = not applicable (no .markdownlint config in repo — deliberate non-fix, dprint owns markdown here) | `git diff TODO_LIST.md`                                                                                 |
| A8  | **Cosmetics:** `/tmp/pbxlens-probe` trashed; the serve_test.go `modernize` warning verified STALE against the golangci-lint binary (line already uses `t.Context()`); one golines reformat of my own test line applied + lint 0                                                                                                          | trash, lint run                                                                                         |
| A9  | **crush-config routing:** pinned-version-vs-HEAD research-trap lesson appended to `references/lessons.md`; daemon stale-`index.lock` hardening row appended to crush-config TODO_LIST                                                                                                                                                    | crush-config repo                                                                                       |
| A10 | **Concurrent-session protocol honored end-to-end:** disjoint files only, foreign edits untouched, dprint's reformat of their 14-16 report reverted, gate moved to a detached worktree at HEAD (see b)                                                                                                                                    | this file §b                                                                                            |

## b) THE GATE (B1 closure attempt)

The shared tree stayed busy all session (concurrent session: templ work → favicon/CSS-MIME fix → recording.go at gate time). Applying 13-49 §e.4 (never gate a tree someone else is editing): the full `buildflow --build-mode dev` gate runs in a **detached git worktree at HEAD** (`/tmp/pbxlens-gate`, a26559f) — immune to interleaved writes, testing exactly what CI will test. Result: see ANNOTATE appendix once complete.

**ANNOTATION (gate outcome, same day):** GREEN at `7404b05` — `buildflow --build-mode dev` passed with warnings (77 steps success / 0 failed, exit 0; the cqrs-lint/statix/file-size counts are the documented advisory set in docs/cqrs-lint-policy.md + docs/buildflow-dev-gate.md). The path there surfaced and closed three REAL repo breaks the busy tree had been hiding, each verified individually per the load doctrine after OOM-kill casualties: (1) HEAD's dashboard.templ carried a templ-fmt violation → flake pre-commit check red → fixed via `templ fmt` (daemon-committed `824efaa`); (2) AGENTS.md at 220 wc-lines counts as 221 under go-structure-linter's trailing-newline counting → trimmed to 219 (`cd1047f`); (3) go.mod drift — templ-components/utils had gone direct without `go mod tidy`, breaking govalid/go-generate → tidied + vendorHash repaired via `nix run .#update-vendor-hash` (green build) → daemon-committed `7404b05`. One transient red was foreign and self-healed: the concurrent session's pagination test landed via daemon ahead of its implementation; their follow-up commit fixed it (verified before the final gate run). A stale buildflow result-cache replay of the pre-tidy go-generate failure was bypassed with BUILDFLOW_NO_RESULT_CACHE=1, matching the documented triage row.

## c) THE THREE §g QUESTIONS — RESOLVED WITH EVIDENCE

1. **Sequencing (Q1):** answered by events — the templ session landed (plan/status committed, `page.templ` in), then kept working on OTHER surfaces (favicon, recording). Disjoint-file last-writer-wins worked all day with zero collisions once respected. Norm now in ROADMAP (pre-flight + ship-state verification) pending the owner's protocol decision.
2. **Browser verification (Q2):** the safe half SHIPPED (A2 — `Cache-Control: no-cache` everywhere HTML is served, with 200+304 assertions); the manual browser/DOM-churn check stays a BLOCKED TODO row for the owner (no browser in the agent env).
3. **Multi-agent protocol (Q3):** owner decision, now a ROADMAP open question with the candidate conventions and the incident evidence; the interim norm (pre-flight + worktree gating + ship-state verification) is documented and was exercised this session.

## d) HARVEST LEDGER (13-49 §f, all 50 items)

| Source (§f item)           | Disposition              | Destination / reason                                                           |
| -------------------------- | ------------------------ | ------------------------------------------------------------------------------ |
| 1-5, 42, 50                | done earlier             | already in CHANGELOG/committed (§a table of 13-49); 50 = this pass             |
| 6                          | done this session        | worktree gate (§b)                                                             |
| 7, 39                      | new row (blocked)        | TODO_LIST browser-level 304 + DOM-churn (owner manual)                         |
| 8                          | done this session        | Cache-Control shipped (A2) → CHANGELOG                                         |
| 9                          | done + non-fix           | alignment via dprint (A7); markdown-lint step N/A in repo                      |
| 10, 11                     | merged row               | TODO_LIST go-etag hooks → /api/ops (hooks verified in pinned v0.6.1 — A5)      |
| 12                         | existing row             | TODO_LIST HTML research report row kept                                        |
| 13, 14                     | merged into existing row | /metrics scrape row extended with real-scraper smoke test                      |
| 15, 16, 25, 46             | merged row               | TODO_LIST e2e cache-semantics hardening                                        |
| 17                         | new row                  | TODO_LIST README ops notes                                                     |
| 18                         | new row                  | TODO_LIST gotchas templ-flush entry                                            |
| 19                         | done this session        | plan annotated (A4)                                                            |
| 20                         | done this session        | AGENTS 220 (A3)                                                                |
| 21                         | routed                   | ROADMAP open question (Q3)                                                     |
| 22, 23                     | routed                   | crush-config lessons.md + TODO_LIST (A9)                                       |
| 24, 48                     | merged row (blocked)     | TODO_LIST fold-in-when-landed row (both verified OPEN)                         |
| 26                         | new row                  | TODO_LIST hash-cost probe                                                      |
| 27, 30, 32, 36, 37, 38, 44 | routed                   | ROADMAP deferred-trigger list                                                  |
| 28                         | new row (blocked)        | TODO_LIST >1MiB transcript policy                                              |
| 29, 49                     | merged row (blocked)     | TODO_LIST dependabot confirmation                                              |
| 31                         | duplicate                | Telnyx etag-client already tracked by the 2027-01 calendar cluster (AGENTS.md) |
| 33, 34                     | existing rows            | TODO_LIST /api/ops stale-pass + staleness-filter rows                          |
| 35                         | new row                  | TODO_LIST SECURITY.md /metrics posture                                         |
| 40                         | declined                 | dashboard 304 rarity ACCEPTED 13-49; revisit on complaint only                 |
| 41                         | new row (blocked)        | TODO_LIST v0.2.0 cut                                                           |
| 43                         | new row                  | TODO_LIST coverage pass                                                        |
| 45, 47                     | done this session        | cosmetics (A8)                                                                 |

## e) STILL OPEN (all now tracked elsewhere)

- TODO_LIST: 12 new/merged rows above; ROADMAP: deferred triggers + Q3; crush-config: 2 items.
- The concurrent session's own wrap (their 14-16 report + whatever follows) is theirs to close.

---

_Snapshot only. Next actions live in TODO_LIST.md; bring this file current only via ANNOTATE._
