<!-- docs/status/2026-10-25_12-55_pareto-execution-gate-truth-and-first-analytics-depth.md
     Execution report for the 2026-10-25 10-46 SUPERB plan (docs/planning/),
     tiers 1% (P1–P3) and the head of 4% (P4–P7, P10), interrupted at P7 by
     owner request. Receipts inline; every claim names its commit. -->

# Pareto execution: gate truth landed, first analytics depth shipped

**Head at interruption:** `ff1f959` (unpushed through `a5f132e` chain — see
§h) · **Load:** calm window opened (15-min ≈ 3–6) after a starving morning
(≈ 100) · **Hooks:** ACTIVE in this clone for the first time since
2026-10-23 · **Plan:** docs/planning/2026-10-25_10-46_SUPERB-gate-truth-
analytics-depth-and-backlog-closeout.md (25 tasks, 118 steps).

## a) Fully done (receipts)

1. **Plan P11-row repair + push** (`57eddbc`): the §3 P11 row's five empty
   cells filled (Impact/Effort/Value/Gates/Evidence); dprint re-aligned the
   whole table; pushed with the daemon's plan commit.
2. **P1 — the gates are real again** (`dab8bf1`): `core.hooksPath` was
   LOCALLY overridden to `.git/hooks` (the machine global says
   `.githooks`), so all four pre-commit checks were dormant since
   ~2026-10-23 while the daemon committed ungated. Re-activated
   (`git config core.hooksPath .githooks`) and proven with THREE seeded
   refusal proofs in a scratch clone, each asserting the gate's MESSAGE
   (not just exit): a 220-line AGENTS (budget refuses), an undefined JS
   symbol in scripts/lib/ui-probe.js (oxlint no-undef refuses), a
   999-report count claim (docs-counts refuses). scripts/README.md's
   "SET since 2026-10-18" claim corrected; the incident + the
   pipeline-exit proof trap recorded in docs/gotchas.md. Every commit since
   runs the chain (including the daemon's).
3. **P2 — concurrent-session norm** (`c713a9d`): docs/concurrency.md (six
   rules the week paid for: claim files, no receipt from a moving tree,
   daemon sweep expectations incl. commit-in-the-same-call and
   never-leave-a-half-edited-hash, flaky-test-is-a-finding, per-clone gate
   arming, proofs assert the message). Linked budget-neutrally from
   AGENTS.md (219/219) and scripts/README.md; `.claims/` gitignored.
   Status: DRAFT pending the owner's nod (§g).
4. **erraudit fix** (`dab8bf1`): four scenario-test rejection sentinels
   declared as `error` (were `*errorfamily.Error`) — the only error-severity
   findings standing between the repo and its first green FULL gate.
5. **P10 — the first green FULL-gate receipt** (run log, `095e62c` docs):
   `scripts/poll-gate.sh` verified START LOAD **15m=6.06** (policy < 40),
   `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode full` → **60
   success / 0 failed / 0 skipped, exit 0**, frozen tree, 2m19s. Three
   runs total: #1 exposed the erraudit findings (exit 69), #2 was green but
   on a tree my own P2 edits moved mid-run (void per the fresh norm), #3 is
   the receipt. The morning's three starved attempts are superseded.
6. **P10 — bench re-measure on the quiet box** (`095e62c`): ingest parse
   **16–26 → 8.7 µs/row**, fresh-stream dispatch **10–12 → 6.1 µs/row**
   (the load-affected numbers were ceilings, as suspected); poll 304 ≈
   88 µs / full-200 ≈ 94 µs (unchanged within noise). docs/http-stack.md
   carries BOTH generations with the load figures; AGENTS points at the
   quiet-box numbers.
7. **P3 — TODO_LIST truth pass** (`095e62c`): the blank line that split
   the Blocked table repaired (rows 87–101 rejoined); versions.json +
   Server-Timing moved to Unblocked (they never had an owner gate); the
   two now-done 🔴 rows deleted (benches, gate receipt); three stale rows
   deleted with evidence in CHANGELOG (probe-residue bundle, prettier
   devShell — the residue is crush-config's, not this repo's — and the
   go-appkit#26 half of the fold-in row, reworded to its live remainder:
   go-etag#2 + quarterly).
8. **P4 — per-segment cost rate** (`2232f25`): Segment gains `PricedSecs`
   (billed time of priced calls only) and `CostPerMinute`; the "Who calls
   what" tables grow a Rate column. Unpriced segments render an em dash,
   never a confident zero. FEATURES flipped PARTIALLY → FULLY. The plan's
   other half (priced-share on the Spend card) verified ALREADY SHIPPED as
   the "N of M calls priced" line — not duplicated.
9. **P5 — no-user-turn week trend** (`1db793d`): DayVolume carries per-day
   Transcripts/NoUserTurn; `QualityTrend` compares last 7 days vs prior 7
   in percentage POINTS (2%→4% is +2 points, not "+100%" — small baselines
   stay honest). Renders only with a prior-week baseline and a nonzero
   prior share; direction spelled out ("down" is good). The volume
   narrative half verified already shipped (calls stat card Change line).
10. **P6 — counterpart history** (`6bd649a`): DetailPage gains
    CounterpartHistory — the outside party (callParties, same grouping as
    the segment tables), the count of representative calls involving that
    number (a CDR+transcript pair = ONE call; proven by test), and the
    newest call's time. "History" definition item drills into `?number=`.
11. **P7 partial — corpus stamp + one golden entry point** (`ff1f959`):
    gen-tc-classes.js writes a `# templ-components@vX` stamp (corpus
    regenerated: v1.24.0, 7637 classes + stamp — styles.css untouched, the
    stamp adds no utilities); TestGoldenOpsPage folded into
    `TestGoldenPages` (own mux, same entry point) so `-run TestGoldenPages`
    can no longer skip the ops golden (the 05-49 §f6 trap).

## b) Partially done

- **P7** (see a11): the corpus FRESHNESS GO TEST (parse the stamp, compare
  against the go.mod pin, floor line-count, census classes) and its seeded
  red proof are NOT yet written. The stamp it will assert on exists.
- **CHANGELOG/FEATURES lag for P4–P6**: the three analytics features are
  committed with tests and goldens, but their CHANGELOG entries (house
  rule: done work lands there) and the FEATURES detail-page row (History
  item) are not yet written. First item of the resume batch.

## c) Not started (plan order)

P8 (domain scenario depth: retire golden, corrected/enriched property
oracle, LookupView property test, -race receipt), P9 (CI hardening:
art-dupl fork tag pin, dependabot sqlite triage, CI job running the four
`--self-test` gates + two docs gates), P11 (poll-gate.sh into AGENTS
Commands + the FULL-gate doc section), P12 (anomaly baseline v1),
P13 (Server-Timing for the export/recording streams), and the whole 20%
tier P14–P25 (saved views, transcript search depth, per-DID page,
ingest-coverage analytics, answer-rate analysis, testing batch 2, docs
hygiene harvest — now FIVE live 2026-10-25 reports including 08-53 —,
versions.json in the sibling, sibling folds, lint residue, analytics docs,
release bodies). Also not run yet this session: `golangci-lint run ./...`
(repo config enables mnd), `-race` over internal/web, and the
`verify-all.sh` browser lane over the changed dashboard/detail surfaces.

## d) Totally fucked up (own failures, this session)

1. **Pipeline-exit masking in the first hook proof**: `git commit … | tail`
   reported "UNEXPECTED-PASS" while the gate HAD refused — the pipe
   swallowed the exit code. Re-run with proper capture; the trap is now a
   gotchas entry (proofs assert the MESSAGE).
2. **Wrong restore order in the scratch clone** (`git restore` before
   `--staged`): proofs 3/4 initially "failed" against a 224-line file the
   index still held. Also asserted the literal "220 lines" instead of a
   pattern — brittle by construction.
3. **AGENTS edit initially +2 lines** → 220 > the 219 budget; caught by my
   own check before committing, compressed to a one-line pointer.
4. **Lost the daemon race THREE times** (dab8bf1 amend, 095e62c squash,
   6bd649a squash, plus ff1f959): I ran `dprint fmt` inside the commit
   chain, which widened the window past the daemon's sweep. All fixed by
   soft-reset squashes while unpushed. Fix going forward: format FIRST,
   then edit → `git add && git commit` immediately.
5. **First "green" gate receipt was from a moving tree** — my own P2 edits
   landed mid-run, violating the norm I was mid-writing. Disclosed, voided,
   re-run frozen (the receipt names run #3).
6. **`url.PathEscape` in the P6 test** vs `QueryEscape` in the href: "+"
   is legal unescaped in paths, so the assertion hunted the wrong string —
   a debug detour to find my own escape-function mismatch.
7. **Debug-slop**: a throwaway test with an unused import (build fail) and
   a grep pattern that missed its log line; both cost a round trip.
8. **P7 interruption state**: for a few minutes `goldenOpsPage` existed as
   a helper no entry point called — the ops golden had NO coverage until
   the wrap-up commit wired it. Disclosed because the daemon could have
   swept exactly that half-edit.
9. **FEATURES edit refused twice** on dprint-mtime staleness (read-state
   invalidated by formatting) — re-viewed each time, but the second one
   cost the f6830bb 1-file commit that needed the squash in (4).

## e) Improve (concrete process changes)

- **Format-first commit discipline**: dprint → re-view → edit →
  add+commit in ONE call; never format inside the commit chain.
- **Pattern assertions over literals** for guard proofs and versioned
  strings.
- **QueryEscape-vs-PathEscape** decision written where the href lives (the
  test now shows both sides).
- **CHANGELOG entry in the SAME commit as the feature** — the b) lag is
  this rule not being followed for P4–P6.
- The plan's verify-first discipline (P4/P5 "already shipped" halves)
  WORKED — keep checking FEATURES before implementing a plan row.

## f) Next 50 (ranked; resume from the top)

1. CHANGELOG entries for P4/P5/P6 + FEATURES detail-page row (b's lag).
2. P7 finish: corpus freshness test (stamp == go.mod pin, ≥3000 lines,
   census classes sm:hidden / dark:hover:bg-gray-800 / bg-blue-600 /
   dark:stroke-gray-800) + seeded red proof in a scratch clone.
3. P7 finish: prove `-run TestGoldenPages` reds on a broken page_ops.golden
   (clone).
4. golangci-lint run ./... (repo .golangci.yml enables mnd).
5. go test -race ./internal/web/ ./internal/calls/.
6. bash scripts/verify-all.sh (dashboard/detail surfaces changed: Rate
   column, trend line, History item).
7. git push (five unpushed commits at interruption).
8. P11.1: swap AGENTS' bare `buildflow` line for `scripts/poll-gate.sh`
   (budget-neutral).
9. P11.2: FULL-gate section of docs/buildflow-dev-gate.md around the
   wrapper; record the receipt (15m=6.06, 60/0/0).
10. P9.1: verify the art-dupl fork tag exists; pin CI to it.
11. P9.2: triage the dependabot sqlite PR (read the run log, classify; do
    not merge — weekly owner confirmation owns that).
12. P9.3: CI job running the four `--self-test` gates.
13. P9.4: + the two docs gates in the same job.
14. P9.5: watch the run; record the receipt.
15. P8.1: golden for the retire path (scenario + ThenGoldenWithPayloads).
16. P8.2: property oracle over corrected/enriched sequences.
17. P8.3: property test for LookupView's legacy-marker fallback.
18. P8.4: -race over internal/calls; record.
19. P12.1: weekday×hour expected bands from HourDay (mean + simple band).
20. P12.2: unit test on a synthetic 4-week matrix.
21. P12.3: today-vs-band verdict (usual/unusual).
22. P12.4: quiet note on the heatmap card (no new chart).
23. P12.5: honest empty state < 2 weeks of history.
24. P12.6: tests + golden + FEATURES row.
25. P13.1: read the recording proxy + export paths + two-hop doctrine.
26. P13.2: design note (which hop, what it must not leak).
27. P13.3: wire servertiming without buffering.
28. P13.4: test — header on first byte, no body buffering.
29. P13.5: document in docs/http-stack.md.
30. P14 saved views (share-link canonicalization + client-side recents).
31. P15 transcript search depth (read FTS5 gotchas first).
32. P16 per-DID page (`/numbers/{number}`, aggregates, links from rows).
33. P17 ingest-coverage analytics (rows/day, gaps, staleness SLO on /ops).
34. P18 answer-rate / short-call analysis (billsec + hangup cause; honest
    "not reported" elsewhere).
35. P19 testing batch 2 (memory↔sqlite equivalence, replay-cost bench,
    parser fuzz seeds).
36. P20 docs hygiene: systemscenario multi-act + AppendBatch rollback
    gotchas (already partly in gotchas via 08-53? verify before writing),
    plan status headers, harvest ALL FIVE live §f lists.
37. P21 versions.json in the go-cqrs-lite checkout (verify the checkout
    exists first).
38. P22 sibling folds (call_direction template, lens.nix wording, webphone
    deep link).
39. P23 lint/gate residue (gated on the lint-policy owner answer).
40. P24 analytics docs (README section, DOMAIN_LANGUAGE denominators).
41. P25 both release bodies rehearsed with cut-release.sh --dry-run.
42. TODO_LIST: delete the Server-Timing row when P13 lands.
43. TODO_LIST: delete versions.json row when P21 lands.
44. Re-run docs gates after the 08-53 harvest (counts must stay true).
45. Anomaly-vs-saved-views ordering is settled (plan: anomaly first) — no
    owner input needed.
46. Consider prettier-fleet residue filing to crush-config (out of repo).
47. Dashboard golden corpus still has no quality-trend coverage (no
    baseline) — a seeded two-week golden would pin the label; decide
    whether the unit test suffices.
48. The 08-53 report's owner questions (§g) — surface with this report's
    §g; three of them overlap mine.
49. Re-verify CI green on the pushed head (gh run watch).
50. Poll the load before any next FULL-gate receipt; the window moves.

## g) Owner questions (three, blocking nothing below them)

1. **Hooks stay active?** The 08-53 report deliberately left
   `core.hooksPath` activation to you; the plan's P1 schedules it and you
   approved execution, so I activated (with proofs). Say the word to
   revert: `git config core.hooksPath .git/hooks`.
2. **docs/concurrency.md** — approve the norm as written (DRAFT), or edit?
   The `.claims/` convention is the only new mechanism; everything else
   codifies what already bit us.
3. **Release shape**: P4–P6 are user-facing analytics; with them Unreleased
   now clearly exceeds a patch. v0.2.1 (stop before analytics) vs v0.3.0
   (split Unreleased, ship the analytics) — P25 prepares both bodies
   either way; your call decides which is rehearsed first.

## h) Git state at interruption

Unpushed (all receipt-grade after daemon-race squashes): `dab8bf1` (P1 +
erraudit), `c713a9d` (P2), `095e62c` (P10+P3), `2232f25` (P4), `1db793d`
(P5), `6bd649a` (P6), `ff1f959` (P7 partial) — plus this report. Tree
clean; every commit ran the re-armed hook chain.
