> **FILED as templ-components#48 (2026-10-12).** Kept as the historical write-up; upstream state lives in `docs/upstream-filings.md`.

# display.BarChart (vertical): documented `Height` alone still renders 0px bars — `items-end` defeats the definite height

Repro below. Filed from the v1.21.0 pin; same code on master (`display/bar_chart.templ` unchanged there for this block).

## Symptom

A vertical `BarChart` with `Height` set — exactly what the field doc says to do — renders every bar at 0px in a real browser. The consumer fix needs a second, undocumented piece: `Class: "!items-stretch"`.

Measured in headless chromium 152, library markup rendered by the pinned v1.21.0 module, Tailwind CSS active:

| Variant                                                   | bar heights       |
| --------------------------------------------------------- | ----------------- |
| `Orient: BarVertical, Height: "8rem"` (per the field doc) | 0px, 0px, 0px     |
| same + `Class: "!items-stretch"`                          | 46px, 108px, 15px |

Reproducer: render `display.BarChart(display.BarChartProps{Orient: display.BarVertical, Bars: three bars, Height: "8rem"})`, serve, measure `div[style*='%']` offsetHeight. First variant measures all zeros.

## Root cause (source)

The field doc promises the fix works (`display/bar_chart.go:93-96`, v1.21.0):

```go
// Height sets the chart container height (CSS value, e.g. "8rem").
// Essential for vertical charts — percentage bar heights need a
// definite parent height. No effect on horizontal.
Height string
```

But the vertical container emits `items-end` (`display/bar_chart.templ:77`):

```go
class={ utils.Class("flex items-end overflow-x-auto", gapClass, props.Class) }
```

`align-items: flex-end` on the row prevents the column flex-items from stretching into that definite container height, so each column is only as tall as its content; the bar area (`flex-1`) has no free space to grow into, and the bar's percentage height (`barPercentHeight`, `display/bar_chart.templ:142`) resolves against an indefinite chain → computed 0px. The `Height` the doc calls essential is real but not sufficient; `items-end` undoes it.

Neither the field doc, the doc comment on `BarChart` (`display/bar_chart.templ:10`), nor `docs/recipes/bar-chart.md` on master mentions `items-end`, `items-stretch`, or the collapse (grep of the recipe for height/stretch/collapse: zero hits). Goldens pass because they assert markup, not layout.

## Design

- [ ] Vertical branch: make the container stretch (drop `items-end` there — bars-at-bottom behavior is already provided by the inner bar-area's own `items-end`, `display/bar_chart.templ:83`), or switch to `items-stretch` explicitly.
- [ ] Consider a default `Height` for the vertical branch when `Height == ""` (e.g. a rem value) — percentage bars stay 0px against an indefinite height even after the stretch fix.
- [ ] Document the contract in the `Height` field doc and `docs/recipes/bar-chart.md` ("vertical bars are percentage-heights; the chart needs a definite height; Tailwind class strings like `h-40` are invalid here — set a CSS value").

## To verify

- [ ] Rebuild the two-variant repro above: with only `Height` set, bars measure > 0 and scale with value.
- [ ] Horizontal orientation unchanged (separate render branch, `display/bar_chart.templ:92+`).
- [ ] Existing vertical-chart goldens still pass (markup deltas only if classes change).

--
💘 Generated with Crush
