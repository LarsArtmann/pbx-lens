# Status Report — pareto plan execution: gate hardening, probes, rehearsals (2026-10-18)

**Trigger:** the approved 2026-10-18 SUPERB pareto plan (T1-T24) executed in
one session; every self-executable task landed, every gated task staged or
documented. Machine load stayed 43-200 all session (full-gate receipt
T2 + cut-release dry-run T13 remain policy-gated: 1-min < 15 AND 15-min < 40).

## a) What shipped (T-numbers reference the plan)

- **T3 statix no-op tripwire** (verify-all lane; fake fail-open binary red,
  real statix green) and **T4 CI nix lane** (statix+deadnix+`nix fmt --ci`,
  lock-pinned via the flake's nixpkgs rev; each lane seeded-red in a
  throwaway clone first; GREEN on remote CI run 39079848906).
- **T6 flake triage — CORRECTED IN-SESSION**: the first hypothesis
  (go-appkit testkit's 5s startTimeout under saturation) was WRONG; the
  dev gate REPRODUCED the failure live (`gzip conditional replay status =
  200, want 304`) and exposed the real root cause: the rendered bodies
  carry a 1-second-granularity stamp, so a conditional replay crossing a
  second boundary legitimately answers 200 with a fresh tag. FIX: all
  conditional legs in serve_test.go now run through a `conditionalReplay`
  helper that re-tags until the representation is stable (a 200 with an
  identical body still fails — that would be a real caching bug). The
  misdiagnosed draft was retracted before filing; the test comment tells
  the corrected story. 20x -race loop green post-fix.
- **T14 `ingest-telnyx -dry-run`** parity (report-only fetch, no index,
  `-api-base` stub knob, stub-API test proving zero writes; README +
  FEATURES updated).
- **T15 csp-sweep per-surface console attribution** (consoleSurface tracking;
  re-run: zero violations, 6 surfaces).
- **T16 `verify-all --baseline` wrapper** + **print-probe break-inside
  assert** (live: 3/3 cards `avoid`) + README doctrine row.
- **T17 art-dupl verdict baseline** (checked in; CI diffs the accepted
  verdict with a refresh receipt; seeded clone stands out 34→35 groups) +
  baseline-freshness warning in verify-all.
- **T18 golden-scrubber contract test** (`TestGoldenScrubberCoversFooterStamp`)
  - the scripts/README codegen-paragraph merge (hash-primary wins).
- **T19 AGENTS extraction**: recording-proxy two-hop budget →
  docs/http-stack.md (verified against recording.go before the move);
  AGENTS at 218/219.
- **T20 `-wired-ops` serve fixture + `verify-ops-verdict.js`** (fourth lane
  probe; stale alert + /api/ops stale:true asserted in a real browser;
  --fast lane green with it).
- **T21 FTS ranking corpus** (106 views; bm25 term-frequency order,
  both-terms-beats-heavyweight, digit-shadow corners + FTS/digit dedupe —
  all pinned in `internal/search/ranking_corpus_test.go`).
- **T22 recording-proxy timing-hop asserts** (full chain: metadata/headers
  in Server-Timing ≤ budget, no body metric, stalling upstream → 502 with
  the budget intact).
- **T23 buildflow doctor re-triage**: 29 marks, all in the documented
  disposition classes, no new relevant tool; gate doc receipt updated; the
  statix invocation contract pinned in `.buildflow.yml`.
- **T7/T8/T9 filing drafts staged** (CopyButton empty-nonce unconditioned
  script [cross-refs #63/#24], Card h3 TitleTag level-skip [12/12 override
  evidence], statix exit-0 footgun [upstream: molybdenumsoftware/statix;
  dupe searches clean]) + **T10 resolved as documentation** (cqrs-lint
  v4.17.0 has no inline suppression; policy doc records the gap).
- **T11 hooks wiring**: `.githooks/pre-commit` chains budget→treefmt and
  `core.hooksPath .githooks` is SET in the primary clone — the auto-commit
  daemon now routes through the budget gate (verified live: the gate ran on
  this session's own commits). CI budget job = T12-gated residue.

## b) Verification receipts

- Full Go suite green (14 packages; final run in the session log).
- verify-all --fast green with all FOUR probes (statix tripwire + 3
  contracts + ops verdict); the FULL browser lane ran green earlier in the
  session (T17 receipt).
- Lint 0 issues on every touched package (cmd, internal/web,
  internal/search); formatting via golangci --fix (gci/golines/mnd).
- Remote CI green on the nix-lane commit; final push watched separately.
- AGENTS.md 218/219 (budget check ran on every commit via the new hook).

## c) Explicitly NOT done (gates, unchanged from the plan)

- **T1 owner answer sheet** (§1-§11): owner decisions; staging doc ready.
- **T2 full-gate 70/70 receipt** + **T13 cut-release dry-run**: load policy
  (43-200 all session; need < 15/40).
- **T5 config-verify** + the T11 CI budget job: gated on the T12 lint-policy
  answer (3d93fdf intent, self-review §g1).
- **Actual filings** (T7-T10): drafts verified + staged; filing is an
  owner-go-ahead public act. go-appkit startTimeout draft also staged (T6).
- **T24 re-ingest rehearsal**: needs the prod Master.csv path (owner).

## d) TODO_LIST shape after this session

2 executable TODO rows (full-gate receipt; lint-policy intent), 22 blocked
(owner/external/load), low-impact section absorbed the rest. Done rows
deleted to CHANGELOG per the list's own discipline.

## e) Owner questions this session adds (none blocking)

1. Go-ahead to FILE the four staged drafts (docs/drafts/README.md LIVE rows)?
2. 3d93fdf lint-policy intent (unblocks T5 residue + the CI budget job).
3. Quiet-window calendar slot for T2 (gate receipt) + T13 (release dry-run).

## f) Risks / notes for the next session

- The daemon's push cadence lagged this session (origin sat at 4b9142f
  until manual pushes) — the new hooksPath means the daemon's commits now
  pass the budget gate, but PUSH still needs watching; verify with
  `git log origin/main..main` before assuming remote state.
- The `.githooks` activation is clone-local by design; fresh clones opt in
  with `git config core.hooksPath .githooks` (documented in scripts/README).
- verify-ops-verdict.js is now a 4th contract probe in every --fast run
  (+~5s); if the lane feels heavy, `--only`-style subsetting is the lever,
  not removal.
