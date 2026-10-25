# Status — 2026-10-25 15:39 — 4% tier completed (P8/P12/P13 shipped), deployment-swap proof, history-collision recovery

Session resumed after the 13-44 report with the owner's "keep going until the
entire list is finished" mandate. This session completed the plan's whole 4%
tier (P8–P13), landed the first 20%-tier items (P19.1, P20.3, P21, P24), and
hit — and recovered from — the worst git state of the week: my local history
rewrite collided with an externally-pushed branch.

## a) Fully done (receipts)

1. **The 13-44 report finished and pushed** (`7a14ec1`): the daemon had
   swallowed the report file into `b145f97` — squashed (unpushed), the
   README index row inserted, docs gates green. Along the way dprint
   mangled report prose (line-initial `+` re-read as a list marker:
   "vendorHash re-derive + gates" became "- gates"); caught by reading the
   formatter diff, fixed by rewording.
2. **PR #4 closed cleanly** — the red receipt was made durable BEFORE the
   branch deletion: run `40194004724` / job `121170835402` (Guard Self-Tests
   fail 7 s; all other jobs pass) recorded in the 13-44 report (`b73d40d`).
   Branch `proof/red-docs-gate` deleted.
3. **PR #3 (dependabot sqlite) classified publicly**: comment names all four
   red jobs (tidy-drift, stale-base lint, phantom `@v0.9.0` art-dupl tag,
   since-fixed styles staleness) and frames the disposition choice. The
   decision stays the owner's (§g).
4. **P8.1 the retire payload golden** (`50f67b8`):
   `TestGoldenRetireScenarioPayloadTrail` pins record+retire as
   payload-hash-stable lines (`retire_scenario_payloads.snap`). First
   attempt failed on the DSL: `markActStart` is first-act-only, so a
   chained act's `Then` still sees act one's events — the fix is one
   `Then(EventObserved, EventRetired)` after both acts. The gotcha was
   already written in docs/gotchas.md (2026-10-25 entry); my failure
   re-confirmed it.
5. **P8.2 the lifecycle property oracle**: record → correct → enrich →
   correct-after-enrich → optional re-enrich → optional retire, every op
   forced to change state, every act converged via `ThenQueryFunc`, whole
   journal faced with `assertFoldParity` + version ordering. The rapid
   shrinker earned its keep twice (see §d5); gocognit 31 forced the
   `driveLifecycleTail`/`enrichmentChecker` extraction.
6. **P8.3 the LookupView quadruple**: bare-key row via real dispatch,
   legacy-key row seeded through `Store.ApplyRecord` with a hand-built
   `EventWithID{ID: LegacyRecordKey(sid)}` (the only write path that
   chooses the row key), clean not-found, and both-exist where the bare
   key must beat a 999-second decoy. Green on first run.
7. **P8.4 the race receipt**: `-race ./internal/app` clean (5.8 s); full
   package lint 0 issues.
8. **P12 the anomaly baseline, complete** (`e14918f`): `volumeNote` derives
   a weekday mean from the window's complete days (a partial today never
   dilutes itself) with a ±2σ Poisson band, rendered as one quiet line on
   the heatmap card (usual / unusually quiet / unusually busy / honest
   no-history under two same-weekdays). Six unit tests, the dashboard
   golden re-pinned (the note is deterministic under the frozen clock:
   "usual for a Tuesday (typically 0-0)"), FEATURES row, CHANGELOG entry.
9. **P13 the streaming Server-Timing work** (`f3aad1f` content; see §d1 for
   the hash saga): the recording proxy was ALREADY instrumented — the real
   gap was (a) export had no metric and (b) the doc's claimed pins had
   never been written. Added `firstpage` (parse + first keyset page,
   stopped at the first row or stream end, body loop never timed), the
   zero-row explicit header commit (the wrapper injects Server-Timing
   lazily at first write, so an empty stream would drop `firstpage` AND
   `total` exactly when nothing matched), `TestExportCarriesFirstPageTiming`
   (both formats + zero-row, through the real middleware wrapper), and
   `TestRecordingProxyTimingHopBudget` — the test docs/http-stack.md has
   claimed since 2026-10-18. The doc's phantom
   `TestRecordingProxySlowUpstreamStaysBounded` corrected to the real
   fast-fail pair; new export section written; TODO row closed.
10. **P19.1 the deployment-swap proof** (`e10e15a`):
    `TestMemoryAndSqliteDeploymentsStayEquivalent` boots memory and sqlite
    over the same DomainConfig, feeds the identical record/correct/enrich/
    retire lifecycle, asserts `systemscenario.AssertJournalEquivalence`
    plus read-model parity. Green on first run.
11. **P21 the sibling versions.json** — regeneration ran IDEMPOTENT (the
    manifest already carried systemscenario v4.2.1; the drift was repaired
    between the gotcha's verification and this run), gate answers "fresh
    (112 module trains, local tags)". Receipts in docs/upstream-filings.md
    - both TODO rows closed (versions.json AND the Server-Timing row).
12. **P20.3 the 19-32 plan's executed header**; P20.1/P20.2 verified as
    already written in gotchas (multi-act window + AppendBatch rollback).
13. **P24 the analytics documentation pass**: README's serve section gained
    the denominator-honest cards walkthrough (spend, segments +
    `?number=` deep links, quality trend, weekday note, counterpart
    history); DOMAIN_LANGUAGE.md gained the canonical "Analytics
    denominators" table. FEATURES re-read: exactly one PARTIALLY row (MOS)
    and it is still accurate.
14. **The 4% tier (P4–P13) is now COMPLETE.** 1% tier was closed before.

## b) Partially done

1. **P19**: P19.1 done (above); P19.2 (idempotent-replay cost bench),
   P19.3 (parser fuzz seeds), P19.4 (-race on ingest) not started.
2. **P20**: P20.3 half done (19-32 header yes; the 10-46 plan's own header
   waits for this report's outcome); P20.4 (harvest the three §f lists
   into TODO/ROADMAP) not started; P20.5 gates green throughout.
3. **Final verification**: unit suites + full lint green all session; the
   FULL buildflow gate and the browser lane (verify-all.sh) have NOT run
   today — the anomaly note and the firstpage metric have unit + golden
   proof but no browser-level probe yet. Push of the final commit at
   report time: pending (§f1).

## c) Not started

- **P14 saved views**, **P15 transcript-mode search**, **P16 per-DID
  pages**, **P17 ingest-coverage analytics**, **P18 answer-rate analysis**
  — the UI/analytics 20% tier, deliberately deferred (each is a surface
  change late in a session already carrying three feature commits).
- **P22 sibling-repo folds** (call_direction CDR template, lens.nix backup
  wording, webphone deep link) — sibling-window gated.
- **P23 lint/gate residue** (gofumpt/v2.14 parity, unknown-`//nolint`
  grep, AGENTS-budget CI job) — low value, deferred.
- **P25 release-body prep** — not started; the shape question is §g2.
- **The 13-44 report's three owner questions** — all three still open
  (they roll into §g here).

## d) Totally fucked up

1. **The history collision (the big one).** I soft-reset the daemon's
   `83647bd` and re-committed its content as `f3aad1f` while — unknown to
   me — that commit and four more daemon commits were being pushed to
   origin. At push-prep time local main was non-fast-forward. Recovery was
   clean and lossless (`reset --mixed origin/main`, verified my tree was a
   strict content superset — the only origin-side extras were stale
   versions of rows I had already edited — re-committed `e10e15a` on top),
   and NO force-push was used. But the collision happened because I never
   fetched between the reset and the rewrite: a five-minute window, and
   the daemon's push landed inside it. The rule I now owe: **fetch
   immediately before ANY history move, not just before push.**
2. **Seven daemon races in one session** (`b145f97`, `53af54d`,
   `9756050`, `4f51f4a`, `b8e4874`, `52e2a24`, `4bef8ee`): every one
   needed a squash/recovery cycle, and two of them swallowed half-finished
   states that then LOOKED committed. The concurrency norm on the books is
   right and still not enough — this needs the owner (§g3).
3. **I nearly trusted the phantom-test doc.** docs/http-stack.md claimed
   two recording timing tests since 2026-10-18; neither existed until I
   wrote one today. During P13 planning I assumed they existed and almost
   scoped the task as "verify" instead of "write". The grep that found the
   emptiness came late, by luck of a failing assertion, not by design.
   Doc-claimed Test names should be gate-checked (§f4).
4. **My P12 generator put Sunday at index 6** — `time.Weekday()` has
   Sunday=0 — six red tests before the fix, plus a wrong expectation
   (all-zero window is Usual [0,0], not NoHistory). Sloppy; the correct
   matrix was the FIRST thing I should have written down.
5. **The lifecycle oracle's shared ID pool**: three fixed native IDs dealt
   randomly across iterations meant a retired stream could be dealt into a
   later lifecycle, whose record then (correctly) rejects. The shrinker
   found it in seconds. Disjoint IDs per iteration should have been the
   design, not the patch.
6. **dprint changed PROSE MEANING** in the 13-44 report (list-marker
   reinterpretation, §a1). I caught it by reading the formatter diff — a
   habit that paid once and must never lapse.
7. **Edit-tool collisions three times** (formatter/daemon rewriting files
   between my view and my edit; one multiedit silently applied only part).
   Each cost a re-view + re-apply round trip. After ANY background
   formatter or daemon touch, re-view before editing.
8. **gocognit 31** on the first lifecycle driver — lint caught it, the
   extraction was mechanical, but the complexity was visible in my first
   draft and I shipped it anyway.

## e) What we should improve

1. **A fetch-before-history-move reflex.** Today's collision was 100%
   preventable by `git fetch origin` before the soft reset. The existing
   "never reset pushed work" rule needs the operational trigger: FETCH
   FIRST, every time.
2. **Gate-check doc-claimed test names.** A tiny script: extract
   backticked Test-prefixed names from docs/*.md, assert each exists in the
   tree. Would have caught the phantom pair 7 days ago. Seed-red proof
   first, then a CI lane.
3. **Daemon claim protocol.** Seven races in one session is the argument.
   The proposal (claim files, no-commit-while-claims-exist) is drafted;
   ratification and a daemon-side implementation are the owner's call.
4. **Property-test generators: disjoint stream pools by default**, weekday
   indexing documented at the array literal, and the zero-case expectation
   derived from the model, not guessed.
5. **Formatter diffs on prose get read line-by-line** — dprint proved it
   can flip meaning, not just layout.
6. **Streaming headers need a "who commits the header" review question.**
   The zero-row export dropping its timing header was invisible until a
   test asked for it; any lazy-injection middleware deserves that question
   per route.

## f) Next (ranked; provenance: 10-46 plan unless marked NEW)

1. Push `e10e15a` (fast-forward) and watch CI — includes the new Guard
   Self-Tests job. (NEW)
2. FULL gate receipt at a calm window: `nix develop -c bash
   scripts/poll-gate.sh -- env BUILDFLOW_NO_RESULT_CACHE=1 buildflow
   --build-mode dev` — none recorded today. (NEW)
3. Browser lane `bash scripts/verify-all.sh` — the anomaly note and the
   firstpage metric have no browser-level proof. (NEW)
4. Doc-claims gate: extract backticked Test-prefixed names from docs/*.md, assert
   existence; seeded red first. (NEW, from §e2)
5. The 10-46 plan's executed-status header + this report's index row.
   (NEW)
6. P25: draft BOTH release bodies (v0.2.1 patch vs v0.3.0 minor +
   Unreleased split) via `scripts/cut-release.sh --dry-run` so the
   owner's call is five minutes.
7. P19.2: idempotent-replay cost bench + receipt row.
8. P19.3 parser fuzz seeds (cdr/webhook/agent) + P19.4 -race receipt.
9. P20.4: harvest this week's three §f lists into TODO/ROADMAP with
   provenance.
10. P14 saved views (recent-views list + canonical share links).
11. P15 transcript-mode search results (docs/gotchas.md FTS5 section
    first).
12. P16 per-DID segment pages (volume/AI/quality per number).
13. P17 ingest-coverage analytics (rows/day, gaps, staleness SLO).
14. P18 answer-rate / short-call analysis where the source supports it.
15. P22 sibling folds: call_direction in the CDR template, backup-rotation
    wording in lens.nix, webphone deep link.
16. P23 lint/gate residue: gofumpt/v2.14 parity, unknown-`//nolint` grep,
    AGENTS-budget CI job (220-line seed).
17. Execute the PR #3 disposition once §g1 answers: `@dependabot rebase`
    or close + hand-rolled bump with tidy + vendorHash re-derive.
18. Ratify or reject the daemon claim protocol (§g3) and implement the
    daemon side if ratified.
19. Re-audit the art-dupl suppressed bucket after the next
    templ-components bump (standing AGENTS trap, `--show-suppressed`).
20. Re-verify versions.json after the next go-cqrs-lite tag wave (today it
    was already fixed; the projection can drift again).
21. CSP nonce enforce flip at the next host deploy (standing owner gate,
    docs/http-stack.md CSP section).
22. MOS poller (T15, 2027-01 window) — the repo's one PARTIALLY row.
23. Telnyx live-API quarterly re-verify (recordings m27.9 window).
24. PII policy: `E164`/`ParticipantHash` exist but stored fields stay raw
    — the enforcement decision is still open.
25. Retention SLO dashboard on top of `/api/ops` (the "is the index
    healthy" half-answer).
26. Ops counters via the etag hooks (`OnETagGenerated`/`On304`, pinned
    v0.6.1 has them; TODO_LIST row).
27. Export streaming compression decision (streams skip ETag; do they want
    q-negotiated gzip on the wire?).
28. Golden expansion: new cards (anomaly note, segments, spend) have unit
    - golden coverage; a screenshot-baseline refresh would pin pixels.
29. Dependabot grouping policy: today's PR re-proved the multi-module
    mangling — consider pinning dependabot to github-actions-only.
30. Backup restore drill (the VACUUM INTO path has never been restored
    under the current schema).

## g) Questions for the owner (cannot be figured out)

1. **PR #3 (dependabot sqlite 1.60.1→1.61.0)**: after the classification
   comment — `@dependabot rebase` (retry through the pipeline, accepting
   it will need a tidy/vendorHash pass anyway), or close and hand-roll
   the bump in a normal PR where the multi-module mangling can't happen?
2. **Release shape**: v0.2.1 patch (fixes + test/gate depth only) or
   v0.3.0 minor (anomaly baseline + firstpage counted as features)? I
   will prep both bodies either way — which one is the DEFAULT the
   choreography should target if you don't answer before the next
   release window?
3. **The daemon norm**: seven intercepted commits today, one of them
   pushed to origin underneath my history rewrite. Do you want (a) the
   claim-file protocol ratified and implemented daemon-side (the daemon
   refuses to commit while a session holds a claim file), or (b) the
   opposite rule — sessions NEVER reset daemon commits, only
   amend-forward on top of them?
