# Status Report — templ-components max-adoption UI pass

**When:** 2026-10-12 13:27 CEST
**Scope of this report:** this session only — the "why is the web app ugly /
are we leveraging templ-components to the MAX" audit + rework of the serve web
app (`internal/web`). The concurrent htpasswd/httputil session's work is
reported only where it collided with this session. No other research done.
**Verification state at writing:** `go build ./...` green; `go test
./internal/web/` green; `nix build` green (vendorHash repaired via
`nix run .#update-vendor-hash`); full-suite has ONE red test owned by the
other session (see d1).

---

## a) FULLY DONE

1. **Skill-gated start.** Loaded `templ-components`, `library-deep-dive`,
   `frontend-design` SKILL.mds before touching code. The consumer guide's
   "track adoption in your AGENTS.md" tip drove the doc updates.
2. **Real-API audit, not doc-trust.** Enumerated the pinned
   `templ-components@v1.21.0` surface from the module cache (every package's
   exported components, props structs, enums, icon names). This caught the
   skill docs drifting ahead of the pinned tag AND a false claim in our own
   AGENTS.md adoption map (see a3).
3. **AGENTS.md adoption-map corrections.** `feedback.ProgressBar` EXISTS in
   v1.21.0 (the old map listed it absent); `utils/format` genuinely absent
   (verified — our hand-rolled duration/percent helpers stay justified);
   `FilterBar`/`SegmentedControl`/`SegmentedControl`/`errorpage` genuinely
   absent; `FilterInput`/`FilterDropdown` exist.
4. **Baseline capture.** Dumped all five rendered pages (list, dashboard,
   search, detail, empty-search) to /tmp via a temporary in-package test
   before changing anything; eyeballed structure and `bg-blue-*` counts.
5. **13-finding gap analysis**, the headline items: three hand-rolled copies
   of the same raw submit-button class string; a raw search input; raw reset
   anchors; hand-rolled main-container/2-col-split/stack wrappers; zero icons
   anywhere; empty `@theme` (default Tailwind blue = the "generic" look);
   export CSV/JSONL routes linked from NOTHING; a duplicate nested `<main>`
   landmark (Base already emits one); transcript as plain gray bubbles while
   `Scrollback` ships in the pin; AI/human hand-rolled flex bar while
   `PieChart(Donut)` ships; MOS as plain text while `CircularProgress` ships;
   StatCards/EmptyStates used without Icon/Tone.
6. **Signal-palette theming.** `app.css` overrides `--color-tc-primary`/
   `-hover`/`--color-tc-info` to teal-600/700/500 in an `@theme` block placed
   AFTER the `templ-components-theme.css` import (deliberate: imports hoist,
   later @theme wins the merge; the theme file binds
   `--color-blue-600: var(--color-tc-primary)`). Verified the compiled
   styles.css chain: `--color-tc-primary:#0d9488` lands, one declaration, no
   cycle. Every library blue re-skins from this one block.
7. **Shell rebuild (page.templ).** Duplicate `<main>` fixed (Base emits the
   landmark; shell now renders `layout.Container` + `navigation.Footer` only).
8. **Calls page.** `forms.Form(FormGet, FormLayoutInline)` + `display.Button`
   Apply (submit) / Reset (ghost, Href) replace all raw controls; Export CSV
   button in `PageHeader.Action` (secondary variant, download icon +
   `download` attr) surfacing the previously-unlinked `/api/export.csv`;
   EmptyState icon.
9. **Search page.** `forms.Form` + `forms.Input(InputSearch, EnterKeyHint)`
   - `display.Button` replace the raw input/button; EmptyState icon; honest
     helper copy unchanged.
10. **Dashboard.** Range form on library Form+Button; StatCards get icons
    (Phone/Calendar/Microphone/Clock) + Tones (blue/green/purple/yellow);
    `layout.Split(1-3)` + `layout.Stack(LG)` replace hand-rolled grid/space-y
    (extracted as `dashboardMain`/`dashboardAside` — templ forbids `@`
    expressions inside struct literals); AI/human mix is now a
    `PieChart` donut with CenterLabel + legend, exact counts preserved by
    baking them into legend labels ("AI · N"/"Human · N") — data-honesty
    contract kept, tests updated; MOS card gained a `CircularProgress` gauge
    (`mosGaugePercent` maps MOS 1–5 → 0–100 clamped; probed
    `aria-valuenow="80"` for MOS 4.2).
11. **Detail page.** `Split`+`Stack`; source kind now a neutral Badge under
    the title (was plain "source: …" text); transcript is `display.Scrollback`
    (terminal-style voice log — the page's signature element), `Stagger:
    false` with a written reason (reading content, not replay animation);
    role vocabulary preserved (user/assistant); tone mapping user=info,
    assistant=success; recording fallback is `SimpleEmptyState`; 404 page
    uses EmptyState Icon + ActionText/ActionHref (dropped the raw wrapper
    div + button).
12. **Tests honest and green.** Donut assertions moved to the new copy
    ("50.0%", "AI · 1", "Human · 2"); flow/detail/staleness tests untouched
    and passing; all `internal/web` tests green; `go build ./...` green.
13. **Nix path green.** `nix build` initially failed on vendorHash drift
    (from the concurrent session's go.mod addition); repaired with the
    documented `nix run .#update-vendor-hash` (never hand-pasted) — green,
    which also proves the preBuild `templ generate` + tailwind codegen path
    handles the new sources.
14. **Docs synced.** AGENTS.md: adoption map rewritten (Container/Split/
    Stack, Footer, Scrollback, PieChart donut, Form/Input, CircularProgress,
    icons rows added; Base-emits-main warning; ABSENT row corrected) + a new
    theme bullet with the import-ordering rationale and a "never hand-write
    bg-blue-* here" rule. CHANGELOG.md: Added entries for the UI pass and
    theming; Changed entry for the test-copy move. TODO_LIST.md: removed the
    now-superseded SegmentBar row per that file's DONE convention.
15. **Cleanup.** Temporary dump harness trashed; locally built styles.css
    trashed so `TestStylesUnbuiltReturns404` holds its clean-checkout
    contract (nix/BuildFlow rebuild the artifact).
16. **Root-caused the one flaky moment.** A silently-hidden `templ generate`
    failure (`>/dev/null 2>&1`) left stale generated code and made
    TestDashboardCountsMatchFixture fail-then-pass across runs; verified the
    generated file content directly, regenerated with visible output — stable.

## b) PARTIALLY DONE

~~1. **Visual verification is structural, not pixel-level.** Every new~~ done — screenshot probe + goldens + style contracts landed (M1/M3/M4)
   component's markup was grepped in rendered HTML (donut center label,
   gauge, scrollback tags, icons, container, footer, single `<main>`), but
   no browser screenshot pass happened — the frontend-design skill explicitly
   says "a picture is worth 1000 tokens" and asks for self-critique from
   renders. Unknown: donut center-label overflow at "100.0%", teal contrast
   in dark mode, 375px behavior.
~~2. **Mobile nav unverified.** `SimpleNav` + `CurrentPath` is wired, but I~~ done — the probe corpus covers 375px (mobile captures eyeballed)
   never rendered/probed a narrow viewport; the library has MobileMenu
   components we do not use — whether SimpleNav degrades acceptably on a
   phone is unchecked.
~~3. **htmx error handling claimed but never wired (pre-existing, preserved).**~~ done — htmx ErrorHandlingConfig shipped (dashboard density pass, CHANGELOG)
   The old page.templ comment said "…htmx error handling" but
   `GlobalErrorHandling` appears NOWHERE in our templates (verified by grep).
   Polled regions have no client-side fetch-failure feedback. I neither
   introduced nor fixed this — it is exactly the kind of gap this session
   was hunting, found one step too late.
~~4. **StatCard Change/Trend fields unused.** The week-over-week delta renders~~ done — week-over-week deltas shipped (R8)
   as a plain `<p>` under the volume chart while `StatCardProps` ships
   `Change`/`Trend`. Arguably the delta belongs to the chart card, but the
   library affordance went unexamined.
~~5. **Search hit cards remain hand-rolled** (intentional similarity — they are~~ done — hover aligned with the calls table (v0.2.0 dark-mode pass)
   semantic link blocks, not cards) but got zero polish: no hover transition,
   no chevron affordance, no focus-visible ring beyond browser default.
~~6. **Lint debt in the package is real but not mine:** wsl_v5/goimports/gci~~ done — lint driven to zero (10-33)
   findings in freshness.go, recording.go, search.go, server.go, timing.go —
   all touched by the concurrent session, all committed by the auto-daemon.
   Not fixed (not my files); BuildFlow gate will own them.
~~7. **`TestBuildServiceConfigWiring` red** (concurrent session's in-flight~~ done — the peer's htpasswd work landed; suite green since
   htpasswd lazy-read TDD work in cmd/pbx-lens). Flagged, not touched.

## c) NOT STARTED

~~1. Browser screenshot/golden regression gate for the UI (nothing exists).~~ done — screenshot-probe + goldens + the verify-all lane
~~2. Dashboard range-picker presets (7/30-day) and digits-only search hint~~ done — presets shipped; the chip superseded by the R6 hint copy
   chip — pre-existing TODO_LIST items, untouched.
~~3. Staleness fragment carrying active filters (pre-existing TODO, untouched).~~ NOT-DO — the banner renders unfiltered-first-page only, by design
~~4. `export_cursor_stalled` surfaced in the export UI (pre-existing TODO).~~ done — export honesty note (ba6ec8f)
~~5. Recording-proxy timeout-test hardening (injected deadlines) — pre-existing.~~ → routed: recording timeout-test row (TODO_LIST)
~~6. `errorpage`/`NotFound404` adoption — package absent in v1.21.0; revisit on~~ → routed: ROADMAP errorpages/security revisit
   next templ-components bump.
~~7. `SegmentBar` swap-in — moot for now (donut covers the need); revisit only~~ → routed: ROADMAP SegmentBar re-eval
   if the donut reads wrong visually.
~~8. `utils/format` adoption — confirmed absent in v1.21.0; revisit on bump.~~ → routed: ROADMAP utils/format bump watch
~~9. library-deep-dive's canonical HTML research report — the session collapsed~~ done — the research report landed (399fd50, docs/research/)
   the audit into fix-work + this status report instead.
~~10. Favicon / theme-color meta on the Base shell.~~ done — favicon.svg shipped + ThemeColor/DarkThemeColor props (calls.templ)

## d) TOTALLY FUCKED UP

Nothing shipped broken — everything below was caught and fixed before the
session ended, but each was avoidable:

1. **Illegal templ syntax, three files at once.** I wrote `@layout.Stack(…) {}`
   inside `SplitProps`/`PageHeaderProps` struct literals — `@` expressions are
   children-position syntax, not Go-expression syntax. Generation failed in
   calls/detail/dashboard; fixed by extracting named component functions.
   Cost: one full generate-build cycle. Should have known templ's grammar
   before writing three call sites in one sweep.
2. **Self-inflicted stale-generation confusion.** Piping `templ generate` to
   `/dev/null 2>&1` hid a failure; the subsequent test run used stale
   generated code and TestDashboardCountsMatchFixture failed-then-passed
   across runs. I briefly suspected test races/globals before finding my own
   command hygiene was the cause. Never hide codegen output.
3. **CSS variable cycle, nearly shipped.** First placement of the @theme
   override was BEFORE the theme-file import. Since CSS `@import` hoists, the
   theme file's `--color-tc-primary: var(--color-blue-600)` would have
   overridden my colors AND formed a resolution cycle (both variables go
   invalid at computed-value time — buttons would have rendered unstyled).
   Caught by reasoning about hoisting before building; the fix (override
   after imports) is documented in AGENTS.md. I initially wrote it wrong.
4. **Dump harness mkdir miss.** First run of the temporary dump test failed
   on a missing /tmp dir; later the dir vanished mid-session (external /tmp
   cleanup) breaking the harness again. Trivial, but two avoidable red runs.
5. **Built the artifact the test contract forbids.** Building styles.css
   locally flipped TestStylesUnbuiltReturns404 red; I should have read that
   test's contract before generating and either built into /tmp or trashed
   the artifact proactively.
6. **Non-atomic page-wide trust in the old comment.** I trusted (and
   reproduced) the old shell comment's claim of "htmx error handling" without
   grepping for the component (see b3). The session's whole thesis was
   "claims ≠ adoption map" — and I relayed one unverified claim in my own
   shell rewrite.

## e) WHAT WE SHOULD IMPROVE

1. **Add a pixel-level feedback loop.** HTML-structure asserts cannot see
   ugliness — the literal user complaint. A screenshot script (light/dark ×
   mobile/desktop × 5 pages) run after UI changes would have caught contrast
   and overflow questions this session left open.
2. **Ban raw palette classes in our .templ files.** The three-copies-of-
   blue-button drift existed for months. A CI grep gate (`bg-blue-`/
   `rounded-md bg-` in internal/web/*.templ fails) keeps the theming token
   chain authoritative. Library classes re-skin; ours don't.
3. **Never swallow codegen output.** `templ generate` failures must surface;
   a `-` redirect of its stderr cost an investigation.
4. **Re-verify the adoption map against the module cache on every version
   bump.** It carried a false row (ProgressBar) — a split brain between docs
   and reality, the exact class of error the map exists to prevent.
5. **Copy-coupled assertions.** Tests asserting exact UI phrasing ("50.0%
   AI") break on every copy tweak. Assert structure/aria where possible;
   reserve copy asserts for honesty contracts (counts, em-dashes).
6. **Wire GlobalErrorHandling (b3).** One component, real UX payoff: polled
   regions currently fail silently when the server blips.
7. **Extract local templ helpers for repeated wiring** (StatCard tiles,
   EmptyState copy pattern) if the pages grow — repetition is starting.

## f) Up to 50 things to get done next

_Braintorm, sorted roughly by impact × ease. UI items first (session
context); the tail is ROADMAP fuel. Not commitments._

**Verify/correct this session's work:**

~~1. Screenshot pass: 5 pages × light/dark × 375px/1280px; fix what looks wrong.~~ done — the probe corpus grew to 41 PNGs (light/dark × mobile/desktop)
~~2. Donut center-label overflow probe ("100.0%", 4+ char strings).~~ done — corpus eyeball + goldens
~~3. Teal-600/500 contrast audit (AA) in light and dark, buttons + badges +~~ done — T22 contrast-audited palette tokens
   focus rings.
~~4. Mobile nav behavior probe (SimpleNav at 375px; adopt MobileMenu if broken).~~ done — mobile captures in the corpus
~~5. Wire `htmx.GlobalErrorHandling` into the shell (polled regions fail~~ done — ErrorHandlingConfig shipped
   silently today).
~~6. StatCard `Change`/`Trend` trial for the week-over-week delta.~~ done — StatCard deltas shipped (R8)
~~7. grep-gate CI check: no raw `bg-blue-*`/palette classes in pbx-lens~~ done — palette_gate_test.go
   .templ files.
~~8. Fix lint debt in internal/web (wsl_v5/goimports/gci — concurrent session's~~ done — lint zero (10-33)
   files; coordinate).
~~9. Fix/await `TestBuildServiceConfigWiring` (concurrent session's red).~~ done — the peer's work landed; suite green
~~10. Codegen-drift guard: CI runs `templ generate` and fails on diff.~~ done — templ_sync_test.go
~~11. Keyboard pass: tab through list rows, filter form, search; confirm~~ done — M5 a11y bundle + the #49 focus ring
    Row.Href keyboard support and visible focus.
~~12. Table `CellPadding` compact trial for the dense call list.~~ NOT-DO — density fine at the current scale
~~13. `tc-content-auto` on the calls table (content-visibility for long lists).~~ NOT-DO — keyset paging keeps lists short

**UI polish (library affordances still unused):**
~~14. `navigation.Pagination` (or LoadMore) instead of the single "Older calls"~~ done-by-decision — two honest numberless buttons (pagerRow rationale)
button.
~~15. `navigation.Breadcrumbs` on detail (Calls / {kind} / {id}).~~ done — breadcrumbs on the detail header (T20/T21)
~~16. Dashboard range presets (7/30-day buttons; SegmentedControl when~~ done — 7/30-day presets
published).
~~17. Digits-only search hint chip under the search form.~~ done-superseded — the R6 hint copy tells the digits truth
~~18. Search hit cards: hover transition, chevron-right icon, focus ring.~~ done — hover aligned (v0.2.0)
~~19. `display.Eyebrow` over page titles for hierarchy personality.~~ NOT-DO — taste call; never adopted
~~20. `CopyButton` for phone numbers / native IDs on the detail page.~~ done — CopyButton on id + phone numbers (T20)
~~21. `display.Heatmap` (hour × day call density) as the dashboard's next~~ done — weekday×hour heatmap shipped (T13-T19)
signature chart.
~~22. `LineChart` trial vs BarChart for the volume trend.~~ NOT-DO — the Sparkline covers the trend
~~23. `DataTable` evaluation (sorting) for the calls table.~~ NOT-DO — no sorting need over keyset cursors
~~24. Scrollback timestamps if transcript turns ever carry timing.~~ NOT-DO — turns carry no timing
~~25. Detail-page search-highlight jump (link "?q=" through to transcript).~~ done — the ?q= jump shipped (T21)
~~26. Recording card: duration/size metadata beside the player.~~ NOT-DO — the facts row carries duration
~~27. Footer: binary version string + "read-only index" note.~~ done — footer + version string (v0.2.0 footer fix)
~~28. Favicon + theme-color meta.~~ done — favicon.svg + theme-color props
~~29. EmptyState ActionText on the calls page (point at the ingest commands or~~ done — Clear-filters EmptyState CTA (T12)
docs).
~~30. Staleness banner carries active filters (pre-existing TODO).~~ NOT-DO — unfiltered-only by design
~~31. Surface `export_cursor_stalled` in the export UI (pre-existing TODO).~~ done — export honesty note
~~32. Print stylesheet for call details (cheap, occasionally useful).~~ done — @media print block

**Upstream/docs hygiene:**
~~33. File the v1.21.0-vs-skill-docs drift upstream (verify-before-filing~~ done — filed as templ-components#50
first): skill lists FilterBar/SegmentedControl/errorpage as existing;
ProgressBar/format absence claimed in consumer AGENTS files.
~~34. Re-verify whole adoption map at the next templ-components bump~~ done — the watchlist ran at the v1.23.0 bump
(SegmentBar/errorpage/utils/format candidates).
~~35. Extract dashboard.go/queries.go nolint rationales into docs (pre-existing~~ done — docs/nolint-ledger.md carries them
TODO).
~~36. Consider `docs/DOMAIN_LANGUAGE.md` entries: "observation", "stream",~~ done — T26 added the rows (voice log etc.)
"voice log" (Scrollback), "signal palette".
~~37. Move this report's section (f) harvestables into TODO_LIST/ROADMAP~~ done — eighth pass + this sweep
(docs-health HARVEST mode).
~~38. CHANGELOG: fold the "Changed" test-copy bullet into the UI-pass Added~~ done — folded under v0.1.0
entry if a release is cut before then (avoid a one-line section).

**Deeper bets (ROADMAP fuel):**
~~39. Golden-HTML snapshot tests for the five pages (library-style goldens).~~ done — T22 golden net
~~40. Visual regression CI job (screenshots) if the LAN tool keeps growing UI.~~ done-in-part — the local verify-all lane is the chosen level; no CI visual job (g2 resolved)
~~41. `errorpage`/`NotFound404` adoption when templ-components publishes it.~~ → routed: ROADMAP errorpages revisit
~~42. `utils/format` adoption when published (delete local duration/percent~~ → routed: ROADMAP utils/format bump watch
helpers — mind the split brain).
~~43. `SegmentBar` if the donut reads wrong (blocked upstream).~~ → routed: ROADMAP SegmentBar re-eval
~~44. `AppShell` evaluation if the nav grows past three links.~~ NOT-DO — SimpleNav holds at 4 links
~~45. webphone read-only deep link (SUPERB plan sibling) — detail-page~~ → routed: ROADMAP theme 3 (webphone, sibling-side)
affordance, data stays one-way.
~~46. nix-international-telephony module: verify the packaged binary ships the~~ → routed: host deploy row (TODO_LIST)
new styles (preBuild codegen) on a real host deploy (pre-existing BLOCKED
deploy task).
~~47. Per-source filter chips on the calls list (FilterChips absent in pin;~~ NOT-DO — no operator need surfaced
hand-roll only if operators actually need it).
~~48. Dashboard "quiet index" honesty: idle-index drift note in the chart card~~ done — an idle index drifts out of the chart (M3 empty-ladder)
(design, not code).
~~49. Accessibility sweep with a real screen reader on the transcript pane.~~ done — Scrollback AriaLabel + the M5 a11y bundle
~~50. Upstream contribution: consumer-tip adoption-map example in the~~ NOT-DO — skill-side idea; never picked up
templ-components skill could cite pbx-lens as the worked example.

## g) Questions I cannot figure out myself

~~1. **Palette taste-call:** is the teal "signal" accent right for pbx-lens, or~~ resolved-by-ship — teal is the shipped theme (v0.1.0/v0.2.0; doctrine in AGENTS.md)
   do you want a different accent family? It is now a three-line change in
   `app.css` (`--color-tc-primary/-hover/-info`), so this is purely a taste
   decision I cannot derive from the codebase.
~~2. **Verification bar:** for a LAN htpasswd'd operator tool, do you want a~~ resolved — goldens + screenshot corpus + browser contracts = the chosen bar (local, not CI)
   screenshot/golden-UI regression gate (CI weight, catches "ugly" early), or
   is HTML-assert + occasional eyeball the right level?
~~3. **Concurrent session ownership:** the htpasswd/httputil session left~~ done — the peer's work landed; suite green since
   `TestBuildServiceConfigWiring` red and lint debt in five web files — do I
   wait for it to land, or take over and finish that work now (risk: clobbering
   an active editor buffer)?

---

_Point-in-time snapshot; goes stale. HARVEST section (f) into
TODO_LIST.md/ROADMAP.md per docs-health if acted on later._
