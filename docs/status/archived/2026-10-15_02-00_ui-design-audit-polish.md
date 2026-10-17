# Status Report — UI Design Audit & Polish (All Pages)

- **When:** 2026-10-15 02:00 CEST
- **Session scope:** design review of every pbx-lens page against the intended
  user (the owner-operator of pbx.artmann.tech), then implementation of every
  actionable fix, with full verification.
- **Method:** skills loaded (`templ-components`, `frontend-design`); all 5
  routes + 2 fragments + 404 read; all 24 screenshot variants (light/dark ×
  mobile/desktop × 6 views) eyeballed BEFORE and AFTER; library v1.21.0
  sources checked for every prop/API claim.
- **Format note:** `.md` at the operator-demanded path (HTML is normally
  canonical for status reports; explicit user instruction wins).

---

## a) FULLY DONE

| #  | Work                                                                                                                                                                                                                                                                                                                                                                        | Evidence                                                                                                 |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| 1  | Full page inventory + code read: shell/page, calls list, call detail, search, dashboard, staleness/dashboard fragments, 404                                                                                                                                                                                                                                                 | `internal/web/*.templ`, `server.go` routes                                                               |
| 2  | 24 before-screenshots captured and eyeballed (light/dark × mobile/desktop)                                                                                                                                                                                                                                                                                                  | `/tmp/lens-shots-before`                                                                                 |
| 3  | **Heatmap palette leak closed.** `display.Heatmap` is the one component off the tc-* token chain: it fills cells via `rgba(var(--ds-brand-rgb), α)` and rings its peak cell with `var(--ds-brand)`, both defaulting to violet-600. app.css now re-bridges both onto teal (teal-700 light / teal-400 dark), after the imports so the override wins                           | `internal/web/static/app.css` `:root`/`.dark` blocks; teal peak cells verified in light+dark screenshots |
| 4  | **Truthful filter selects.** Direction/Who showed "Inbound"/"AI" while unfiltered (browsers render option 1 as chosen — the control claimed a filter that wasn't set). Explicit `All` options (value `""`, selected when empty) make the control state honest; verified `templ` emits `value=""` literally and `parseFilters` treats `""` as unfiltered (server.go:152-217) | `calls.templ` filterForm; golden diff shows `<option value="" selected>All</option>` ×2                  |
| 5  | **MOS alert de-jargoned.** Removed the internal plan ID "(T15)" from user-facing copy; now states what will make data appear                                                                                                                                                                                                                                                | `dashboard.templ` mosCard                                                                                |
| 6  | **Duration histogram noise off.** Five printed "0"s over empty buckets drowned the one real number; `ShowValues: false` matches the volume chart's established pattern, tooltips carry exact counts                                                                                                                                                                         | `dashboard.templ` durationCard; golden diff drops the spans                                              |
| 7  | **Search hint trimmed.** Dropped the redundant floating "digits-only finds numbers" chip; the prose hint stays under the one-row form (label + input + Search button)                                                                                                                                                                                                       | `search.templ`                                                                                           |
| 8  | **Detail facts consistency.** Direction/Who on the detail page now render the same badges the calls table uses — one vocabulary across pages                                                                                                                                                                                                                                | `detail.templ` detailItems reusing `directionBadge`/`whoBadge`                                           |
| 9  | Test pins updated to the new contracts: `TestSearchHintChip` → `TestSearchHintProse` (pins hint present AND chip absent); new All-option contract pinned in `TestOneFlowFromSeedToDetail`                                                                                                                                                                                   | `dashboard_window_test.go`, `flow_test.go`                                                               |
| 10 | Goldens regenerated (`-update`), diff **reviewed line-by-line via word-diff** (6 files, exactly the intended changes, nothing else)                                                                                                                                                                                                                                         | `internal/web/testdata/*.golden`                                                                         |
| 11 | Full verification: `go build ./...` clean, **14/14 packages green**, `golangci-lint run ./...` → **0 issues**, `verify-ui-contract.js` → **ALL PASS** (copy buttons, `?q=` → 302 → `#turn-N` → amber mark → real fragment scroll), 24 after-screenshots eyeballed                                                                                                           | session runs                                                                                             |
| 12 | AGENTS.md palette doctrine updated with the Heatmap bridge contract (incl. the comma-tripplet legacy-rgba constraint); also fixed the drifted teal values in that bullet (said teal-600/teal-500, reality is teal-700/teal-800)                                                                                                                                             | `AGENTS.md` UI theme bullet                                                                              |
| 13 | Pre-existing `lll` lint violation in `cmd/pbx-lens/serve.go:244` (224-char CSP line) fixed on sight — CI lane would have failed on it                                                                                                                                                                                                                                       | wrapped string concat; tests re-run green                                                                |

## b) PARTIALLY DONE

~~1. **The design audit itself.** Every page was audited and every actionable~~ done — all fixes landed and verified (this file's §a)
in-repo fix landed — but several observations are deliberately parked, not
done (see c) because they need an upstream move, a data state that doesn't
exist yet, or a taste decision that belongs to the owner.
~~2. **Dark-mode verification.** After the final change I eyeballed dark~~ done-in-part — later corpora (39→41 PNGs + the dark sweep flag) closed the sampled gap
desktop dashboard + calls + detail and dark mobile dashboard; I did NOT
re-eyeball every dark variant (e.g. dark mobile search/detail/`?q=`). The
changes were color-token-level and the sampled dark surfaces were correct,
but the sweep is sampled, not exhaustive.
~~3. **`go.sum` modified by `go tool templ generate`** (tool-dep hashes). Done~~ done — committed; harmless
as a side effect of using the go.mod-pinned codegen; committed by the
daemon; harmless but worth knowing it happened.

## c) NOT STARTED (noticed this session, deliberately not begun)

~~1. Volume chart readability during quiet windows: 29 zero-days are honest~~ done-in-part — the M3 empty-ladder made quiet windows honest; the axis filing stays a ROADMAP idea
gaps but render invisibly — one floating bar, no baseline. Library BarChart
has no axis option; needs an upstream filing or an accepted tradeoff.
~~2. Donut at 100%: single slice covers the ring; center label sits on solid~~ NOT-DO — gated on real mixed data
teal (light-mode contrast borderline). Mixed AI/human data would look
right; no action until real mixed data exists.
~~3. Mobile filter-form stacking: library `FormLayoutInline` wraps labels~~ → routed: ROADMAP FilterBar/SegmentBar bump watch
awkwardly on narrow screens. `FilterBar` is ABSENT in v1.21.0 — parked
until a bump ships it.
~~4. Footer copyright line ("© 2026 pbx-lens. All rights reserved.") is~~ NOT-DO — library-baked; revisit only on a real objection
library-baked and not prop-overridable; hand-rolling was judged not worth
a maintenance surface. Revisit only with a real objection.
~~5. Dashboard control rows (presets + range form) kept as two rows — merging~~ → routed: owner decision pack D2
was considered and rejected this session; not implemented.
~~6. T15 MOS poller: the UI placeholder is now clean copy, but the poller/data~~ → routed: T15 MOS poller row
itself remains unbuilt (pre-existing plan item).
~~7. PII policy (E164/ParticipantHash exist but unenforced) — pre-existing~~ → routed: PII enforcement row (one-pager ready)
TODO_LIST.md item, untouched.
~~8. Upstream filings state (#48 BarChart collapse, #49 TableRow focus ring,~~ done — re-verified 2026-10-15 (#48/#49/#51 OPEN, zero movement)
#51 ScrollbackLine hook) not re-checked this session.
~~9. `/api/ops` poller counters are JSON-only; no human-visible surface.~~ done — the /ops page shipped (v0.2.0)
~~10. 404 page and empty-index first-run states were never captured in the~~ done — the probe grew to 39→41 pages incl. 404 + empty-index ladder
24-PNG screenshot set — visually unverified this session.

## d) TOTALLY FUCKED UP

Nothing destroyed: no data loss, no source of truth touched, no reverts of
others' work, no broken state left behind. But three real stumbles, reported
honestly:

1. **Search help-text variant made the page worse before it got better.**
   Moving the hint into `Input.HelpText` pushed the Search button onto its
   own row — I had already ticked the todo "done" before the re-screenshot
   caught it, costing one extra templ+golden+test cycle. Lesson applied
   going forward: a UI change is not done until the after-screenshot is
   eyeballed, and the screenshot belongs INSIDE the todo, not after it.
2. **Ran the wrong templ CLI first.** The devShell templ is v0.3.1020 while
   go.mod pins v0.3.1070 (CI runs `go tool templ generate`). The version
   warning fired and I re-ran with the pinned CLI before building; no
   drifted output landed in a build, but the mistake was real. The devShell
   pin drift is itself a small repo issue worth fixing (see f-26).
3. **Screenshot probe failed on a stale port** (18099 busy) — burned one
   background run, retried with `--port`. Trivial, but the script could
   auto-pick a free port (f-50).

## e) WHAT WE SHOULD IMPROVE

1. **CSS has no palette gate.** `TestNoRawPaletteClasses` only scans `.templ`
   files for `-blue-N`. The ds-brand leak lived in CSS _defaults_ and reached
   production-visible pixels for weeks. A companion gate should assert the
   bridge exists (e.g. styles.css contains the teal `--ds-brand-rgb`
   triplets, or a computed-style probe on a rendered peak cell).
2. **Goldens don't pin pixels, so missing/rotted CSS is invisible to the
   whole suite** (T17 note says exactly this). The cheap mitigation is a
   browser-level computed-style probe for the handful of bridge contracts
   (heatmap cells, peak ring) rather than a full pixel-diff regime.
3. **Golden diffs are single-line HTML and barely reviewable.** This session
   I needed `--word-diff=porcelain` plus grepping to confirm exactly what
   changed. A tiny pretty-diff helper (or per-region goldens) would make
   golden reviews honest instead of skimmable.
4. **Screenshot verification is fully manual.** 24 PNGs per pass, eyeballed.
   A side-by-side before/after montage (or a change-region report) would
   cut the review cost and make the loop tighter.
5. **The full buildflow dev gate was not run this session** — only
   golangci-lint + full test suite. House standard is the gate; next UI
   session should run it (or confirm the 15-min load rule).
6. **UI todo discipline:** fold "eyeball after-screenshot" into each UI
   todo itself (see d-1).
7. **The auto-commit daemon commits without the lint gate** (serve.go lll
   sat in the tree until I hit it). A pre-commit lint hook or a periodic
   gate sweep would stop daemon-committed drift accumulating.

## f) Up to 50 things we should get done next

_Brainstorm ranked roughly by impact, not a commitment list — most items
below the first ~10 are ROADMAP fuel. Items marked (pre-existing) were
noticed, not researched, this session._

**UI/UX (this session's domain):**

~~1. Add a browser-level probe for the All-select: unfiltered list renders~~ done — TestOneFlow pins the All-option contract
`All` as the chosen option; `?direction=inbound` renders `Inbound`.
~~2. Add a computed-style probe pinning the heatmap bridge (peak-cell~~ done — verify-style-contract.js + TestCSSGateBridgesHeatmapBrandToTeal
background/ring = teal) so CSS bridges cannot silently rot.
~~3. Fold upstream templ-components#51 (ScrollbackLine id/class hook) when it~~ → routed: fold-in watch row (templ-components#51)
lands; delete the hand-rolled `transcriptMatchLog` class duplication.
~~4. Verify the stale-banner live transition visually (poll → disappears on~~ done — verify-staleness-live.js (M3)
recovery) — never actually watched it happen.
~~5. Capture the 404 page and the empty-index first-run states (calls +~~ done — 404 + empty-index captures in the corpus
dashboard) into the screenshot set; eyeball them once.
~~6. Verify the recording audio card visually with a seeded recording (the~~ done — the recording-audio detail card capture
24-PNG set only ever showed the empty state).
~~7. Keyboard/AT pass: tab through the calls table (role=link rows), copy~~ done — M5 keyboard/print a11y bundle
buttons, pager buttons; confirm the focus-ring CSS and aria-disabled
semantics survive the changes.
~~8. Verify print output of the detail page after the badge change (badges~~ done — print verified (M5 + probe)
now render in print; print CSS hides buttons).
~~9. Decide the dashboard control-row question once: merge window presets~~ → routed: owner decision pack D2
into the range form, or keep two rows (current) — and record it.
~~10. Upstream filing: BarChart vertical requires `Height` + `!items-stretch`~~ done — already filed as templ-components#48 (2026-10-12)
to not collapse to 0 px (documented as "upstream file candidate" in
dashboard.templ since 2026-10-12 — actually file it).
~~11. Check upstream #49 (TableRow focus ring) status vs the local~~ done — re-verified 2026-10-15 (fold-in watch)
`tr[role="link"]:focus-visible` CSS workaround.
~~12. Consider Eyebrow/SectionHeading grouping on the dashboard (library has~~ done — SectionHeading trial rejected as D6; hover aligned (M11)
both; the dashboard is currently one long card grid).
~~13. Dark-chrome check: pageProps notes the dark theme-color meta is not~~ done — DarkThemeColor IS a prop; pinned #111827 (M11)
prop-overridable in v1.21.0 — verify what mobile browsers actually
show in dark mode.
~~14. Mobile calls table: horizontal scroll hides "Opening line" entirely;~~ → routed: mobile card-list row (owner taste)
consider a mobile card-list layout (bigger change, ROADMAP).
~~15. Decide date/number vocabulary once (see g-1): US "Oct 9, 2025" vs EU~~ → routed: owner decision pack D1
"09.10.2025" across RelativeTime, search hits, detail facts.
~~16. Export honesty note: consider `CollapsibleSection` to cut page noise —~~ → routed: owner decision pack D4
decide once with the operator.
~~17. Accessibility pass on the new detail badges inside DefinitionList~~ done — M5 reasoned AT-safe
(badge inside `<dd>` — confirm screen-reader reading order is sane).
~~18. Long-excerpt truncation check: calls.DefaultExcerptMax is 200 runes —~~ done — wraps cleanly at the ceiling (F61)
visually confirm the table handles the longest real excerpts.
~~19. Theme-toggle aria state after htmx swaps on the dashboard (library~~ done — aria-checked verified surviving swaps (M11)
syncs on render; verify once in-browser).
~~20. Search hit-card hover affordance vs table-row hover — confirm the two~~ done — hover aligned 100/800 (M11)
interactive surfaces feel like one system.

**Verification/DX infra:**
~~21. CSS palette gate (see e-1/e-2): assert the ds-brand bridge in CI.~~ done — verify-style-contract.js + the Go CSS gate
~~22. Before/after screenshot montage tool for review economy (see e-4).~~ NOT-DO — the alert-only pixel-diff (--diff) landed instead
~~23. Golden pretty-diff helper (see e-3).~~ done — scripts/golden-pretty-diff.sh
~~24. Auto-pick a free port in `scripts/lib/ui-probe.js` (see d-3).~~ done — findFreePort + --port in every probe (M4)
~~25. Run the FULL buildflow dev gate (lint+format+test) after the next UI~~ done — green gates followed
session; this session ran only golangci-lint + tests.
~~26. Fix the devShell templ pin drift: nix devShell templ v0.3.1020 vs~~ done-as-doctrine — the flake documents pkgs.templ as FMT-ONLY (drift note)
go.mod tool pin v0.3.1070 — the version warning fires on every regen;
either bump the flake input or document `go tool templ` as the only
supported regen path.
~~27. Sweep `cmd/` + `internal/` for other daemon-committed lint drift~~ done — lint zero (10-33 lineage)
(serve.go lll was pre-existing; golangci is clean now, confirm the
rest of the gate too).
~~28. Ensure the screenshot/verify scripts regenerate `_templ.go` before~~ done — ensureCodegen bootstrap (M4)
building (a stale generated file would build stale markup).
~~29. Consider alert-only pixel-diff on the 24-PNG set (tolerance-based,~~ done — --diff/--save-baseline alert-only pixel diff
not a gate) — catches regressions the goldens cannot see.
~~30. Pin the exact `--port` used by verify/screenshot scripts in AGENTS.md~~ done-superseded — free-port default made pinning moot
commands so stale-port failures are rarer.

**Data/domain (noticed, not started):**
~~31. (pre-existing) T15 MOS poller — UI is ready; data still never arrives.~~ → routed: T15 MOS poller row
~~32. (pre-existing) E164/ParticipantHash PII enforcement (TODO_LIST.md).~~ → routed: PII enforcement row
~~33. Re-check `hasFilters`/`aiValue` redundancy now that selects carry All~~ done — evaluated: already minimal (F59)
(behavior unchanged; code may simplify).
~~34. One time-zone vocabulary check: heatmap/day-buckets are Europe/Berlin;~~ done-as-documented — AGENTS documents the Berlin buckets; server-local renders deliberate
RelativeTime and search hits render server-local — confirm the mixed
vocabulary is intentional and documented.
~~35. `/api/ops` counters: decide whether they deserve a human-visible~~ done — the /ops page (v0.2.0)
surface (currently JSON-only behind auth).
~~36. Verify ETag/304 revalidation behavior after a styles.css regen~~ done — the nav-id 304 probe covers /app.css
(http-stack.md contract — never actually exercised after a CSS change).
~~37. Measure fragment Server-Timing (dashboard 30s / staleness 60s pollers)~~ done — perf rails + BenchmarkConcurrentPoll
to confirm the polling budget holds with the current stats query.
~~38. (pre-existing) retention purge surfacing — Policy/OldViews/Retire run~~ done — the /ops retention card (v0.2.0)
behind `purge`; nothing operator-visible reports what was retired.

**Upstream/library leverage:**
~~39. Re-run the 5-step version-bump watchlist on the next templ-components~~ done — the watchlist ran at v1.23.0
bump (gen-tc-classes → map re-audit → hand-rolled re-check → filings
re-read → golden `-update`).
~~40. When SegmentBar ships upstream (ABSENT in v1.21.0), re-evaluate the~~ → routed: ROADMAP SegmentBar re-eval
AI/human donut against a segment bar with real mixed data.
~~41. When FilterBar ships upstream, re-evaluate the hand-rolled filter row~~ → routed: ROADMAP FilterBar re-eval
and the mobile stacking complaint (c-3).
~~42. Check the three open filings' upstream state (#48/#49/#51) quarterly~~ done — re-verified 2026-10-15 + the fold-in watch table
per docs/upstream-filings.md.

**Docs/process:**
~~43. Harvest this report's section (f) into TODO_LIST.md / ROADMAP.md via~~ done — eleventh pass + this sweep
docs-health HARVEST (the skill flags the loop as not closed until then
— pending your go-ahead, see below).
~~44. Re-render the T23 LineChart-vs-BarChart spike comparison per~~ done — docs/assets/volume-chart-spike-comparison.png
docs/assets/README.md evidence convention (the render "was not
preserved" — noted in the adoption map).
~~45. Check AGENTS.md still respects its 220-line budget after the palette~~ done — AGENTS held at 219 by later passes
bullet grew (it absorbed an overwrite, net +8 lines).
~~46. Record the devShell templ drift as a TODO_LIST entry (or fix in flake)~~ done-as-doctrine — the flake drift note documents it
so the next session does not repeat d-2.
~~47. Add a one-line note to docs/templ-components-adoption.md that Heatmap~~ done — the adoption map carries the Heatmap ds-brand bridge row
rides the ds-brand bridge (the map is the component truth table; the
bridge currently lives only in app.css comments + AGENTS.md).
~~48. Consider a session lessons entry: "UI todos are not done until the~~ NOT-DO — recorded in the reports; the gotchas eyeball lesson covers it
after-screenshot is eyeballed" — it generalizes (per the global memory
rules this goes in the project where learned, i.e. here).
~~49. Re-verify the full 24-PNG dark sweep after the next color-token change~~ NOT-DO — the sampled sweep stood; no color-token change since forced it
(make the sampled dark check exhaustive for color-only diffs).
~~50. Schedule the quarterly Telnyx live-API re-verify window (next opens~~ → routed: Telnyx quarterly row
2027-01 per docs/telnyx-live-api.md — calendar entry already exists;
confirm it is actually on the calendar).

## g) Questions I cannot figure out myself

~~1. **Date/number vocabulary:** the operator context is German (Europe/Berlin~~ → routed: owner decision pack D1
day buckets, German PBX), yet every rendered date is US-style ("Oct 9,
2025 22:31"). Should the UI switch to EU format (09.10.2025 22:31), or is
the current style fine? This touches RelativeTime text, search hits,
detail facts, and chart tooltips — a one-decision, all-surfaces change I
should not make on taste alone.
~~2. **Dashboard control rows:** keep the 7/30-day presets and the from/to~~ → routed: owner decision pack D2
range form as two separate rows (current), or merge them into one denser
control row? I chose "keep" for separation of navigation vs filtering;
it is a taste call and yours to overrule.
~~3. **Volume chart during quiet weeks:** the 30-day chart with one spike~~ done-in-part — the empty-state honesty ladder landed; the axis ask parked with #48
renders 29 invisible zero-days — honest but axis-less, and it can read as
"broken" at a glance. Do you ever look at this chart during quiet
stretches (i.e., is an upstream axis/baseline filing worth pursuing), or
is the spike-only reading acceptable?

---

_Report written per the status-report skill; `.md` format at the explicitly
demanded path (user instruction overrides the HTML default). No manual
commit: the auto-commit daemon picks this file up (Crush harness forbids
unrequested commits)._
