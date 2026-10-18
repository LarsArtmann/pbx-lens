> [!NOTE]
> **LIVE DRAFT — do NOT file yet.**
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation (the 2026-10-18 filings queue, pareto plan T9), not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# statix: a config parse error prints "config error" and EXITS 0, silently linting nothing (fail-open default)

## Verified repro (statix 0.5.8-unstable-2026-07-17, nixpkgs)

```console
$ printf 'disabled = [20]\n' > statix.toml        # invalid: rule codes, not rule-name strings
$ statix check
config error: unable to parse statix.toml: ...     # (stderr)
$ echo $?
0
$ git status                                       # nothing was linted
```

Any statix.toml that fails to parse (wrong schema shape, typo'd key,
integer where the schema wants strings) produces the same shape: one
stderr line, exit code 0, zero findings. The exit code is what every CI
gate, pre-commit hook, and wrapper script composes on, so a broken config
degrades a lint gate to a no-op that reads green.

## The cost when it bites (real incident)

A repo carried `disabled = [20]` — a TOML-parse error — for five days
while the lint gate (BuildFlow's statix step, `statix check` with cwd =
repo root) reported success on every run. Discovered only when a human
ran `statix check .` manually and read the stderr. The repair commit plus
the audit that followed cost more than the feature the config tweak was
for.

## The ask

Make a config parse error fail closed: exit non-zero (e.g. 2, distinct
from the findings exit 1) while still printing the parse error. Anyone
who wants the old soft behavior can keep it explicitly (a flag or a
config knob), but the default of a LINTER is that a broken lint
configuration is a failure — every mainstream linter (eslint, golangci-lint,
ruff, shellcheck) treats config load failure as fatal.

Secondary nicety: a machine-readable marker on stderr (the current
"config error: " prefix works if it is contractual) so wrappers can build
tripwires — pbx-lens's verify lane now greps stderr for exactly that
prefix as a band-aid, which is brittle if the wording ever changes.

## Dupe search

Searched `config error` across molybdenumsoftware/statix issues (all
states, 2026-10-18): nothing covers exit-code semantics on config parse
failure (nearest hits are walker-error tests and repo chores).

## Consumer context

statix rides this repo's treefmt + BuildFlow lanes and a new CI job; the
incident record is docs/gotchas.md (statix entry) and CHANGELOG
2026-10-17. If the exit-code contract changes, the tripwire and the
gotchas entry will be updated to the new truth.
