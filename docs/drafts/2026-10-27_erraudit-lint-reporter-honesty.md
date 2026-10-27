> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation (the 2026-10-27 erraudit filings staging), not at my request. **LIVE DRAFT — do NOT file yet.**
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# erraudit lint: the reporter misreports what happened — tree format contradicts text output, the failure banner carries no count, JSON filesScanned is always 0, and directory promotion noise lands on stderr

**TL;DR:** Four reporter defects make `erraudit lint` hard to script against: `--format tree` shows zero findings where text mode reports one and exits 2; the violations-found banner prints an unfilled template instead of a count; the JSON summary's `filesScanned` is 0 even on runs that found violations; and every directory argument prints a "promoting directory" notice to stderr. The ask: make every format agree on the finding set, put the count in the banner, populate the summary counters, and move the promotion notice off stderr.

All repros use the same one-file probe module (binary `61a744c`, 2026-10-27):

```go
// main.go
package main

import "fmt"

func boom() error {
	return fmt.Errorf("boom: %w", errNested())
}

func errNested() error { return nil }

func main() { _ = boom() }
```

## 1. `--format tree` contradicts text mode

```console
$ erraudit lint ./... --max            # text: 1 finding (generic_return), exit 2
$ erraudit lint ./... --max --format tree | head -4
Error Metrics
=============
Total Errors Found: 0
Error Returns: 0
```

The tree strategy renders metrics from the graph object, but the lint
runner builds a minimal `&types.ErrorGraph{Violations: violations}` with
zeroed metrics (`lint_runner.go`); `output_strategy.go` hands that graph
to `viz.NewTreeVisualizer(graph)`. Same run, two truths — anyone gating or
summarizing on tree output reads "nothing happened" while the process
exits 2.

## 2. The failure banner prints no count

```console
$ erraudit lint ./... --max
...
  [Rejection:violations.found] violations found.
```

`analyzer.go`:

```go
var ErrViolationsFound error = errorfamily.NewRejection("violations.found", "violations found")
```

The count exists at the call site but the banner renders the sentinel's
template, so every failure looks identical. Scripts (and humans) have to
re-count findings from the list above to know whether one or fifty
violations tripped the gate. Put the number in the banner line.

## 3. JSON summary `filesScanned` is always 0 under lint

```console
$ erraudit lint ./... --max --format json | grep filesScanned
    "filesScanned": 0,
```

Same run reports `"findings": [ ... 1 item ... ]`. The summary counters are
never populated on the lint path, so any consumer diffing summaries
("did this run actually scan my tree?") gets a guaranteed false zero —
the exact "invisibly shrunken scan must not look like full coverage"
case the telemetry doc warns about.

## 4. Directory-argument promotion noise on stderr

```console
$ erraudit lint . --type-aware
erraudit: promoting directory "." to package pattern "./..." (--type-aware requires package mode)
```

`pipeline/pipeline.go`:

```go
fmt.Fprintf(os.Stderr,
	"erraudit: promoting directory %q to package pattern %q (--type-aware requires package mode)\n",
	spec.Path, promoted)
```

This is informational (the run proceeds normally) but lands on the same
stream as real diagnostics — noise-to-signal for every wrapper that
treats stderr content as failure evidence. Move it to stdout, or behind
`-v`.

## The ask

Per item: (1) tree metrics must derive from the same merged violation set
as text; (2) banner line carries the count; (3) lint populates the summary
counters it emits; (4) promotion notice goes to stdout or verbose only.
All four are observable contract fixes, no behavior change to detection.

## Dupe search

Searched LarsArtmann/erraudit issues (all states, 2026-10-27): nothing on
reporter output formats, banner rendering, summary counters, or stderr
noise (nearest: #2 nolint namespace — different topic).

## Consumer context

pbx-lens's acceptance gate reads text-mode lint output; findings 2 and 3
were first observed live there on 2026-10-27 (docs/status reports 19-17
§c3 and 20-08 §b3), and finding 1 surfaced while staging this filing.

💘 Generated with Crush
