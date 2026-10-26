# Status — 2026-10-25 17:12 — the claim gate caught its own author; hook closed; CI green

Short follow-up to the 16-58 report: finishing its handoff exposed a
two-hole failure the new gate was built to prevent, closed both within the
hour, and left main green through CI (run `40210321319` on `0a83df4`).

## a) Fully done (receipts)

1. **The meta-failure found and fixed.** My own 16-58 report named the
   caught phantoms in backticks — the gate (correctly) counted four claims
   (TestGolden, TestGoldenOpsPage, TestRecordChangedObservationCorrects,
   TestRecordIdempotentReplay as historical names) — and the commit had
   STILL landed: my shell pipeline (`… | tail -1`) swallowed the gate's
   exit 1, and the pre-commit hook did not run the new gate. Both holes
   closed: the report's phantom names are now unbackticked prose (their
   real successors keep the code spans) with a §d5 owning the incident, and
   `check-doc-test-claims.sh` runs at every commit beside the other docs
   gates (`.githooks/pre-commit`). Gate state: 62 names checked, 0 phantom,
   exit 0.
2. **The whole day's work is pushed as receipts**: `22c9fc9` (claim gate +
   parser property + release bodies, 13 files, daemon piecemeal squashed),
   `80f2bc4` (16-58 report), `0a83df4` (this fix) — HEAD == origin, tree
   clean.
3. **CI green on the final head** (run `40210321319`, all jobs; watched to
   completion). AGENTS 219/219, docs counts 19 live reports, doc links 0
   broken, claim gate 0 phantom.

## b) Partially done

1. **The 16-58 §f list**: items 1 (push) and 15 (hook wiring, pulled
   forward by §d's incident) done; items 2 (browser lane) and 3 (FULL gate
   receipt) still pending — the machine's 15-minute load was 62.66 against
   the <40 policy earlier, and the session order spent the calm budget on
   the incident fix.
2. **P19**: 19.3 done (16-58); 19.2 (replay bench) and 19.4 (-race ingest)
   untouched.

## c) Not started

- Browser lane (`scripts/verify-all.sh` — now three docs gates + four
  probes; the claim gate has unit + hook proof but no lane pass).
- FULL gate receipt (load-policy gated).
- P19.2, P19.4, P20.4 (harvest), P14–P18, P22, P23.
- Release cut and PR #3 disposition — owner-gated (§g).

## d) Totally fucked up

1. **The gate's first day caught its own author, and the author nearly
   got away with it.** Two independent holes let a red-per-my-own-gate
   state reach main: a piped gate (`| tail -1`) whose exit code the
   pipeline discarded, and a hook that didn't know the new gate existed.
   The gate had no enforcement surface for LOCAL commits until tonight —
   between the batch commit (~16:53) and the hook fix (~17:08), every
   local commit could carry phantom claims silently. Fixed same-hour, but
   the pattern is the repo's oldest lesson wearing a new hat: a gate that
   only runs where you remember to run it is a hope.
2. **`| tail -1` on a gate is a lie machine.** Any pipeline stage after a
   failing command replaces its exit code unless pipefail is set. The
   chained `&&` looked green because tail was green. Rule: gates run
   bare, or the whole chain runs under `set -o pipefail`.
3. **The daemon intercepted again** (`3d867dd` carried my report edit;
   benign this time because the tree was otherwise clean, but the seventh
   and eighth interceptions of the day keep the §g3 question live).

## e) What we should improve

1. **A gate lands complete, or it does not land**: script + CI job +
   verify-all + scripts/README + pre-commit hook in ONE commit. Tonight's
   gate shipped with three of five surfaces and the gap was exploited by
   its own author within the hour.
2. **Never pipe gates** — bare invocation, or `pipefail` for the whole
   chain. Add the rule to scripts/README's gate contract paragraph.
3. **Reports are docs too**: the claim gate reads status reports, so
   report prose must follow the same convention (phantom/historical names
   unbackticked, real successors backticked). Cost: one fix cycle; the
   convention is now demonstrated twice.
4. Leave-green remains absolute (held tonight: the rapid.Bytes fix landed
   before the previous report; the gate fix landed before this one).

## f) Next (ranked)

1. ~~Browser lane `bash scripts/verify-all.sh` (three docs gates + four~~ done — verify-all.sh:100 runs the claim gate; lane green 2026-10-26 (cf6e393 pass)
   ~~probes; first lane proof for the claim gate).~~
2. FULL gate receipt at the calm window (poll-gate → buildflow; 15-min
   load was 62.66 at 15:44 — re-check first).
3. P19.2 idempotent-replay bench + receipt row.
4. P19.4 `-race ./internal/ingest` receipt.
5. P20.4 harvest the three §f lists into TODO/ROADMAP with provenance.
6. ~~Annotate the 16-58 report's §f item 15 (hook wiring) with its done-at~~ done — in the 2026-10-26 truth sweep (16-58 f15 annotated with the hook-wiring receipt)
   ~~marker (resolved by this session, §a1).~~
7. P14 saved views.
8. P15 transcript-mode search (gotchas FTS5 section first).
9. P16 per-DID segment pages.
10. P17 ingest-coverage analytics.
11. P18 answer-rate / short-call analysis.
12. Execute the release per §g2's answer (bodies + mechanics are drafted).
13. PR #3 disposition per §g1's answer.
14. Daemon norm per §g3's answer (claim files vs never-reset).
15. Screenshot-probe baseline refresh (new cards).
16. Dependabot github-actions-only grouping.
17. Backup restore drill.
18. FTS tokenizer coverage table test.
19. Print-stylesheet audit for the new cards.
20. Export streaming compression decision.
21. Ops counters via the pinned etag hooks.
22. Retention SLO surface on /api/ops.
23. PII enforcement decision.
24. CSP nonce enforce flip at next host deploy.
25. MOS poller (2027-01 window); Telnyx quarterly re-verify (m27.9).

## g) Questions for the owner (cannot be figured out)

1. **PR #3 (dependabot sqlite)** — `@dependabot rebase`, or close and
   hand-roll (tidy + vendorHash + gates in a normal PR)?
2. **Release shape** — v0.2.1 or v0.3.0? Both bodies are drafted; your
   pick is the only missing input.
3. **The daemon norm** — claim-file protocol (daemon refuses to commit
   under an active claim), or never-reset (sessions amend forward only)?
   Eight interceptions across two sessions now; the question is no longer
   hypothetical.
