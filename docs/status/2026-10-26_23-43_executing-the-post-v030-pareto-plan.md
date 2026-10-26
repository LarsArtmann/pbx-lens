<!-- docs/status/2026-10-26_23-43_executing-the-post-v030-pareto-plan.md
     Status report: executing docs/planning/2026-10-26_20-10_SUPERB-post-
     v030-execution-and-truth-closeout.md end to end (T1-T19), same evening.
     Provenance: the plan's Step-0 errata table; receipts cited inline. -->

# The post-v0.3.0 Pareto plan executed — truth, ops proof, P11, and a banked receipt (2026-10-26 20:15 → 23:43)

## a) Fully done (each verified against code/commits)

1. **T1 — the truth sweep** (`ff603f3` + `3dcd582`): the five 19-53-executed
   rows struck in 16-00/17-29, and evidence-verified resolved leads struck
   across all ten 2026-10-25 live reports (claim gate, CI self-tests,
   etag/retention counters, gotcha entries, hooks activation, the 10-46
   plan's EXECUTED header). Partial rows keep their open half visible
   (routed notes, no over-strikes). check-rows: 12/12 complete, dprint-
   stable. Nothing archived — every touched report still carries open rows,
   so ARCHIVE would have been wrong.
2. **T2 — the orchestrator receipt, and the bug it caught on the way**:
   `poll-gate.sh -- <command>` APPENDED the args to the default gate, so the
   documented invocation (AGENTS Commands AND cut-release.sh — both landed
   in `cf6e393` with mechanics proven only via the `POLL_GATE_EXEC` stub)
   built `buildflow --build-mode full env … buildflow …` and died on
   "Unknown command env". Fixed at the script: `--` args now REPLACE the
   gate (the contract both callers encode), the self-test seeds that exact
   case (5/5), shellcheck clean, and the REAL receipt banked solo:
   **run `20261026-223553-b4759eaa`, START LOAD 29.70/27.16/23.12, 51
   success / 0 failed, exit 0, wall 59.7s, over commit `49de047`** —
   recorded beside the 1b5208a-era receipt in docs/buildflow-dev-gate.md.
3. **T4 — ops browser-truth**: `verify-ops-verdict.js` asserts the retention
   section live (active boolean; the four SLO keys present, number-or-null);
   /ops joined the screenshot corpus (36-PNG main sweep + `empty-ops` + a
   new `ops-wired` phase: stale alert + active retention window in one
   frame; 50 PNGs per full sweep); and the wording lie is fixed — an empty
   index under an active window renders "nothing is due — the index is
   empty" instead of "Purge lag: unknown", while a genuinely failed scan
   keeps "unknown" (the page tracks the distinction via a scan-succeeded-
   and-empty bit; the JSON null contract unchanged). New posture test; the
   golden stayed byte-stable; full verify-all green.
4. **T5 — P11 answer-rate** (the plan's next north-star slice): answer rate
   (answered over calls whose KIND reports a billsec — agent-native rows sit
   OUTSIDE the denominator because their zero duration means "not reported",
   never "unanswered"), short-call share (< 10 billed seconds over
   ANSWERED), and the top zero-billsec hangup cause on the quality card.
   Denominators registered in DOMAIN_LANGUAGE; unit + render tests; goldens
   re-pinned (dashboard + segments); lint 0; verify-all green.
5. **T3 — release closure prep**: the paste-ready v0.3.0 notes (the shipped
   draft minus its meta header), the v0.3.1-vs-wait one-pager (facts table,
   both options, default: wait — nobody consumes the tagged binary's
   self-report), and the first-tag CI watch checklist (with the
   workflow-at-tag trap and the new concurrency-group behavior). The drafts
   index also absorbed three drafts that had slipped out of it.
6. **T6 — sibling folds** (window checked first: clean, synced): lens.nix
   carries the backup-rotation opinion in the unit descriptions themselves
   (staging only / never rotates / host owns retention; purge tombstones
   index rows, sources untouched); the call-deep-link handoff row landed in
   the sibling DOMAIN_LANGUAGE. The CDR call_direction template fold is a
   RECORDED NO-OP: the sibling owns no cdr_csv template (stock FreeSWITCH
   Master.csv), so the fold's own precondition is unmet — decided, not
   skipped. Sibling commit `04c9d61`.
7. **T7-T15, T17-T18 — the 20% tier**: T7 the nolint census (15 directives,
   all reasoned; erraudit is a cross-tool directive, not dead) plus the REAL
   find — the devShell's `pkgs.golangci-lint` is UNPINNED (v2.15.0) while CI
   pins v2.14; the v2.14 probe ran **0 issues** on this tree, so the skew is
   hygiene, not correctness (TODO row). T9 the FTS tokenizer trap table test
   (case/diacritic folding, apostrophe/hyphen adjacent-phrase matches, and
   the raw-FTS proof that mid-number runs match nothing). T10 the VACUUM
   INTO restore drill re-run green on the current schema. T13 the advisory
   triage (branching hotspots and file-size ACCEPTED with reasons; search.go
   already split; jsonv2 flip surface ordered P0-P3 with ContentHash LAST).
   T14 the cqrs-lint policy pass (census 19, unchanged; the one uncovered
   code, B010, dispositioned). T15 the vuln windows (govulncheck down to 1
   symbol-level finding: GO-2026-6809, fixed in go1.27.3; vulnix's 22 are
   host packages — no repo action between channel bumps). T16 the pre-push
   ritual written into scripts/README.md, the SHA-keyed CI concurrency
   group, and the `-update` quirk diagnosed (the go tool drops the package
   pattern before an unknown flag; flag-last is the fix) and pinned as a
   gotcha. T17 the cache.db VACUUM (194 MB → 33 MB); the binary rebuild is
   PARKED — the BuildFlow checkout carries a concurrent session's dirty
   go.mod and BuildFlow#55 forbids hash repair on dirty trees. T18 the
   stamp-churn harness (`scripts/poll-304-probe.js`) with BEFORE numbers:
   **91.7% within-minute, 75.0% across a 72 s span** — the quantized pill's
   rollover cost, now measured instead of guessed.
8. **T8, T11, T12**: the bundled PII + export-compression owner message
   (letter answers) landed in docs/decisions/ and linked from the PII row;
   the upstream watchlists verified at-latest against the LOCAL go-cqrs-lite
   checkout (system v4.15.4, metaengine v4.20.0, scenario v4.4.3,
   systemscenario v4.2.1; no v5 tags exist; versions.json 109/112 truthful
   — the remaining 3 were my checker's prefix artifacts, not lies); the
   dependabot github-actions grouping turned out ALREADY LANDED in
   .github/dependabot.yml; go-licenses' 17 errors re-confirmed live, ALL
   larsartmann-owned, and the license-check step deliberately STAYS (it
   guards third-party licenses) — the real fix (upstream LICENSEs) is a
   TODO row.

## b) Partially done

1. **T19's archive half**: 19-53 (33 rows) and 19-54 (24 rows) are
   annotated with receipts and check-rows complete, but NOTHING was
   archived this pass — the two reports still carry owner-gated rows, and
   the ten 10-25 reports keep their un-routed tails. The archive move rides
   the next docs-health cycle (f50's own schedule).
2. **T17's rebuild half**: see §a7 — the VACUUM landed, the rebuild is
   parked on the dirty sibling tree.

## c) Not started (carried)

The Blocked table (owner walls) is untouched and correctly parked; the
gh-token cluster (release object, tag-CI watching) keeps its gates; the
2027-01 Telnyx window rows are calendar-bound.

## d) Totally fucked up

1. **I armed the T2 gate with a command that could not work and did not
   notice until it failed**: the plan's own fine step 2.1 (and AGENTS)
   carried the broken append-form since `cf6e393`, and I executed it
   verbatim instead of reading the script's arg handling first. The failure
   was cheap and diagnostic — but two runs burned before the contract fix.
2. **I ran heavy probes PARALLEL to the gate** (the golangci v2.14 probe and
   the 304 measurement during the first receipt run): the gate lost
   test-coverage to spawn starvation (exit 75) — my own contention, exactly
   what the starvation message said. The clean solo re-run is the receipt;
   the parallel habit is the lesson now written into the gate doc.
3. **The machine rebooted under my first poller** (uptime 0:18 when I
   noticed): a background job vanished without a trace and I initially read
   it as a tool glitch. The vulnix flake invocation failed for the same
   environmental reason. External, but I should have checked uptime FIRST
   when a background job disappeared.
4. **The 304 probe never called `req.end()`** — three timeouts before I
   read my own script instead of blaming the serve. The fixed harness is
   the deliverable; the debugging detour was avoidable.
5. **My first versions.json check was wrong** (112/112 "stale"): I compared
   every tag against HEAD in a monorepo where per-module tags legitimately
   lag. The refined check (claimed tag vs the module's LATEST tag) gave the
   honest 109/112. First answer confidently wrong, second check actually
   designed.

## e) What we should improve

1. **Run the orchestrator SOLO, always** — now doctrine (the gate doc
   receipt row says it; the starvation message always said it).
2. **Read the script before running the documented command** — the
   documentation was the bug; a 30-second read of poll-gate's arg loop beat
   two failed runs.
3. **Probe binaries at BOTH pins when versions drift** — the v2.14-vs-v2.15
   probe (two commands) turned a hygiene scare into a non-event.
4. **Monorepo tag math needs the per-module question**, never the
   repo-wide one (the versions.json lesson).
5. **Background jobs get an uptime check on disappearance** before any
   tool-blame.

## f) Next (ranked)

1. Owner: `gh auth login` → publish the v0.3.0 release object (paste-ready
   notes waiting) → watch the first tag per the checklist.
2. Owner: the v0.3.1-vs-wait call (the one-pager is decision-ready).
3. Next docs-health cycle: archive the 19-53/19-54 pair + the fully-
   dispositioned 10-25 reports once the owner rows drain (check-rows BEFORE
   git mv; manifest after).
4. golangci-lint pin alignment (the TODO row): pin the flake to v2.14 or
   bump CI to v2.15 — hygiene either way, the probe says both agree.
5. The remaining 20%-tier tails that survived triage: T12's upstream
   LICENSEs (owner repos), the BuildFlow binary rebuild once the sibling
   tree is clean, the 19-54 docs-health toolkit rows (bullet annotate mode,
   the rg-hook misfire filing, the archiving runbook line).
6. AFTER stamp-churn lands client-side: re-run `poll-304-probe.js` for the
   AFTER numbers (the harness exists; same minute-span discipline).

## g) Owner questions (not self-answerable)

1. The v0.3.0 release object + v0.3.1-vs-wait (both decision-ready; the
   one-pager's default is wait).
2. Probe tag for the tag-CI trigger vs waiting for v0.4.0 (unchanged, §g1
   of 19-53).
3. The golangci-lint pin alignment preference (flake-down vs CI-up; both
   verdict-identical on this tree).
