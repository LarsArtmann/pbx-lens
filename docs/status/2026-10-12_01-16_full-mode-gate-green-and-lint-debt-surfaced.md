# Status Report — pbx-lens — 2026-10-12 01:16 — Full-Mode Gate Green, Lint Debt Surfaced

Session scope: buildflow gate closure (bare-shell `--fix --build-mode full` run through
final green 74/74), individual casualty verification, 3 real lint fixes in
`internal/web/recording.go`, and dev-gate doc updates. Concurrent sessions were active
throughout (load 73-160 on 32 cores; another session refactored `telnyx.go` mid-session —
untouched by this session, committed by the daemon at `5a805fd`/`8a0a9e1`).

Quick stats: **8 fully done · 4 partially done · 15+ not started · 0 repo-state fucked up ·
3 self-inflicted missteps (all corrected)** · final gate: **74/74 steps, 0 failed, 100%**.

---

## a) FULLY DONE

1. **First green FULL-mode buildflow gate** (2026-10-12 00:45):
   `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --fix --build-mode full
   --max-time 15m` → 74/74 steps, 0 failed. Verified via `buildflow history`
   (run `20261011-224529-c`, 100% pass rate). All previous green close-outs were dev-mode.
2. **Root-caused the original 4-step failure** (user's bare-shell run, exit 69):
   - `go-generate`, `govalid-generate`, `golangci-lint` → go 1.27.0-vs-1.27.1 toolchain
     skew (bare-shell artifact; `GOTOOLCHAIN=auto` cannot help — buildflow pins
     `GOTOOLCHAIN=local` for tool spawns).
   - `nix-build` → killed by the user-imposed 5m `--budget/--max-time`, NOT OOM (full mode
     is documented ~5-10 min). It had already passed the vendorHash fetch.
3. **3 real lint findings fixed** (masked by the skew in the bare-shell run):
   - `internal/web/recording.go` cyclop 13>12 → extracted `copyRecordingResponse`
     (complexity now 10).
   - `internal/web/recording.go:178` `forcetypeassert` → `, ok` assertion with zero-
     Transport fallback in `newUpstreamClient`.
   - `internal/web/recording_test.go:331` `wsl_v5` → missing blank line.
     Verified: `golangci-lint run ./internal/web/...` → **0 issues**; web race tests pass.
4. **All casualties individually verified green**: `gofmt -l` clean; `go build ./...` OK;
   `go test -race ./...` all 14 packages OK; `go test -race -cover ./...` all 13 packages
   OK (81.2% stmt coverage on the e2e pkg; wire 100%); `nix build` completes (both
   derivations); `license-check` re-verified green standalone (13.6s, 1/1).
5. **Load-starvation flake diagnosed and dismissed with evidence**: two gate runs each
   lost a DIFFERENT test step (`test-race`, then `test-coverage`) under load 80-160.
   Reproduced the exact shuffled order via the seed printed in the step log
   (`go test -race -shuffle=1791758100252644268 ./internal/web/`) → PASS. Not code.
6. **go.sum trim from run 1 verified CI-safe**: `go mod tidy -diff` clean; nix build
   accepted the existing vendorHash (FOD survived the trim, as documented).
7. **`docs/buildflow-dev-gate.md` updated** with 3 new facts: (1) bare-shell skew can
   MASK real findings — devShell rerun before dismissing; (2) `-test.shuffle` seed-replay
   triage for gate test-step failures; (3) full-mode green close-out +
   `--max-time ≥ 15m` requirement. dprint-formatted.
8. **Repaired artifacts landed**: daemon committed lint fixes + flake.nix fmt at
   `084d0c4`; run-1 repairs (go.sum trim) at `3c43433`. Working tree clean except this
   report.

## b) PARTIALLY DONE

1. **"9 tools unavailable (health check failed)"** — appears in EVERY gate run; I never
   ran `--verbose` to enumerate them (only 3 are explained: markdown-lint, gitleaks,
   codespell are on-demand). Unknown whether the other 6 matter.
2. **Buildflow binary freshness** — 0eec15d-dirty vs BuildFlow HEAD 5d34fae (advisory
   WARN every run). The documented fix (`nix build . && nix run .#reinstall` in the
   BuildFlow repo) was NOT executed this session.
3. **file-size-check residuals** — 5 files over the 350-line max
   (telnyx_test.go 749, dashboard_templ.go 652, detail_templ.go 497, calls_templ.go 390,
   recording_test.go 371). Observed and reported; no splits done. Note: 3 of the 5 are
   GENERATED `*_templ.go` files; the other session already fixed `telnyx.go` (380 → off
   the list) via the `dto.go` extraction.
4. **TODO_LIST harvest** — section (f) below is report-only; not yet routed into
   `TODO_LIST.md`/`ROADMAP.md` (docs-health HARVEST), pending owner go-ahead.

## c) NOT STARTED (carried, unchanged, from before this session)

1. encoding/json v1→v2 swap (13 go-auto-upgrade findings; blocked on T4 stack decision).
2. Rebuild/reinstall buildflow binary (see b2).
3. Quarterly Telnyx live-API re-verification cluster — including the enrichment
   endpoints (`conversations-insights`, `detail_records`) that have NEVER been
   live-verified (no key at implementation time).
4. PII policy: `E164` + `ParticipantHash` exist but nothing enforces them on stored rows.
5. E003 owner question (flat `internal/calls` domain) — still open, decision documented
   as HOLD.
6. Upstream D6 filings status: go-cqrs-lite#62/#63/#64, go-nix-helpers#7,
   treefmt-nix#545 — no upstream-state check this session.
7. art-dupl (20+) / branching-flow (~69 infos) dedicated dedup/branching passes.
8. statix findings (15) on `deploy/nixos/module-draft.nix` — the superseded design
   record; delete or fix.
9. ~~`nix-flake-check`: `update-vendor-hash` app lacks `meta.description`~~ done at the 02:34 session (flake `meta.description` landed; `nix flake check` warning count 0)..
10. nix-store CVE advisories (vulnix, 22 findings: binutils 2.46/2.47, bison, coreutils,
    gcc 10.4) — nixpkgs bump territory, advisory-class.
11. Inbound AI calls / DID interception path (direction is caller-declared today).
12. T9 agent transcripts as a second aggregate → module-boundary revisit trigger.

## d) TOTALLY FUCKED UP

Nothing in the repo is fucked: gate green, tests green, tree clean. Three
self-inflicted missteps this session, all corrected, recorded for honesty:

1. **Initial misattribution (the dangerous one)**: I first read the run-1
   `golangci-lint` failure as pure toolchain skew. It was the skew AND it masked
   3 real code findings. Only the AGENTS.md "verify casualties individually" doctrine
   saved it. First instincts about "known failures" are exactly where real work hides.
2. **Doomed config executed without upfront flagging**: ran the user's bare-shell 5m-
   budget full-mode command knowing the docs predict both the skew failure and the
   budget kill. Running it was correct (explicit command); not stating the predicted
   outcome first cost ~5 blind minutes.
3. **Stale in-context AGENTS.md**: I edited toward a doc structure that another session
   had already restructured (gate policy moved to `docs/buildflow-dev-gate.md`);
   first edit attempt targeted vanished sections. Re-read before writing fixed it —
   but I should have re-read the file before the FIRST edit, not after `rg` returned
   nothing.

## e) WHAT WE SHOULD IMPROVE

1. **Never accept "environment artifact" until the individual rerun is green** — make
   the doctrine's step (individual verification) a reflex BEFORE writing the narrative,
   not after. Today the narrative was written first and quietly corrected later.
2. **Check the working tree for concurrent-session edits before ANY build/gate**, not
   just before git operations (caught the telnyx/flake.nix edits by luck before the
   first `nix build`, would have misattributed any breakage).
3. **Run `buildflow doctor` before the first gate** when prior context mentions binary
   skew — it prints the freshness delta and available tools in one shot.
4. **Ignore stale LSP diagnostics in tool output** — the recording.go warnings kept
   reappearing post-fix; the authoritative check is the fresh `golangci-lint run`
   (0 issues). Consider an LSP restart when diagnostics contradict a verified-clean run.
5. **Gate hygiene**: full mode needs `--max-time ≥ 15m` (5m kills nix-build); always
   `BUILDFLOW_NO_RESULT_CACHE=1` when triaging; reproduce shuffled test failures with
   the printed seed before diagnosing.

## f) Up to 50 things we should get done next (brainstorm, impact-ordered-ish)

Gate & environment:

1. Enumerate the "9 tools unavailable" via `buildflow --verbose` and triage each.
2. Rebuild/reinstall buildflow from HEAD; delete the freshness caveat in the docs.
3. Make the recording-proxy timeout tests load-robust (inject clock/budget) so
   `-test.shuffle` under load can't flake them.
4. Exclude GENERATED `*_templ.go` from file-size-check, or raise their cap (policy).
5. Split `telnyx_test.go` (749) along ingest/enrichment seams.
6. Split `recording_test.go` (371) — proxy vs local-serving files.
7. Install or explicitly skip the missing `interrogate` tool.
8. Investigate the transient result-cache `SQLITE_BUSY` checkpoint warning (concurrent
   gates share state?) — possible upstream filing.
9. Decide GITHUB_TOKEN vs lychee-exclude for private-repo link checks (documented
   undecided fleet policy).
10. ~~Add `meta.description` to the `update-vendor-hash` app.~~ done at the 02:34 session.
11. Bump nixpkgs for the vulnix CVE set (binutils/bison/coreutils/gcc) or document
    acceptance.
12. ~~Delete `deploy/nixos/module-draft.nix` (superseded design record) to zero the
    statix findings, or annotate it as a record.~~ resolved: annotated as the SUPERSEDED
    historical design record (header note, LANDED 2026-10-11 in the sibling).

Code quality:
13. Dedicated art-dupl pass (20+ findings; commands.go mixins, Observation/CallView
13-field overlap — likely intentional, needs judgment).
14. branching-flow review (69 infos) — fold genuinely-complex functions only.
15. encoding/json v2 swap behind T4 (13 findings).
16. Audit `.golangci.yml` `run.go` floor-tracking automation (currently manual).
17. Cyclop headroom check on `proxyTelnyxRecording` (now 10; fine, but note it).

Docs:
18. ~~HARVEST this report's (f) into TODO_LIST.md/ROADMAP.md.~~ done — third-pass TODO
closure (02:34) + the 2026-10-12 docs-health AUDIT pass (TODO_LIST/ROADMAP rebuilt).
19. Record the `-test.shuffle` seed-replay trick in the session-status pattern for
future gate triage (done in buildflow-dev-gate.md; cross-link from TODO_LIST).
20. Update the CI lane docs if golangci-lint pin policy changes.

Upstream (D6 cluster):
21. ~~Check status of go-cqrs-lite#62 (sqlite time binding) — close in-repo note if fixed.~~
done at the 02:34 session: all five filings re-verified OPEN (docs/upstream-filings.md).
22. ~~Check go-cqrs-lite#63 (memory engine field-name filters).~~ done at the 02:34 session (OPEN).
23. ~~Check go-cqrs-lite#64 (cqrs-lint C009/E005).~~ done at the 02:34 session (OPEN; covers the C009 bare-`must` gap).
24. ~~Check go-nix-helpers#7 (templ formatter go wrap).~~ done at the 02:34 session (OPEN).
25. ~~Follow treefmt-nix#545 corroborating comment.~~ done at the 02:34 session (OPEN).
26. File the `-test.shuffle` + load observation upstream IF it reproduces outside this
box (low confidence, needs evidence first).

Domain/product (carried):
27. Quarterly Telnyx live re-verify incl. FIRST live test of enrichment endpoints.
28. PII enforcement policy (E164/ParticipantHash) — design + enforce.
29. E003 second-aggregate decision (agent transcripts T9).
30. Inbound AI calls / DID interception → direction no longer caller-declared.
31. Webphone read-only deep link (planned integration).
32. nix-international-telephony: exercise the landed NixOS module against a real
backup/purge timer run.
33. Search UX: revisit FTS5 number-tokenization hint (maybe prefix-index the E.164).
34. Dashboard: replace hand-rolled flex bar when templ-components publishes
SegmentBar.
35. Cost display: verify MicrosFromCost rounding against real Telnyx detail records.
36. Retention: exercise `purge` + tombstone convergence on a long-lived index.
37. Backup: scheduled VACUUM INTO verification (integrity check in monitor path?).

Process:
38. Add "check git status for foreign edits" to the pre-gate checklist (AGENTS.md).
39. Pre-commit the status-report convention: session reports land as `.md` here (this
report follows the explicit `.md` instruction; skill default is HTML).
40. Consider a `scripts/gate.sh` wrapper encoding the devShell + NO_RESULT_CACHE +
15m invocations (one blessed entry point).
41. Track golangci-lint CI pin (v2.14, go1.27.1-built) against future go bumps.
42. Dependabot PRs from the CI lane: schedule a sweep.

Nits noticed in passing:
43. `internal/web/recording_test.go` blank-line nit fixed — scan for other wsl_v5
stragglers the gate's module fan-out may not have covered.
44. The daemon's heuristic commit messages (`chore: auto-commit N changed file(s)`)
make history archaeology hard — consider richer daemon messages (global config).
45. `flake.nix` was touched by nix-fmt during repairs (committed in `084d0c4`) — skim
that diff once for unintended semantic drift.
46. Load-shedding: gate runs during multi-session load average >80 reliably lose a
random step — consider documenting a "wait for <50" norm (was followed today).
47. `interrogate` WARN (run 1 only) — if it's a buildflow-provided tool, its absence
should be in the unavailable-tools list, not a bare WARN.
48. Result-cache hit rate 91% with NO_RESULT_CACHE bypasses — verify cache isn't
masking stale steps in normal runs.
49. `docs/upstream-filings.md` — add the SQLITE_BUSY observation as a candidate filing
pending repro.
50. Celebrate: gate 96% 30-day pass rate, 1m20s avg duration, and today the first
100% full-mode pass — the fleet pattern works.

## g) Questions I cannot figure out myself (3)

1. **Buildflow binary rebuild**: rebuilding from BuildFlow HEAD replaces the binary used
   by ALL concurrent sessions on this box (3+ crush instances active today). Rebuild
   now, or wait for a quiet window?
2. **file-size-check vs generated code**: `calls_templ.go` (390), `dashboard_templ.go`
   (652), `detail_templ.go` (497) are templ-GENERATED — splitting them means splitting
   the .templ sources. Should the gate exclude generated files, or do you want the
   templates themselves decomposed (real refactor, real risk)?
3. **Timing-test policy**: should the 1 ms-budget recording tests be made
   load-robust (injectable clock, larger budgets with artificial stall — semantics
   change, ~1h work), or is "reproduce with the printed seed" an acceptable standing
   triage when the box is loaded?

---

_Point-in-time snapshot 2026-10-12 01:16. Evidence: gate history run
`20261011-224529-c` (74/74), `golangci-lint run ./internal/web/...` = 0 issues,
`go test -race -cover ./...` 13/13 ok, `nix build` completed, `license-check` 1/1._
