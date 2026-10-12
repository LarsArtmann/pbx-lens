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
| display.PieChart Donut (AI/human mix)                                                       | used 2026-10-12   | Donut+CenterLabel+ShowLegend replaced the hand-rolled flex bar; Slice.Color is a text-* class (fill=currentColor)               |
| display.Grid + StatCard (dashboard stat row)                                                | used              | icons (Phone/Calendar/Microphone/Clock) + Tones added 2026-10-12; StatCard.Value is a string; renders `>N </dd>` — assert so    |
| display.BarChart (vertical) + Sparkline (dashboard volume/trend)                            | used              | MinBarWidth `min-w-1` + Gap `gap-px` for dense 30-day series; Tooltip carries the full date                                     |
| forms.Select/DatePicker + Form(FormGet, FormLayoutInline)                                   | used 2026-10-12   | FormLayoutInline == the old hand-rolled `flex flex-wrap items-end gap-3` row; Form renders the `<form>`, Apply/Reset are Button |
| forms.Input (InputSearch)                                                                   | used 2026-10-12   | replaced the raw search input on /search                                                                                        |
| feedback.Alert + CircularProgress                                                           | used              | Alert: staleness + MOS placeholder; CircularProgress: MOS gauge ((MOS-1)/4*100, aria-valuenow asserted in tests)                |
| icons                                                                                       | used 2026-10-12   | EmptyState.Icon takes `icons.Name`; Button.Icon takes a rendered `icons.Icon(name, class)`; ~100 names in icon_names.go         |
| htmx.PolledRegion                                                                           | used              | dashboard cards 30s, staleness banner 60s                                                                                       |
| display.SegmentBar / utils/format / errorpage / FilterBar / SegmentedControl                | ABSENT in v1.21.0 | feedback.ProgressBar EXISTS (contrary to earlier note) — SegmentBar alone is unreleased upstream; the rest stays unimported     |
