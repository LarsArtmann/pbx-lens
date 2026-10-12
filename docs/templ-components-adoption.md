# templ-components v1.21.0 adoption map (consumer truth for THIS repo)

<!-- Extracted from AGENTS.md 2026-10-12 (220-line budget). The skill docs
     drift ahead of the pinned tag — this table is the truth for pbx-lens.
     Stay on the PUBLISHED templ-components version (no local replace —
     breaks hermetic nix); hand-roll missing thin pieces instead. -->

| Component                                                                                   | Status            | Note                                                                                                                            |
| ------------------------------------------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| layout.Base/DefaultPageProps                                                                | used              | `.Title` set; CSSPath defaults to /app.css; Base EMITS the `<main>` landmark — never nest a second main in the shell            |
| layout.Container / Split / Stack                                                            | used 2026-10-12   | Container replaces the hand-rolled main wrapper; Split(1-3)+Stack(LG) replaced hand-rolled grid/space-y in dashboard + detail   |
| navigation.SimpleNav + ThemeToggle + Footer                                                 | used              | Footer (BrandText single-row) anchored to Base's Footer slot since 2026-10-12                                                   |
| display.Table/TableRow.Href/TableCell                                                       | used              | row-click JS included                                                                                                           |
| display.Badge, PageHeader, EmptyState, ListNote, RelativeTime, Card, DefinitionList, Button | used              | DefinitionList takes `Items []DefinitionItem{Term, Detail, DetailComponent}` — NOT children; PageHeader.Action = Export CSV     |
| display.Scrollback (transcript)                                                             | used 2026-10-12   | replaced hand-rolled bubbles; `Stagger: false` (reading content, not replay); exact counts ride in legend Labels                |
| display.Scrollback — `?q=` match path                                                       | hand-rolled 2026-10-12 | ScrollbackLine has NO id/class hook at v1.21.0, so the T21 search-jump log (`transcriptMatchLog`) copies the block/line classes; visual parity pinned by goldens — re-check on every bump |
| display.CopyButton                                                                          | used 2026-10-12   | detail page: PageHeader.Action (native id) + DefinitionList DetailComponent (from/to); button variant only, Href variant deliberately unused; singleton CSP-safe script runs once per page |
| display.LineChart                                                                           | rejected 2026-10-12 | SVG fixed-pixel (Width/Height ints, 600×300 default) in a responsive card, no per-point tooltips; the one-spike volume series renders flatline+jump vs BarChart's labeled day bar (T23 spike, 2026-10-12); adopt trigger: multi-series quantitative comparison where axes/ticks earn their keep |
| display.DataTable                                                                           | rejected 2026-10-12 | sort-link shell over Table — server-side sort must exist behind the sort URLs, and ListCallsQuery is keyset-paginated date-desc by design; adopt trigger: operators demand column sorting AND a sort-capable query surface lands |
| layout.AppShell                                                                             | rejected 2026-10-12 | sidebar grid earns its keep at ≥6 destinations; 3-item SimpleNav + ThemeToggle + BottomBar already cover shell duties; revisit on real IA growth (operator/admin split) |

## Version-bump watchlist (run on EVERY templ-components bump, in order)

1. **Regenerate the Tailwind corpus:** `node scripts/gen-tc-classes.js` — the
   class list is pinned to module-cache sources; a stale corpus silently
   drops new/renamed library classes (this shipped the 2026-10-12 hamburger/
   dark-shade breakage). Rebuild CSS after: `nix develop -c tailwindcss -i
   internal/web/static/app.css -o internal/web/static/styles.css --minify`.
2. **Re-audit this map against the new pin:** `ls` the new module cache for
   every ABSENT/rejected row (SegmentBar, FilterBar, SegmentedControl,
   utils/format, LineChart, DataTable, AppShell) — flip rows that shipped,
   with a one-line why-not for flips that stay rejected.
3. **Re-check the hand-rolled thin pieces** for library equivalents:
   `transcriptMatchLog` (waits on a ScrollbackLine id/class hook),
   SegmentBar stand-in, anything else carrying a "hand-rolled" status here.
4. **Re-read the open templ-components filings** (#48 BarChart collapse,
   #49 TableRow focus ring, #50 skill drift) — a fix upstream deletes a
   local workaround (`!items-stretch` class args, the focus-visible CSS).
5. **Goldens:** markup changes from the bump are deliberate — refresh with
   `GOEXPERIMENT=jsonv2 go test ./internal/web/ -run TestGoldenPages
   -update`, REVIEW the diff, then commit; the plain suite must go green
   without `-update` afterwards.
| display.PieChart Donut (AI/human mix)                                                       | used 2026-10-12   | Donut+CenterLabel+ShowLegend replaced the hand-rolled flex bar; Slice.Color is a text-* class (fill=currentColor)               |
| display.Grid + StatCard (dashboard stat row)                                                | used              | icons (Phone/Calendar/Microphone/Clock) + Tones added 2026-10-12; StatCard.Value is a string; renders `>N </dd>` — assert so    |
| display.BarChart (vertical) + Sparkline (dashboard volume/trend)                            | used              | MinBarWidth `min-w-1` + Gap `gap-px` for dense 30-day series; Tooltip carries the full date                                     |
| forms.Select/DatePicker + Form(FormGet, FormLayoutInline)                                   | used 2026-10-12   | FormLayoutInline == the old hand-rolled `flex flex-wrap items-end gap-3` row; Form renders the `<form>`, Apply/Reset are Button |
| forms.Input (InputSearch)                                                                   | used 2026-10-12   | replaced the raw search input on /search                                                                                        |
| feedback.Alert + CircularProgress                                                           | used              | Alert: staleness + MOS placeholder; CircularProgress: MOS gauge ((MOS-1)/4*100, aria-valuenow asserted in tests)                |
| icons                                                                                       | used 2026-10-12   | EmptyState.Icon takes `icons.Name`; Button.Icon takes a rendered `icons.Icon(name, class)`; ~100 names in icon_names.go         |
| htmx.PolledRegion                                                                           | used              | dashboard cards 30s, staleness banner 60s                                                                                       |
| display.SegmentBar / utils/format / errorpage / FilterBar / SegmentedControl                | ABSENT in v1.21.0 | feedback.ProgressBar EXISTS (contrary to earlier note) — SegmentBar alone is unreleased upstream; the rest stays unimported     |
