# Status — 2026-10-25 16:58 — doc-claims gate (8 phantoms caught day one), parser fuzz property, release bodies drafted

Follow-up session to the 15-39 report, executing its §f list under the
standing "keep going" mandate. Three §f items closed (the doc-claims gate,
the plan header, the release bodies), one §f item partially closed (P19.3),
two blocked by load, and the batch landed as ONE clean push (`22c9fc9`).

## a) Fully done (receipts)

1. **CI verified green on the 15-39 report push** (`65355b4`, run
   40203906668). The daemon's intermediate `83647bd` had been red on Lint —
   it carried the P13 test files without the P13 implementation — and my
   completed commit turned it green. No action needed; the piecemeal-daemon
   pattern just cost CI one red run.
2. **The doc-claims gate exists and earned its keep on run one** (plan item,
   15-39 §f4): `scripts/check-doc-test-claims.sh` extracts every backticked
   `Test…` name from the living docs and asserts it is a real Go test in
   the tree. First live run: **8 phantoms** — FEATURES rows citing
   TestRecordIdempotentReplay/TestRecordChangedObservationCorrects
   (real coverage: `TestScenarioRecordPaths`, now cited), the 18-30 audit's
   pre-adoption name for the fold-parity oracle (annotated with the landed
   `TestPropertyProjectedRowEqualsPureFold`), three reports' historical
   TestGoldenOpsPage mentions (annotated/unbackticked with the
   `TestGoldenPages` successor), and the 10-46 plan's aspirational
   TestGolden entry point. All fixed same-session under
   annotate-don't-rewrite: history keeps its story and names its real
   successor. Self-test (tree-green precondition, seeded phantom
   extraction, Go-only matching, fence skipping) plus a LIVE seeded-red
   proof on the real tree (phantom → exit 1 naming file and name → revert →
   green). Wired into CI's Guard Self-Tests job (`--self-test` AND a live
   lane), `verify-all.sh` (now three docs-truth gates before any browser
   boots), and scripts/README.md with its own row.
3. **P25 both release bodies drafted**:
   `docs/drafts/2026-10-25_release-bodies-v0.2.1-vs-v0.3.0.md` — body A
   (v0.2.1 patch: the six lint fixes, styles regen, pill tradeoff,
   phantom-doc correction, ALL test/gate integrity, the FULL receipt) with
   its CHANGELOG split mechanics; body B (v0.3.0 minor: everything since
   v0.2.0 — analytics surface, catalog, platform wave) with the simple
   rename mechanics; plus the cut-release.sh execution runbook for either.
   The owner's §g2 answer is now a five-minute choice, exactly as planned.
4. **P19.3 the parser fuzz property**:
   `TestParsersSurviveArbitraryInput` drives ARBITRARY bytes through all
   three parsers (cdr, webhook, agent) and asserts no panic, no error
   (parsers skip — one bad row never fails the index), and every emitted
   observation passes the domain's own `Validate` — an unrecordable
   emission now breaks the build at the parse boundary, before any
   dispatcher sees it. Green.
5. **The 10-46 plan carries its executed-status header**: 4% tier complete
   (execution reports 12-55/13-44/15-39), 20% tier partial (P19.1, P20.3,
   P21, P24 done; the rest open).
6. **AGENTS budget held at 219/219** all session; docs gates green
   throughout; the batch pushed as ONE receipt commit (`22c9fc9`, 13 files)
   after squashing the daemon's three piecemeal interceptions.

## b) Partially done

1. **P19**: 19.3 done (above). P19.2 (idempotent-replay cost bench) and
   P19.4 (-race receipt for ingest) not started — both are next-session
   quick wins now that the property test exists.
2. **15-39 §f execution**: items 4, 5, 6 done; item 2 (FULL gate receipt)
   and item 3 (browser lane) not reached — the machine's 15-minute load
   was 62.66 against the <40 policy at check time, and the report demand
   arrived before the browser lane started. The new third docs gate has
   unit-level proof but no browser-lane pass yet.

## c) Not started

- **Browser lane** (`bash scripts/verify-all.sh`) — blocked only by session
  order, not by anything discovered.
- **FULL gate receipt** — load-policy gated (15-min 62.66 > 40); poll-gate
  will wait for the calm window whenever launched.
- **P19.2** (replay bench), **P19.4** (-race ingest).
- **P20.4** (harvest the three §f lists into TODO/ROADMAP with provenance).
- **P14–P18** (saved views, transcript search, per-DID pages,
  ingest-coverage, answer-rate) — the UI 20% tier.
- **P22** sibling folds, **P23** lint residue.
- **PR #3 disposition** and the **release cut** — both owner-gated (§g).

## d) Totally fucked up

1. **I wrote a property test against an API that does not exist**:
   `rapid.Bytes` is not in pgregory.net/rapid v1.3.0 (`Byte()` + SliceOf is
   the shape). The build failed, then I grepped the module cache, then the
   one-line fix. The verify-before-use discipline I apply to external
   claims applies to my own assumed API names too — a five-second grep
   before writing would have saved the round trip.
2. **Formatter/daemon edit races continue to burn round trips**: the
   scripts/README.md row insert silently missed (file reflowed between my
   read and my edit — the edit tool demanded a re-view), the claim-fix
   python script asserted mid-way (my quoted string didn't match the
   formatter's reflow), and the final batch got split across three daemon
   commits before I squashed it. Every single one was recoverable in
   seconds, but the pattern is now the session tax.
3. **My first claim-fix pass used the wrong annotation shape**: suffixing
   the historical name (`OldName (since folded into NewName)`) still
   leaves the OLD name backticked, and the gate correctly flags every
   backticked name. Cost a second edit round across three files. The rule
   I should have decided FIRST: a historical name survives prose
   UNBACKTICKED, its successor carries the code span.
4. **The report demand arrived with a build-broken tree** (the rapid.Bytes
   failure was mid-fix). Had the session ended five minutes earlier, main
   would carry a non-compiling test. The leave-green rule needs to be
   absolute: no report, no pause, no handoff until `go build ./...` is
   clean — one-line fixes finish FIRST.
5. **My own 16-58 report tripped the gate it announces**: section d1 named the
   caught phantoms in backticks, the gate (correctly) counted four claims, and
   the commit STILL landed because my shell pipeline swallowed the gate's exit
   code AND the pre-commit hook did not run the new gate. Both fixed same-hour:
   the phantom names are unbackticked here, and check-doc-test-claims.sh is
   wired into .githooks/pre-commit after the doc-links gate. The gate caught
   its own author within the hour — which is the argument for it.
6. **The daemon's piecemeal commit cost CI a red run** (`83647bd`: the P13
   test files without the P13 implementation). Not my commit, but my
   session produced the half-state it swallowed. The squash-on-sight
   habit is right; the daemon will keep finding gaps until the norm
   question (15-39 §g3) is answered.

## e) What we should improve

1. **Grep-the-module-before-you-write**: any rapid/servertiming/etc.
   generator or helper name gets a module-cache grep BEFORE the first
   compile. Same muscle as verify-external-claims, pointed at my own
   assumptions.
2. **Gate-wiring checklist**: a new guard script touches four places
   (script+chmod, CI job, verify-all, scripts/README row). I hit all four
   this time by luck of the plan's phrasing — write the checklist down in
   scripts/README so the fifth gate cannot skip one.
3. **Decide annotation shapes before editing docs**: the claim gate forced
   two passes because I improvised the annotate style mid-fix. A
   one-line convention ("historical names unbackticked; successor carries
   the code span") belongs in the gate script's header comment — added
   there now, but conventions decided up front cost less than conventions
   retrofitted.
4. **Leave-green is absolute**: build clean before any pause, whatever the
   interrupt.
5. **Load-aware ordering**: CPU-heavy work (property tests, benches,
   -race) earlier in a session; the load-gated FULL receipt LAST, when
   the machine is actually quiet. Today's order was right by accident —
   make it deliberate.

## f) Next (ranked; NEW = this session's additions, else 15-39 §f / 10-46)

1. ~~**Browser lane** `bash scripts/verify-all.sh` — now runs three docs~~ done — verify-all.sh:100 runs the claim gate; lane green 2026-10-26 (cf6e393 pass)
   ~~gates + four probes; first proof of the claim gate in the lane.~~
2. **FULL gate receipt** at the calm window: `nix develop -c bash
   scripts/poll-gate.sh -- env BUILDFLOW_NO_RESULT_CACHE=1 buildflow
   --build-mode dev` (or full mode when load < 40).
3. **P19.2**: idempotent-replay cost bench + receipt row in
   docs/http-stack.md (the dispatch no-op is measured at ≈6.1 µs/row;
   the bench makes it a standing receipt).
4. **P19.4**: `-race ./internal/ingest` receipt (the property test makes
   the package race-clean claims checkable).
5. **P20.4**: harvest this week's three §f lists (05-49, 12-55, 15-39)
   into TODO_LIST/ROADMAP with provenance.
6. **P14 saved views** (recent-views list + canonical share links).
7. **P15 transcript-mode search** (docs/gotchas.md FTS5 section first).
8. **P16 per-DID segment pages**.
9. **P17 ingest-coverage analytics** (rows/day, gaps, SLO on /api/ops).
10. **P18 answer-rate / short-call analysis** where sources support it.
11. **Execute the release** per §g2's answer: CHANGELOG split per the
    drafted mechanics, then `scripts/cut-release.sh v0.2.1|v0.3.0`.
12. **PR #3 disposition** per §g1's answer (rebase vs close+hand-roll).
13. **Daemon norm** per §g3's answer (claim files vs never-reset).
14. Annotate the 15-39 report's §f items 4/5/6 with `done at` markers
    (they are resolved inline now).
15. ~~AGENTS Commands: budget-neutral swap adding the claim gate beside the~~ done — check-doc-test-claims.sh runs in .githooks/pre-commit (the 17-12 incident fix, stronger than the AGENTS line)
    ~~docs gates (219/219 held; the swap must stay neutral).~~
16. Re-audit the art-dupl suppressed bucket after the next
    templ-components bump (`--show-suppressed`, AGENTS standing trap).
17. Re-verify versions.json after the next go-cqrs-lite tag wave.
18. CSP nonce enforce flip at the next host deploy.
19. MOS poller (T15, 2027-01 window) — the repo's one PARTIALLY row.
20. Telnyx live-API quarterly re-verify (recordings m27.9 window).
21. PII enforcement decision (`E164`/`ParticipantHash` exist, stored
    fields stay raw).
22. ~~Retention SLO dashboard on /api/ops.~~ done at `cf6e393`
23. ~~Ops counters via the pinned etag hooks (OnETagGenerated/On304).~~ done — internal/web/caching.go:78-79 (OnETagGenerated/On304)
24. Export streaming compression decision (streams skip ETag; wire gzip?).
25. Screenshot-probe baseline refresh (new cards: anomaly note, segments,
    spend).
26. Dependabot policy: github-actions-only grouping (the sqlite PR
    re-proved the multi-module mangling).
27. Backup restore drill (VACUUM INTO under the current schema).
28. FTS tokenizer coverage pass (the gotchas section lists the traps;
    a table test would pin them).
29. Detail-page print stylesheet audit (the @media print rules predate
    the new cards).
30. Poll-gate load telemetry (record the wait time, not just the verified
    window — the receipt gets more honest per run).

## g) Questions for the owner (cannot be figured out)

1. **PR #3 (dependabot sqlite 1.60.1→1.61.0)** — `@dependabot rebase`, or
   close it and hand-roll the bump (tidy + vendorHash re-derive + gates in
   a normal PR)? The classification comment is posted; only the disposition
   is yours.
2. **Release shape** — v0.2.1 (patch: fixes + integrity) or v0.3.0 (minor:
   everything since v0.2.0)? Both bodies + CHANGELOG mechanics are drafted
   in docs/drafts/2026-10-25_release-bodies-v0.2.1-vs-v0.3.0.md; I will not
   cut either without your pick.
3. **The daemon norm** — ratify the claim-file protocol (daemon refuses to
   commit while a session holds a claim file), or mandate the opposite
   (sessions never reset daemon commits, only amend forward)? Seven
   interceptions last session, three more today, one red CI run between
   them: the current arrangement taxes every session.
