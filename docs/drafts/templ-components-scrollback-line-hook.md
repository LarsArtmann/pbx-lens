> **FILED as templ-components#51 (2026-10-12).** Kept as the historical write-up; upstream state lives in `docs/upstream-filings.md`.

# display.ScrollbackLine: no per-line id/class hook — line anchors and highlights are impossible without re-rolling the component

`Scrollback` carries `BaseProps` on the container, but the line struct has no `ID`/`Class` escape hatch, so a consumer cannot anchor or style a single line. We hand-rolled a copy of the loop in pbx-lens for the search-jump deep link; a per-line hook would retire it.

## Symptom

Our call-detail page deep-links from search hits to a specific transcript turn: `/calls/…?q=…` 302s once to `?q=…&at=N#turn-N`, and the browser scrolls to the matching line, which is also highlighted. With `Scrollback` that URL is unbuildable — lines render no `id`, and no class hook exists to mark the matched ones. `fragment` scrolling is also how we keep the jump working without JavaScript.

## Root cause (source)

The line struct (`display/scrollback.templ:25-30`, v1.21.0) has only content fields:

```go
type ScrollbackLine struct {
	Timestamp string
	Tag       string
	Text      string
	Tone      ScrollbackTone
}
```

The line div (`display/scrollback.templ:80-89`) renders a fixed class list and no per-line attributes — `ID`, `Class`, and `Attrs` exist only on `ScrollbackProps` (lines 67-78), i.e. the container:

```go
for _, line := range props.Lines {
	<div class={ utils.Ternary(props.Stagger, "tc-log-line", "tc-log-line tc-log-line-still") }>
```

Same shape on master as of 2026-10-12 (fetched via the GitHub contents API), so this is not fixed upstream yet.

Consumer copy we carry (`internal/web/detail.templ:197-212`, pbx-lens) — the loop duplicated only to add `id` + highlight classes:

```go
if matchQuery != "" && line.Matched {
	<div id={ "turn-" + strconv.Itoa(line.Index) } class="tc-log-line tc-log-line-still rounded bg-amber-100 dark:bg-amber-900/40">
} else {
	<div class="tc-log-line tc-log-line-still">
}
```

## Design

- [ ] Add `ID` and `Class` (string) fields to `ScrollbackLine`, merged into the line div: `id={ line.ID }` when set, classes via the existing `utils.Class`. That covers anchor navigation, highlight marks, and per-line test hooks without touching `Attrs`/ARIA scope.
- [ ] Alternative considered and rejected: `Attrs []templ.Attribute` per line — more general but heavier for the two-line use case, and the container already shows the convention of explicit `ID`/`Class` fields.
- [ ] Document on the struct: ids are caller-owned (uniqueness is the caller's problem), `Stagger`'s `nth-child` delays are unaffected by added classes.

## To verify

- [ ] Golden delta: line div gains `id`/class only when the fields are set; default rendering byte-identical.
- [ ] pbx-lens fold-in: delete the hand-rolled match log, render `Scrollback` with per-line `ID`/`Class`, keep the redirect + fragment jump golden green.

--
💘 Generated with Crush
