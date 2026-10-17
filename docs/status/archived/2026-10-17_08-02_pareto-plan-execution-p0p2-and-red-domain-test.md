<!-- docs/status/2026-10-17_08-02_pareto-plan-execution-p0p2-and-red-domain-test.md
     Status report for the 2026-10-17 morning session: execution of the
     05-07 SUPERB Pareto plan (P0-T3/T4, P1 trust bundle, P2-T14..T20-partial).
     Ends with ONE red test committed locally, 17 unpushed commits. -->

# Pareto Plan Execution — P0/P1 complete, P2 partial, one red domain test

Date: 2026-10-17 08:02 CEST · Baseline: the 05-07 Pareto plan (27 tasks) over
the post-archive TODO_LIST · Machine: load policy blocked the full gate all
session (1-min 25-65, 15-min 26-85; policy needs 1<15 AND 15<40).

## 0. The one-paragraph truth

Eleven of the plan's tasks are fully done and verified (the whole P1 trust
bundle minus the owner-gated rebuild, P0-T3/T4, and P2-T14..T19), the CSP
sweep caught and fixed two REAL nonce holes, the a11y sweep caught and fixed
a real heading-order violation, and styles.css determinism is now proven and
CI-guarded. The session ended mid-task P2-T20 (coverage floor): a
newly-written domain test (`TestRetireDeciderPaths` in internal/calls) is
RED and was auto-committed by the daemon — local main is 17 commits ahead of
origin (unpushed, so remote CI is still green) and MUST NOT be pushed until
that test is fixed.

## a) FULLY DONE (verified this session)

| Task                                    | What landed                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Verification                                                                                              |
| --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| P0-T4 planning-doc closure              | EXECUTED headers on all four executed plans (00-09, 01-32, 19-05, 03-43) pointing at archived execution reports + CHANGELOG                                                                                                                                                                                                                                                                                                                                                              | dprint clean; headers render                                                                              |
| P0-T3 decision-pack staging             | `docs/decisions/2026-10-17_owner-answer-sheet.md` (7 questions: D1-D5, PII boundary, mobile card-list — options + recommendations + ANSWER slots); linked from the 3 TODO_LIST rows                                                                                                                                                                                                                                                                                                      | File live; owner answers pending by design                                                                |
| P1-T5 security headers                  | `httputil.SecurityHeaders` in the extras chain (Referrer-Policy: no-referrer, nosniff, X-Robots-Tag: noindex); appkit default XFO:DENY survives; http-stack.md chain now 5 items; AGENTS.md chain line updated (219-line budget held)                                                                                                                                                                                                                                                    | `TestSecurityHeadersContract` (page, 304 replay, app.css, export.csv, export.jsonl) + wiring counts 4/5/5 |
| P1-T6 X-Robots-Tag                      | Rode the same middleware via `Custom` — covers non-HTML endpoints uniformly (the row's whole point)                                                                                                                                                                                                                                                                                                                                                                                      | Same contract test asserts it on exports                                                                  |
| P1-T7 CSP report-only sweep             | NEW instrument `scripts/csp-sweep.js` (nonce + report-only, 6 surfaces, securitypolicyviolation events); **found 2 real holes**: calls-Table row-href script and CopyButton scripts rendered un-nonced; fixed by threading the nonce (`tableProps`, `listPage`, `nativeIDCopy`, `copyableValue`, `copyDetail` chains); serve.go comment corrected (report-only alone is meaningless); verdict recorded in http-stack.md                                                                  | Re-run: ZERO violations across 6 surfaces; full browser lane (`verify-all.sh`) green after the fix        |
| P1-T8 buildflow tooling triage          | doctor's 32 fleet-wide ✗ marks dispositioned; KEY finding: eslint/markdownlint/prettier are NOT registered for this repo (dead skip config would warn); shellcheck → devShell (0 findings on scripts/); `.buildflow.yml` skips lychee (fleet GITHUB_TOKEN policy pending) + nix-build-verify (12/12 env failures; `nix build` stays the manual receipt step); nix-hash-fix 82% accepted as environmental; markdownlint trial-run → deliberate non-adoption (MD013/MD060 vs house tables) | `buildflow --dry-run`: 37 ok / 1 warn / 0 fail, no dead-config warnings; flake check green                |
| P1-T9 recordings lookback               | Decision: the 300-recording window is a CONTRACT, not a guarantee (silent permanent RecordingRef miss documented); comment upgraded in `telnyx.go`; bound paragraph in docs/telnyx-live-api.md; remedy = one-off raised-constant pass if it ever bites                                                                                                                                                                                                                                   | Comment/doc only — behavior unchanged by design                                                           |
| P1-T10 styles freshness guard           | Determinism PROVEN: nixpkgs tailwindcss 4.3.3 byte-identical (rerun + committed copy + npm `@tailwindcss/cli@4.3.3`); NEW CI job `styles-freshness` (scratch npm install → regen → diff, error names `nix run .#update-styles`); css-gate error messages repointed                                                                                                                                                                                                                       | 3-way cmp byte-identical; ci.yml job added                                                                |
| P1-T11 cut-release.sh                   | `scripts/cut-release.sh`: clean-tree + load-policy + pushed-HEAD quiesce (BuildFlow#55 guard), CHANGELOG sweep, nix build, full gate, verify-all, remote-CI-green check (the tag-red lesson), annotated tag + push, `go list -m` proxy check                                                                                                                                                                                                                                             | bash -n + shellcheck clean; guard rails proven (no-arg fail, dirty-tree fail)                             |
| P1-T13 ops visibility                   | README alerting recipe: jq on `last_pass_age_seconds` (with the -1-never-ran guard) + host-side `systemctl list-timers`/`journalctl` for the purge timer; the /ops card's in-process facts (window, would-retire) verified already present; host-side boundary documented                                                                                                                                                                                                                | README section; wire shape checked against `SnapshotAny`                                                  |
| P2-T14 FTS quality pass                 | `TestMultiWordQuerySemantics`: AND semantics, order-free, case-folding, absent-term, words+number mix with digit-shadow dedupe — all clean, no ranking surprises (bm25 double-mention ranking was already pinned)                                                                                                                                                                                                                                                                        | Test green in internal/search                                                                             |
| P2-T15 export lane tests                | Consumer-side parsing (real `csv.Reader` + line-by-line JSON decode) of STALLED streams: header integrity + marker record well-formed; `BenchmarkExportStream10k` (real sqlite, 10k rows: ~268 ms/op, ~24.8 MB allocs — streaming evidence)                                                                                                                                                                                                                                              | Both tests + 1x benchmark run green                                                                       |
| P2-T16 recording negative paths         | fakeTelnyx gained metadata-500 + signed-URL-403; `TestRecordingNegativePathsAnswerHonestly`: honest 502 bodies, no upstream-URL leak, no audio content-type; throttle decision: defaults stay hardcoded (2/s, burst 60, loopback exempt — flags only if ops asks)                                                                                                                                                                                                                        | Test green                                                                                                |
| P2-T17 verify-all --fast + probe polish | `verify-all.sh --fast` (contracts-only); screenshot-probe: discovery SKIPPED when no detail-*/print surface selected + KNOWN_SURFACES drift guard                                                                                                                                                                                                                                                                                                                                        | `--only dashboard` run: 2 PNGs, no discovery scrapes                                                      |
| P2-T18 pixel-diff practice              | Baseline corpus saved (41 PNGs → ~/lens-baselines/v1); zero-tolerance self-diff measured the noise: ≤81 px time-text churn at content-dependent positions → implemented a **volatile-px budget (128)** instead of brittle geometric masks, tightened default tolerance 0.5% → 0.05%; aged-baseline alerts documented as the refresh signal; `LENS_DIFF_BASELINE` hook in verify-all                                                                                                      | Same-session diff: 0 alerts, all absorbed; doctrine in scripts/README                                     |

## b) PARTIALLY DONE

~~1. **P2-T19 /ops hardening** — done: /ops joined the e2e lane (fifth~~ done — the close-out leg: full browser lane green twice post-h2 + baseline v2 diff-proven (§a3/§a4); the stale-verdict probe keeps its TODO row
surface), a11y contract leg added to verify-ui-contract.js, which caught
a REAL h1→h3 skip; fixed by `TitleTag: display.HeadingTagH2` on all 12
page-level Cards; goldens regenerated (-update, 4 files); browser probe
ALL PASS (headings 1,2,2,2). NOT done: the stale-verdict _browser_
probe — the probe stack has no poller-wired serve composition (serve
without ingest sources 503s /api/ops), and the stale markup is already
pinned server-side (`TestOpsPageStaleVerdictAlerts`); documented, not
built.
~~2. **P2-T20 coverage floor** — cmd wiring tests landed~~ done — floor 70 landed (§a5), measured 73.0; the red domain test fixed (§a1)
(`indexsession_test.go`: openIndexSession/finalizeIngest/gateAndFinalize

- age gate, printRejected, requireDB/requireOneArg happy paths, smoke +
  backupCmd flows): cmd 9.2% → **27.6%**, total 70.4% → **71.9%**. The
  follow-up domain test (`internal/calls/commands_test.go`:
  constructors + Excerpt + Retire decider) is **RED** — see (d). The CI
  floor is still 50 (raising it is the unfinished half of the task).
  ~~3. **Full browser lane after the h2 change** — verify-ui-contract +~~ done — close-out §a3 (all four suites green twice) and §a4 (baseline v2 saved)
  internal/web tests ran green after the TitleTag fix, but the FULL
  verify-all (style contract, staleness-live, 41-PNG sweep) has not been
  re-run since; also the pixel baseline v1 now predates the h2 markup
  change and needs a re-save.

## c) NOT STARTED

~~- P0-T2 full buildflow gate receipt — load policy blocked it all session~~ routed → the gate-receipt TODO row (load policy)
(1-min never under 15); `cut-release.sh` and the receipt procedure wait
on a quiet machine.
~~- P1-T12 BuildFlow#56 binary rebuild — owner-gated (cross-repo build +~~ routed → answer-sheet §9 (owner)
install from ~/projects/BuildFlow; doctor currently warns the binary
trails that checkout by some commits).
~~- P2-T21 timeout-test hardening (audit remaining `time.Sleep`, cancellation~~ done — T21 audit closed (close-out §a7)
pattern).
~~- P2-T22 docs bundle (DOMAIN_LANGUAGE MOS/cost rows, Telnyx 2027-01~~ done — T22 (close-out §a8)
quarterly checklist file).
~~- P2-T23 UX bundle (search hint example query, RelativeTime day-precision~~ done — T23 (close-out §a9)
tooltip, ops print page-breaks).
~~- P2-T24 hygiene bundle (pre-commit AGENTS.md ≤219 check, `ingest-cdr~~ done — T24 (close-out §a10)
  -dry-run`).
~~- P2-T25 verification hygiene (dark-theme eyeball over the 41-PNG corpus,~~ done — T25 (close-out §a11)
true clean-clone ensureCodegen rehearsal).
~~- P2-T26 R27.x advisory triage (art-dupl/branching/file-size verdicts).~~ done — T26 (close-out §a12)
~~- P3-T27 watch register pass (external gate states, re-rank).~~ done — T27 (close-out §a13)
~~- Close-out ledger: TODO_LIST rows not yet closed (T5-T19 work is done but~~ done — close-out §a14 (rows closed, CHANGELOG written, adoption map, lint re-run)
the rows still say open), CHANGELOG Unreleased entry not written,
docs/templ-components-adoption.md does not yet record the TitleTag
adoption, dprint/cqrs-lint/golangci-lint not re-run over the late edits.

## d) TOTALLY FUCKED UP (own failures, no gloss)

1. **A red test is COMMITTED and local main is 17 commits unpushed.**
   `internal/calls/commands_test.go` `TestRetireDeciderPaths` fails with
   `call.retire.before_observed` — my retire leg folded the Record events
   and assumed `Observed=true`, but the folded state doesn't satisfy the
   decider the way I built it (stream-id construction and/or which event
   Record emits at v0 — undiagnosed; the existing domain_test helpers show
   the working pattern I didn't re-check). The daemon auto-committed it.
   Remote is safe (nothing pushed), but the tree is red and the fix is the
   mandatory first action. Lesson re-learned: run the new test BEFORE
   writing the next one, and never let a session end with an unverified
   test in the tree.
2. **sed surgery on templ struct literals broke syntax twice** (the
   TitleTag insertion landed outside the braces on the first pattern, then
   my repair pattern was also wrong before the third worked). Caught each
   time by grep before any build — but this is exactly the "edit tools, not
   sed" rule; multiedit with exact matches would have been slower and
   correct the first time.
3. **Two wrong-first-tries in test fixtures**: the e2e /ops assertion
   expected the Retention card in the UNWIRED state (it renders only when
   wired); the first export-consumer file had a gocognit violation and a
   fake `testing.T` inside the benchmark (rewritten: separate tests +
   cli.OpenSystem wiring). Both caught by tests/lint before landing — but
   they were avoidable by reading the branch structure first.
4. **npx blind alleys**: `npx tailwindcss@4` (v4 moved the CLI to
   `@tailwindcss/cli`), then a missing `tailwindcss` resolution for the CSS
   import (fixed with the scratch node_modules symlink) — three attempts
   for a one-command proof.
5. **csp-sweep console attribution is coarse**: console violations
   accumulate under a "(console)" label instead of per-surface (the
   securitypolicyviolation events are per-surface and precise). Works, but
   the first sweep's output was harder to read than it should have been.

## e) WHAT WE SHOULD IMPROVE

- **Never end a step with an unrun test** — the red domain test existed for
  exactly one edit-cycle, but the daemon commits faster than my discipline.
  Run `go test ./...` (or at least the touched package) between edits when
  the tree is being auto-committed.
- **Baseline hygiene is now load-bearing**: the pixel baseline must be
  re-saved after ANY markup change (the h2 fix aged it within the hour);
  the README doctrine should say "re-save after any golden -update".
- **Gate waiting**: the load policy blocked P0-T2 all session; a quiet-hour
  run (or `--max-concurrency`) should be scheduled rather than retried
  opportunistically.
- **grep-before-sed**: for templ/Go struct literals, exact-match editing
  beats pattern surgery — twice burned.
- **The coverage ceiling is structural**: cmd's remaining gap is
  os.Exit-guarded flag paths (need exec-based tests or a refactor);
  internal/calls' gap is the projection fold (integration-covered).
  Getting to a truthful 80 needs decisions, not just more tests.
- **Adoption-map lag**: the TitleTag adoption (and the CSP nonce-threading
  pattern) belong in docs/templ-components-adoption.md the moment they
  land, not "at close-out".

## f) NEXT — ranked (49 items)

**P0 — unblock the tree (do first, in order)**

~~1. Fix `TestRetireDeciderPaths` (mirror domain_test's working stream-id/~~ done — close-out §a1 (folded the full history; domain semantics correct)
record-fold pattern; or split the Retire leg out until diagnosed).
~~2. `GOEXPERIMENT=jsonv2 go test ./...` full-suite green.~~ done — close-out §a2 (14/14 packages)
~~3. Re-run the full browser lane post-h2 (`bash scripts/verify-all.sh`).~~ done — close-out §a3
~~4. Re-save the pixel baseline (`--save-baseline ~/lens-baselines/v2`) —~~ done — close-out §a4 (v2, 0 alerts)
v1 predates the h2 markup change.
~~5. dprint the session's late edits (templ files, scripts, md).~~ done — close-out §a6 (formatted; scoped lesson recorded in §d5)
~~6. `nix develop -c golangci-lint run ./...` (clear the stale-LSP warnings~~ done — close-out §a6 (0 issues repo-wide)
for real: css_gate golines, export_consumer gocognit).
~~7. `nix develop -c cqrs-lint .` per the findings policy.~~ done — close-out §a6 (F009 reconfirmed as the documented policy)
~~8. Push the 17-commit tail once green; watch remote CI to green.~~ done — close-out §a16 (pushed, remote CI success)
~~9. Update TODO_LIST: close T4/T5/T6/T7/T8/T9/T10/T11/T13/T14/T15/T16/T17/~~ done — close-out §a14 (18 rows closed with receipts)
T18 rows (move to CHANGELOG), annotate the T19 partial, keep T20 open.
~~10. CHANGELOG Unreleased entry for the whole session (headers, CSP fixes,~~ done — close-out §a14
a11y h2, nonce threading, instruments, guard, triage).
~~11. Record TitleTag + nonce-threading in docs/templ-components-adoption.md~~ done — close-out §a14
(and note Card's h3 default in the version-bump watchlist context).
~~12. Finish P2-T20: decide the CI floor (see question 1), edit ci.yml~~ done — close-out §a5 (floor 70 as default; answer sheet §8)
MIN_COVERAGE, watch the run.
~~13. P0-T2: full buildflow dev gate on the settled tree when load < 15/40;~~ routed → the gate-receipt TODO row (BuildFlow#56 CLOSED upstream; still load-gated — 1-min 92 at this pass)
bank the receipt in docs/buildflow-dev-gate.md; close/keep the
BuildFlow#56 workaround sentence accordingly.

**P1 — remaining trust/quality work**
~~14. P2-T21: audit remaining `time.Sleep` in tests; convert one candidate to~~ done — close-out §a7 (exactly two await-pattern sites; row closed)
the cancellation-driven pattern; generalize the helper.
~~15. P2-T22: DOMAIN_LANGUAGE MOS/cost rows; Telnyx quarterly checklist file~~ done — close-out §a8
(date + commands, 2027-01 window).
~~16. P2-T23: search hint example query; RelativeTime day-precision tooltip;~~ done — close-out §a9
ops print page-breaks (with probe capture).
~~17. P2-T24: pre-commit AGENTS.md ≤219 wc check; `ingest-cdr -dry-run` flag~~ done — close-out §a10

- test.

~~18. P2-T25: dark-theme eyeball over the fresh corpus; true clean-clone~~ done — close-out §a11
ensureCodegen rehearsal in a scratch worktree.
~~19. P2-T26: R27.x triage (art-dupl + branching-flow verdicts, file-size~~ done — close-out §a12
accept-or-split for server.go/search.go/telnyx_test.go, vulnix note).
~~20. P3-T27: watch-register pass (verify each Blocked row's external gate;~~ done — close-out §a13
re-rank TODO_LIST after P0/P1 land).
~~21. Consider an upstream filing candidate: Card's h3 default makes~~ routed → the TODO upstream-filings row (verify-before-filing first; this pass)
heading-order violations the natural outcome (we now pass TitleTag in
12 call sites) — verify-before-filing first; the library documents
TitleTag, so this is a DX suggestion, not a bug.
~~22. Improve csp-sweep: attribute console violations to the current surface~~ routed → the TODO probe-polish row (this pass)
(reset the console buffer per navigation like the DOM events).
~~23. cut-release.sh: dry-run the full body on a quiet machine (it has only~~ routed → the TODO release-rehearsals row (this pass)
had its guard rails exercised, not the gate+probe sequence).
~~24. README: the new `--fast`/LENS_DIFF_BASELINE/csp-sweep entries exist in~~ Won't implement — audience split is deliberate: scripts/README is the dev surface, README stays user-facing
scripts/README — mirror the one-liner usage into README.md's dev
section if the owner wants them surfaced.

**Watch / owner-gated (scheduled, not fake-scheduled)**
~~25. P1-T12 BuildFlow#56 rebuild on approval → then re-run the full gate for~~ routed → answer-sheet §9 + the gate-receipt row
the literal full receipt; close the ledger note.
~~26. Owner answer sheet (D1-D5, PII, mobile) → record outcomes in~~ routed → the owner rows (outcomes recorded on arrival per the F7 contract)
TODO_LIST + AGENTS.md per the F7 contract.
~~27. Fold-ins when upstream lands: templ-components #48/#49/#50/#51/#55,~~ routed → the TODO fold-in rows (next check 2027-01)
go-etag#2, go-appkit#26, BuildFlow#55.
~~28. 2027-01 Telnyx live-verify window + MOS poller execution (design done).~~ routed → the 2027-01 window row (checklist file ready)
~~29. Host deploys (`aiDids` options; the NixOS module is ready).~~ routed → the TODO deploy row (owner)
~~30. Production re-ingest for ContentHash v2 (owner path).~~ routed → the TODO re-ingest row (-dry-run rehearsal enabled)
~~31. Schema-v2 length-prefixed ContentHash encoding (frozen at v1 until the~~ routed → the TODO schema-v2 row
gate).
~~32. Dependabot weekly run: confirm bumps for the newly-direct deps.~~ routed → the TODO dependabot row
~~33. dependabot-core#16538 maintainer answers (owner).~~ routed → the TODO watch row (owner)
~~34. Browser-level 304 check (owner manual; server side pinned).~~ routed → the TODO browser-304 row (owner manual check)
~~35. >1MiB transcript policy (owner decision, default documented).~~ routed → the TODO owner row (option-1 default stands)
~~36. Sibling repo backup-rotation fold into lens.nix comments.~~ routed → the TODO sibling row
~~37. /tmp cleanup decision (48G tmpfs was 100% at session start, recovered~~ Won't implement — machine-local scratch, not repo work
to 54% by external cleanup — the cross-session logs are not ours to
trash unilaterally).
~~38. lychee fleet GITHUB_TOKEN policy (BuildFlow fleet question) — revisit~~ routed → the tooling-triage row (fleet policy)
the skip_steps entry when it exists.
~~39. vulnix re-triage (rides R27.x / next bump cycle; govulncheck is the~~ routed → ROADMAP (rides the next flake bump)
trusted CVE gate).
~~40. markdownlint: revisit only if the fleet adopts a curated config that~~ Won't implement — deliberate non-adoption recorded by P1-T8
tolerates wide tables.
~~41. Coverage: the calls-projection fold + cmd exec-level flag tests are the~~ routed → the coverage owner row (answer sheet §8)
path from 71.9% toward 80 — decide whether that is worth a dedicated
pass (see question 1).
~~42. serve.go: the CSP flip (enforcing nonce mode) waits on the next host~~ routed → answer-sheet §11 (CSP enforce-mode; staged this pass)
deploy — bundle it with the `aiDids` deploy.
~~43. Stale-verdict browser probe: revisit ONLY if a poller-wired serve~~ routed → the TODO stale-verdict probe row
composition ever exists in the probe stack (otherwise it tests nothing
a golden doesn't).
~~44. golden-volatile scrubber: `Updated HH:MM:SS` replacement hardcodes the~~ routed → the TODO probe-polish row
stamp format — verify it still catches after any footer copy change.
~~45. cut-release.sh `go list -m` sleep-10 proxy lag: make it a retry loop if~~ routed → the TODO release-rehearsals row
the first check ever flakes.
~~46. .buildflow.yml skip_steps: re-audit if BuildFlow registers~~ routed → the tooling-triage row (re-audit trigger)
markdownlint/prettier/eslint providers for this repo shape (the
disposition rationale says why they're excluded today).
~~47. scripts/README: two "Codegen bootstrap" paragraphs tell the same story~~ routed → the TODO probe-polish row (docs merge)
twice (M4/F14 + hash-primary) — merge them next docs touch.
~~48. AGENTS.md 219-line budget: held this session; the next addition needs~~ done — budget enforced (check-agents-budget.sh + .githooks/pre-commit); the extraction pre-plan is a TODO row
a compensating trim (the pre-commit check in P2-T24 would automate
this).
~~49. Clean up scratch dirs from this session (/home/lars/tmp/pd-run*,~~ Won't implement — machine-local scratch, not repo work
csp-sweep-run*, tw-*, shots-fast) once the receipts above are banked.

## g) Questions for the owner (cannot self-answer)

~~1. **CI coverage floor**: total is 71.9% after the cmd wiring tests. Do I~~ done — floor 70 implemented as default (answer sheet §8; close-out §a15)
pin MIN_COVERAGE=70 now (truthful, 1.9pt headroom), hold at 50 until the
80 push is real, or invest the dedicated pass (projection-fold +
exec-level cmd tests) to legitimately reach the fleet's 80 gate?
~~2. **BuildFlow#56**: may I build + reinstall the buildflow binary from~~ routed → answer-sheet §9 (owner bookkeeping; the upstream fix landed)
~~/projects/BuildFlow on this machine (clean checkout, `nix build . &&
   nix run .#reinstall`)? It gates the literal full-gate receipt and the
stale-binary advisory.
~~3. **Pixel-baseline strategy**: keep baselines machine-local~~ routed → answer-sheet §10 (owner; v2 saved machine-local per the recommendation)
(~~/lens-baselines/<gen>, re-save after any markup change — my
recommendation, zero repo weight) or commit a baseline generation into
the repo (reviewable drift, ~41 PNGs per generation)?

---

_Verified state at writing: local main 02ee134, 17 ahead of origin/main
(UNPUSHED, remote green), working tree clean, internal/calls RED
(TestRetireDeciderPaths), everything else green as itemized above._

---

## Resolution addendum (same day, ~10:30)

The red `TestRetireDeciderPaths` is FIXED: the test folded only the retire
event into an empty state, so the idempotent-re-retire leg saw
`Observed=false`; the domain semantics were correct. Fixed by folding the
full history (commands_test.go). Close-out landed the same morning:
TODO_LIST swept (18 rows closed with receipts, watch table re-verified —
go-appkit#26 and BuildFlow#55/#56 CLOSED upstream), CHANGELOG Unreleased
written, the adoption map gained the TitleTag + nonce rows, the three §g
questions got answer-sheet slots (§8-10; floor 70 implemented as default),
the gate receipt stayed load-blocked (1-min load never dropped under 15;
peak 112), and the tail pushed with remote CI watched to green. Pixel
baseline v2 saved and diff-proven (0 alerts / 41 PNGs).
