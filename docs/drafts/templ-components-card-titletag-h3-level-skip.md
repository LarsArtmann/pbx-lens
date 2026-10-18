> [!NOTE]
> **LIVE DRAFT — do NOT file yet.**
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation (the 2026-10-18 filings queue, pareto plan T8), not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# display.Card: the h3 TitleTag default bakes in a heading-level skip for the dominant page-card pattern

## Verified at v1.23.0 (source-level)

`CardProps.TitleTag` documents its default
(`display/card.templ:51-55`): "Defaults to h3 (HeadingTagH3) when empty.
Set to HeadingTagH2 when the…". `headingTag` resolves the empty value to
the h3 branch (`display/shared_templ.go:262+`).

## The consumer evidence

pbx-lens (v0.2.x, templ-components v1.23.0) renders **12 `display.Card`
call sites** across its three card-bearing pages — dashboard (6),
call detail (3), ops (3). **All 12 override the default with
`TitleTag: display.HeadingTagH2`.** None relies on the h3 default.

The reason is document structure: every pbx-lens page has ONE `<h1>`
(the page title); cards are its SECTIONS, so accessible outline (WCAG
1.3.1 / heading-order linters) wants `<h2>` — h1 followed by card-level
h3 is a level skip. With the h3 default, the outline-correct spelling is
the boilerplate `TitleTag: display.HeadingTagH2` on every single card,
which is what a 12-for-12 override rate means: the default serves a
pattern the library's own consumers don't render.

A skip is not merely stylistic: axe-core's "heading-order" and IBM's
checkpoint 1.3.1 both flag h1→h3 as a violation, so every consumer that
accepts the default ships a finding their a11y linter reports.

## The ask

Presented as a DX/a11y question with three options, owner's pick:

1. **Flip the default to h2 in the next major** — matches the observed
   pattern (page card = section), zero cost for consumers who must
   override today, a breaking change for any consumer that (silently)
   wanted h3.
2. **Keep h3, document the level-skip consequence** on the prop (smallest
   change; the docs currently present the default neutrally).
3. **A section-context helper** (e.g. a heading-level the Card inherits)
   — heavier; only worth it if other components show the same pull.

## Dupe search

No existing issue addresses Card's heading default (searched
card|heading|h3|level over all states, 2026-10-18).

## Consumer context

The 12 call sites: `internal/web/dashboard.templ:112,213,233,305,323,354`,
`internal/web/detail.templ:126,179,312`, `internal/web/ops.templ:50,87,158`
(pbx-lens at HEAD 2026-10-18).
