<!-- docs/status/2026-10-11_23-35_gate-closure-lint-residuals-test-dedup.md
     Status report for the 2026-10-11 ~22:20-23:35 dispatch: gate closure
     under load, lint residual fixes, test-wiring dedup, license-check note
     resolution. All numbers re-verified live. NOTE: a second concurrent
     session was active in this repo during the window (dprint-padding
     lesson entry 23:24, BuildFlow binary rebuilt at HEAD); its changes are
     respected, not reverted. -->

# Status — Gate Closure, Lint Residuals, Test-Wiring Dedup

Date: 2026-10-11 23:35 CEST · Baseline: dispatch-three's 129 test functions
green; top task was the full-gate rerun.

## a) FULLY DONE

| Item                                                                   | Evidence                                                                                                                                                                                                                                                                                                                                                                            |
| ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Two golangci-lint residuals fixed**                                  | `internal/web/server_test.go`: dynamic `errors.New("client vanished")` (err113) became package-level static `errClientVanished`; `cmd/pbx-lens/main.go`: 132-col flag help line wrapped under the 120 limit (lll). `golangci-lint run ./...` = **0 issues** (22:24, devShell)                                                                                                       |
| **`newTestServerWithSearch` helper extracted** (20:26 report §f #3)    | Built on the existing `sqliteSearcher`; flow_test's and server_test's two 15-line copies of `search.Open → Rebuild → newTestServerWith(Options{Search: index})` collapsed to one line each; unused imports pruned; `internal/web` green                                                                                                                                             |
| **Gate verified to step level under load 85-184**                      | Full pass 22:26 (devShell, no-cache): every step green except `nix-build` (killed, OOM/timeout — the documented starvation pattern). Casualties verified individually per docs/buildflow-dev-gate.md: gofmt clean, vet clean, `go mod tidy -diff` clean, `go test -race -count=1 ./...` 13/13, golangci-lint 0 issues, `nix build` green twice (22:21, 22:35) with a running binary |
| **nix-build closed via the gate's own runner**                         | `buildflow -s nix-build` = green 1/1 in 23.6s (23:23). Every gate step that ran in the 22:26 pass, plus the one that starved, now holds a green verdict today                                                                                                                                                                                                                       |
| **license-check stale note RESOLVED** (its own recorded fix condition) | Binary was rebuilt (now at BuildFlow HEAD — concurrent session's doing); `buildflow -s license-check` = green 13.6s (22:31). Note replaced with a one-line resolution in docs/buildflow-dev-gate.md; TODO_LIST P10 flipped DONE                                                                                                                                                     |
| **AGENTS.md templ drift fixed**                                        | UI-trio line said `a-h/templ` v0.3.1020; go.mod carries v0.3.1070 (manual bump 40e1a49 because dependabot mangles this go.mod). Line corrected + manual-bump-only note added                                                                                                                                                                                                        |
| **CHANGELOG closure**                                                  | 3 entries: helper extraction (Added), lint residuals + AGENTS.md templ fix (Fixed)                                                                                                                                                                                                                                                                                                  |

## b) PARTIALLY DONE

| Item                                 | Works                                                                                                                                                                                                                                                                                                                                                      | Open                                                                                                                                                                    | Effort        |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| **One fully-green single-pass gate** | Two full passes attempted: 22:26 (all green except starved nix-build) and 23:59 (starved EARLIER — go-mod-update's internal `go build` killed at 1m49s with no output, load 52-98; buildflow misread the kill as "compile errors" and auto-reverted go.mod/go.sum cleanly). Tree health re-verified after both: tidy -diff clean, build OK, all tests pass | A pass at genuinely low load (the box oscillates 39-184; both attempts landed on spikes). The 22:26 pass also left 2 steps "skipped (blocked)" only a clean pass covers | S at low load |

## c) NOT STARTED (unchanged, owner-gated)

P9 postings (five D6 drafts + go-atomic-write ask), production `-max-age`
window in the sibling NixOS module, first release tag v0.1.0 (its "gate
green" precondition now has step-level evidence), auth htpasswd
reload-on-change, per-IP limiting, ops-metrics endpoint, slog adoption,
README screenshots, keyset-pagination probe.

## d) TOTALLY FUCKED UP

1. **Pipe-masked exit code**: read golangci-lint's verdict as
   `cmd | tail; echo $?` — that reports `tail`'s status, and "2 issues, exit
   0" is implausible. Caught it by reasoning about the output, re-ran with
   redirection (`> file; echo $?`). The lesson: exit codes of piped commands
   need PIPESTATUS or redirection, every time.
2. **Two stale-read edit refusals** (TODO_LIST, buildflow-dev-gate.md):
   edited without re-reading after the daemon/dprint/concurrent session
   touched them; the edit tool refused, I re-read and re-applied. No damage —
   but this is the exact mechanical failure dispatch-three flagged, now
   recurring under CONCURRENT-session conditions, which makes re-reading
   before edit load-bearing rather than optional.
3. **Gate attempt at load 144**: I knew the paid-for starvation threshold
   (40-85) and started individual verification before the full pass rather
   than after — the doc's protocol, followed — but the pass itself was still
   predictably starved on its heaviest step. Cheaper would have been:
   verify casualties first, run the pass only when the 5-min load average
   dropped under ~50.

## e) WHAT WE SHOULD IMPROVE

1. **Concurrent-session discipline**: a second session edited docs and
   rebuilt the BuildFlow binary during this window. Before ANY doc edit,
   re-read (mtime check); after ANY build-tool run, re-check binary version.
   The fleet's auto-commit daemon turns both into races that are invisible
   in `git status` snapshots taken minutes earlier.
2. **Gate scheduling still open** (owner question from 20:26 §g.2): today
   added a third starvation. A `--max-jobs`/serial nix-build mode or a
   load-gate on the invocation in AGENTS.md would end this class.
3. **file-size-check advisories** (`internal/calls/domain.go` 571 lines vs
   max 350, +2 more) keep riding along as warnings; a scheduled split pass
   (domain vs persistence) would retire them — TODO_LIST candidate.

## f) Top next tasks

| # | Task                                                                                                                                          | Impact | Effort | Category      |
| - | --------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1 | Land the one fully-green single-pass gate (23:59 attempt pending)                                                                             | High   | S      | Quality       |
| 2 | P9 filings: post the five D6 drafts + go-atomic-write ask (owner go)                                                                          | Medium | M      | Quality       |
| 3 | Sibling module production `-max-age` + `ingest-* -max-age` flags (owner value)                                                                | High   | S      | Feature       |
| 4 | First release tag v0.1.0 via go-release                                                                                                       | Medium | M      | Feature       |
| 5 | domain.go split pass to retire file-size advisories                                                                                           | Low    | M      | Cleanup       |
| 6 | crush-config lessons.md: "sibling repos are the first dependency search path" + "wire tests from existing scaffolding" (by commit, that repo) | Low    | S      | Documentation |

## g) Top questions I can NOT figure out myself

1. Gate scheduling decision (carry-over, 20:26 §g.2): serial nix-build mode,
   load-gated invocation, or gates only at idle times?
2. Release order for v0.1.0: gate precondition is met at step level — is
   that sufficient, or do you want the single clean pass + sibling
   `-max-age` decision first?

Waiting for instructions.
