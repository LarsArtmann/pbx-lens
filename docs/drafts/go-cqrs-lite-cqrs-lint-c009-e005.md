> [!NOTE]
> **FILED upstream 2026-10-12 (R12): go-cqrs-lite#64 (filed 2026-10-11; C009 confirmed covered). Draft historical.**
>
> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

Two cqrs-lint false positives from a real consumer: C009 flags the bare `must` idiom, and E005 doesn't recognize `system.RegisterCommand` — so every ADR-0123 system-rooted app gets a "no registered handler" per command.

## C009: `len(name) > 4` contradicts the rule's own doc

`cmd/cqrs-lint/pkg/rules/correctness/c009.go:95`:

```go
if (strings.HasPrefix(name, "must") || strings.HasPrefix(name, "Must")) && len(name) > 4 {
```

The file's doc (lines 13-16) says the rule "skips panics inside functions prefixed with must" — but a helper literally named `must` (len 4) fails the length guard and gets flagged. The bare `must` free function is the standard panic-on-error idiom (the `template.Must` shape as a package-local helper). Consumer workaround: renamed the composition-root helper to `mustRegister` purely to satisfy the lint.

`New` has the same shape at line 99 (`len(name) > 3`), excluding a bare `New`.

Suggested fix: drop the length guard (prefix match alone), or document why the bare names are excluded.

## E005: `system.RegisterCommand` is not a recognized registration

`cmd/cqrs-lint/pkg/rules/architecture/rules.go:87` suggests "Register a handler via command.RegisterTyped or dispatcher.Register", and the scanner tracks registrations "via RegisterTyped" (`pkg/analyzer/scanner_consts.go`, `CommandTypesRegistered`). But the system composition root API (ADR-0123: `system.New` + `system.RegisterCommand[T, S](sys, ...)`) is the documented wiring path for system users — pbx-lens registers every command that way in `internal/app`, and E005 still reports "Command type has no registered handler — dispatching it will return ErrNoHandler" for each. The warning is wrong for every consumer of the `system` module.

Suggested fix: teach the registration detector the `system.RegisterCommand` generic call shape (the type parameter names the command), and extend the suggestion text to mention it.

Both observed on master `f2c892b` against pbx-lens (github.com/LarsArtmann/pbx-lens).

💘 Generated with Crush
