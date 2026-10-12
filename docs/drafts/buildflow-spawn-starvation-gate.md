> **FILED as BuildFlow#50 (2026-10-12).** Kept as the historical write-up; upstream state lives in `docs/upstream-filings.md`.

# The findings gate fails the run on steps BuildFlow itself tagged `[transient:tool.timeout]` — spawn starvation should retry or degrade, not gate-fail

BuildFlow's own step outcome already carries a transient classification (`[transient:tool.timeout] … no output at all (hung at or right after spawn)`), but the run still exits error when the only failures carry that tag. Under load, every gate run goes red on environment noise, and the operator pays triage for zero code signal. One automatic retry for no-output spawn kills — or a distinct exit class for all-transient failures — would keep the gate honest under load.

## Symptom

Two consecutive gate runs on pbx-lens failed with steps killed at or right after spawn, each at a machine load the box was also running unrelated GPU/database jobs on:

- Run 1 (15-min load ~54): `go-mod-update` killed after 13.3 s, `lychee` after 2.4 s — both `[transient:tool.timeout] … no output at all (hung at or right after spawn)`.
- Run 2 (15-min load ~47): `test-coverage` killed after 1.1 s plus one more step, same class.
- Run 3 at load 4–9 (the first quiet window): everything green, no code changed between runs.

Across 2026-10-11/12 we logged five such runs; each lost DIFFERENT steps (`go-fix`, `test-race`, `nix-build`, `go-mod-tidy`, `golangci:detect`) — the loser set is random per run, which is the starvation signature, not a code signature. Every casualty verified green when run individually (`go test -race ./...`, `nix build`, `go mod tidy -diff`, `buildflow -s golangci-lint`).

## Root cause (source)

The step executor already produces the right classification — `tools/providers/cmdrun_streaming.go:86` wraps the deadline kill with `errorfamily.Transient`, class `tool.timeout`, and the diagnosis even distinguishes the two hang phases ("no output at all (hung at or right after spawn)" vs "went silent after working output"). What nothing does is act on that class:

- The error registry's `tool.timeout` entry (`internal/cli/error_registry.go:45-50`) advises "Increase the timeout in .buildflow.yml or optimize the tool's input" — for the starvation flavor neither helps: the tool never got scheduled, so a bigger timeout burns the same budget against a saturated scheduler, and there is nothing to optimize.
- No retry exists for the transient class anywhere in the pipeline, and the findings gate (`internal/cli/fail_on_gate_*`) treats a run whose failed steps are all transient-tagged exactly like one with real tool failures: exit error, `step_failure` verdict, operator triage.

## Design

- [ ] On a no-output spawn kill (the `transient:tool.timeout` class), retry the step once after a short backoff before recording a failure — the second attempt on a loaded box either inherits the quiet window or reproduces fast enough to be a real hang.
- [ ] If all failed steps in a run carry a transient class, exit with a distinct code (or a `--fail-on` verdict of `environment`) instead of the generic step failure, and say so in the summary: "N steps lost to spawn starvation (load E[15] = 54); no code findings — rerun or use --step-filters."
- [ ] Surface the 15-min load average in the failure summary — the operator currently has to guess whether load was the cause from context alone.

Related: #45 covers the go-mod-update-specific misreport (a starved compile gate blamed on the dependency bump); this is the general gate-level behavior underneath it.

## Verification

- [ ] A run where every step failure is `[transient:tool.timeout]` exits with the environment class, not error.
- [ ] The retry fires exactly once per step per run and is visible in the step log.
- [ ] Genuine tool failures (real findings, real compile errors) still gate-fail identically to today.

--
💘 Generated with Crush
