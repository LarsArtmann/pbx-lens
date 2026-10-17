<!-- docs/status/2026-10-17_10-28_pareto-closeout-red-test-fix-and-tail-tasks-status.md
     Point-in-time snapshot. Written on explicit .md-path demand (HTML is the
     status-report skill's canonical format; the dispatch prompt named .md). -->

# Status — Pareto close-out: red-test fix, T21–T27 tail, ledger, push (2026-10-17 10:28)

**Scope of this report:** the session leg from the 08:02 report to now —
fixing the red domain test, executing the remaining Pareto tasks T21–T27,
the close-out ledger, the lint/format sweep, and the push of the whole
commit tail (30 commits) with remote CI watched to green.

**Headline state:** local main `d78291a` == origin/main `d78291a`, working
tree clean (daemon keeps committing), remote CI **success**. Test suite
14/14 packages green, golangci-lint 0 issues, all browser probes pass.
The only open work item from the plan is the full-gate receipt, which is
**load-blocked** (policy: 1-min < 15; today's 1-min load ranged 25→112,
never under 15).

---

## a) FULLY DONE

1. **Red `TestRetireDeciderPaths` fixed** (the session's first blocker).
   Root cause was in the TEST, not the domain: `foldAll` folds events into
   an empty `CallState{}`, so folding only the `call.retired` event made
   the idempotent-re-retire leg see `Observed=false` and hit the
   `call.retire.before_observed` rejection. The domain decider semantics
   are correct. Fixed by folding the complete history
   (`internal/calls/commands_test.go`; also simplified to use `mustSID`
   and dropped the now-unused `id` import).
2. **Full test suite green:** `GOEXPERIMENT=jsonv2 go test ./...` → 14/14
   packages ok (re-verified after every edit batch).
3. **Full browser lane green twice:** all four suites
   (ui-contract / style-contract / staleness-live / screenshot 41-PNG)
   ALL PASS; `ops headings: 1,2,2,2` confirms the h2 fix live in a real
   browser; re-ran `--fast` after the T23 markup changes — all pass.
4. **Pixel baseline v2 saved and diff-proven:**
   `/home/lars/lens-baselines/v2` (41 PNGs, post-h2-change), compared
   against a fresh capture → **0 alerts**, volatile budget absorbing all
   churn. v1 (pre-h2) is now retired.
5. **CI coverage floor raised 50→70** in `.github/workflows/ci.yml` with
   rationale comment; measured total is **73.0%** (cmd 9.2%→27.6% from the
   prior leg's wiring tests). The coverage job passed in CI.
6. **Lint/format sweep to zero:** golangci-lint **0 issues repo-wide**
   (cleared wsl_v5 ×2, golines ×4, usestdlibvars ×1 — including two
   stale LSP-only warnings that the real lint run disproved); dprint
   formatted; cqrs-lint's only finding (F009 poller) re-confirmed as a
   documented decision in `docs/cqrs-lint-policy.md`.
7. **P2-T21 timeout-test audit closed:** exactly two `time.Sleep` sites in
   tests (`internal/app/app_test.go:74`, `internal/cli/ingest_test.go:52`)
   — both are deadline-bounded convergence-poll loops (await pattern with
   5s deadline), NOT budget+sleep timing assertions. No conversion needed;
   the "injected deadlines for NEW timing tests" rule stands. Row closed.
8. **P2-T22 docs bundle:** `docs/DOMAIN_LANGUAGE.md` gained the MOS row
   (zero-means-unreported is now written law) and the
   CostMicros/CostCurrency row; new `docs/telnyx-quarterly-checklist.md`
   — the 2027-01 re-verify bundle (setup commands, per-cluster expected
   shapes, Range-support first-test, close-out steps, calendar-confirm
   bullet as its first checkbox).
9. **P2-T23 UX bundle:** search hint now teaches multi-word AND semantics
   with a worked example (`thermostat broken`); RelativeTime behavior
   pinned by `TestRelativeTimeDayPrecisionContract` (exact-timestamp
   tooltip always present; day-precision text ≥30 days; 7–30 days stays
   week-granular + tooltip by design — the library already did the right
   thing, now it's contract-pinned against bumps); ops cards print-
   protected via `.lens-ops-card` marker class + `break-inside: avoid` in
   the `@media print` block + a `print-ops.png` probe capture.
10. **P2-T24 hygiene bundle:** `scripts/check-agents-budget.sh`
    (AGENTS.md ≤219 `wc -l` check; both pass and fail paths executed and
    verified) + checked-in `.githooks/pre-commit` (activation is a single
    owner config line — documented, not silently set); `ingest-cdr
    -dry-run` (parse + age-gate report, **no -db required, no index
    created** — proven by test) for the F63 production re-ingest
    rehearsal.
11. **P2-T25 verification hygiene:** dark-theme eyeball over the fresh
    corpus — dashboard (desktop+mobile), calls list, detail page: **no
    defects** (teal chain intact, badges/contrast readable, mobile stacks
    cleanly); true clean-clone rehearsal — scratch worktree + `git clean
    -xfd` (verified 0 `*_templ.go`, no codegen ledger) → probe bootstrap
    self-healed (generated 6 files, built, served, captured 4 PNGs).
12. **P2-T26 R27.x triage:** art-dupl → 36 clone groups, all
    non-actionable/filtered (templ signature idioms); judged the two real
    candidates — `hasFilters` vs `hasTypedFilters` (deliberately different
    predicates, doc comments say so) and `clockDuration` vs
    `compactDuration` (different formats/rounding/call-sites) —
    accept-with-rationale; branching-flow splitbrain ✅ / boolblind ✅ /
    dupe → 2 self-classified false positives (`AgeGate` vs
    `retention.Policy`, one shared field name, documented layering);
    file-size triage: **serve.go 301 lines (was 617 — resolved, under the
    350 gate)**, search.go 443 and telnyx_test.go 835 accept-and-document;
    govulncheck: **0 vulnerabilities affecting code** (1 module-level,
    uncalled).
13. **P3-T27 watch register, all filings re-verified live today:**
    **CLOSED upstream:** go-appkit#26 (testkit drain cost),
    BuildFlow#55 (nix-hash-fix), BuildFlow#56 (gate tools go floor),
    go-cqrs-lite#62/#63/#64. **Still OPEN:** go-etag#2, go-nix-helpers#7,
    treefmt-nix#545 (it's an ISSUE, not a PR — the filings doc's ledger
    row wording corrected by observation), templ-components
    #48/#49/#50/#51/#55.
14. **Close-out ledger:** TODO_LIST rewritten — 18 rows closed with
    receipts, the Unblocked section now leads with the (now-unblocked-
    upstream, load-gated) clean-checkout 70/70 receipt, 3 new owner-gated
    rows (coverage chase, baseline home, BuildFlow#56 local rebuild),
    Blocked watch rows updated with today's upstream states. CHANGELOG
    Unreleased gained three sections (quality tail / UX+docs / corpus
    archive pointer). Adoption map gained: Card `TitleTag` row (all 12
    page-level Cards), CopyButton nonce row (CSP sweep finding + fix),
    RelativeTime contract row.
15. **Answer sheet extended (§8–10):** CI floor 70 (implemented as
    default), BuildFlow#56 local rebuild (superseded upstream, bookkeeping
    question remains), pixel-baseline home (v2 saved machine-local,
    recommendation A).
16. **Pushed + CI green:** 30-commit tail pushed; final HEADs `70ff42f`
    and `d78291a` both **CI success** — including the new styles-freshness
    job (committed styles.css byte-matches the pinned Tailwind build) and
    the 70% coverage floor.
17. **Morning report annotated:** the 08:02 report's "RED test / UNPUSHED"
    verified-state footer now carries a resolution addendum pointing at
    the fix and this close-out.
18. **scripts/README.md** gained the budget-script row (including the
    hooksPath activation note).
19. **AGENTS.md budget held:** 219/219 lines (verified by the new script).

## b) PARTIALLY DONE

1. **P0-T2 / clean-checkout 70/70 gate receipt.** Upstream is unblocked
   (BuildFlow#56 CLOSED today — the fix landed), but the load policy
   (1-min < 15 AND 15-min < 40) never permitted a run all day (1-min
   ranged 25→112). The row now sits FIRST in TODO_LIST Unblocked, ready
   for a low-load window. Nothing about the tree blocks it.
2. **/ops hardening (T19 leftover).** Authed lane membership + a11y
   (headings, focus) landed; the stale-verdict browser probe was
   deliberately not built — the e2e serve boots without ingest sources,
   so /ops renders the unwired EmptyState. The blocker is buildable-away
   (a poller-wired serve composition or a `serve -wired-ops` dev flag);
   markup is pinned by `TestOpsPageStaleVerdictAlerts` meanwhile.
3. **FTS quality pass.** Multi-word AND semantics pinned
   (`TestMultiWordQuerySemantics`, prior leg); ranking spot checks on a
   larger corpus remain (row rewritten in TODO_LIST).
4. **Buildflow tooling triage.** lychee + nix-build-verify skips
   rationalized in `.buildflow.yml`, shellcheck joined the devShell;
   the doctor-entry disposition (markdown-lint/gitleaks/codespell
   on-demand vs install) still wants a pass after the BuildFlow#56
   rebuild.
5. **Coverage story.** Floor 70 landed; the chase to the fleet-80 target
   is owner-gated (answer sheet §8; measured 73.0%).
6. **Pixel-diff practice.** Baseline v2 + alert-only `--diff` + volatile
   budget all landed; the durable HOME of baselines is owner-gated
   (answer sheet §10).
7. **CSP posture.** Report-only sweep is enforce-clean (prior leg);
   enforce-mode flip is a decision not yet made (see §g, question 3).

## c) NOT STARTED (verified — all live in TODO_LIST with evidence)

1. Recording-proxy timing-hop budget asserts (Server-Timing must not
   perturb upstream timeout budgets).
2. Stale-verdict browser probe (blocked on the wired-ops composition —
   see b.2).
3. FTS ranking spot checks.
4. Buildflow tooling triage remainder.
5. Production re-ingest for ContentHash v2 (owner path; the `-dry-run`
   flag now enables the rehearsal step).
6. Host deploys via the nix-international-telephony module (`aiDids`
   newly available) — owner.
7. Owner decision pack D1–D5 execution (answers → one-decision
   all-surfaces date change etc.).
8. T15 MOS poller execution (2027-01 Telnyx window).
9. PII enforcement at the read boundary (owner decision, recommendation
   B on the table).
10. Browser-level htmx 304 revalidation check (owner manual check).
11. 1MiB marathon-transcript buffer policy (owner).
12. Dependabot confirmation for newly-direct deps (weekly cycle).
13. dependabot-core#16538 maintainer answers + gomod pinning decision.
14. Sibling backup-rotation fold into `modules/telephony/lens.nix`.
15. Mobile card-list (375px) — after owner taste call.
16. Telnyx live-verify cluster (2027-01; checklist file ready).
17. Schema v2 length-prefixed ContentHash encoding (at the v2 gate).
18. README screenshots (R27.x remnant), nix-checker cleanup, vulnix
    closure scan (govulncheck covers Go; the nix closure is unaudited
    this cycle).

## d) TOTALLY FUCKED UP (own goals this leg, all caught and fixed)

1. **Pushed with uncommitted work in the tree (the real one).** The push
   of the 29-commit tail went out while two lint fixes were still
   uncommitted (the daemon hadn't snapshot them yet) — CI on that SHA
   (`bafd463`) went RED on exactly those findings. Caught within a
   minute: committed + pushed `70ff42f`, final HEAD green. The damage is
   a permanently red SHA in history and a few minutes of confusion risk
   for anyone watching CI. Root cause: I combined `git status` and
   `git push` into one command and read the status output too late.
2. **A red test lived in history for ~2.5 hours** (the morning leg's
   `TestRetireDeciderPaths`, daemon-committed at 05:0x, found red at
   session resume ~08:0x, fixed ~09:00). The daemon commits faster than
   test discipline; "run the touched package's tests immediately" was in
   the resuming-assistant notes and still the fix came first, tests
   second.
3. **Wrote `parsed.Total()`** — a method that doesn't exist on
   `ingest.Result` (plain fields). Compile caught it in seconds; still a
   sloppy API assumption I should have grepped first.
4. **Wrong arithmetic in my own new test** (Oct 7 vs Sep 7 — 40 days
   before Oct 17). First run failed on MY expectation, not the code.
   Trivial, but it's the exact "never trust your own test's first green"
   class inverted.
5. **dprint blast radius:** running the formatter repo-wide reflowed ~28
   ARCHIVED status reports (strips continuation-indent under
   strikethrough lines; +1258/−1251 diff noise mixed into a session with
   real changes). Mechanically correct per the formatter, but it should
   have been scoped to session-touched files; the archived prose is
   marginally less readable as a bonus cost.
6. **Accidental throwaway baseline:** passed `--save-baseline
   /tmp/throwaway-v2b` during the diff proof (didn't need it), wrote a
   full 41-PNG corpus to /tmp, then trashed it. Wasted I/O, zero harm,
   pure sloppiness.
7. **Stale LSP warnings chased longer than needed:** the golangci_lint_ls
   diagnostics kept claiming golines/gocognit issues in
   `export_consumer_test.go` after the file had been split and formatted
   — the authoritative `golangci-lint run` was clean. Lesson recorded:
   trust the lint run, not the stale LSP cache.

## e) WHAT WE SHOULD IMPROVE

1. **Push protocol:** `git status` MUST be its own command with its own
   read before any push — the daemon means "uncommitted at push time" =
   "silently missing from the push".
2. **Test-run cadence:** run the touched package's tests IMMEDIATELY
   after writing a test file, before the daemon can commit a red state.
   The 2.5h red test (d.2) is the receipt.
3. **Formatter scoping:** when the tree holds a large untouched corpus
   (archived docs), scope dprint to the session's touched files; ask
   before reflowing history-docs (or exclude `docs/status/archived/` —
   owner call, §f.42).
4. **Gate scheduling:** the full gate has now slipped a full workday on
   load policy. An off-hours catch-up (systemd timer / cron / at) or a
   "first session of the day runs the gate before anything else" rule
   would stop P0-class receipts from aging.
5. **cqrs-lint signal hygiene:** F009 fires on the poller every run and
   is policy-documented; an inline suppression (if cqrs-lint supports
   one) would keep the finding list signal-only.
6. **LSP/lint trust order:** when they disagree, the lint run wins;
   consider restarting the LSP after formatter runs to stop stale-warning
   chase cycles.
7. **Baseline freshness coupling:** markup changes require a manual
   baseline re-save (v1 went stale silently). `cut-release.sh` or
   verify-all could warn when HEAD contains markup changes newer than
   the baseline generation.
8. **art-dupl baseline:** the CI lane is warning-only with 36 known
   groups; a checked-in verdict baseline would make NEW clones stand out
   instead of re-judging the known idiom noise each run.
9. **Upstream the two library findings:** CopyButton singleton-script
   nonce gap and Card h3 default (level-skip) are both templ-components
   hardening/filing candidates discovered by our instruments — file them
   while fresh.

## f) Top 50 things to get done next (brainstorm, impact-ordered; 1–18 are

already-routed TODO_LIST rows, 19+ are proposal-grade ROADMAP fuel)

1. Run the full buildflow gate at low load → bank the 70/70
   clean-checkout receipt (BuildFlow#56 landed); retire the v0.2.0
   workaround note.
2. Owner: answer the three §g questions below + the standing answer
   sheet (D1–D5, PII, mobile) — most blocked High rows gate on this.
3. Prod re-ingest REHEARSAL with the new flag: `ingest-cdr -dry-run`
   against the production Master.csv (F63 prep, zero risk).
4. Host deploys (owner) via `modules/telephony/lens.nix`; verify
   `aiDids` + `-max-age`/`-recording-timeout` on real hosts.
5. Production re-ingest for ContentHash v2 (the one-time wave, after 4).
6. Build a poller-wired ops composition for the e2e serve (small
   serveProbe option or `serve -wired-ops` dev flag) → unlock the
   stale-verdict browser probe.
7. PII enforcement at the read boundary (if owner picks B): E164
   rendering path + gate-style pinning test.
8. FTS ranking spot checks on a larger corpus (digits-shadow corners).
9. Recording-proxy timing-hop budget asserts.
10. Buildflow tooling triage remainder (doctor entries disposition).
11. Mobile card-list implementation (after the owner taste call).
12. Fold the go-appkit#26 drain-cost insight into our testkit usage +
    the upstream ledger.
13. Watch ledger: go-etag#2, go-nix-helpers#7, treefmt-nix#545,
    templ-components #48/49/50/51/55 (quarterly re-verify).
14. 2027-01 Telnyx window: run `docs/telnyx-quarterly-checklist.md` top
    to bottom; confirm the calendar entry NOW.
15. T15 MOS poller execution in the same window.
16. 1MiB transcript buffer policy decision + implementation.
17. Confirm dependabot bumps for newly-direct deps; answer
    dependabot-core#16538.
18. Sibling fold: backup-rotation opinion into lens.nix comments; schema
    v2 ContentHash encoding at the v2 gate.
19. CSP enforce-mode decision (needs §g Q3) + flip mechanics if go.
20. csp-sweep improvement: per-surface console attribution (§d5 of the
    08:02 report).
21. Activate `.githooks` (owner one-liner) AND wire the budget check into
    the daemon path (the daemon bypasses local hooks unless hooksPath is
    set) — otherwise the pre-commit check only guards human commits.
22. Add the AGENTS.md budget check as a lightweight CI job (guards all
    committers without local config).
23. cqrs-lint inline suppression for F009 (if supported upstream —
    possible filing).
24. Off-hours gate catch-up timer (see e.4).
25. art-dupl verdict baseline file (see e.8).
26. cut-release.sh has never been exercised end-to-end — dry-run it once
    against a non-release commit.
27. Baseline-freshness warning in cut-release.sh / verify-all (e.7).
28. File the CopyButton nonce gap upstream (templ-components).
29. File the Card TitleTag default (h3 level-skip) upstream.
30. Decide the archived-docs dprint policy: exclude
    `docs/status/archived/` or accept future reflows.
31. Check whether branch protection can require HEAD-green only (cosmetic
    mitigation for mid-flight red SHAs like `bafd463`).
32. README screenshots (the R27.x remnant — README is still text-only).
33. nix-checker cleanup (R27.x remnant).
34. vulnix scan of the full nix closure (govulncheck covers only Go) —
    pair with the next flake lock bump.
35. D3 follow-through: quiet-week volume-chart filing if owner picks B.
36. D4 follow-through: export-note collapse if chosen.
37. D5 follow-through: AI-share tone change if chosen.
38. Extend the print probe to assert `break-inside` computed style on
    `.lens-ops-card` (today the rule's presence is only in the CSS).
39. Long-excerpt PNGs: quarterly truncation-regression eyeball.
40. `verify-all --baseline <dir>` convenience wrapper over
    LENS_DIFF_BASELINE.
41. AGENTS.md 219/219: pre-plan the next extraction (the serve/UI
    paragraph's recording-proxy details could move to docs/http-stack.md)
    so the next addition doesn't stall.
42. Session-start drift guard: `gh run list` + one package test as a
    two-second "is HEAD actually green" check before continuing work.
43. ROADMAP sweep: fold f-items 19+ that survive triage into ROADMAP
    raw-ideas (docs-health HARVEST rigor: don't entomb them here).
44. CHANGELOG: consider splitting the Unreleased pile at v0.3.0 once the
    gate receipt lands (the pile is 3 sections deep already).
45. Pixel baseline: re-save to v3 automatically as part of the next
    markup-touching change's definition-of-done.
46. Coverage: if owner chases 80, the biggest gaps are serve/UI handler
    branches (web 73.9%) — plan extraction of pure helpers first.
47. `ingest-telnyx -dry-run` parity with ingest-cdr (fetch is network-
    bound; a report-only mode still helps window rehearsals).
48. Ops page: surface purge-timer state (deployment config visibility
    row partially done; last-purge age still absent by design decision —
    revisit if ops asks).
49. Docs: mirror the `-dry-run` flag into README's ingest examples.
50. Celebrate v0.2.x hygiene: the repo is at lint-0 / test-14-14 /
    CI-green / probes-green — schedule the next Pareto pass BEFORE new
    work accretes (the plan-after-plan pattern that worked twice now).

## g) Three questions I cannot answer myself

1. **Coverage: keep the floor at 70 (landed, measured 73.0%) or chase
   the fleet-80 gate?** Chasing 80 means mostly serve/UI handler-branch
   tests — real work with diminishing returns for a single-host internal
   tool. Your risk appetite decides; I can't derive it.
2. **Pixel-baseline home: stay machine-local (`~/lens-baselines/<gen>`,
   my recommendation, zero repo weight) or commit a baseline generation
   (~41 PNGs, reviewable drift, ~10–20 MB repo weight)?** This is a
   repo-weight-vs-reproducibility taste call, and it decides whether a
   second machine can ever meaningfully diff.
3. **CSP: flip to enforce-mode in v0.3.0 after one more clean sweep, or
   stay report-only until real-browser usage exists on the deployed
   hosts?** The sweep is enforce-clean in the probe environment, but the
   probe browser is not your browser mix — enforce-mode breaks pages
   silently for real users if any surface was missed, and only you know
   the operator risk tolerance for that.

---

_Verified state at writing (2026-10-17 10:28 CEST): local main ==
origin/main at `d78291a`, CI success, working tree clean, tests 14/14,
golangci-lint 0 issues, browser probes green, AGENTS.md 219/219. Waiting
for instructions._
