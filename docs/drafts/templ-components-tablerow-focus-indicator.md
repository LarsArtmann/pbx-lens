# display.Table: keyboard-focusable clickable rows have no visible focus indicator (WCAG 2.4.7)

Clickable rows are focusable and activatable by keyboard, but nothing on screen shows where focus is. Consumer patch below.

## Symptom

`TableRow.Href` renders a row with `data-tc-row-href`, `role="link"`, `tabindex="0"`, `cursor-pointer`, hover styling — and zero focus styling. Tabbing through a table in Chromium/Firefox gives no visible indicator on the focused row (browsers draw no default outline on `<tr>`), so a keyboard user cannot tell which row Enter will activate.

## Root cause (source)

Row markup (`display/table.templ:280-290`, v1.21.0):

```go
<tr
    if row.Href != "" {
        data-tc-row-href={ row.Href }
        role="link"
        tabindex="0"
    }
    class={
        templ.KV("tc-content-auto", ...),
        templ.KV("bg-gray-50 dark:bg-gray-800/50", props.Striped && i%2 == 1),
        templ.KV("hover:bg-gray-100 dark:hover:bg-gray-800", props.Hover || row.Href != ""),
        templ.KV("cursor-pointer", row.Href != ""),
    }
>
```

Hover and cursor are handled; no `focus-visible:*` class exists anywhere in the row's class list. Activation is fine — `tableRowHrefJS` navigates on Enter/Space (`display/shared.go:217-224`) — which makes the invisible focus state worse, not better: the row is a keyboard button whose focus position is invisible.

Consumer patch we carry (`static/app.css`, pbx-lens):

```css
tr[role="link"]:focus-visible {
  outline: 2px solid var(--color-tc-primary);
  outline-offset: -2px;
}
```

## Design

- [ ] Add focus-visible classes to the clickable-row branch of the class list, matching the library's own focus language (Button uses `focus-visible:ring-2 focus-visible:ring-blue-500`): e.g. `templ.KV("focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-blue-600 dark:focus-visible:outline-blue-400", row.Href != "")`.
- [ ] `tableRowHrefJS`'s htmx `afterSettle` re-attach path (`display/shared.go:226-231`) synthesizes `role=link` rows in swapped content — confirm those get the same class treatment (classes are server-rendered, so swapped rows only match if the server rendered them with the fix — noting here so the checklist covers lazy rows).
- [ ] Document the a11y contract in the `TableRow.Href` field comment (`display/table.templ:99-105`).

## To verify

- [ ] Keyboard Tab through a table with `Href` rows: focused row shows a ring in light and dark mode.
- [ ] `Tab` focus + Enter navigates (already true, unchanged).
- [ ] Table goldens: class-list delta only.

--
💘 Generated with Crush
