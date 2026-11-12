# cmdguard — adoption-fit report (pbx-lens)

**Date:** 2026-11-11 · **Refreshed 2026-11-12** (tag re-verified current: nothing stale, stamp-completeness only — plan P09.3) · **Upstream:** `github.com/larsartmann/cmdguard/v5` v5.1.0 (checkout last commit 2026-11-11; == proxy latest for the /v5 module path; v4 line retired at v4.3.1, wraps Cobra) · **Status in pbx-lens:** **NOT USED** (not in the graph — `go.sum` carries zero entries)
**Verdict:** **REJECT** — the dependency policy bans Cobra (`AGENTS.md` stack decisions: "no cobra"), cmdguard wraps Cobra, and pbx-lens' CLI is too small for cmdguard's value to clear the dependency bar. The stdlib flag stack stays.
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

The trinity no other Go CLI framework ships together: struct-tag typed flags (no string lookups), dependency injection with lazy services + lifecycle, and a zero-panic error contract validated at construction — plus graceful shutdown in reverse registration order, `DoctorCommand`/health checks, styled output (fang + lipgloss), duplicate-command/missing-handler/invalid-flag detection at registration instead of 2am, error printed exactly once with correct exit codes, and gradual migration (raw cobra + typed runtime). v5's `NewCLI[AppConfig]` + `Provide(cli.Scope(), …)` + `NewCommand` shape is a real composition story.

## Why pbx-lens declines it

1. **Policy:** "no cobra, no ORM" is a recorded stack decision (AGENTS.md). cmdguard is a Cobra wrapper — adopting it imports the banned dependency with extra steps.
2. **Scale:** pbx-lens is ONE binary with a handful of leaf commands (`serve`, `refresh`, `smoke`, `ingest-cdr`, `ingest-telnyx`, `ingest-agent`, `catalog`, `version`), all stdlib `fs.FlagSet` with two shared validators (`requireDB`/`requireOneArg`, messages derived from `fs.Name()` — `internal/cli`). There is no per-command flag-struct duplication, no DI need (commands take an `app.App`-shaped value directly), and no lifecycle beyond serve's appkit `Run`.
3. **The pain points cmdguard deletes don't exist here:** no subcommand tree deep enough to mis-wire, no service graph to shut down (appkit owns drain), no panic paths (stdlib flag parse + explicit fatal). Construction-time validation would validate a surface too small to get wrong.
4. **What pbx-lens got instead, hand-rolled and paid-for:** exit-code contracts (`fatalCode` → errorfamily `HandleError`, EX_TEMPFAIL for transient — the cmdguard exit-code story achieved via [go-error-family.md](go-error-family.md)), flag validation at startup, and appkit owning signals/lifecycle for the one long-running command.

## Flip trigger (falsifiable)

Adopt (and re-litigate the no-cobra policy in the same act) when ANY of: (a) a SECOND binary joins the module and the two CLIs start duplicating flag/plumbing shapes; (b) `internal/cli` grows per-command flag structs duplicated across ≥3 commands, or commands need DI'd services with their own lifecycles; (c) the fleet's cmdguard-lint rules become valuable here (the cmdguard skill's lint layer). None is close: the last CLI change added a command, not structure.

## Version currency

v5.1.0 == latest tag (2026-11-11; v5 line is current, v4 frozen). Upstream moves fast (commit today). Nothing to pin.

## How to verify

```console
rg -n "cmdguard" go.mod go.sum internal/ cmd/   # zero hits
rg -c "flag.NewFlagSet" cmd/pbx-lens            # the stdlib stack, per command (the command layer lives in cmd/pbx-lens)
rg -n "requireDB|requireOneArg" cmd/pbx-lens | head -4
rg -n "no cobra" AGENTS.md                      # the policy
```
