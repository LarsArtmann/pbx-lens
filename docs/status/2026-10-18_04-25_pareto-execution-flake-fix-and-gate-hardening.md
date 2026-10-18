<!-- docs/status/2026-10-18_04-25_pareto-execution-flake-fix-and-gate-hardening.md
     Status report for the 2026-10-18 pareto-plan execution session
     (T1-T24 of the 01-01 SUPERB plan). Format: task-queue dispatch .md at
     the operator's explicit a-g structure. Supersedes the 02-40 report as
     the newest live snapshot; that report's §e owner questions carry
     forward. -->

# Status Report — pareto execution: gate hardening, the flake fix, and the corrected diagnosis (2026-10-18 04:25)

**Session scope:** the approved 01-01 SUPERB pareto plan executed end to end
in one run. Thirteen of the twenty-four comprehensive tasks landed fully;
three landed as staged-but-gated artifacts; the session's centerpiece was
accidental — the dev gate REPRODUCED the repo's one CI "flake" live, which
falsified the session's own first root-cause diagnosis and produced the real
fix. Machine load fell 143 → 13 during the run; the full-gate policy window
(1-min < 15 AND 15-min < 40) is OPEN as of this writing (13.3/15.9/18.7).

## a) FULLY DONE

1. **T3 — statix no-op tripwire** (`scripts/verify-all.sh`): the lane fails
   loudly on statix's fail-open signature ("config error" on stderr, exit 0,
   lints nothing — the five-day October outage class). Proven both ways: a
   fake fail-open statix on PATH turns the lane red BEFORE any probe runs
   (exit 1); real statix passes through to green contracts. Skips cleanly
   when statix is off PATH. Documented in the scripts/README lane row.
2. **T4 — CI nix lane** (`.github/workflows/ci.yml` `nix-lint` job): statix
   (with the no-op tripwire logic) + `deadnix --fail` + `nix fmt --ci`, every
   tool resolved through the FLAKE'S LOCKED nixpkgs rev (via
   `nix flake metadata` → `NIXPKGS_REF`) so a tool release can never flip a
   verdict mid-green. Each lane was proven red on a seeded failure (broken
   statix.toml, unused binding, unformatted file) inside a throwaway
   `--no-hardlinks` clone before landing. GREEN on remote CI. Also surfaced
   an upstream-adjacent fact: deadnix exits 0 on UNPARSEABLE files (the fmt
   lane covers that hole — noted in the job comment).
3. **T14 — `ingest-telnyx -dry-run` parity** with ingest-cdr: report-only
   fetch ("N conversations fetched, N would-record, N age-refused, N
   fetch-rejected — nothing written"), no `-db` required, no index opened,
   enrichment correctly absent (it reads the read model). New `-api-base`
   flag for stub-server rehearsals. Pinned by `TestIngestTelnyxDryRunReports
   WithoutWrites`: an httptest Telnyx stub, both conversations' message
   lists walked, zero entries added to the temp dir. README + FEATURES rows.
4. **T15 — csp-sweep per-surface console attribution**: console CSP messages
   now carry the navigating surface (`${surface} (console)`) via a
   `consoleSurface` variable reset per navigation, including a dedicated
   attribution for the copy-click phase. Full re-run: zero violations across
   all 6 surfaces, exit 0.
5. **T16 — `verify-all --baseline <dir>` wrapper** (the LENS_DIFF_BASELINE
   hook as a lane flag; wins over the env var; missing-arg guard exits 2 —
   observed live) **+ print-probe `break-inside` computed-style assert** on
   every `.lens-ops-card` (live run: `["avoid","avoid","avoid"]`, RC 0, no
   card → loud failure). README doctrine updated.
6. **T17 — art-dupl verdict baseline**: `scripts/art-dupl-baseline.txt`
   (exact-output, 34 clone groups / 2 shown at pin 0.9.0) checked in; the CI
   duplication job diffs fresh output against it and warns with the refresh
   procedure. Seeded-clone proof: a real 10-line duplicate pair stands out
   (34→35 groups, new block in the diff). **Baseline-freshness warning** in
   verify-all: when internal/web markup is newer than the pixel baseline
   dir, the lane warns (never fails) with the re-save command — proven with
   a back-dated dir.
7. **T18 — golden-scrubber contract test** (`TestGoldenScrubberCoversFooter
   Stamp`): pins the scrubber↔footer-copy contract by exact-match output;
   if the footer copy drifts, this test fails with a pointer to update the
   pattern+replacement pair instead of the goldens silently capturing raw
   wall-clock stamps. Bonus verification: confirmed the optional-tabindex
   group is consumed by design (goldens' established shape).
   **scripts/README codegen paragraphs merged** (the two competing
   "Codegen bootstrap" blocks → one hash-primary paragraph).
8. **T19 — AGENTS.md recording-proxy extraction**: the two-hop upstream
   budget details moved to `docs/http-stack.md` — verified against
   `internal/web/recording.go` first (hop 1 = `context.WithTimeout` on the
   metadata fetch; hop 2 = `transport.ResponseHeaderTimeout`; body
   unbounded; Server-Timing pre-first-write only). AGENTS at 218/219
   (budget check green, now enforced on every commit — see #14).
9. **T20 — `-wired-ops` serve fixture + fourth lane probe**: `serve
   -wired-ops` swaps the live counters for a canned STALE snapshot (mnd-
   clean named constants); `scripts/verify-ops-verdict.js` boots that
   composition in a real browser, asserts the "Poller pass is stale" alert
   with the canned 2h age, and checks `/api/ops` answers `stale: true` with
   both canned source rows. Wired into verify-all (both modes); `serveProbe`
   gained a backward-compatible `extraServeArgs` parameter. ALL PASS live.
10. **T21 — FTS ranking corpus test** (`internal/search/ranking_corpus_test
    .go`): a 106-view synthetic corpus (100 fillers with digit-restricted
    number tails + 6 instrumented targets) pins bm25 ORDER (term frequency:
    rank-hot before rank-cold; both-terms document outranks the single-term
    heavyweight) and the digit-shadow corners (4-digit run inside a longer
    number, shared-prefix runs hitting both members, discriminating runs
    isolating one, full-number query deduping the FTS and digit legs).
    Expectation corrected once against reality (rank-both legitimately
    matches "thermostat" too). Full package green; gci-fixed.
11. **T22 — recording-proxy timing-hop asserts** through the FULL served
    chain (`cmd/pbx-lens/serve_test.go`): `TestRecordingProxyTimingHopBudget`
    proves the metadata and headers hops appear in Server-Timing, each ≤ the
    UpstreamTimeout, no body metric, proxied bytes intact;
    `TestRecordingProxySlowUpstreamStaysBounded` proves a stalling upstream
    (500ms withhold vs 60ms budget) fails fast at 502 with the header budget
    intact.
12. **T23 — buildflow doctor re-triage**: 29 ✗ marks, every one inside the
    2026-10-17 disposition classes (cargo/python/js-ts/docker/proto
    no-match, codespell on-demand, gci via golangci-lint, vulnix advisory
    row, lychee config skip); dry-run health re-verified (9 unavailable =
    cosmetic, 5 skipped: 3 on-demand by design + 2 deliberate config
    skips). Gate-doc receipt updated. **Statix invocation contract pinned
    in `.buildflow.yml`** (bare `statix check`; targets bypass the config
    `ignore`; re-audit on nixpkgs bumps).
13. **T7/T8/T9 — three upstream filing drafts staged** (docs/drafts/, all
    verify-before-filing checked with dupe searches, none filed):
    CopyButton's empty-`Nonce` unconditionally-emitted UN-NONCED singleton
    script (source-verified in `utils.ScriptComponent`; cross-refs #63 and
    #24), Card's h3 TitleTag default (12/12 pbx-lens call sites override
    it — a level-skip under every page h1), statix's exit-0-on-config-error
    footgun (upstream identified as molybdenumsoftware/statix; repro
    re-verified in the seed clone; dupe search clean). **T10 resolved as
    documentation**: cqrs-lint v4.17.0 has NO inline suppression and no
    per-path exclude (verified at source), so `docs/cqrs-lint-policy.md`
    records the gap and the repo-wide-disable tradeoff.
14. **T11 — hooks wiring**: `.githooks/pre-commit` now chains the AGENTS
    budget check AND delegates to any flake-installed hook (treefmt), and
    `git config core.hooksPath .githooks` is SET in the primary clone —
    the auto-commit daemon is routed through the budget gate (observed
    live: the gate line printed on this session's own commits). CI budget
    job = T12-gated residue.
15. **THE FLAKE — root-caused (for real), fixed, and proven**: the
    2026-10-16-era `TestCacheSemanticsHardening` CI one-off reproduced live
    in the dev gate (`gzip conditional replay status = 200, want 304`,
    5.04s). Real cause: rendered bodies embed a 1-second-granularity stamp
    (the PolledRegion "Updated HH:MM:SS" footer, RelativeTime chips), so a
    conditional replay crossing a second boundary legitimately answers 200
    with a fresh tag. Fix: a `conditionalReplay` helper re-fetches and
    re-tags until the representation is stable (a 200 carrying an IDENTICAL
    body still fails as a real caching violation), applied across ALL
    conditional legs (`assertConditionalGET`, the EndToEnd "/" replay, the
    gzip leg, the identity leg, and a new `assertConditionalHEAD`). 20x
    -race loop green; 3x standard loop green.
16. **CI lint hardening of the fix**: remote lint caught gocyclo 22>20,
    ineffassign, and mapsloop on the first fix commit — resolved by
    extracting `assertConditionalHEAD`, restructuring the helper's return
    tracking, and `maps.Copy`. Local lint 0 issues; tests green 2x.
17. **Final verification battery, all green**: full Go suite 14/14 packages;
    `verify-all --fast` with all FOUR probes; the FULL browser lane (41-PNG
    sweep) green earlier in the session; dev gate **RC=0** on the fix;
    remote CI **green** on `9c94f85` (run 39083743958, watched with
    --exit-status); AGENTS 218/219 on every commit via the new hook.
18. **Living docs re-synced**: CHANGELOG gained Added + Changed 2026-10-18
    sections (incl. the corrected flake narrative); TODO_LIST rebuilt to 2
    executable TODO rows + 22 blocked (done rows deleted to CHANGELOG per
    the list's discipline); status report 02-40 written and its live-set
    table row added; 5 markdown files dprint-formatted.

## b) PARTIALLY DONE

1. **T2 — the literal 70/70 full-gate receipt**: the session ran the DEV
   gate to green (twice, the second time on the fix), and the load policy
   window is OPEN right now (13.3/15.9/18.7) — but the FULL buildflow gate
   from a clean checkout was never STARTED (the dispatch arrived as the
   window opened). F6 (bank the receipt in docs/buildflow-dev-gate.md) and
   F7 (close the v0.2.0 workaround note in CHANGELOG) wait on that run.
2. **T13 — cut-release.sh full-body dry-run**: quiesce conditions are ALL
   met at this moment (load in policy, clean tree, pushed HEAD `9c94f85`,
   remote CI green) — the dry-run has not been attempted. F51 (the
   `go list -m` sleep-10 → retry loop) is unknown until it runs; F52
   (runbook receipt) waits on it.
3. **T7-T10 filings**: drafting, dupe-searching, and ledger staging are
   100% done; the actual FILED steps (issue creation, FILED banners, the
   gotchas statix issue-link F36) are 0% — an owner-go-ahead public act by
   design.
4. **T11 hooks**: clone-local activation done; the CI budget-job variant
   (F41-F44) is gated on the lint-policy answer.
5. **T5 — `golangci-lint config verify`**: not wired (gated on T12); the
   residue row is staged in TODO_LIST with the gohumanize proof plan.
6. **T12 — lint-policy intent (3d93fdf)**: untouched — owner question.
7. **T1 — owner answer sheet**: staging doc ready; zero answers recorded
   (owner input).
8. **T24 — production re-ingest rehearsal**: needs the prod Master.csv
   path (owner); `-dry-run` machinery proven ready by T14's twin.

## c) NOT STARTED

- The T2 full-gate run itself (window open, decaying — time-sensitive).
- The T13 dry-run itself (same window).
- All FILED steps for the staged drafts (owner-gated).
- T5 wiring + T11's CI job (T12-gated).
- T1, T24 (owner-gated).
- Everything in the plan's §6 external register: 2027-01 Telnyx window (MOS
  poller, DTO live-verify, fold-ins), host deploys (`aiDids`), CSP
  enforce-mode flip, mobile implementation, PII enforcement, schema v2,
  v0.3.0 cut, vulnix/nix-checker pass, quarterly cadences — untouched by
  design.

## d) TOTALLY FUCKED UP

1. **The T6 misdiagnosis — the session's defining error.** The first
   triage attributed the 2026-10-16 `TestCacheSemanticsHardening` flake to
   go-appkit testkit's `startTimeout = 5s` (the 5.06s duration matched the
   constant, and the file's own leakSettleTimeout precedent made the story
   feel canonical). I wrote a complete upstream filing draft, committed the
   wrong narrative into the permanent record (test comment, CHANGELOG,
   status report), and it took the dev gate ACCIDENTALLY reproducing the
   failure with its real assertion visible (`gzip replay = 200, want 304`)
   to expose the truth: a body-churn race on the 1-second Updated stamp,
   nothing to do with listener startup. Mitigations that held: the draft
   was LIVE (owner-review box unchecked) so nothing was filed publicly; the
   retraction + corrections landed in the same session (draft deleted,
   ledger row removed, test comment and CHANGELOG and report rewritten).
   Process root causes below (§e).
2. **Diagnosis from duration-shaped circumstantial evidence**: I never
   fetched the 2026-10-16 CI log to read the ACTUAL assertion — the
   archived report recorded only "5.06 s, flaky-looking". The correct
   assertion was sitting in a CI log one `gh run view --log-failed` away,
   and the archived report's own phrasing ("real-time-dependent timing
   worth a look") was generic enough to fit my theory instead of
   falsifying it.
3. **False-confidence loop**: 50x -race under load 140-200 went green and I
   recorded it as triage support. A green loop under load bounds
   frequency; it validates NOTHING about a causal hypothesis. I even wrote
   that caveat into the draft — and still over-trusted the diagnosis.
4. **Lint drift onto remote CI**: the conditionalReplay commit was pushed
   after local lint passes that predated the final edits — CI failed on
   gocyclo/ineffassign/modernize, all introduced in the last few edits. A
   red public CI run spent on refactor hygiene. (Compounding confusion:
   the daemon's push cadence lagged origin for stretches, so "is it
   pushed?" needed checking before watching runs.)
5. **Markdown reflow churn**: hand-written long table rows in
   scripts/README.md and the status reports needed dprint reformatting
   after the fact (the gate flagged them); one dprint re-wrap of the
   02-40 report landed mid-edit and forced a scripted exact-match repair.
   The "run dprint immediately after md edits" lesson from the 2026-10-17
   pass was re-learned the expensive way.
6. **Minor**: the first statix seed test diffed against a relative path
   inside the clone (invalid test, caught and redone with an absolute
   path); FEATURES.md hit modified-since-read races twice from the daemon
   (dropped to a scripted atomic edit); `verify-all --baseline` with a bad
   invocation wastes the three contract probes before failing at the
   screenshot stage (observed, documented, left as-is — borderline call).

## e) WHAT WE SHOULD IMPROVE

1. **Flake triage rule: read the actual failing assertion first.** Fetch
   the CI log (`gh run view --log-failed`) before forming ANY hypothesis;
   a duration is circumstantial. Consider a docs-health convention:
   archived flake records must quote the assertion line.
2. **Reproduce the failure mode before claiming the cause.** Green-under-
   load loops bound frequency and are not confirmation. The gate's
   accidental repro was worth more than all 70 green loop iterations.
3. **Lint the exact diff immediately before every push** (not just after
   each edit) — the CI lint failure was 100% preventable; the diff was
   never linted after its last change.
4. **Reconcile dev-gate vs CI lint coverage**: the dev gate went green on
   the same commit CI lint failed (gocyclo/ineffassign/modernize). Either
   dev mode skips those linters or the gate's golangci ran a stale pass —
   audit the step matrix and record it in docs/buildflow-dev-gate.md.
5. **The stamp churn is a real product issue, not just a test nuisance**:
   with a 1-second body granularity, the dashboard's 30s polls almost
   never hit their 304s (the ETag changes every second). Rendering the
   "Updated" stamp and/or RelativeTime chips client-side would make HTML
   bodies second-stable and make the whole conditional-GET economy real.
   Owner doctrine call (§g3).
6. **Seed-clone technique is now proven** for destructive/experimental
   tests (twice this session) — record the pattern in scripts/README so
   future sessions don't gamble on the daemon.
7. **Push-lag hygiene**: after each commit, check `git log origin/main..main`
   before watching CI — the daemon's push is not immediate and grabbed-the-
   wrong-run happened once.
8. **Run dprint on touched markdown immediately** after each edit, not in
   a cleanup batch (the reflow-then-repair dance cost three tool calls).
9. **verify-all --baseline should parse early** and fail before the
   contract probes (small polish).
10. **TODO_LIST scripted edits for daemon-hot files** (FEATURES.md race)
    — prefer atomic script edits over the edit tool on frequently-touched
    files.

## f) Top 50 things we should get done next

1. **T2 NOW — the policy window is open (13.3/15.9/18.7) and decaying**:
   run the full buildflow gate from the clean checkout; bank the literal
   receipt (exit code + step counts + load at start) in
   docs/buildflow-dev-gate.md.
2. F7 immediately after: close the v0.2.0 workaround note in the CHANGELOG
   release section, pointing at the fresh receipt.
3. **T13 in the same window**: cut-release.sh full-body dry-run (guard
   rails: dirty tree, unpushed HEAD, CI red); implement the `go list -m`
   retry loop if it flakes; record the runbook receipt in scripts/README.
4. Audit dev-gate vs CI lint step coverage (§e4) and record the matrix in
   the gate doc — the dev gate must not be passable where CI is red.
5. Owner: answer sheet §1-§11 (T1) — unblocks six High/Medium rows.
6. Owner §g1: the 3d93fdf lint-policy intent (T12) — unblocks T5 and the
   CI budget job.
7. T5: wire `golangci-lint config verify` (surface per #6's answer); prove
   it rejects the historical gohumanize config (reverted).
8. T11 CI variant: AGENTS-budget CI job + 220-line seed proof (after #6).
9. Owner go-ahead → FILE the CopyButton empty-nonce draft (cross-ref
   templ-components#63/#24) (F28).
10. Owner go-ahead → FILE the Card TitleTag h3 level-skip draft (F32).
11. Owner go-ahead → FILE the statix exit-0 draft at
    molybdenumsoftware/statix (F35); then update docs/gotchas.md with the
    issue link (F36).
12. Owner call on the RETRACTED startTimeout draft: file nothing, or leave
    a scoped comment on go-appkit#63 with only the 1ms-poll/5s-constant
    observation (no causal claim)?
13. T24: owner provides the prod Master.csv path → `ingest-cdr -dry-run`
    rehearsal; record would-record/reject counts in the F63 runbook.
14. Watch the next real CI runs for a conditional-replay 200 recurrence —
    the fix's field trial; revisit the 3-attempt budget if it fires.
15. Pursue the real stamp fix (§e5): client-side "Updated" stamp (and/or
    RelativeTime chips) so HTML bodies are second-stable and 30s polls
    actually earn their 304s (owner doctrine call — see §g3).
16. Measure today's 304-hit rate on /frag/dashboard (cheap log probe)
    before/after #15 to quantify the win.
17. Add the "quote the failing assertion" convention to the docs-health
    flake-record practice (notes in the skill's repo or AGENTS).
18. Document the hooksPath opt-in one-liner for fresh clones in the main
    README (currently only scripts/README carries it).
19. Consider a gocyclo policy for test functions (raise threshold vs
    nolint-ledger row) after #4's audit.
20. Adopt the seed-clone pattern into scripts/README (from §e6).
21. dependabot weekly confirm (blocked row) when the cycle lands.
22. Fold-in watch: templ-components#48/49/50/51/55, go-etag#2,
    go-nix-helpers#7, treefmt-nix#545 (quarterly re-verify).
23. go-appkit#26 fold-in (testkit drain cost) when a bump lands.
24. After #15: update docs/http-stack.md's revalidation contract to the
    new body-stability truth.
25. Harvest + archive the 00-18/00-55/02-40 report trio in the next
    docs-health pass (this report becomes the newest live set member).
26. v0.2.1 consideration: the flake fix + gate hardening + dry-run parity
    are a coherent patch release (owner release call).
27. Wire the art-dupl baseline refresh into the templ-components bump
    watchlist (docs/templ-components-adoption.md order of operations).
28. Add a stale-window timer idea to ROADMAP if load-gated receipts keep
    blocking (the gate-timer raw idea already exists — cite this session).
29. After a durable pixel-baseline home exists (owner §10), wire
    verify-all's `--baseline` default to it.
30. Extend the FTS corpus to 1000 views only if ranking becomes a product
    question (cheap lever, noted).
31. Surface `serveProbe` extraArgs as a screenshot-probe flag for ad-hoc
    fixture captures.
32. Confirm static-asset conditional GETs 304 first-try (no retry path) —
    one assertion in assertConditionalGET's callers.
33. Extract a shared quiesce check if cut-release.sh duplicates verify-all
    preconditions (post-#3 refactor).
34. AGENTS.md commands block: consider a one-line statix-contract pointer
    if budget allows (218/219 — needs an extraction first).
35. docs/drafts: when filings land, flip ledger rows + FILED banners in the
    same commit as the `gh issue create` output.
36. The `-wired-ops` fixture: consider also serving canned CACHE counters
    so the /ops cache card gets live browser coverage too.
37. csp-sweep: after upstream CopyButton fixes land, re-run the 6-surface
    sweep and re-verify the nonce threading docs.
38. Look at `TestCacheSemanticsHardening`'s post-fix duration in CI (the
    5s signature should be gone; a residual 1-2s means the retry is
    firing — watch it).
39. README: mention `-dry-run` parity for ingest-telnyx in the commands
    quick-reference block if one exists beyond the prose section.
40. Retracted-draft residue: grep the corpus next session for any straggler
    citations of the startTimeout draft (report + CHANGELOG corrected;
    verify nothing else).
41. The `-api-base` flag: consider documenting a stub fixture script for
    Telnyx rehearsals (the test's httptest handler is the seed).
42. Consider `go test -count=1` in the dev gate if result-cache staleness
    ever masks the stamp race again (the gate reproduced it with
    NO_RESULT_CACHE=1; note the flag dependency in the gate doc).
43. Pixel baselines: after the next UI change, re-save and re-diff to
    exercise the freshness warning end-to-end.
44. `.golangci.yml`: the modernize linter flagged maps.Copy — sweep the
    repo for remaining mapsloop findings proactively once.
45. verify-ops-verdict: add a check that /ops WITHOUT the fixture renders
    the honest "nothing wired" state (guards the fixture from leaking
    into normal serves).
    46.docs/http-stack.md: record the two-hop budget section's test pointer
    (TestRecordingProxyTimingHopBudget) next to the doctrine.
46. Consider retry-loop extraction if more stamp-sensitive assertions
    appear (a tiny `stability` helper over conditionalReplay's loop).
47. Pre-draft the T2 receipt paragraph (F6) so banking is a 2-minute edit
    once the gate finishes.
48. ROADMAP: the off-peak gate-timer idea gains a concrete trigger (this
    session's window-waiting) — promote to a raw-idea with data.
49. Celebrate the daemon-routing win: the budget hook fired on every
    commit this session — verify it also fires for DAEMON-authored
    commits (it should, hooksPath is clone-global; watch one).

## g) My Top 3 questions I can NOT figure out myself

1. **The 3d93fdf lint-policy intent (self-review §g1, now blocking two
   rows)**: were the `lll` exclusion scope, `run.go`'s "1.27" (vs the
   1.27.2 floor), and the tagliatelle rule changes DELIBERATE policy or
   daemon accident? Formalize-with-comments or revert? (Unblocks T5 and
   the CI budget job.)
2. **Filing go-aheads, per draft**: file the CopyButton empty-nonce issue?
   The Card TitleTag h3 issue? The statix exit-0 issue at
   molybdenumsoftware/statix? And for the retracted startTimeout
   diagnosis — total silence, or a scoped comment on go-appkit#63 with
   only the polling-cadence observation?
3. **The stamp-churn doctrine call**: the ETag economy on stamped pages is
   currently decorative (the body changes every second, so 30s polls
   re-download). Do you want the client-side-render fix for the "Updated"
   stamp / RelativeTime chips (real 304s, less churn, a small CSP/JS
   surface addition), or is per-second revalidation acceptable for a
   single-operator internal tool and the tests should simply tolerate it
   (as they now do)?

——

**Waiting for instructions.** The load window for T2/T13 is open NOW and
decaying; say the word and the full gate runs first.
