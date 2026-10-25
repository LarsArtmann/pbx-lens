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

| #  | Item                                                                                                                                                                                                                                                       | Evidence                                                  |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| 1  | Both matching skills loaded before any tool action                                                                                                                                                                                                         | session start                                             |
| 2  | Phase 0: prior reports located (2, no index README); adoption map read — its broken table spotted immediately                                                                                                                                              | docs/research/, docs/templ-components-adoption.md:69-80   |
| 3  | Version currency: pin v1.23.0 (go.mod:24-27), published v1.24.0 (2026-10-23), upstream HEAD = 3 docs commits past release; v1.24.0 body read from CHANGELOG                                                                                                | go list -m -versions, CHANGELOG.md:34+                    |
| 4  | Full component census: 33 distinct @-calls across 6 .templ surfaces + Go-side calls (SimpleNav/ThemeToggle/RelativeTime/DefaultPageProps)                                                                                                                  | grep census (session log)                                 |
| 5  | Capability ground truth read at the exact pin, not from drifting docs: utils@v1.23.0/format/format.go:75-155 read line-by-line and diffed against the 4 local twins                                                                                        | module cache                                              |
| 6  | Live upstream filing states: #48 #49 #50 #51 #52 #59 #60 #61 #63 — all OPEN; workarounds located in-code (dashboard.templ:441, app.css:124)                                                                                                                | gh issue list + in-repo greps                             |
| 7  | The split-brain discovery + output diff: local compactDuration renders "1m00s"/"12h05m" vs library "1m"/"12h5m"; clockDuration/stringOrDash byte-identical sub-day; percent identical today (all ratio sources zero-guarded) but missing NaN→"—" hardening | format.go vs calls.templ:392-422, dashboard.templ:712-714 |
| 8  | 38-row capability inventory, 10 N/A rows each with a stated reason, score derived and published (23×1 + 5×0.5 of 28 = 91)                                                                                                                                  | report appendix                                           |
| 9  | 7 findings cards + version-currency table + 10-row priority matrix (sorted desc) + how-to-verify appendix                                                                                                                                                  | report                                                    |
| 10 | HTML composed by copying the editorial template and editing in place (no line-splicing); one failed edit (HTML entity mismatch) caught and fixed in the next step                                                                                          | multiedit log                                             |
| 11 | Self-consistency gate: denominators agree everywhere (91 / 23/28 / N=38 / A=28), priority table descending, 4 spot-checked citations in range, zero demo leftovers                                                                                         | gate greps (session log)                                  |
| 12 | dprint check attempted per repo discipline (no HTML plugin configured — nothing to format); git state verified (daemon owns commits)                                                                                                                       | nix develop -c dprint check                               |

## b) PARTIALLY DONE

| # | Item                                                                                                                                                                                      | State                                                                                                                                                                                                                                                                           |
| - | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | The audit's own action plan (items 1–4)                                                                                                                                                   | Reported and prioritized, **not executed** — the ask was "are we", not "make it so". All four are ready-to-run with known cost (goldens + browser lane for #1–#3).                                                                                                              |
| 2 | Component-count precision                                                                                                                                                                 | "33 components" = the @-prefixed census; ~37 counting Go-expression calls (SimpleNav, ThemeToggle, RelativeTime, DefaultPageProps). Conservative direction, still imprecise.                                                                                                    |
| 3 | Two immunity claims inferred, not read: #54 (Badge "Dot unused") and #61 (Grid "typed enums only") rest on the adoption map's word — the map whose stale format row is itself finding F5. | report F7                                                                                                                                                                                                                                                                       |
| 4 | Cross-document consistency                                                                                                                                                                | This morning's dedup verdict (accepted clockDuration/compactDuration pair, recorded in docs/status/2026-10-24_18-59 and the baseline doc's rationale) is superseded by finding F1 — but only the new HTML report says so. The two older artifacts still assert the old verdict. |
| 5 | Deep-dive coverage of the UI trio                                                                                                                                                         | templ-components audited; go-appkit (the third member) has never been deep-dived.                                                                                                                                                                                               |

## c) NOT STARTED (deliberate waits / out of scope)

| # | Item                                                                                    | Why                                        |
| - | --------------------------------------------------------------------------------------- | ------------------------------------------ |
| 1 | Action ① — adopt utils/format, delete 4 twins (~40 lines), golden refresh, browser lane | Awaiting user go-ahead (see g1/g2)         |
| 2 | Action ② — v1.24.0 bump via the 6-step watchlist                                        | Same                                       |
| 3 | Action ③ — ops "Last pass" zero-time guard ("never" copy)                               | Same                                       |
| 4 | Action ④ — repair the adoption map's broken table + stale format row                    | 2-line fix, bundled with ①                 |
| 5 | HARVEST of today's two status reports into TODO_LIST.md                                 | User instruction: report, then wait        |
| 6 | docs/research/README.md index (3 reports, none indexed)                                 | Candidate, not ordered                     |
| 7 | CHANGELOG entry                                                                         | Docs-only session — likely none; undecided |

## d) TOTALLY FUCKED UP

**Nothing at that severity this session.** Near-misses, honestly:

1. **The biggest error adjacent to this session is this morning's, which this session had to correct**: the dedup pass accepted the clockDuration/compactDuration pair as "different documented business rules" without checking the dependency graph — the library shipped all four twins at the pinned version all along. The dedup skill's own doctrine ("hand-rolled code duplicating built-in functionality") was applied without its cheapest step. This session found it, quantified it, and recorded the correction — but the correction is prose in a third file until action ① lands.
2. First multiedit batch: 1 of 5 edits failed (I sent the literal `·` where the template has `&middot;`) — caught by grep, fixed immediately.
3. Session workflow friction: the rg-protection hook blocked four command shapes; worked around via the Grep-tool/plain-grep patterns. Cost: a few minutes, no damage.

## e) WHAT WE SHOULD IMPROVE

1. **Dedup passes must grep the dependency graph before accepting any "similar shape" as intentional.** The library check is one grep and today it flips a verdict recorded six hours earlier. Rule candidate: no acceptance without a `go list -m all`-informed look at what the pinned deps already provide.
2. **When an audit supersedes an earlier verdict, annotate the earlier artifact in the same step** (or explicitly accept owning the contradiction until the fix lands) — two docs currently disagree silently, which is exactly the split-brain class this repo writes guard tests against.
3. **Verify immunity claims at the call site, not from the map** — the map is authoritative for intent, not for current code; F5 proves maps rot.
4. **When editing templated HTML, replace whole elements and check entity encoding first** (`&middot;` vs `·` cost one retry).
5. **Start docs/research/README.md at report #3** — three point-in-time reports, no index; Phase 0 of the next audit depends on finding these.
6. **Precision hygiene in reports**: state the census definition when quoting counts ("33 @-calls" vs "33 components").

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
12. [C] Adopt the "grep the dependency graph before accepting" rule into the dedup flow (this session's lesson).
13. [C] Zero-group gate contract stands: new multi-line clone → extract, never baseline-accept without review.
14. [C] After any art-dupl pin bump: regenerate baseline + suppressed audit in the same change.
15. [C] Re-litigate any accepted 1-line group when a third function shares that signature.
16. [C] Quarterly suppressed-bucket review (calendar, not ritual).
17. [C] Re-check internal/ingest for clone pressure on next mapper change.

**Docs health:**
~~18. [S] Create docs/research/README.md index (3 reports: http-stack, go-cqrs-lite, templ-components).~~ v docs/research/README.md created 2026-10-25 (3 reports indexed)
19. [C] ANNOTATE reports non-destructively as items land.
~~20. [S] Decide CHANGELOG policy for docs-only sessions (likely: none).~~ v decided: none for docs-only sessions; policy recorded in docs/research/README.md
21. [C] FEATURES.md sync on next feature change.
22. [C] Archived reports stay CLOSED — never harvest.

**Deep-dive program:**
~~23. [S] Re-run this audit after ①+② land — expect ~94/100 (pin row + ARIA rows flip green); the delta proves the actions moved the number.~~ v census re-run + delta appendix appended to the report (score 91→~93-94: format row flips, pin current; #49/#51 rows stay half)
24. [S] Deep-dive go-appkit (the unaudited third of the UI trio).
25. [S] Deep-dive a-h/templ itself (codegen usage: props patterns, `templ.WithChildren` escape hatch — never audited).
26. [B] Keep the adoption map's per-component truth ahead of the skill docs (it drifted once; F5 shows it can rot locally too).

**Testing depth:**
27. [S] Actions ①–③ are UI-visible: full browser lane (`scripts/verify-all.sh`) + golden review + pixel-diff baseline refresh are part of the work, not optional.
28. [C] Poll benchmark (≈82 µs/op 304) after any serve-path change.
29. [C] Golden `-update` only after the full templ-components bump checklist order.
30. [C] Browser lane before any UI-visible change (standing rule; ①–③ trigger it).
31. [R] Fuzz/property tests for ingest parsers (candidate, existence unverified).
32. [R] Byte-exact fixture wire-shape assertion (likely over-engineering; judge later).

**Standing backlog (carried, unverified this session):**
33. [C] Telnyx live-API quarterly re-verification (recordings 2026-10-12; window 2027-01, m27.9).
34. [C] Enrichment path: spec-verified only — live-verify when the window opens.
35. [B] Guard the dependabot lane: templ bumps are manual-only.
36. [B] golangci-lint pin vs go language-level parity on next bump.
37. [B] After any app.css change: rebuild ONLY via `nix run .#update-styles`.
38. [B] go.mod/go.sum: `nix run .#update-vendor-hash` on a CLEAN tree; never nix-hash-fix.
39. [C] E003 second aggregate: hold-flat — revisit only on recorded triggers.
40. [C] `retention.Policy.Admits` ↔ serve-poller `-max-age` interaction on retention changes.
41. [C] auth.go invariant: htpasswd timing pad hashes at the LOADED entries' bcrypt cost.
42. [C] Exports/recording stream unbounded — revisit posture if exposure widens.
43. [C] htmx cadences (30s/60s) tune only on staleness-UX evidence.
44. [B] Consult docs/gotchas.md before search/FTS5/projection work.
45. [B] New projections keep BARE record keys via calls.LookupView.
46. [B] server_timing: manual wrap stays until upstream ships a middleware constructor.

**Roadmap fuel:**
47. [C] PII policy: E164 + ParticipantHash exist but NOT enforced on stored fields.
48. [R] Webphone read-only deep link (sibling integration, no data dependency).
49. [C] nix-international-telephony lens module sync with serve config knobs.
50. [C] Periodic progress review against the SUPERB plan.

## g) Questions I cannot figure out myself

1. **Execute now or plan only?** Actions ①–④ are scoped, costed (goldens + browser lane + pixel-diff refresh for ①–③), and priority-ranked. Do you want them implemented in this session, or is this audit planning input?
2. **Whose duration dialect wins?** Action ① as written adopts upstream's format ("1m", "12h5m") — shorter, one dialect, deletes ~40 lines, but changes rendered copy in the staleness banner. If you prefer the local "1m00s"/"12h05m" copy, ① flips to "don't adopt; file an upstream format-option request" instead. I assumed upstream wins — confirm.
3. **The superseded morning verdict:** annotate the 18-59 status report + baseline doc now (two small doc edits), or only when action ① actually lands?

---

_Point-in-time snapshot; goes stale by design. Harvest section (f) into TODO_LIST.md via docs-health when the session continues; annotate this file non-destructively as items land._
