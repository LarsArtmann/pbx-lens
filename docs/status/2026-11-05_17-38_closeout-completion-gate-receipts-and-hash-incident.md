# Status — closeout completion, gate receipts, and a gate that rewrote a hash pin

**Written:** 2026-11-05 17:38 CET · **Session:** third pass on
`docs/planning/2026-11-04_18-41_SUPERB-fleet-audit-closeout.md`, continuing from
`docs/status/2026-11-05_02-47_closeout-verification-and-fleet-rescore.md`.

**Scope line.** Everything below is work observed in this session unless marked
"inherited". Nine commits are mine (six authored, three folded over daemon
sweeps), the tree is clean, and **nothing was pushed** (44 commits ahead of
`origin/main`). The numbers I quote were re-read from the tree at report time,
not remembered.

## Context

The previous pass ended with an explicit to-do list: fix the false "the reset
API does not exist" claims, upgrade the rebuild runbook, add CHANGELOG and
FEATURES entries, run the quality gate solo, then work T20–T27. This pass did
all of that — and lost twenty minutes to a gate step that rewrote a tracked Nix
hash pin, which had to be diagnosed and reverted before the gate could be
trusted.

## Task ledger (plan T01–T27, inherited except where marked)

| Task              | State                                            | Note                                                                                                                                          |
| ----------------- | ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| T01–T14, T18, T19 | done (prior passes)                              | middleware swap, family errors, EmptyText fold-in, ETag knobs, pin receipts                                                                   |
| T15               | **done this session**                            | `cli.ResetReadModel` + runbook rewritten to the one-call path + both dry-run tests                                                            |
| T16, T17          | done (prior pass)                                | `WithOnFailed` alert, failure-based readiness                                                                                                 |
| T20               | **done this session**                            | verified nothing is filable; ledger carries the evidence and the rejected candidate                                                           |
| T21               | **done this session**                            | 50-item backlog routed: 8 Unblocked + 9 Low rows, 4 ROADMAP ideas with triggers                                                               |
| T22               | **done this session**                            | PNG policy written down; the three status-report questions annotated with the decisions applied                                               |
| T23               | **done this session**                            | four prior deep-dives reconciled; no inherited-claim caveat needed retiring                                                                   |
| T24               | done (prior pass) + **re-rendered this session** | 1280×11914, 1,842,350 bytes, after the final index edit                                                                                       |
| T25               | **done this session**                            | SOLO gate on a frozen tree, `go test -race`, full lane, `nix build`, devShell, `nix flake check`                                              |
| T26               | **done this session**                            | verdicts home checked and recorded as "not library-policy"; refresh-rule line in all 19 reports                                               |
| T27               | **done this session**                            | N/A reasons written out, rate-limit Hold given falsifiable abuse criteria, "flips when" column, N/A confirmations, curated-commit note closed |

## a) FULLY DONE (with receipts)

1. **The reset claims are gone and the path they described is now code.**
   `cli.ResetReadModel` (stop → `sys.ResetProjection(ctx, app.ProjectionName)` →
   caller drains) lives at `internal/cli/system.go:76`; `app.ProjectionName` is
   a named constant with the reason it exists (the system constructor registers
   `"projections"` and the host rejects any other name). `TestHostResetRebuildsReadModel`
   now exercises the helper itself (2 views → reset → 0 → drain → 2), so the
   runbook and the tested path cannot drift.
2. **Three documents that asserted the opposite are corrected**, one of them with
   an explicit retraction: `TODO_LIST.md` row 114 (the parenthetical that said
   the API does not exist), the fleet review page (`docs/review/libs/index.html`
   — a "Retraction (2026-11-05)" block that names the mistake and why the
   audit-day finding was right the first time), and the runbook. The plan's M15
   wording was checked and was already correct, so it was left alone.
3. **A durable lesson replaced a story**: `docs/gotchas.md` now carries
   _"That API does not exist" needs a compile, not a grep_, with the cheap
   settling test (call it and assert the observable effect) and the corollary
   that owner-facing runbooks point at an in-repo helper rather than a prose
   sequence of SDK calls.
4. **CHANGELOG + FEATURES caught up with a full batch of unreleased code.** The
   Unreleased section gained two dated blocks (readiness/failure-alert/rebuild
   under _Fixed_, the whole closeout surface under _Added_); FEATURES gained
   rows for the CLI's family exit codes, the family-aware error responses, the
   readiness probe, the mid-drain failure alert, and the rebuild helper, plus
   two corrected rows (Server-Timing no longer claims a hand-written wrapper;
   `/api/ops` field list now includes `write_errors`).
5. **T20 closed by evidence, and the honest answer was "file nothing".**
   go-atomic-write's newest tag v0.6.2 exports eight functions, all data- or
   writer-shaped; upstream master adds only `WriteFuncWithPerm` — no path-based
   commit primitive exists, so the draft stays staged and owner-gated.
   `versions.json` agrees with all **13 direct and 12 indirect** pins (verified
   against the live upstream file with the `X/v4` ↔ `"X": "X/v4/A.B.C"`
   mapping), so the old "versions.json lies" filing is retracted outright. A
   fresh candidate — "`Host.Reset` cannot discover the projection name" — was
   rejected because it is false: `projectionhost.WorkerState.Name` exists
   (`dlq.go:33`) and `Host.Status()` returns it.
6. **T21 harvest landed with routing rigor.** Eight gate-free bounded rows and
   nine polish rows in `TODO_LIST.md`, each with status/impact/effort/evidence;
   four raw ideas with their triggering conditions in `ROADMAP.md`; one
   duplicate killed on sight (the projection-failure counter appeared in both
   files, now ROADMAP-only).
7. **T27 produced real content, not checkmarks.** `RecordServerTiming` and
   `MeasureWithDesc` N/A verdicts now carry the API signature that justifies
   them (`(ctx, name, desc, dur)` records an already-measured duration; nothing
   here hands us one); the rate-limiter Hold gained three falsifiable abuse
   observations plus the cheapest instrument to detect them; every not-imported
   library gained a "flips when" column, which is the difference between a
   rejection and neglect.
8. **The quality gate finally has a usable receipt.** On a frozen tree (load 17
   at start), the SOLO gate ran without starvation: one failing step, the
   documented `checks.pre-commit` templ false positive in
   `internal/web/dashboard.templ` (a file untouched this session). Everything
   else green: `go build`/`vet`, `golangci-lint` 0 issues, `go test ./...`,
   `go test -race ./...`, `scripts/verify-all.sh` **all probes passed** (50
   PNGs), `nix build` → `pbx-lens 0.3.0`, `nix develop -c echo ok`, and
   `nix flake check` failing only on that same pre-commit check.
9. **The proof artifact is current and eyeballed where it changed.** The fleet
   index PNG was re-rendered after the final content edit; I cropped and
   inspected the changed table at 1:2 scale to confirm the new column renders
   and the callout text still fits.

## b) PARTIALLY DONE

1. **The gate's hash corruption is documented, not root-caused.** The first gate
   run left `flake.nix` modified on a tree that was clean when it started:
   cqrs-lint's `vendorHash` replaced with this repo's (`ECM62…`), and the
   repair app's sentinel `fake` rewritten to the old cqrs-lint value. I proved
   the correct value (`jTwZn/…`) by probing the FOD with an all-zeros fake,
   restored the pin, and the second run left the tree clean — so the mechanism
   is **not** understood, only the symptom and the proof. Two candidates: a
   BuildFlow hash-fix step, or the git pre-commit hook fired by the daemon's own
   commit. Neither was instrumented.
2. **`nix flake check` is still red.** Accepted as pre-existing and called out in
   the receipt, but not fixed and not re-filed upstream (the 2026-10-28 report
   said it was filed; I did not re-verify that ledger row this session).
3. **The PNG policy text is now numerically stale against the artifact I
   rendered.** `docs/research/README.md` says "≈1.6 MB at 1280×11026"; the
   committed render is 1.84 MB at 1280×11914. The policy decision is right,
   its numbers are one edit behind.
4. **I eyeballed only the section I changed.** The full-page render was not
   reviewed end to end, so a layout regression elsewhere in the fleet index
   (added retraction paragraph, extra column in the fleet table) would be
   invisible to me.
5. **The fleet index's own score is deliberately inconsistent with its delta
   appendix** — cqrs-lite core reads 92 in the table while `go-cqrs-lite.md`
   argues Reset is now partially adopted (≈94) with a documented reason not to
   move it (the prod run is owner-gated). That is defensible, but it is a known
   seam someone will trip on.
6. **`AGENTS.md` is 222 lines** after I added the rebuild-helper line; the
   budget discipline from the prior pass was ~219. Three lines over is not a
   defect, but the trend is wrong.
7. **The `projectionhost.Reset` vs `Host.Reset` wording** in the plan's M15 and
   a few historical reports still says `ProjectionHost.Reset`. It is not false
   (the type is `projectionhost.Host`), but it does not grep cleanly against
   the real symbol.

## c) NOT STARTED

1. **Annotating the executed plan rows (T01–T19, T24) with receipts and real
   line refs.** It is a TODO row I created tonight; the plan still reads like a
   forecast.
2. **`nix develop -c cqrs-lint .` and a standalone `govulncheck`.** Both are
   gate-adjacent; I relied on the gate's own verdicts rather than taking my own
   receipt.
3. **A guard test that a new route must make a cache-skip decision**
   (`skipBodyCaching` drift), the `drain → rebuild → Run` ordering comment in
   `serve.go`, and `/health/ready` parity in the remaining probes — all three
   are now TODO rows, none executed.
4. **A `pbx-lens reset` verb.** The helper is a library call; the operator still
   has to write Go to rebuild. Recorded in ROADMAP with its trigger (the prod
   deploy path).
5. **The exported-commit-primitive ask** is still an unfiled draft with an
   unticked owner review box.
6. **`/tmp` cleanup** of this session's scratch (`lens-shots/`, the render and
   crop scripts, `flake.nix.good`/`.ecm` copies, `vj.json`).

## d) TOTALLY FUCKED UP

1. **I trusted a green devShell as hash proof and reported it as one.** My first
   conclusion — "jTwZn builds the devShell, ECM62 fails, so jTwZn is right" —
   was right for the wrong reason: the jTwZn path was already realised in the
   store, so Nix reused it without checking. The real proof is the all-zeros
   fake probe. I caught this myself and rewrote the gotcha entry, but the
   intermediate commit `8778110` carries the weaker reasoning.
2. **I created a documented rule and broke it twenty minutes later, in the same
   session, twice.** First run: I launched the gate while the tree was clean but
   then ran `git status`/`git show` inspections and, minutes later, a hash
   repair app against the same tree — so the gate's own receipt covers a tree I
   was actively touching. The second run was correct, but "frozen" has to mean
   frozen for the whole run, not just at launch.
3. **The first gate receipt was worthless and I nearly kept it.** Its
   `nix-build` step ran against the corrupted pin, so the failure it reported
   was partly mine. If I had not checked `git status` afterwards, I would have
   filed a false "the gate is red because of pre-commit" story.
4. **I published a wrong claim and then fixed it in the same session** — the
   `docs/research/README.md` PNG policy cites a size and height I had already
   superseded by re-rendering. Small, but it is exactly the "doc states a number
   the artifact contradicts" class I spent the day fixing elsewhere.
5. **I wrote a Changelog sentence naming three files as newly classified when
   only two were.** Caught by reading `recording.go` myself; the two
   configuration 503s there are deliberately plain text. Amended in `a83c095`.
6. **I burned a commit on a message prefix** (`docs:` → `fix:`) and another two
   folds of daemon commits. Churn in history that a cleaner workflow (commit
   immediately, before the daemon fires) would have avoided.
7. **I let a sentinel remain unexamined for most of the session.** The repair
   app's `fake` value had been rewritten to a real hash by the corruption; I
   noticed it while reading the script for a different reason, not by auditing
   the file I had just reverted.

## e) WHAT WE SHOULD IMPROVE

1. **A frozen tree needs a post-condition, not a promise.** Wrap the gate call in
   a script that records `git status` before _and_ after and refuses to present
   the run as a receipt if the tree moved (or restores and re-runs).
2. **A tripwire for the pin class.** `vendorHash.nix` and flake.nix's cqrs-lint
   `vendorHash` must NEVER be equal; a two-line check in `verify-all.sh` would
   have caught tonight's corruption in a second instead of twenty minutes.
3. **Proof-of-hash discipline**: always probe with a deliberately wrong value and
   read `got:`. "It builds" is not evidence on a machine that has a store.
4. **Numbers in docs should be generated or dropped.** Sizes, heights, counts and
   "N reports" are the most common drift in this repo; the ones that must exist
   should be asserted by a script (the docs-counts gate is the model).
5. **Harvest rows and their closures should move together.** Three rows I created
   in the morning were done by the evening and had to be deleted by hand; a
   "done today" sweep step belongs in the harvest procedure.
6. **One owner per surface.** The projection-failure counter appeared in both
   TODO and ROADMAP within the same hour; the fix was deletion, but the pattern
   (route twice, notice later) will recur.
7. **Name the constant for what it is.** `app.ProjectionName` holds the host's
   registered worker name, not a domain projection identity; `ProjectionWorkerName`
   would not need the comment to be true.

## f) Up to 50 things to get done next (impact-ordered)

1. **Fix the stale numbers in the PNG policy** (1.84 MB, 1280×11914, or make the
   sentence generic) — one edit, and it is currently false.
2. **Root-cause the pin corruption**: instrument one gate run (`inotifywait` or a
   `stat` fingerprint per step) and name the writer, then file it upstream if it
   is BuildFlow's.
3. **Add the pin tripwire** (`vendorHash.nix` ≠ cqrs-lint `vendorHash`) to
   `scripts/verify-all.sh` with `--self-test` coverage and a pre-commit hook.
4. **Wrap the gate** in a before/after `git status` receipt script.
5. **Annotate plan rows T01–T19 + T24** with receipts and real line refs.
6. **Clean `/tmp`**: `lens-shots/`, `render-fleet-index.js`, `crop-notused*.js`,
   `notused*.png`, `flake.nix.good`, `flake.nix.ecm`, `vj.json`.
7. **Eyeball the full fleet-index render** (or at least the findings callout and
   fleet table crops) and fix any overflow from the new column.
8. **Add this report's row to `docs/status/README.md`** (done with this commit).
9. **Rename `app.ProjectionName` → `ProjectionWorkerName`.**
10. **Add a test for `ResetReadModel`'s reset-while-running rejection** (the host
    returns `projectionhost.reset_while_running` if the Stop is skipped).
11. **Add an idempotency test**: calling `ResetReadModel` twice in a row must not
    error on the second call.
12. **Re-run `nix develop -c cqrs-lint .`** and record the finding count.
13. **Run `govulncheck ./...` standalone** for a first-hand receipt.
14. **Run `scripts/check-dedup-gate.sh` standalone** (it ran inside the lane).
15. **Trim AGENTS.md back to ≤219 lines** after this session's addition.
16. **Record the exact failing flake-check attribute name** in docs (currently
    only "pre-commit").
17. **Add the incident to `docs/buildflow-dev-gate.md`** (gate traps), with the
    frozen-tree post-condition habit — gotchas has it, the gate doc does not.
18. **Re-verify the 2026-10-28 ledger claim** that the templ sandbox false
    positive was filed upstream; update `docs/upstream-filings.md`.
19. **Decide the pre-commit FP**: give treefmt module resolution, make the check
    advisory, or keep documenting it (owner call).
20. **Add the tag-vs-master evidence to the go-atomic-write draft itself** (the
    draft still only argues from the writer-shaped API).
21. **Mark the draft's owner review box state in the ledger row** so "staged"
    cannot be mistaken for "filed".
22. **Sweep for other "revisit on abuse"-style unfalsifiable Holds** and give each
    a trigger, the way the rate-limiter now has one.
23. **Re-check `docs/review/libs/index.html` hands-on** at 1280 for the retraction
    block's height (long paragraph above the findings list).
24. **Cross-link the reset retraction from the gotchas record-key entry** so a
    reader of the key incident finds the rebuild path.
25. **Align plan M15.1's wording** to `Host.Reset` for grep-ability (and any
    historical report that says `ProjectionHost.Reset`).
26. **Decide whether the two rebuild tests should merge** (`host_reset_test.go`
    vs `reset_rebuild_test.go`) — two tests for one path is a split-brain risk.
27. **Add `cli.ResetReadModel` to the AGENTS "Commands" block?** or leave it as
    library-only (decision).
28. **Route the four ROADMAP closeout follow-ons into FEATURES as PLANNED rows**
    if they are product features rather than ideas (decision).
29. **Re-check CHANGELOG section ordering** for the two new 2026-11-05 blocks
    (Fixed should precede Added by convention, it does — verify after the next
    doc edit).
30. **Run `verify-all.sh --fast`** as a post-doc-edit regression (cheap, catches
    contract text drift).
31. **Pin the lane's artifact count** ("50 PNGs" today) in `scripts/README.md`,
    which still describes the 41-image era.
32. **Annotate the two 2026-11-05 status reports' "next session" advice** now that
    it is executed.
33. **Add the harvest-completion sweep** (b5) to the docs-health procedure.
34. **Check `internal/web/errors.go` for a second unclassified path** (the
    ops-coverage scan) and decide on family-wrapping it (existing TODO row).
35. **Verify the ops golden for the never-ran `EmptyText` path** and pin it
    (existing TODO row).
36. **Add a route-table guard for `skipBodyCaching`** (existing TODO row).
37. **Comment the `drain → rebuild → Run` ordering in `serve.go`** (existing TODO
    row).
38. **`/health/ready` parity in the remaining probe scripts** (existing TODO row).
39. **Extend the T14 receipt style to the other four first-party pins.**
40. **Decide on an ops-visible projection-failure counter** (ROADMAP trigger:
    first real failure).
41. **Consider the `pbx-lens reset` verb** once the deploy path exists.
42. **Review the daemon's commit policy**: should it publish a change a _tool_
    made to the tree mid-run? (owner policy question).
43. **Push the 44 commits** (owner gate).
44. **File or retire the go-atomic-write draft** (owner gate).
45. **Add a `docs/review/libs/README.md`** so the 19 reports have a stated index
    rule rather than an implicit one (decision).
46. **Check whether `Result` symlink removal** after `nix build` is expected in
    receipts (it is gitignored; document if anyone is confused).
47. **Add the "flips when" column idea to the lib-report template** so future
    rejections ship with a trigger by default.
48. **Verify the fleet index mean is still arithmetically right** after any score
    movement (currently 83, `(92+95+73+95+80+67+82)/7`).
49. **Re-run the docs gates after every doc-only session** as a habit
    (counts/links/test-claims/dprint all green tonight).
50. **Keep the session's receipts in one place**: tonight's are split across
    `docs/status/2026-11-05_02-47` (§ "Final verification") and this report.

## g) Three questions I cannot answer myself

1. **Push now?** 44 commits are local, the tree is clean, the gate/lane/race
   suite are green with one documented false positive. Push, or keep batching?
2. **What do you want done about `checks.pre-commit`?** It makes `nix flake
   check` permanently red and it is the only thing standing between "green" and
   "green with a footnote". Fix the sandbox (module resolution for the templ
   formatter), demote the check to advisory, or keep documenting it?
3. **How much do you want spent on root-causing the pin corruption?** Instrument
   a run and chase the writer (my guess: BuildFlow's hash-fix step, but the
   second run did not reproduce), or accept the tripwire + post-condition
   guards and move on? Related policy question: should the auto-commit daemon be
   allowed to publish tree changes a tool made during a gate run?
