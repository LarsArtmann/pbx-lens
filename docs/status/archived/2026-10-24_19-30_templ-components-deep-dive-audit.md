# Status Report — 2026-10-24 19:30 CEST — templ-components Deep-Dive Audit

**Scope:** This session only (user directive: no unrelated research). The session had
one task: "Are we using /home/lars/projects/templ-components/** superbly?" — answered
with a full library-deep-dive audit (skills loaded first: `library-deep-dive` +
`templ-components`).

**Deliverable:** `docs/research/2026-10-24_templ-components-deep-dive.html` —
38-capability scored inventory, 7 evidence-cited findings, adoption 91/100 (very
high). Not committed manually (harness contract); auto-daemon picks it up.

**Headline verdict:** Yes, superb where it counts — 23/28 applicable capabilities fully
leveraged, 0 anti-patterns, theming/a11y/CSP handled ahead of upstream defaults. Two
real gaps: (1) `utils/format` hand-rolled in four places that the pinned version
already ships — including a **two-dialect split brain** ("/ops" shows the library's
"1m", the staleness banner shows the local helper's "1m00s"); (2) pin is one minor
behind, and v1.24.0 fixes ARIA nesting in Heatmap + CopyButton, both rendered today.

---

## a) FULLY DONE

| #                                         | Item                                                                                                                                                                                                                                                           | Evidence                                                      |
| ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| ~~1~~ done (docs-health pass 2026-10-26)  | ~~Both matching skills loaded before any tool action~~                                                                                                                                                                                                         | ~~session start~~                                             |
| ~~2~~ done (docs-health pass 2026-10-26)  | ~~Phase 0: prior reports located (2, no index README); adoption map read — its broken table spotted immediately~~                                                                                                                                              | ~~docs/research/, docs/templ-components-adoption.md:69-80~~   |
| ~~3~~ done (docs-health pass 2026-10-26)  | ~~Version currency: pin v1.23.0 (go.mod:24-27), published v1.24.0 (2026-10-23), upstream HEAD = 3 docs commits past release; v1.24.0 body read from CHANGELOG~~                                                                                                | ~~go list -m -versions, CHANGELOG.md:34+~~                    |
| ~~4~~ done (docs-health pass 2026-10-26)  | ~~Full component census: 33 distinct @-calls across 6 .templ surfaces + Go-side calls (SimpleNav/ThemeToggle/RelativeTime/DefaultPageProps)~~                                                                                                                  | ~~grep census (session log)~~                                 |
| ~~5~~ done (docs-health pass 2026-10-26)  | ~~Capability ground truth read at the exact pin, not from drifting docs: utils@v1.23.0/format/format.go:75-155 read line-by-line and diffed against the 4 local twins~~                                                                                        | ~~module cache~~                                              |
| ~~6~~ done (docs-health pass 2026-10-26)  | ~~Live upstream filing states: #48 #49 #50 #51 #52 #59 #60 #61 #63 — all OPEN; workarounds located in-code (dashboard.templ:441, app.css:124)~~                                                                                                                | ~~gh issue list + in-repo greps~~                             |
| ~~7~~ done (docs-health pass 2026-10-26)  | ~~The split-brain discovery + output diff: local compactDuration renders "1m00s"/"12h05m" vs library "1m"/"12h5m"; clockDuration/stringOrDash byte-identical sub-day; percent identical today (all ratio sources zero-guarded) but missing NaN→"—" hardening~~ | ~~format.go vs calls.templ:392-422, dashboard.templ:712-714~~ |
| ~~8~~ done (docs-health pass 2026-10-26)  | ~~38-row capability inventory, 10 N/A rows each with a stated reason, score derived and published (23×1 + 5×0.5 of 28 = 91)~~                                                                                                                                  | ~~report appendix~~                                           |
| ~~9~~ done (docs-health pass 2026-10-26)  | ~~7 findings cards + version-currency table + 10-row priority matrix (sorted desc) + how-to-verify appendix~~                                                                                                                                                  | ~~report~~                                                    |
| ~~10~~ done (docs-health pass 2026-10-26) | ~~HTML composed by copying the editorial template and editing in place (no line-splicing); one failed edit (HTML entity mismatch) caught and fixed in the next step~~                                                                                          | ~~multiedit log~~                                             |
| ~~11~~ done (docs-health pass 2026-10-26) | ~~Self-consistency gate: denominators agree everywhere (91 / 23/28 / N=38 / A=28), priority table descending, 4 spot-checked citations in range, zero demo leftovers~~                                                                                         | ~~gate greps (session log)~~                                  |
| ~~12~~ done (docs-health pass 2026-10-26) | ~~dprint check attempted per repo discipline (no HTML plugin configured — nothing to format); git state verified (daemon owns commits)~~                                                                                                                       | ~~nix develop -c dprint check~~                               |

## b) PARTIALLY DONE

| #                                                                                                      | Item                                                                                                                                                                                          | State                                                                                                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~ done — executed — actions ①–④ landed 2026-10-25 (05-49 report)                                   | ~~The audit's own action plan (items 1–4)~~                                                                                                                                                   | ~~Reported and prioritized, **not executed** — the ask was "are we", not "make it so". All four are ready-to-run with known cost (goldens + browser lane for #1–#3).~~                                                                                                              |
| ~~2~~ done — held — the census definition is stated in later reports                                   | ~~Component-count precision~~                                                                                                                                                                 | ~~"33 components" = the @-prefixed census; ~37 counting Go-expression calls (SimpleNav, ThemeToggle, RelativeTime, DefaultPageProps). Conservative direction, still imprecise.~~                                                                                                    |
| ~~3~~ done — verified at call sites 2026-10-25 (zero Dot usages; the typed-enum Grid call)             | ~~Two immunity claims inferred, not read: #54 (Badge "Dot unused") and #61 (Grid "typed enums only") rest on the adoption map's word — the map whose stale format row is itself finding F5.~~ | ~~report F7~~                                                                                                                                                                                                                                                                       |
| ~~4~~ done — the correction propagated (18-59 postscript + the baseline doc)                           | ~~Cross-document consistency~~                                                                                                                                                                | ~~This morning's dedup verdict (accepted clockDuration/compactDuration pair, recorded in docs/status/2026-10-24_18-59 and the baseline doc's rationale) is superseded by finding F1 — but only the new HTML report says so. The two older artifacts still assert the old verdict.~~ |
| ~~5~~ done — go-appkit deep-dived 2026-10-25 (docs/research/2026-10-25_go-appkit-templ-deep-dive.html) | ~~Deep-dive coverage of the UI trio~~                                                                                                                                                         | ~~templ-components audited; go-appkit (the third member) has never been deep-dived.~~                                                                                                                                                                                               |

## c) NOT STARTED (deliberate waits / out of scope)

| #                                                                          | Item                                                                                        | Why                                            |
| -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| ~~1~~ done — utils/format adopted (ec89cce; AGENTS ADOPTED 2026-10-24)     | ~~Action ① — adopt utils/format, delete 4 twins (~40 lines), golden refresh, browser lane~~ | ~~Awaiting user go-ahead (see g1/g2)~~         |
| ~~2~~ done — the v1.24.0 bump (de87205)                                    | ~~Action ② — v1.24.0 bump via the 6-step watchlist~~                                        | ~~Same~~                                       |
| ~~3~~ done — the "never" guard (c1c2ed0)                                   | ~~Action ③ — ops "Last pass" zero-time guard ("never" copy)~~                               | ~~Same~~                                       |
| ~~4~~ done — the map repaired (ec89cce)                                    | ~~Action ④ — repair the adoption map's broken table + stale format row~~                    | ~~2-line fix, bundled with ①~~                 |
| ~~5~~ done — harvested                                                     | ~~HARVEST of today's two status reports into TODO_LIST.md~~                                 | ~~User instruction: report, then wait~~        |
| ~~6~~ done — docs/research/README.md created 2026-10-25                    | ~~docs/research/README.md index (3 reports, none indexed)~~                                 | ~~Candidate, not ordered~~                     |
| ~~7~~ done — decided — none for docs-only sessions (the README policy row) | ~~CHANGELOG entry~~                                                                         | ~~Docs-only session — likely none; undecided~~ |

## d) TOTALLY FUCKED UP

**Nothing at that severity this session.** Near-misses, honestly:

1. ~~**The biggest error adjacent to this session is this morning's, which this session had to correct**: the dedup pass accepted the clockDuration/compactDuration pair as "different documented business rules" without checking the dependency graph — the library shipped all four twins at the pinned version all along. The dedup skill's own doctrine ("hand-rolled code duplicating built-in functionality") was applied without its cheapest step. This session found it, quantified it, and recorded the correction — but the correction is prose in a third file until action ① lands.~~ **NOT-DO — process note; corrected by this session and resolved by the adoption.**
2. ~~First multiedit batch: 1 of 5 edits failed (I sent the literal `·` where the template has `&middot;`) — caught by grep, fixed immediately.~~ **NOT-DO — process note; caught and fixed immediately.**
3. ~~Session workflow friction: the rg-protection hook blocked four command shapes; worked around via the Grep-tool/plain-grep patterns. Cost: a few minutes, no damage.~~ **NOT-DO — process note; the hook belongs to crush-config.**

## e) WHAT WE SHOULD IMPROVE

1. ~~**Dedup passes must grep the dependency graph before accepting any "similar shape" as intentional.** The library check is one grep and today it flips a verdict recorded six hours earlier. Rule candidate: no acceptance without a `go list -m all`-informed look at what the pinned deps already provide.~~ done — adopted — the dependency-graph grep is the recorded lesson (18-59 correction)
2. ~~**When an audit supersedes an earlier verdict, annotate the earlier artifact in the same step** (or explicitly accept owning the contradiction until the fix lands) — two docs currently disagree silently, which is exactly the split-brain class this repo writes guard tests against.~~ done — adopted — same-step annotations practiced since
3. ~~**Verify immunity claims at the call site, not from the map** — the map is authoritative for intent, not for current code; F5 proves maps rot.~~ done — adopted — verified at call sites (f9)
4. ~~**When editing templated HTML, replace whole elements and check entity encoding first** (`&middot;` vs `·` cost one retry).~~ done — adopted
5. ~~**Start docs/research/README.md at report #3** — three point-in-time reports, no index; Phase 0 of the next audit depends on finding these.~~ done — docs/research/README.md created 2026-10-25
6. ~~**Precision hygiene in reports**: state the census definition when quoting counts ("33 @-calls" vs "33 components").~~ done — adopted

## f) Up to 50 things to get done next

Brainstorm, not commitment. Provenance: **[S]** this session, **[C]** carried open from this morning's report, **[B]** standing backlog, **[R]** roadmap fuel. Extra items need docs-health HARVEST routing rigor.

**Execute the audit (ready-to-run):**
~~1. [S] Action ①: adopt utils/format; delete clockDuration/compactDuration/stringOrDash/percent; keep mosGaugePercent; goldens `-update` + review copy change ("1m00s"→"1m"); browser lane.~~ h ec89cce,05e1f30,c1c2ed0,de87205
~~2. [S] Action ②: bump templ-components v1.23.0→v1.24.0 via the 6-step watchlist (corpus regen → map re-audit → hand-rolled re-check → filings re-read → goldens → art-dupl baseline).~~ h de87205,ec89cce,60dc010
~~3. [S] Action ③: ops "Last pass" guard — `p.LastPass.IsZero()` renders "never" instead of zero-time RelativeTime.~~ h c1c2ed0,ec89cce
~~4. [S] Action ④: restore the adoption map's 12 orphaned rows under the main table; split the format row into adopted vs remaining.~~ h ec89cce,c1c2ed0
~~5. [S] Annotate this morning's dedup report + baseline doc: the clockDuration/compactDuration acceptance is superseded by F1 (until ① lands, then the note becomes "resolved by adoption").~~ h ec89cce,c1c2ed0
~~6. [S] `buildflow --build-mode dev` before next push (still not run today, two sessions of changes).~~ v
~~7. [S] Full `golangci-lint run ./...` before next push.~~ h ec89cce
~~8. [S] Comment on upstream #52 with the ops-poller "never ran" use case (EmptyText escape belongs upstream).~~ v posted 2026-10-25 as issuecomment-6393164087; draft in docs/drafts/
~~9. [S] Verify the two inferred immunity claims at call sites: Badge Dot truly unused (#54); Grid typed enums only (#61).~~ v zero Dot: usages; single Grid call site uses the typed enum display.GridCols4; recorded in docs/upstream-filings.md
~~10. [S] Rule for CopyButton: never render inside an htmx-swapped fragment until #63 ships a page-level script mount.~~ v recorded in the adoption map CopyButton row (docs/templ-components-adoption.md)

**Dedup program (carried):**
~~11. [C] HARVEST both status reports into TODO_LIST.md (docs-health HARVEST).~~ h ec89cce
12. ~~[C] Adopt the "grep the dependency graph before accepting" rule into the dedup flow (this session's lesson).~~ done — adopted — the dependency-graph grep is the recorded lesson (18-59 correction)
13. ~~[C] Zero-group gate contract stands: new multi-line clone → extract, never baseline-accept without review.~~ done — standing — the contract held (no red through 2026-10-26)
14. ~~[C] After any art-dupl pin bump: regenerate baseline + suppressed audit in the same change.~~ done — standing — the pin-bump duty; ran with v1.24.0
15. ~~[C] Re-litigate any accepted 1-line group when a third function shares that signature.~~ done — standing — the trigger is recorded in the baseline doc
16. ~~[C] Quarterly suppressed-bucket review (calendar, not ritual).~~ **NOT-DO — event-driven replaced the calendar idea.**
17. ~~[C] Re-check internal/ingest for clone pressure on next mapper change.~~ done — standing — watch-only; no pressure surfaced

**Docs health:**
~~18. [S] Create docs/research/README.md index (3 reports: http-stack, go-cqrs-lite, templ-components).~~ v docs/research/README.md created 2026-10-25 (3 reports indexed)
~~19. [C] ANNOTATE reports non-destructively as items land.~~ done — this pass (2026-10-26 docs-health)
~~20. [S] Decide CHANGELOG policy for docs-only sessions (likely: none).~~ v decided: none for docs-only sessions; policy recorded in docs/research/README.md
21. ~~[C] FEATURES.md sync on next feature change.~~ done — FEATURES is current through 2026-10-26
22. ~~[C] Archived reports stay CLOSED — never harvest.~~ done — standing

**Deep-dive program:**
~~23. [S] Re-run this audit after ①+② land — expect ~94/100 (pin row + ARIA rows flip green); the delta proves the actions moved the number.~~ v census re-run + delta appendix appended to the report (score 91→~93-94: format row flips, pin current; #49/#51 rows stay half)
24. ~~[S] Deep-dive go-appkit (the unaudited third of the UI trio).~~ done — the go-appkit+templ deep dive landed 2026-10-25
25. ~~[S] Deep-dive a-h/templ itself (codegen usage: props patterns, `templ.WithChildren` escape hatch — never audited).~~ **NOT-DO — not audited — a-h/templ codegen usage stays a candidate.**
26. ~~[B] Keep the adoption map's per-component truth ahead of the skill docs (it drifted once; F5 shows it can rot locally too).~~ done — standing — the map stayed authoritative through every bump

**Testing depth:**
27. ~~[S] Actions ①–③ are UI-visible: full browser lane (`scripts/verify-all.sh`) + golden review + pixel-diff baseline refresh are part of the work, not optional.~~ done — the lane ran green over actions ①–③ (05-49/19-09)
28. ~~[C] Poll benchmark (≈82 µs/op 304) after any serve-path change.~~ done — receipts re-measured 2026-10-25 (docs/http-stack.md)
29. ~~[C] Golden `-update` only after the full templ-components bump checklist order.~~ done — ran with v1.24.0
30. ~~[C] Browser lane before any UI-visible change (standing rule; ①–③ trigger it).~~ done — standing — held
31. ~~[R] Fuzz/property tests for ingest parsers (candidate, existence unverified).~~ done — the rapid parser property landed (P19.3)
32. ~~[R] Byte-exact fixture wire-shape assertion (likely over-engineering; judge later).~~ **NOT-DO — not built — the payload-hash golden covers the shape need.**

**Standing backlog (carried, unverified this session):**
33. ~~[C] Telnyx live-API quarterly re-verification (recordings 2026-10-12; window 2027-01, m27.9).~~ done — routed → the Telnyx quarterly row (TODO_LIST)
34. ~~[C] Enrichment path: spec-verified only — live-verify when the window opens.~~ done — routed → the same quarterly row
35. ~~[B] Guard the dependabot lane: templ bumps are manual-only.~~ done — standing — the AGENTS dependabot-templ note stands
36. ~~[B] golangci-lint pin vs go language-level parity on next bump.~~ done — standing — the v2.14/go1.27 parity is documented (AGENTS)
37. ~~[B] After any app.css change: rebuild ONLY via `nix run .#update-styles`.~~ done — standing — the gotcha stands (docs/gotchas.md)
38. ~~[B] go.mod/go.sum: `nix run .#update-vendor-hash` on a CLEAN tree; never nix-hash-fix.~~ done — standing — exercised for real 2026-10-26 (18382a2)
39. ~~[C] E003 second aggregate: hold-flat — revisit only on recorded triggers.~~ done — standing — the HOLD decision stands (AGENTS)
40. ~~[C] `retention.Policy.Admits` ↔ serve-poller `-max-age` interaction on retention changes.~~ done — standing — the invariant holds (AGENTS)
41. ~~[C] auth.go invariant: htpasswd timing pad hashes at the LOADED entries' bcrypt cost.~~ done — standing — preserved; untouched through 2026-10-26
42. ~~[C] Exports/recording stream unbounded — revisit posture if exposure widens.~~ done — standing
43. ~~[C] htmx cadences (30s/60s) tune only on staleness-UX evidence.~~ done — standing
44. ~~[B] Consult docs/gotchas.md before search/FTS5/projection work.~~ done — standing
45. ~~[B] New projections keep BARE record keys via calls.LookupView.~~ done — standing
46. ~~[B] server_timing: manual wrap stays until upstream ships a middleware constructor.~~ done — standing — the manual wrap stands (AGENTS)

**Roadmap fuel:**
47. ~~[C] PII policy: E164 + ParticipantHash exist but NOT enforced on stored fields.~~ done — routed → the PII enforcement row (TODO_LIST)
48. ~~[R] Webphone read-only deep link (sibling integration, no data dependency).~~ done — standing — sibling-side (ROADMAP theme 3)
49. ~~[C] nix-international-telephony lens module sync with serve config knobs.~~ done — standing — module options verified (the deploy row)
50. ~~[C] Periodic progress review against the SUPERB plan.~~ done — standing — the plan corpus is archived; the reviews are continuous

## g) Questions I cannot figure out myself

1. ~~**Execute now or plan only?** Actions ①–④ are scoped, costed (goldens + browser lane + pixel-diff refresh for ①–③), and priority-ranked. Do you want them implemented in this session, or is this audit planning input?~~ done — answered — executed 2026-10-25 (actions ①–④ landed)
2. ~~**Whose duration dialect wins?** Action ① as written adopts upstream's format ("1m", "12h5m") — shorter, one dialect, deletes ~40 lines, but changes rendered copy in the staleness banner. If you prefer the local "1m00s"/"12h05m" copy, ① flips to "don't adopt; file an upstream format-option request" instead. I assumed upstream wins — confirm.~~ done — answered — the upstream dialect won (AGENTS: never hand-roll formatting)
3. ~~**The superseded morning verdict:** annotate the 18-59 status report + baseline doc now (two small doc edits), or only when action ① actually lands?~~ done — answered — annotated same-day (f5; the 18-59 postscript)

---

_Point-in-time snapshot; goes stale by design. Harvest section (f) into TODO_LIST.md via docs-health when the session continues; annotate this file non-destructively as items land._
