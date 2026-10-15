# CSS-gate recurrence fix + BuildFlow#55 restore — session report

- **Written:** 2026-10-15 18:45 CEST
- **Scope:** this session only — remediation of the 18:12:54
  `buildflow --fix --build-mode=full --budget 5m --log-level warn --max-time 5m
  --lifecycle-summary --go-upgrade-blame all` run (exit 69, test-race failed),
  plus everything that failure turned out to hide. No unrelated research.
- **Outcome:** the failing step and both hidden follow-on breaks (go.sum tidy
  drift, stale vendorHash) are healed and verified scoped-green; a recurrence
  guard landed (flake app `update-styles`); one NEW live corruption was caused
  and restored within the same session (BuildFlow#55 recurrence, my fault,
  details in d). The full-mode gate rerun is deferred by the documented load
  policy (15-min loadavg ~52 > 40 at 18:37; not re-checked at close).

---

## a) FULLY DONE

1. **test-race root-caused (2nd occurrence same day).**
   `TestCSSGateBridgesHeatmapBrandToTeal` failed because daemon commit 577b6cb
   (18:04:58, styles.css mtime 18:03:59) replaced the canonical minified
   stylesheet (104,666 B, `--ds-brand-rgb:15, 118, 110`) with an unminified
   v4.3.3 build (132,600 B, `--ds-brand-rgb: 15, 118, 110`, space after
   colon). This invalidated the 18:06 triage session's heal (2b0bcb2, 17:12)
   two minutes before that report was written. Evidence: git show 577b6cb
   diff, file header, token grep.
2. **styles.css canonically rebuilt.** `GOEXPERIMENT=jsonv2 go tool templ
   generate ./internal/web/` (go.mod pin) + devShell `tailwindcss_4` v4.3.3
   `--minify` → **byte-identical to 2b0bcb2** (`cmp` clean), violet defaults
   absent. Evidence: token grep, cmp, wc.
3. **Verification stack green.** CSS gate `ok 0.006s`; full
   `go test -race -shuffle=on ./...` 14/14 packages exit 0 — run twice
   (~18:21, ~18:37, after every go.sum change); scoped `buildflow -s
   test-race` exit 0 ("BuildFlow passed 1/1"); `buildflow --failed-only`:
   "no failures recorded in the last run; nothing to rerun".
4. **Recurrence guard landed: flake app `update-styles`** (flake.nix, mirroring
   `update-vendor-hash`): go.mod-pinned templ codegen → flake-pinned tailwind
   `--minify` → 3-token bridge self-verify that fails the app on a broken
   artifact. Verified: `nix flake check --no-build` exit 0; end-to-end
   `nix run .#update-styles` green and idempotent (output byte-identical
   again). This closes prior-session recommendation f.9.
5. **go.sum tidy drift repaired.** Preflight flagged "root needs go mod
   tidy" (2 stale `/go.mod` lines). `buildflow -s go-mod-tidy --fix` exit 0;
   `go mod tidy -diff` now clean; preflight warn gone (18:37 dry-run).
6. **BuildFlow#55 wrong-attribute corruption — caused, caught, restored.**
   My `buildflow -s nix-hash-fix --fix` run (18:33, stale binary 518739c,
   docs-dirty tree) wrote the real vendor hash into cqrs-lint's
   `fetchFromGitHub` `hash`, placeholders into cqrs-lint's `vendorHash` AND
   vendorHash.nix (daemon-committed as 62bcca6) while self-reporting
   "✔ verify / 1 fixed". Caught by reading the post-repair diff; restored
   all three attributes (real `got: sha256-eKVz…` from the probe log into
   vendorHash.nix; original cqrs-lint hashes from git). Proven: `nix build .`
   exit 0, `nix develop -c true` exit 0, HEAD content verified.
7. **Docs truthed up.** 18:06 triage report carries the 18:26 ADDENDUM
   (recurrence + heal + guard); docs/gotchas.md and AGENTS.md now point at
   `nix run .#update-styles` as the single rebuild entrypoint (AGENTS.md held
   at 219 lines); docs/upstream-filings.md #55 entry carries the RECURRENCE
   block + hardened rule ("hash repairs via `nix run .#update-vendor-hash`
   on a clean tree until the binary carries the fix"); vendorHash.nix
   provenance records the 18:33 recurrence.
8. **Preflight at close:** gomod-freshness and FOD-hash-freshness warns
   cleared. Remaining advisories: BuildFlow binary-freshness (their repo) and
   the legacy buildflow.db (documented, deferred).

## b) PARTIALLY DONE

1. **The user's exact full-gate command has never been re-run green.**
   Deferred by the documented load policy (15-min loadavg ~52 vs threshold 40
   at decision time; docs/buildflow-dev-gate.md). Everything the 18:12 run
   failed on or skipped-behind was verified scoped-green instead — but the
   "3 skipped (blocked by failures above)" steps from that run (unnamed in
   its summary) were never individually rerun, and no end-to-end full pass
   exists post-fix. Blocker: load. Effort: S (one command) once load < 40.
2. **Recurrence guard is entrypoint + prose, not enforcement.** Nothing
   technically prevents a third flag-less tailwind clobber of the committed
   styles.css; the CI-side freshness guard (regenerate + `git diff
   --exit-code`, TODO_LIST row dd50513) is still open. Effort: M.
3. **flake.lock never deliberately reconciled** after the 18:12 run's
   `nix-flake-update` (prior session c.7/f.14 also skipped it). Indirect
   consistency only: my rebuild byte-matches the 17:12 artifact, so the
   tailwind pin is effectively unchanged. What else it bumped is unknown.
   Effort: S.

## c) NOT STARTED

1. **Browser lane** (`bash scripts/verify-all.sh`; verify-style-contract M1,
   screenshot-probe) after the CSS heal — not run. Risk ≈ 0 (artifact is
   byte-identical to the long-good state) but it is the documented UI
   verification. Priority: still wanted. Effort: S.
2. **`nix flake check` with builds** — only `--no-build` + package build +
   devShell realization verified this session. Effort: M (load-sensitive).
3. **TODO_LIST.md harvest** of this report (and the 18:06 one) — deliberately
   report-then-wait per instruction. docs-health HARVEST owes: prior f.9 is
   now DONE (recorded only in the triage addendum), plus this file's f-list.
4. **cqrs-lint 17 findings vs docs/cqrs-lint-policy.md diff** (prior f.5) —
   untouched this session.
5. **AGENTS.md line ~192 fix** — "(buildflow -s nix-hash-fix --fix does the
   same)" is now actively misleading (with a stale binary it corrupts, see
   d.1); I noticed it only while writing this report. Effort: S, line-neutral
   edit possible.

## d) TOTALLY FUCKED UP

1. **I caused the exact corruption the file I was editing warned about.**
   I ran `buildflow -s nix-hash-fix --fix` with (a) a tree dirty with my own
   doc edits and (b) a binary that preflight had warned THREE times is stale —
   the precise preconditions of BuildFlow#55, whose provenance warning
   ("never run the hash fixer on a dirty tree") sits in vendorHash.nix, the
   very file I then went to edit. I read it AFTER the daemon had committed
   the corruption (62bcca6). The step's own failure-rate warning (104/124)
   was printed at me twice. Severity: high — live wrong hashes in two files,
   committed to history; restored same session with proof (nix build +
   devShell green). Root cause (mine): I classified the "advisory" severity
   as irrelevant-to-me without mapping it onto the step I was about to run;
   and I used the SECOND-named repair path (the step) when AGENTS.md names
   `nix run .#update-vendor-hash` first. Mitigation until binary rebuild:
   documented in upstream-filings + vendorHash.nix.
2. **First `update-styles` app shipped a broken verifier.** `grep -qF
   "$token"` with tokens starting `--` parsed them as flags — the verify
   stage failed on its first token. Caught only because I ran the app
   end-to-end instead of trusting `nix flake check`. A verifier that cannot
   pass is worse than none. Fixed (`grep -qF --`); the failure path is now
   proven too (that same run exercised it).
3. **A void verification was briefly counted as done.** My first chain
   (`nixfmt flake.nix && nix flake check …`) short-circuited at missing
   nixfmt (127) — flake check never ran — and I moved on before noticing the
   output ordering was wrong. Redone properly. Repeat of prior session's
   lesson e.3 (capture per-command exit codes, not chain tails).
4. **Overclaim in the triage addendum.** "the clobber was a one-shot run
   without --minify" — "no tailwind process alive at 18:15" does not
   distinguish one-shot from an exited `--watch` loop. The version-match
   inference (v4.3.3 both sides → flag-less invocation) is solid; "one-shot"
   is not. Left in place with this correction recorded here (annotate, never
   rewrite).
5. **Near-misses, zero damage:** typo'd "fileed" into the vendorHash.nix
   rewrite on first attempt (blocked by the edit tool's stale-read guard);
   interpreted dprint's silence as "clean" without confirming the check ran
   (rectified by running `dprint fmt` after).

## e) WHAT WE SHOULD IMPROVE

1. **Read the guardrails inside the artifact before repairing it.** The
   provenance comment was one `cat` away and names the exact preconditions.
   Pre-flight for any repair step: read the target file's comments first.
   (Recurring-class: prior session e.1 "verify before writing".)
2. **Map advisories to the step at hand.** "Binary stale" is noise for
   dprint and a hard stop for nix-hash-fix (known filed bug, version-pinned).
   Triage advisories BY TOOL, not by severity alone.
3. **Diff after every repair step, unconditionally.** #55 was caught by
   habit, not by tooling — the step reported success while writing the wrong
   attributes. Make `git diff` post-repair a reflex step in the loop, even
   (especially) on green.
4. **One command, one exit code.** Stop chaining new verifications behind
   unproven invocations (nixfmt-then-check); chains hide which part was void.
5. **Test the failure path of every new verifier** (feed the guard a broken
   input once). My app's first verifier could only fail — proving only its
   success path is half a test.
6. **Re-check deferral conditions at close.** Loadavg was checked at decision
   time, not at session end; a final re-check + timestamp would have either
   unblocked the full gate or made the deferral current.
7. **AGENTS.md equivalence lines age badly.** "X does the same as Y" claims
   rot when one side changes (stale binary). Prefer pointing at ONE canonical
   entrypoint per task — which is exactly what `update-styles` now is for CSS.

## f) Next things (impact-ordered; brainstorm-grade beyond ~10, HARVEST must

apply routing rigor)

1. Fix AGENTS.md ~192: drop "(buildflow -s nix-hash-fix --fix does the
   same)", add "clean tree only; BuildFlow#55" pointer — line-neutral.
   Impact: Critical (prevents repeat of d.1). Effort: S. Category: Docs.
2. Full-gate rerun (user's exact command) once 15-min load < 40, ideally
   `BUILDFLOW_NO_RESULT_CACHE=1`; verify the 3 previously-blocked steps.
   Impact: Critical. Effort: S-M. Category: Quality.
3. Rebuild + reinstall the BuildFlow binary from BuildFlow HEAD (carries the
   #55 fix; also un-breaks #56 gate steps). Impact: Critical. Effort: S.
   Category: Tooling.
4. Run `bash scripts/verify-all.sh` (browser lane) against the healed CSS.
   Impact: High. Effort: S. Category: Verification.
5. Answer "who ran flag-less tailwind at 17:0x and 18:03" (shell history /
   concurrent sessions); if a `--watch` habit exists, retire it.
   Impact: High. Effort: S. Category: Prevention.
6. CI-side styles.css freshness guard (TODO_LIST dd50513 row): regenerate
   with pinned builder + `git diff --exit-code` in CI; prove nixpkgs-pin
   byte-determinism first. Impact: High. Effort: M. Category: Quality.
7. TODO_LIST harvest: mark prior f.9 DONE (update-styles landed); pull this
   f-list per docs-health HARVEST. Impact: High. Effort: S. Category: Docs.
8. Reconcile flake.lock deliberately (what did 18:12's nix-flake-update
   bump?). Impact: Medium. Effort: S. Category: Hygiene.
9. Failure-path test for update-styles (break app.css in a sandbox copy,
   expect nonzero). Impact: Medium. Effort: S. Category: Testing.
10. Update css_gate_test.go's error message to name `nix run
    .#update-styles` (it prints the now-sub-canonical raw tailwind command).
    Impact: Medium. Effort: S. Category: Docs.
11. `nix flake check` with builds once load permits. Impact: Medium.
    Effort: M. Category: Verification.
12. Diff the 17 cqrs-lint findings vs docs/cqrs-lint-policy.md (prior f.5).
    Impact: Medium. Effort: M. Category: Quality.
13. After binary rebuild: verify BuildFlow HEAD actually carries
    attribute-aware hash matching (read the source, then re-tag the
    upstream-filings entry). Impact: Medium. Effort: S. Category: Tooling.
14. Record the new BuildFlow binary hash in the gate ledger (prior f.24).
    Impact: Low. Effort: S. Category: Tooling.
15. Migrate + delete legacy ~/.cache/buildflow/buildflow.db (prior f.13).
    Impact: Low. Effort: S. Category: Hygiene.
16. Enumerate the 9 unavailable buildflow tools; per tool: devShell install /
    exclude with rationale / accept (prior f.6). Impact: Medium.
    Effort: M. Category: Tooling.
17. Add prettier/shellcheck/vulnix/lychee to the devShell (they currently run
    via bare `nix run nixpkgs#…` without project deps). Impact: Medium.
    Effort: S. Category: Tooling.
18. Triage nix-build-verify 12/12 and nix-hash-fix 104/124 failure records
    after the binary rebuild (prior f.11/12). Impact: Medium. Effort: M.
    Category: Tooling.
19. go-auto-upgrade: schedule the encoding/json v2 adoption decision for the
    22 flagged json-v1 boundaries (AGENTS: env var becomes load-bearing
    then). Impact: Medium. Effort: L. Category: Feature.
20. jscpd: extract scripts/lib launch + wait-for-server helpers (prior
    f.15). Impact: Low. Effort: S. Category: Cleanup.
21. file-size-check triage: split internal/web/server.go (618) and
    internal/search/search.go (444); exclude generated `*_templ.go` with
    rationale (prior f.16). Impact: Low. Effort: M. Category: Cleanup.
22. branching-flow: judge CallView 17 fields vs threshold (prior f.17).
    Impact: Low. Effort: S. Category: Quality.
23. art-dupl 98 findings: schedule the dedup pass (prior f.18).
    Impact: Low. Effort: L. Category: Cleanup.
24. Annotate the 16-32 / 17-40 / 18-06 reports + this one once the full gate
    goes green (docs-health ANNOTATE). Impact: Low. Effort: S.
    Category: Docs.
25. Sweep other committed codegen artifacts for dual-format risk
    (.codegen-inputs.sha256 consumers, templ-components-classes.txt) —
    prior f.28; styles.css was case #1 of the class.
    Impact: Medium. Effort: M. Category: Prevention.
26. Verify .github/dependabot.yml really excludes templ (prior f.20).
    Impact: Low. Effort: S. Category: Hygiene.
27. Confirm the 2027-01 Telnyx re-verify calendar entry exists (prior f.21).
    Impact: Low. Effort: S. Category: Ops.
28. CI parity decision for advisory-only checks (file-size/branching/dup) or
    document BuildFlow-as-only-gate (prior f.23). Impact: Low.
    Effort: S. Category: Quality.
29. Pre-commit guard idea: refuse a styles.css commit that is not the pinned
    minified build (cheap content probe; complements f.6).
    Impact: Medium. Effort: M. Category: Prevention.
30. Pre-commit hook health check (templ version drift, prior f.29).
    Impact: Low. Effort: S. Category: Hygiene.

## g) Questions I cannot figure out myself

1. **Who/what produced the two flag-less tailwind builds (17:0x and
   18:03:59)?** You manually, another agent session, or a `--watch` dev
   loop? I checked: no tailwind/templ process was alive at 18:15, and the
   builds carry the pinned v4.3.3 — so it was a `--minify`-less invocation,
   but only you know the actor. The answer decides whether the entrypoint
   guard suffices or needs the hard pre-commit/CI enforcement (f.29).
2. **Full-gate rerun policy: wait or override?** 15-min loadavg was ~52
   (> 40 threshold) at 18:37. Strictly wait for < 40, or do you want the
   full gate run NOW despite the documented load-starvation risk?
3. **May I rebuild + reinstall the BuildFlow binary from
   /home/lars/projects/BuildFlow HEAD as part of pbx-lens remediation?** The
   buildflow skill reserves BuildFlow development for BuildFlow sessions, but
   the stale binary is what bit this repo twice today — your call whether
   that crosses the line or you handle it in that repo.
