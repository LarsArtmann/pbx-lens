> [!NOTE]
> **LIVE DRAFT — do NOT file yet.**
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation (the 2026-10-18 filings queue, pareto plan T7), not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# display.CopyButton: an empty Nonce still emits the singleton `<script>` — un-nonced inline JS that strict CSP silently blocks, with no opt-out

## Verified at v1.23.0 (source-level)

`CopyButton` ends with `@copyButtonScriptComponent(props.Nonce)`
(`display/copy_button.templ:110`), which is
`utils.ScriptComponent(nonce, copyButtonJS(), "copy button script")`
(`display/shared.go:139-143`). `ScriptComponent` renders:

```go
nonceAttr := ""
if nonce != "" {
	nonceAttr = fmt.Sprintf(" nonce=\"%s\"", html.EscapeString(nonce))
}
...
fmt.Fprintf(w, "<script%s>\n%s</script>\n", nonceAttr, script)
```

So with `Nonce: ""` (zero value — every consumer that has not been told
about the nonce contract gets it), the component emits a **bare
`<script>` block containing the whole singleton JS**. Under an enforcing
CSP with `script-src 'nonce-…'` (no `'unsafe-inline'`) the browser blocks
that script: the `tcCopyAttached` delegation listener never installs, and
every copy button on the page goes **silently dead** — clicks do nothing,
no console error surfaces on the button itself beyond the CSP violation
entry.

The library's own tests only pin the WITH-nonce case
(`bdd_new_test.go:52`, "includes singleton script with nonce"); the
no-nonce case is tested only for the execCommand fallback, never for what
the script tag looks like.

## Why it bites in practice

The consumer contract today is "pass the nonce, always" — but the
component does not defend the invariant. One call site that forgets
`Nonce:` compiles, renders, and works in every non-CSP environment, then
breaks invisibly the day CSP flips to enforce. pbx-lens hit the adjacent
class during its report-only CSP sweep: every `CopyButton` call site
(3 in `internal/web/detail.templ`) must thread the nonce by hand, and the
sweep is the only instrument that would catch a miss.

## The ask (either or both)

1. **An opt-out for script emission** — e.g. `DisableScript bool` on
   `CopyButtonProps` (mirroring how the singleton was designed to be
   attach-once): pages that mount the copy JS globally (bundled asset, or
   #63's proposed page-level mount) render zero inline script tags
   regardless of nonce plumbing.
2. **Fail loud, not silent**: when `Nonce == ""`, either skip the script
   (documented) or leave as-is but document the CSP consequence on the
   prop. The dangerous case is not "renders un-nonced" — it is that
   nothing anywhere tells the consumer this couples to CSP mode.

Cross-reference: #63 (page-level script mount — an accepted mount would
subsume the opt-out half), #24 (the omit-empty nonce sweep — same
problem space, different emission shape: those render `nonce=""`, this
renders NO nonce attribute at all).

## Repro

```go
output := utils.Render(t, CopyButton(CopyButtonProps{Text: "x"})) // Nonce unset
// output contains `<script>\nif(!window.tcCopyAttached)...` — no nonce attribute.
```

Serve that page with `Content-Security-Policy: script-src 'nonce-Z'` and
the copy button does nothing; the devtools console shows one CSP
violation per rendered button script.

## Consumer context

pbx-lens v0.2.x uses `display.CopyButton` (templ-components v1.23.0) on
the call-detail page; the CSP sweep lives in `scripts/csp-sweep.js`.
