# Status & Self-Review — art-dupl Deduplication Pass (2026-11-05 21:13)

**Scope.** This report covers ONE session: the `art-dupl -t 7` deduplication pass
(prompt: "deduplicate! fully!") and its verification. What was VERIFIED, with
commands: dedup gate `bash scripts/check-dedup-gate.sh` (pin 93432fd enforced,
re-run fresh at 21:13 → **0 clone groups**); `go build ./...` + `go test ./...`
green (run twice, plus buildflow lanes); art-dupl rescans at `-t 7`, `-t 4
--type-aware --show-suppressed`, `-t 10` (2 remaining groups = the two
documented accepted signature pairs); buildflow gate via
`scripts/poll-gate.sh` twice (run 2: every lane I touched green). What was NOT
verified: the browser probe lane (`scripts/verify-all.sh`,
`screenshot-probe.js`) — Go-level golden/ETag tests cover the touched rendering
paths, but no pixel/browser probe ran; CI on the pushed commits (nothing
pushed); the three daemon commits' pre-commit hook executions (assumed from
clean tree + green local gate, not individually inspected). Section (f) items
marked **[gate-out]** are observations from this session's buildflow output —
noticed, not researched.

## What the session did

Three real duplications extracted (found by body-by-body judgment, not by any
single threshold setting):

1. **`eachTypedFilter`** (`internal/web/calls.templ:105`) — the typed-filter
   wire walk (direction/ai/number/from/to) was copy-pasted **3×**
   (`recentViewFiltersJSON`, `listPageHref`, `filtersQueryString`). One
   vocabulary now; `url.Values.Set` consumes the visitor directly.
2. **`hasFilters` composes `hasTypedFilters || walk cursors`**
   (`internal/web/calls.templ:291`) — the 5-term disjunction was spelled twice
   across files; the subset relation (already documented in the `staleness`
   comment) is now structural.
3. **`seedFoldedPair`** (`internal/app/reset_rebuild_test.go:91`) — a 42-line
   seed-and-prove block duplicated across both reset proofs
   (`host_reset_test.go` / `reset_rebuild_test.go`), found via a `-t 10` sweep
   of the suppressed bucket (the 2026-10-24 `openPollHarness` precedent class).

All other `-t 7` groups re-verified body-by-body as deliberate signature-shape
coincidence (the documented 2026-10-24 remainder list held up:
`mosBars`/`volumeBars` zero-policies, the ops trio's three switches,
`noTranscriptTitle`/`Description`, param lists). The utils/format dependency
check from the baseline doc's supersession lesson came back clean — the ops
helpers already ride `format.CompactDuration`. History recorded in
`docs/art-dupl-baseline.md` (2026-11-05 entry).

Commits (daemon): `e4c2535` (calls.templ), `f0f52d7` (test files + baseline
doc), `57b1052` (golines fix). Tree clean at report time.

## Brutal self-review

1. **What did you forget?**
   - I did not re-run `check-dedup-gate.sh` after the LAST code change
     (seedFoldedPair + golines fix) before declaring done — closed at 21:13,
     green, but the claim in my final summary leaned on the earlier run.
   - I never opened `TODO_LIST.md` (project-discovery checklist) — low cost
     for this gate-scoped task, but skipped, not decided.
   - The interrupted first gate run: I restarted it without checking 1-minute
     load myself (the poll script only gates on 15-min load; 1m was 39.77/40).
2. **What is stupid that we do anyway?** **[gate-out]**
   - `checks.pre-commit` false-positives on templ alignment in a file nothing
     touched, every gate run, documented as "never commit its drift" — a
     permanently red lane we've normalized.
   - `nix-hash-fix` has failed 238/293 times (81%) and still runs every gate.
   - The buildflow binary is stale (e81c118 vs HEAD 42d5ebd) with an advisory
     warning every run.
3. **What could you have done better?**
   - **I shipped a 122-char line into the full gate and paid a whole gate run
     (~5 min) to learn about golines max-len 120.** I ran `go build`/`go test`
     after editing but no lint lane (`buildflow -s golangci-lint`) on the
     changed files. The run-1 gate failure was otherwise purely environmental
     (test-race killed at 53s below its own p95 at load 39.77/40) — my format
     finding was the only avoidable part of that wasted round.
   - I dismissed the 17 `templruntime.NewAttributeExpression` LSP errors as
     "stale LSP" with indirect evidence (untouched files also red, toolchain
     green) but never diagnosed the actual LSP/codegen version mismatch —
     noticed, not closed.
4. **What could you still improve?**
   - **Mini split brain I left in place:** `hasTypedFilters` tests
     `filters.AI != nil` while `eachTypedFilter` tests `aiValue(AI) != ""` —
     equivalent today (verified: nil→"", true→"yes", false→"no") but two
     derivations of "is the AI filter active" that can diverge if `aiValue`
     ever grows a state. Unifying (e.g. predicate derives from the same
     source) would close it.
   - The wire-key vocabulary is now 1× on the WRITE side but the READ side
     (`server.go` `query.Get("direction")`… parser) and the form field names
     remain independent spellings — inherent to the HTTP round-trip, worth a
     conscious accept or a shared key-constant set, not silence.
5. **Did you lie to you?** No false claims found on re-check; one claim was
   under-fresh (dedup gate, see 1a) and is now command-backed. The "full test
   suite green" claim is true for `go test ./...` + gate lanes; browser lane
   was not run and I did not imply otherwise — but I also did not SAY so
   until this report.
6. **How can we be less stupid?** Cheap pre-gate lint lane on changed files
   before every full gate run; decide the fate of the permanently-red
   pre-commit check instead of re-reading its documentation each session.
7. **Ghost systems?** None created; none encountered in scope.
8. **Scope creep?** No — I resisted restructuring the signature-shape
   remainder (renaming helpers to game the scanner would be worse code).
9. **Removed something useful?** No; net −54/+27 lines in calls.templ with
   behavior pinned by existing tests.
10. **Split brains?** One small one left (see 4, AI predicate duality);
    one reduced 3-way→2-way (filter keys).
11. **Tests?** Existing web/app tests (incl. golden pages, ETag determinism
    via sorted-key JSON) passed unchanged — the refactors were deliberately
    behavior-identical, so no new tests were added; the honest gap is that
    nothing NEW pins `eachTypedFilter` directly (it is pinned transitively
    through its three consumers).
12. **Re-verify inherited claims?** Yes — the 2026-10-24 "deliberate
    remainder" list was re-verified body-by-body against source, not trusted
    from the doc; that re-verification is precisely what surfaced the 3× walk
    the doc's own audit had missed.

## a) FULLY DONE

- `eachTypedFilter` extraction (3× walk → 1×) in calls.templ — tests green.
- `hasFilters` composition over `hasTypedFilters` — behavior-identical, tests green.
- `seedFoldedPair` extraction (42-line test clone) — both reset proofs green.
- Suppressed-bucket audit: `-t 4 --type-aware --show-suppressed` (only test
  scaffolding multi-line spans) and `-t 10` sweep (only the 2 accepted pairs).
- Dedup gate: fresh run 0 groups, pin 93432fd enforced.
- buildflow gate (run 2): golangci-lint, test-compile, test-coverage,
  test-race, erraudit, workspace-build-verify all green on touched lanes.
- `docs/art-dupl-baseline.md` history entry for the pass.
- All work committed (daemon): e4c2535, f0f52d7, 57b1052. Tree clean.

## b) PARTIALLY DONE

- Gate receipt: green on every lane I touched, but the overall buildflow run
  still exits 69 on `nix-build` → `checks.pre-commit` (pre-existing,
  untouched-file templ-alignment FP; identical deterministic diff both runs).
- "Fully deduplicated": zero harmful clones, but the `-t 7` no-`--min-lines`
  report still shows the 1-line signature class BY DESIGN (gate calibration
  drops it). If the operator wants those report lines gone too, that is a
  different (and, I'd argue, wrong) trade — see question 3.

## c) NOT STARTED (in-scope leftovers noticed this session)

- Unify the AI-filter predicate duality (hasTypedFilters vs aiValue).
- Pin `eachTypedFilter` with direct unit tests (currently only transitively).
- Diagnose/document the templ LSP `NewAttributeExpression` false errors.
- docs/gotchas.md note on the poll-gate 1m-vs-15m load window near-miss.

## d) TOTALLY FUCKED UP

- Nothing in the deliverable. One avoidable process miss (golines line-length
  found by the expensive gate instead of a cheap lane) and one under-fresh
  verification claim (both detailed above, both recovered). No data loss, no
  reverts, no behavior changes.

## e) WHAT WE SHOULD IMPROVE

1. Pre-gate habit: run `buildflow -s golangci-lint` on changed packages
   before every full poll-gate run (this session paid ~5 min for one 122-char
   line).
2. Decide the `checks.pre-commit` templ FP's fate (drop templ from the
   sandbox check / keep documented / chase upstream) — a permanently red lane
   erodes the gate's signal.
3. Close the AI-predicate duality before it diverges.
4. The `-t 7`-style ad-hoc scans confuse "actionable" with "all lines" —
   consider documenting the one-liner that maps a user scan onto gate
   settings (the baseline doc almost says this already).
5. Harvest this report's (f) into TODO_LIST (report-then-wait honored; not
   done unilaterally).

## f) NEXT (up to 50; tiered by impact, [gate-out] = observed in session gate output, not researched)

**Decisions / gates**

1. Resolve `checks.pre-commit` templ-alignment FP permanently [gate-out].
2. Raise `test-race` step timeout (healthy p95 > default budget; tool itself
   prescribes 2m11s) or accept load-dependent flakes [gate-out].
3. `cue-vet`: add `permissions:` to `.github/workflows/ci.yml` (fleet policy
   error) [gate-out].
4. docs-gates red ×4 (annotate/audit-all/cadence/freshness) — run
   `docs-gates` for the full list [gate-out].
5. Harvest this report's (f) into TODO_LIST.md / ROADMAP.md.

**Small code/doc fixes**
6. Unify AI-filter predicate (hasTypedFilters ↔ aiValue).
7. Direct unit tests for `eachTypedFilter`.
8. AGENTS.md is 223/220 lines — trim 3 [gate-out].
9. README.md stale >240h — refresh pass [gate-out].
10. docs/gotchas.md: poll-gate 1m vs 15m load-window note.
11. Document/diagnose templ LSP `NewAttributeExpression` false positives.
12. `nix-checker`: extract inlined hash/vendorHash to dedicated files
[gate-out].
13. `go-version-auto-configure`: go.mod patch pin (1.27.2) vs major.minor
floor — reconcile with the AGENTS.md go-appkit-floor decision [gate-out].

**Backlog surfaced by gate output (bigger)**
14. `go-auto-upgrade`: 39 files still on encoding/json v1 with the documented
foreign-type caveat — triage which boundaries are actually swappable
[gate-out].
15. `file-size-check`: 21 oversized files (dashboard.go 551, server.go 771,
search.go 570, telnyx_test.go 839 …) — split plan [gate-out].
16. `vulnix`: 22 advisories (binutils/bison/brotli/gcc …) — mostly build-time
closure; assess runtime relevance [gate-out].
17. `flake-meta-checker`: add homepage/mainProgram/maintainers/platforms
[gate-out].
18. dprint remote plugin fetches → local wasm pinning [gate-out].
19. buildflow cache DB VACUUM (47% free pages) [gate-out].
20. Rebuild/reinstall stale buildflow binary (e81c118 → 42d5ebd) [gate-out].
21. Investigate or exclude `nix-hash-fix` (81% historical failure) [gate-out].
22. Formatter-fingerprint drift → run `buildflow format` once to re-verify
cached verdicts [gate-out].

**Nice-to-have / hygiene**
23. Shared key-constant set for filter wire keys (write/read/form) or a
conscious accept note in DOMAIN_LANGUAGE.
24. `seedFoldedPair` secondNativeID param: distinct ids are cosmetic (separate
memory engines) — collapse or document.
25. Browser lane run over the changed list/pager fragments for belt-and-braces.
26. Consider a tiny doc paragraph in art-dupl-baseline.md mapping ad-hoc scan
thresholds to gate settings for future operators.

(26 grounded items — padding to 50 with un-researched guesses would violate
the "no unrelated research" instruction.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **The permanently-red pre-commit check:** do you want it fixed by REMOVING
   the templ formatter from the sandbox check (accepting CI as the only templ
   lane), kept as a documented false positive, or escalated upstream (templ
   fmt determinism without module resolution)? All three are defensible; the
   trade is yours.
2. **test-race budget:** bake `--step-timeout test-race≈2m11s` into the
   documented gate entry (every future session inherits headroom), or keep
   the canonical command untouched and treat kills as load-signal? This
   changes the AGENTS.md contract line, so it's an operator call.
3. **"Fully":** the 1-line signature groups in your `-t 7` report remain BY
   CALIBRATION (distinct business rules, extraction would be a net loss). Is
   "fully deduplicated = zero harmful clones, remainder documented" the
   acceptance you want — or do you want the raw `-t 7` line count down too
   (which I'd push back on)?

— Reported 2026-11-05 21:13; awaiting instructions.
