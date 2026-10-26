# Session status — truth-first analytics surfaces, non-browser gates, docs audit (2026-10-25 08:08)

**Scope:** one session on `/home/lars/projects/pbx-lens`, concurrent agent sessions
active the whole time (shared tree, auto-commit daemon sweeping). Input: the
TODO_LIST after the 2026-10-24 harvest + the 2026-10-23 Pareto plan's
truth-first tiers. **Everything below is this session's own work**, except where
a concurrent session's landing is named explicitly (I verified those, I did not
author them).

**One-line verdict:** the three north-star analytics slices shipped on top of
correct counts (duplicate source rows collapse, CDR direction derives from the
dialplan), four non-browser gates landed with self-tests, the five unaudited
living docs were audited (one archived), and a real 25 s → 0.81 s test-cost
defect was found and fixed in the go-appkit deep dive. Full browser lane green
(42 PNGs) and `nix build` green. One self-inflicted destructive near-miss
(empty TODO_LIST, recovered from git).

---

## a) FULLY DONE

1. **Duplicate source rows collapse on the list and the dashboard**
   (`internal/web/representative.go`, `queries.go`, `dashboard.go`). One real
   call described by a CDR + an agent transcript used to render twice and count
   twice. The representative row is the CDR (transcript only as fallback); the
   representative absorbs what only its sibling knew (the opening line). The
   export deliberately stays per-source. Six tests, including a page-boundary
   case, a timestamp-tie case and the stall guard.
2. **Pagination stays honest under collapse.** The list walk scans one batch
   ahead with a probe row (`WithLimit(batch+1)`), keeps the tie set, and fails
   loudly (`lens.web.list_cursor_stalled`) instead of silently truncating when a
   timestamp group is wider than a batch — the export's documented failure mode
   applied to the page.
3. **CDR direction derivation** (`internal/ingest/cdr/cdr.go`): a
   template-declared `call_direction` column wins, read **by value** (so extras
   can be re-ordered and the MOS column keeps working), then
   `context == "public"`, then the transfer heuristic (external caller → internal
   extension = inbound). The live 2026-10-23 mislabel (4084757593 → 9100 indexed
   outbound) is fixed and pinned by fixtures.
4. **Dashboard "Spend" card** (slice 1): window spend, cost per billed minute,
   week-over-week delta, daily sparkline. Honest empty state, "N of M calls
   priced" denominator, and a mixed-currency window refuses to name one currency.
5. **"Who calls what" segmentation** (slice 2): top 5 our-numbers and
   counterparties with calls / AI share / avg duration / spend, each row a
   drill-through into the filtered list via a **new `?number=` filter** that
   matches either side of a call (`metaengine.WithOr`), plus a Number input in
   the filter form and length validation (400 with an explanation).
6. **"How calls end" quality rollup** (slice 3): hangup-cause distribution
   (long tail folded into `other`) plus standing rates for `no user turn`,
   `self-dial` and the recording tail — each over its OWN population
   (transcripts / calls with both parties / timed transcripts), with
   pluralized denominators so "1 transcript" never reads as "1 transcripts".
7. **JS lint gate** (`scripts/check-js-lint.sh` + `.oxlintrc.json`): oxlint with
   `no-undef` + `no-unused-vars`, browser+node globals, wired into
   `.githooks/pre-commit`, `--self-test` seeds an undefined reference. **It found
   a real latent bug on the first run**: `fail()` was called four times in
   `screenshot-probe.js` and never defined (the print/empty checks would have
   thrown a ReferenceError).
8. **Docs count gate** (`scripts/check-docs-counts.sh`): re-derives AGENTS.md's
   archived-report count, the status index's closed-report count, the planning
   archive's closed-plan count, and the live-table rows ↔ live-report files
   (both directions). `--self-test` proves a drifted claim and a missing claim
   are red.
9. **Docs link gate** (`scripts/check-doc-links.sh`): every relative Markdown
   link in the living docs resolves; fenced code blocks and inline code spans are
   skipped (a Go generic signature was the first false positive), URLs/anchors/
   absolute paths are out of scope, archived corpora stay closed history.
10. **Load-poll → exec gate wrapper** (`scripts/poll-gate.sh`): reads the 15-min
    load, prints the verified triple, `exec`s the full gate in the SAME process
    (no poll→launch gap), refuses to arm on an unreadable source, exit 75 when
    not ready. `--self-test` covers over-policy / ready / unreadable.
11. **Static assets must revalidate on the FIRST replay**
    (`assertFirstTryRevalidation` in `cmd/pbx-lens/serve_test.go`, used for
    `/app.css` and `/static/htmx.min.js`): the old path tolerated a re-tag
    because HTML embeds a timestamp; a static asset never changes, so a 200 on
    the first conditional GET is now a failure.
12. **`verify-all.sh` parses its own flags before doing anything** — a typo'd
    `--baseline` used to surface after four browser suites.
13. **`prettier` + `oxlint` in the devShell**: the prettier-format step now runs
    in 777 ms inside `nix develop` instead of the 2.3 s `nix run` fallback.
14. **Detect-only findings triage, decided in writing**
    (`docs/buildflow-dev-gate.md`): art-dupl 101 (the CI zero-group gate is
    authoritative), branching-flow 116 (one false-positive class + taste calls),
    file-size 16 (fleet heuristic, cohesive files). Plus **all ten "tools
    unavailable" entries named** with a disposition each
    (`buildflow --dry-run --verbose`).
15. **cqrs-lint policy completed** (`docs/cqrs-lint-policy.md`): F021, F026
    (inapplicable — `WithPrefetch` is a reader cache for CURSOR paging, not a
    ScanOption), F031 (helper-built option slices), the time-typed-field
    entries, and two real fixes: a named-return finding in the CDR mapper and
    AGENTS.md's stale go-cqrs-lite version reference.
16. **Five living docs audited** (`docs/followons-2026-10-15.md`,
    `process-prep-2026-10-15.md`, `backup-rotation-opinion.md`,
    `telnyx-live-api.md`, `did-direction-design.md`):
    F56 marked EXECUTED (the enrichment, not the MOS window, supplied cost);
    the scanner ceiling corrected to `scannerMaxLine` and the re-ingest runbook
    now names `refresh`; the **backup doc's rotation advice was WRONG** (the app
    publishes ONE fixed-path snapshot, so there is nothing to rotate) and is
    corrected; the Telnyx direction section now records the `-ai-did` override;
    the DID direction design note was corrected (its "the CDR side is already
    true" claim was disproven live) and archived with a manifest row.
17. **go-appkit + a-h/templ deep dive** (`docs/research/2026-10-25_go-appkit-templ-deep-dive.html`,
    method of the templ-components dive: ground truth at the pin, usage census,
    gap analysis, score, appendix). **F1 was a real defect**: every stack test
    paid appkit's production 5 s drain delay because it built the service with
    `appkit.NewService`; `testkit.NewService` (v0.9.0's test-safe defaults) is
    adopted at all three sites — `cmd/pbx-lens` went **10.5 s → 0.81 s** and
    `TestServeStackEndToEnd` **5.04 s → 0.02 s**.
18. **Flaky scenario tests fixed** (`internal/app/*_test.go`): the 5 s library
    default await timeout reddened the gate under `-race`; raised to 15 s with a
    comment saying the assertion is about eventual convergence, not latency
    (verified: four consecutive `-race` runs green).
19. **docs/research/README.md** updated with the new report and the CHANGELOG
    policy for docs-only sessions decided (no entry).
20. **Living docs updated for the new surfaces**: FEATURES (four analytics rows
    flipped + new gate-helpers row), CHANGELOG (a full 2026-10-25 Added/Changed
    pair), scripts/README (four new rows + verify-all note), DOMAIN_LANGUAGE
    (representative row, direction precedence), gotchas (the `Exists` trap),
    http-stack (the nonce-source warning), AGENTS (one compressed sentence,
    budget held).
21. **TODO_LIST closed its whole Unblocked section** (12 rows deleted as done);
    the BLOCKED rows were re-verified (pixel baseline v4 recorded, prettier row
    narrowed to the fleet-PATH half).
22. **Full browser lane green**: `scripts/verify-all.sh` ran the docs gates,
    the statix tripwire, four probe suites and the 42-PNG sweep — all pass,
    including the print chrome/`break-inside` assertions.
23. **v4 pixel baseline saved and verified** (`~/lens-baselines/v4`, 42 PNGs;
    a fresh dashboard capture diffs clean: 0 alerts, ≤128 px time-text volatile).
24. **`nix build` green** and **`go test ./...` green**; `golangci-lint` 0
    issues; `dprint check` clean; AGENTS at 218/219; all four new gates
    self-test green.
25. **THE DEV SHELL WAS BROKEN AND IS FIXED** (found while chasing the gate):
    the `cqrs-lint` fixed-output derivation's `vendorHash` in `flake.nix` no
    longer matched the vendored tree (a Go toolchain bump changed
    `go mod vendor` output; the old hash had been cached, so nothing noticed
    until the FOD had to rebuild). `nix develop -c true` failed with
    `got: sha256-jTwZn/TZJZ22iSwmLRHG/Hc9lE6L9O6fncmKxkiriNE=`. Re-derived in
    ONE atomic edit (no fake-hash placeholder — the 2026-10-25 "interrupted
    repair left a placeholder" incident happened exactly that way), then
    verified: `nix develop -c true` exit 0, and every devShell tool runs
    (dprint, golangci-lint, cqrs-lint).
26. **`dprint check` green across the whole tree** — two files left unformatted
    by concurrent sessions (`docs/upstream-filings.md`, the 06-51 report) were
    formatted (semantically neutral).

## b) PARTIALLY DONE

1. **BuildFlow gate receipt.** Three runs this session were red, all on
   ENVIRONMENT or CASUALTY grounds, not on a repo defect:
   - run 1: `test-race` (the app-scenario flake — FIXED), `nix-build`
     (vendorHash left as a placeholder by an interrupted concurrent repair — a
     concurrent commit re-landed it, `nix build` green), `nix-hash-fix:diagnose`
     (its own 1 s deadline).
   - runs 2/3: 15-min load **100–106**, 1-min load **160** (concurrent agent
     sessions). Failures were starvation: `nix-build` killed at its 35 s step
     deadline, `govalid-generate` killed after 42 s, `nix-flake-check`
     unrecognized eval failure, `dprint-format:detect` and
     `golangci-lint:detect` failing inside a starved shell. BuildFlow itself
     reported "1 lost to starvation".
     Every casualty was then verified individually and green: `dprint check` 0,
     `golangci-lint run ./...` 0 issues, `cqrs-lint .` 19 findings (all
     policy-documented), `go test ./...` green, `nix build` green,
     `nix develop -c true` green. **No green gate receipt exists for this
     session** — the documented rule (FULL only under 15-min load 40) was not
     met once.
2. **Per-segment spend** renders as a total only; spend-per-minute per segment
   is not rendered (FEATURES marks that row partially functional).
3. **The `?number=` filter** is wired into the list, the pager, the staleness
   fragment and the filter form — but it is not yet documented in README usage
   or in the dashboard's own help text.
4. **Quality rollup coverage**: the three documented signals are aggregated;
   `source markers` (a fourth per-call signal) is deliberately not a rate.
5. **The spend sparkline** has no tooltip/labels (the volume chart does).
6. **docs-counts gate scope**: it verifies the archived/planning/live counts and
   the live-table ↔ file match, but NOT a TODO row count (the TODO list stopped
   hardcoding one, so there is nothing to verify).
7. **poll-gate.sh** is implemented and self-tested but has not been exercised in
   a real FULL-gate receipt (the load policy never opened long enough).
8. **The v4 baseline** is machine-local; the durable-home decision is the
   owner's (answer-sheet Q10).
9. **AGENTS.md** carries the analytics facts only as one compressed sentence
   (budget 219); the detail lives in FEATURES/DOMAIN_LANGUAGE.

## c) NOT STARTED

Everything owner- or calendar-gated in TODO_LIST's `BLOCKED` section:

1. Production re-ingest / correction wave (`refresh -apply`) and the one-time
   read-model rebuild (legacy keyed rows).
2. Host deploys with `aiDids`, the CSP enforce flip, D1–D5 decisions, the PII
   boundary, the coverage-floor call, mobile card layout, the >1 MiB policy, the
   browser 304 check, the pixel-baseline home.
3. Release cut (patch vs minor), stamp-churn doctrine (client-side timestamps),
   the staged upstream filings, the buildflow advisory-vs-fatal decision.
4. Upstream fold-ins (go-appkit#26 is CLOSED and now partially folded by the
   F1 fix; go-etag#2 + ten templ-components issues are OPEN).
5. Sibling-repo work: the corrected backup-rotation wording into `lens.nix`; the
   `call_direction` column in the CDR template (the parser side is ready and
   value-typed, the template side is a sibling task).
6. The 2027-01 Telnyx window (DTO live-verify, MP3 Range, MOS poller).

## d) TOTALLY FUCKED UP (self-inflicted, this session)

1. **I truncated `TODO_LIST.md` to 0 bytes.** A Python edit did
   `open(path,'w')` and then failed to encode an emoji (surrogate pair), so the
   write emptied the file. Recovered from `git show HEAD:TODO_LIST.md` and
   re-applied the two intended edits; nothing was lost, but it was a real
   destructive near-miss and it cost ~10 minutes.
2. **I designed the collapse around `reader.Exists` first** — which silently
   returns false for map-backed rows in sqliteengine (sets live in another
   table) while `Get` finds the row. The wrong version "passed" the
   no-such-sibling path; a scratch debug test exposed it, and the fix is now a
   documented gotcha.
3. **My first tie-group test hung the suite for 600 s** because I assumed the
   keyset could page a timestamp group wider than a batch. The right answer was
   the probe row + an explicit stall error, not a bigger limit.
4. **I wrote a code comment claiming a proof I had not run** ("prove it stays
   honest: seed a broken path…") and removed the claim only on re-reading.
5. **I edited `internal/calls/viewkey.go` after reading it with `cat`** instead
   of the View tool; the edit tool correctly refused. Process violation.
6. **I did not run `-race` locally** before believing the tests were green; the
   gate's race lane found the flake in someone else's new scenario tests.
7. **I set `GOEXPERIMENT=jsonv2` after `timeout` in several commands**
   (`timeout 300 env …` vs `GOEXPERIMENT=… timeout`), which is harmless but sloppy.
8. **I moved the `--baseline` parse below the docs gates first**, then moved it
   to the very top on reflection — the first placement was still "fail late".
9. **I ran the dev gate twice before reading its failure detail**, burning ~2
   minutes of a loaded machine to learn what `--dry-run --verbose` would have
   told me.

## e) WHAT WE SHOULD IMPROVE

1. **Never truncate-then-encode.** Build the string, encode/serialize it, then
   open the file — or write to a temp file and rename.
2. **Verify the library API before designing around it.** Three of this
   session's detours (`Exists`, `WithPrefetch` as a ScanOption,
   `WithAwaitTimeout`) were assumption-first.
3. **`go test -race ./...` is part of "green"**, not an extra: the gate's race
   lane is the one that found the flake.
4. **Read symptoms literally before re-running.** BuildFlow's own
   `--dry-run --verbose` / `explain <step>` names the failing cause in one call.
5. **The pixel baseline must be refreshed in the same change as the visual
   change** — otherwise the next sweep reports intended changes as alerts.
6. **Timeouts in integration-style tests belong to the harness, not to luck**: a
   5 s poll budget on a loaded box is a flake generator.
7. **The docs gates should run in CI too** (they are subsecond); today they run
   locally in `verify-all.sh` + pre-commit.
8. **Analytics denominators deserve a shared helper**: "N of M priced", "1
   transcript", "mixed" are three separate ad-hoc formatters today.

## f) Up to 50 things we should get done next

Architecture / correctness

1. Replace the per-call sibling lookup in the list/dashboard with a read-model
   projection of "call is represented" if profiling shows the extra Get hurts
   (measure first: 50 rows/page today).
2. Add a `rebuild` subcommand (`ProjectionHost.Reset` + replay) so the legacy
   keyed rows can be retired; dry-run classification first.
3. Decide whether the export should offer a `--collapse` mode (per-source stays
   the default).
4. Make the CDR template carry `call_direction` in the sibling repo (the parser
   already reads it by value) and de-risk the heuristic.
5. Add a `?number=` chip to the active-filter area so the drill-through is
   visibly reversible.
6. Extend the number filter to accept a substring/prefix match (today exact).
7. Add per-segment cost-per-minute and a segment drill-through that preserves
   the dashboard window.
8. Surface `source markers` as a fourth quality rate.
9. Give the spend sparkline a tooltip + per-day labels.
10. Add an "unpriced calls" count next to spend so the denominator is visible
    at a glance.

Gates / tooling
11. ~~Wire `check-docs-counts.sh` + `check-doc-links.sh` into CI.~~ done — both gates run in CI (.github/workflows/ci.yml:256-257)
12. ~~Add a CI job that runs the four `--self-test` modes (`poll-gate`,~~ done — five --self-test red-path proofs run in CI (ci.yml:239-249)
~~`check-js-lint`, `check-docs-counts`, `check-doc-links`).~~
13. Add `shellcheck` to the pre-commit chain (it is in the devShell already and
reported zero findings when trialled).
14. ~~Add `oxlint` to CI (it is now in the devShell and the config is committed).~~ done — oxlint installed and check-js-lint gate run in CI (ci.yml:222-226)
15. Teach `scripts/verify-all.sh` a `--no-browser` mode so CI can run the
non-browser half without chromium.
16. ~~Make `poll-gate.sh` the documented receipt entry point in~~ done — docs/buildflow-dev-gate.md:354 "Run the gate through scripts/poll-gate.sh, never bare"
~~docs/buildflow-dev-gate.md and scripts/README (done in README; gate doc~~
~~still describes the old loop).~~
17. Add a `--strict-await` lint idea: fail tests whose scenario awaits exceed a
budget.
18. Grow the docs-count gate to verify the FEATURES row counts per status
(green/yellow/grey) — they drift silently too.

Product / analytics
19. Ship the next analytics slice: anomaly/baseline detection (volume, MOS,
failure rate) — needs the trend baselines that now exist.
20. Saved views: encode the filter set in the URL (mostly done) + a small
"recent views" cookie/localStorage layer, no server write path.
21. A per-DID detail page (segment → its own trend card) instead of a filtered
list.
22. Quality rollup: per-number quality (which DIDs produce the bad calls).
23. Cost intelligence: currency-aware totals with a daily FX note if Telnyx ever
mixes currencies.
24. Demand-gated outcome analytics (documented in ROADMAP theme 0) — needs a
business question first.

Docs / memory
25. Add a README "Analytics" section describing the three dashboard cards and
the `?number=` drill-through.
26. Document the analytics denominators in DOMAIN_LANGUAGE (a short
"denominators" subsection).
27. Record the `assertFirstTryRevalidation` contract in docs/http-stack.md.
28. Add the v4 baseline generation to scripts/README's pixel-diff doctrine.
29. Extract the CDR direction rules from the `AGENTS`-compressed sentence into a
dedicated docs section (they now live in code comments + DOMAIN_LANGUAGE).
30. Add a `docs/status/README.md` reminder that the count gate enforces it.

Owner-gated (still blocked, in priority order)
31. Production `refresh -apply` correction wave.
32. Read-model rebuild + retire the legacy key fallback.
33. Deploy both hosts with `aiDids`; bundle the CSP enforce flip.
34. Answer D1–D5 (date/number vocabulary and the five UI calls).
35. Decide the PII boundary (read-side recommended).
36. Decide the coverage floor (70 accepted vs chase 80).
37. Pick the pixel-baseline home (v4 is machine-local).
38. Decide the >1 MiB transcript policy.
39. Cut the next release (patch vs minor).
40. Approve the staged upstream filings.

Upstream / sibling
41. Fold go-etag#2 when it lands; re-run the filing ledger quarterly.
42. Fold templ-components #48/#49/#51/#55/#59/#60/#61/#63 as they close (the
watchlist in docs/templ-components-adoption.md is the re-check list).
43. Land the corrected backup-rotation wording in `lens.nix` timer comments.
44. Land `call_direction` in the sibling CDR template.
45. Re-run the templ-components audit delta after the next bump (~94 predicted).

Verification depth
46. Run the browser 304/htmx DOM-churn check the owner still owes (server-side
proof exists).
47. Add a golden for the dashboard's populated-spend state (today only the empty
state is golden).
48. Add a benchmark for the segment aggregation over a large window.
49. Add a test that the `?number=` drill-through survives a page walk.
50. Re-measure `BenchmarkConcurrentPoll` after the analytics cards (they add
render work to the polled fragment).

## g) Questions only the owner can answer

1. **Which analytics surface is the product's center of gravity now?** The three
   slices made the dashboard a real analytics view; the next investment is
   either anomaly/baseline detection (needs history + baselines) or saved
   views/drill-downs (pure UI). Which one would you actually use next week?
2. **Should Spend become a headline stat card?** The headline grid is a fixed
   four-up (calls, today, AI share, avg duration) and Spend lives in the aside.
   Promoting it means restructuring the grid to five (or replacing one).
3. **Where does the durable pixel baseline live?** v4 is machine-local
   (`~/lens-baselines/v4`); the alternatives are committing the 42 PNGs (repo
   weight ~10 MB) or pinning the baseline to a CI artifact. This is the standing
   answer-sheet Q10, and it now blocks any CI pixel gate.

---

### Verification receipts (this session, in order)

```console
go test ./... -count=1                      # all 14 packages ok (cmd 0.78s after the F1 fix)
go test -race ./internal/app/ -count=1 ×4   # green after the await-timeout fix
golangci-lint run ./...                     # 0 issues
dprint check                                # clean
cqrs-lint .                                 # 19 findings, every one a documented decision
bash scripts/verify-all.sh                  # docs gates + statix tripwire + 4 probes + 42 PNGs, all green
node scripts/screenshot-probe.js --only dashboard --diff ~/lens-baselines/v4   # 0 alerts
bash scripts/check-docs-counts.sh --self-test   # green (tree + seeded drift + missing claim)
bash scripts/check-doc-links.sh --self-test     # green (tree + seeded link + fence/URL skips)
bash scripts/check-js-lint.sh --self-test       # green (undefined ref red, clean file green)
bash scripts/poll-gate.sh --self-test           # green (over-policy / ready / unreadable)
bash scripts/check-agents-budget.sh AGENTS.md   # 218/219
nix build                                       # green, result/bin/pbx-lens 20.7 MB
```

Late additions after the first draft, both verified: the **cqrs-lint vendorHash
repair** (devShell was broken: `nix develop -c true` failed; fixed and proven)
and the **whole-tree dprint check** (two concurrent-session files formatted).

Final gate verdict: **no green BuildFlow receipt** — three runs were red on
environment (15-min load 100–106, 1-min 160) and on steps since fixed
(`test-race` flake, cqrs-lint vendorHash); each casualty was verified green
individually. That is the honest state, not a green claim.
