> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation (the 2026-10-27 erraudit filings staging), not at my request. **LIVE DRAFT — do NOT file yet.**
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# erraudit lint: --enforce-* flags silently report nothing — the AST path only runs for generic_return and the merge drops every other violation type

**TL;DR:** Under `erraudit lint`, `--enforce-go-error-family`, `--enforce-samber-oops`, `--enforce-coded-errors`, and `--enforce-deferred-close` never produce a finding — the AST analysis that carries them only runs when generic-return enforcement is on, and the merge afterwards keeps only `generic_return` violations. The ask: run the AST path whenever any enforce flag is set, keep all enforce-policy violation types in the merged set, and pin it with an integration test.

## Verified repro (binary `61a744c`, 2026-10-27)

```console
$ mkdir probe && cd probe && go mod init probe
$ cat > main.go <<'EOF'
package main

import "fmt"

func boom() error {
	return fmt.Errorf("boom: %w", errNested())
}

func errNested() error { return nil }

func main() { _ = boom() }
EOF
$ erraudit lint ./... --enforce-go-error-family
$ echo $?
0                                          # expect: fmt.Errorf flagged as stdlib constructor
$ erraudit lint ./... --max --format json | grep -c StdlibConstructor
0                                          # even --max (which enables every enforce flag) emits none
```

`--format table` and `--format json` agree: zero findings. The detector
itself works — erraudit's own unit tests in `internal/ast/enforce_stdlib_test.go`
prove `WithEnforceErrorFamily(true)` + `Analyze(file)` emits
`stdlib_constructor` violations at the unit level.
The loss happens in the lint runner.

## Mechanism (source at `61a744ce`)

Two independent drops, both in `cmd/erraudit/cmd/lint_runner.go`:

1. The AST analysis — the only path that runs the enforce policies — is
   gated on generic-return alone:

   ```go
   if cfg.EnforceGenericReturn {
   ```

   With any other enforce flag set but `--enforce-generic-return` off, the
   AST path never executes.

2. Even when it runs (e.g. under `--max`), the merge keeps one type only:

   ```go
   for _, v := range astViolations {
       if v.Type == types.ErrorTypeGenericReturn {
           violations = append(violations, v)
       }
   }
   ```

   Every other AST-policy violation (`stdlib_constructor`,
   `coded_errors`, `deferred_close`, oops patterns) is discarded before
   filtering/output.

`--max` expands to all enforce flags (`appconfig.go` `expandMaxMode`), so
users running `lint --max` reasonably believe constructor/coded-error
enforcement is active. It is not, and nothing says so.

## The cost when it bites (real incident)

pbx-lens runs `erraudit lint ./... --max` as an acceptance gate. Its
census recorded 8 stdlib-constructor sites as "known-good silence" under
the constructor check; source verification tonight showed the silence was
this gap, not a pass. Any consumer gating on `--enforce-*` under `lint`
has been gating on nothing.

## The ask

- Run the AST path when ANY `--enforce-*` flag is set (or fold the enforce
  policies into the default lint path).
- Keep all enforce-policy violation types in the merged set, not just
  `generic_return`.
- One integration test: a module with a bare `fmt.Errorf` must produce a
  finding under `lint --enforce-go-error-family` (exit 2), covering both
  drops.

## Dupe search

Searched LarsArtmann/erraudit issues (all states, 2026-10-27): nothing
covers enforce-flag behavior under `lint` (nearest: #11 legacy_is guard,
#10 sentinel rewrite — different rules).

## Consumer context

pbx-lens `docs/DOMAIN_LANGUAGE.md` § "Typed error carriers" records the
gap and the standing do-not-migrate answer; when this lands, its census
table becomes the triage list and the acceptance gate may go red on the 8
sites (expected; owner call at that point).

💘 Generated with Crush
