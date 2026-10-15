# BuildFlow full-gate triage — session report

- **Written:** 2026-10-15 18:06 CEST
- **Scope:** this session only — triage of the 17:08:34
  `buildflow --fix --build-mode=full --budget 5m --max-time 5m --go-upgrade-blame all`
  run (exit 69, 59/70 steps, 1 failed step, 5 tools with remaining findings).
  No other research was done; everything below was observed in this session.
- **Outcome:** the one hard failure (test-race CSS gate) was already healed on
  disk before diagnosis; both gate-blockers verified green after scoped reruns;
  all remaining findings are pre-documented advisories. Full-mode gate rerun
  deferred by the load policy (loadavg 70-77 vs 15-min threshold 40).

---

## a) FULLY DONE

1. **test-race CSS gate failure root-caused and verified green.**
   `TestCSSGateBridgesHeatmapBrandToTeal` failed at 17:08 because the on-disk
   `internal/web/static/styles.css` was an **unminified, foreign-tailwind
   build** (`--ds-brand-rgb: 15, 118, 110`, space after colon) while the gate
   pins the minified token format (`--ds-brand-rgb:15,`). The committed copy
   was already canonical (regenerated 17:11 with the flake-pinned
   tailwindcss `--minify`, daemon-committed as 2b0bcb2 at 17:12:54 — 1 line,
   −5413). Verified three ways: direct `go test` on the gate, full
   `GOEXPERIMENT=jsonv2 go test -race -shuffle=on ./...` (14/14 packages ok,
   internal/web included), and a scoped
   `buildflow -s test-race` rerun (exit 0, all packages ok).
2. **AGENTS.md size error cleared.** The 17:08 run errored on 221 lines
   (max 220). A concurrent session had already trimmed to 218 by 17:22
   (a7dd1bf); this session added the `--minify` rebuild note (218→220), then
   compressed the buildflow-dev-gate bullet to one line (220→219). Scoped
   `buildflow -s go-structure-linter` rerun: exit 0.
3. **AGENTS.md 221-off-by-one root-caused at source level.** BuildFlow
   counts lines as `strings.Count(data, "\n") + 1`
   (BuildFlow HEAD d114f28, `discovery/doctor/checks/agents_md.go:51`) —
   +1 phantom line for any POSIX trailing-newline file. The bug exists at
   HEAD, not only in the stale binary. Repo-side convention recorded:
   keep AGENTS.md at ≤219 `wc -l` lines until fixed upstream
   (docs/buildflow-dev-gate.md).
4. **go 1.27.2 directive re-verified as dep-forced (non-fix).** go-appkit
   v0.9.0's own go.mod declares `go 1.27.2` (checked in the module cache), so
   the main module must carry the patch. The stale `go 1.27.1 / v0.8.0`
   version references in docs/buildflow-dev-gate.md were refreshed to the
   current reality, with a note that the 2026-10-15 re-flag needs
   re-verification after the fleet binary rebuild.
5. **Stale "gitignored styles.css" claims fixed on sight.**
   docs/gotchas.md (embed/MIME entry) and the flake.nix preBuild comment both
   still claimed styles.css is gitignored; it is COMMITTED since the
   2026-10-15 v0.2.0 CI fix. Both rewritten; the gotcha entry now carries the
   full failure signature (missing → MIME trap; mis-built/unminified →
   "lost the ds-brand bridge" with every token actually present).
6. **flake.nix hash findings dispositioned.** Line 41 already imports
   `vendorHash.nix` (main package, per convention + repair app); the line-103
   inline hash is cqrs-lint's own FOD from the pinned upstream rev — kept
   inline deliberately (a second hash file would conflate the repair app's
   target), matching the documented "nix-checker hash bookkeeping = advisory"
   policy.
7. **Repair-step side effects reconciled.** go.sum rewrites by
   go-mod-tidy/go-mod-update left no residue: no go.mod/go.sum commits since
   16:00, `go mod tidy -diff` clean, and `nix build` (exit 0, binary runs,
   `pbx-lens 0.2.0`) re-proved the vendorHash still matches.
8. **Format hygiene.** dprint fmt clean on all edited files; flake.nix
   parses; daemon committed all edits (577b6cb last); working tree clean.

## b) PARTIALLY DONE

1. **Gate-green claim is scoped, not full.** Under load average 70-77
   (threshold: 15-min < 40 for full runs, paid-for doctrine), only the two
   formerly-failing steps were rerun under the gate's own harness plus the
   direct full test suite. A full `--build-mode full` rerun (with
   `--max-time 15m`, NOT the 5m used at 17:08 — the doc says full needs ≥15m)
   is still owed when the machine quiets down.
2. **BuildFlow binary rebuild (BuildFlow#56).** The preflight warns the
   binary (518739c) is 101 commits behind BuildFlow HEAD (d114f28);
   `buildflow upgrade` refuses on a dirty checkout. Documented, not done.
   Until rebuilt, the go/govalid gate steps stay broken against the 1.27.2
   floor and advisory severities may mis-report.
3. **The go-version severity claim in the ledger is UNVERIFIED.** I wrote
   that the 2026-10-15 warning flag is "from the stale fleet binary" — I
   verified the binary's AGE but never verified that the dep-forced-floor
   severity policy differs at HEAD. Honest flag: that sentence is a
   hypothesis, not a fact; verify after rebuild.
4. **cqrs-lint 17 findings assumed documented.** I did NOT diff the actual 17
   against docs/cqrs-lint-policy.md this session (the first five shown match
   the known classes). Scoped `nix develop -c cqrs-lint .` + policy diff
   still owed.
5. **9 unavailable gate tools only half-triaged.** The run warned
   ("use --verbose for details"); I saw interrogate missing + prettier/lychee
   not in the devShell, but never ran `buildflow -s test-race --verbose` /
   `buildflow doctor` to enumerate all 9 and decide install-vs-exclude each.
6. **Upstream BuildFlow fixes.** The agents_md counter off-by-one is
   diagnosed but not fixed/filed in the BuildFlow repo (out of this repo's
   scope per the buildflow skill).

## c) NOT STARTED

1. Full-mode gate rerun under load threshold with a 15m budget.
2. BuildFlow binary rebuild + re-baseline of all advisory warnings.
3. Upstream fix (or issue) for the agents_md.go:51 counter.
4. jscpd's 2 duplications in scripts/ (screenshot-probe vs
   verify-staleness-live; verify-style-contract vs verify-ui-contract) —
   extract a shared scripts/lib helper.
5. HARVEST of section (f) below into TODO_LIST.md (docs-health) — the user
   instructed report-then-wait, so it is deliberately deferred.
6. Legacy buildflow.db cleanup at ~/.cache/buildflow/buildflow.db (preflight
   storage-layout warn; migrate via a new-binary pipeline run first).
7. Explicit reconciliation of `nix-flake-update ran` — flake.lock state was
   never deliberately inspected this session (only indirectly: clean tree,
   green build).

## d) TOTALLY FUCKED UP

1. **styles.css artifact provenance is uncontrolled.** Within one hour the
   repo committed two wildly different builds of the same file (5413-line
   unminified → 1-line minified) and a gate failed on the difference. Any
   PATH tailwind silently clobbers the pinned-bundle artifact; nothing in the
   tooling prevents a repeat — the only guard is prose (.gitignore note,
   gotchas entry, AGENTS.md line). The failure was caught (gate worked as
   designed), but the class is one lucky commit away from recurring.
2. **My go-directive probe was methodologically wrong.** I tested `go 1.27`
   with `GOTOOLCHAIN=local` without first checking WHICH go binary that
   would be — the bare shell resolves go 1.26.8 (user profile) with
   auto-switch to 1.27.2, so the probe failed for the wrong reason and I
   briefly misread the result. The conclusion (patch is dep-forced) was right
   via module-cache evidence, but the experiment itself was waste and
   confusion. Should have run `GOTOOLCHAIN=local go version` first.
3. **An unverified causal claim slipped into the ledger** (the
   "stale binary caused the go-version re-flag" sentence, see b.3). That is
   exactly the verify-external-claims failure mode the fleet doctrine bans.
   Self-caught during this report's writing; marked in the doc as needing
   re-verification, but it should not have been written as a statement.

## e) WHAT WE SHOULD IMPROVE

1. **Verify before writing.** Causal attributions (what the tool would do at
   HEAD, why a warning appeared) must be source-checked before entering
   docs/buildflow-dev-gate.md — that file is the fleet's memory and a wrong
   entry misleads every future triage.
2. **Probe design discipline.** Establish tool identity (`go version` under
   the exact env of the experiment) BEFORE designing a directive experiment;
   one command would have saved a flawed test + revert cycle.
3. **Scoped step runs must capture exit codes**, not `| tail` excerpts — the
   first `buildflow -s test-race` rerun hid its verdict and had to be rerun.
4. **Artifact single-entrypoint.** Give styles.css the same treatment as
   vendorHash: one named entrypoint (e.g. a flake app `update-styles`
   mirroring `update-vendor-hash`) so the pinned build is one command, and
   the canonical/minified property is enforced by the tool, not by prose.
5. **Post-`--fix` reconciliation checklist.** Every repair step that ran
   (go-mod-tidy, go-mod-update, nix-flake-update, nix-fmt, templ-fmt) should
   get a 30-second side-effect check before the session closes; this session
   did it for go.sum but skipped flake.lock.
6. **Findings-vs-policy diffs.** When a tool reports N findings and a policy
   doc claims to document them, diff the sets instead of pattern-matching the
   first few (owed for cqrs-lint, see b.4).
7. **Concurrent-session hygiene.** Three other sessions were committing while
   this one edited AGENTS.md/flake.nix/gotchas.md. No collision occurred, but
   the count of overlapping doc edits today (three sessions touched AGENTS.md
   within an hour) argues for re-reading a file immediately before editing
   even if it was read minutes ago — the 220→219 trim only worked because the
   count was re-checked after the concurrent 17:22 trim.

## f) Next things (brainstorm — impact-ordered, not commitments; HARVEST

should route these into TODO_LIST.md / ROADMAP.md)

1. Rebuild the BuildFlow binary from a clean checkout (BuildFlow#56), then
   `buildflow upgrade` — unbreaks go/govalid gate steps and re-baselines all
   advisory severities.
2. After rebuild: re-run the full gate (`--build-mode full --max-time 15m`)
   when 15-min load < 40, ideally with BUILDFLOW_NO_RESULT_CACHE=1.
3. Fix the agents_md.go:51 off-by-one upstream (drop the `+1` when data ends
   with `\n`), then relax the repo convention back to ≤220 real lines.
4. Re-verify the go-version-auto-configure severity behavior at HEAD
   (b.3) and correct the ledger sentence if wrong.
5. Diff the 17 cqrs-lint findings against docs/cqrs-lint-policy.md; file or
   document any undocumented survivor.
6. Enumerate the 9 unavailable tools (`buildflow doctor` + verbose run) and
   decide per tool: install into devShell, exclude with rationale, or accept.
7. Add prettier + lychee to the devShell (buildflow runs them via bare
   `nix run nixpkgs#…` WITHOUT project deps today).
8. Decide interrogate's fate (missing from PATH; warning every run).
9. Give styles.css a single pinned rebuild entrypoint (flake app
   `update-styles` mirroring `update-vendor-hash`); make AGENTS.md point at
   it only.
10. Consider making the CSS gate format-tolerant (normalize whitespace before
    comparing tokens) IF unminified builds keep happening — tradeoff noted in
    g.2; default remains strict.
11. Investigate nix-build-verify's 12/12 (100%) failure record — OOM per the
    run's own triage hint; fix, budget, or exclude with rationale.
12. Investigate nix-hash-fix's 104/124 (84%) failure record — likely the same
    OOM/budget class; the run's summary says "consider investigating or
    excluding".
13. Migrate the legacy buildflow state DB (run one pipeline with the rebuilt
    binary), then delete ~/.cache/buildflow/buildflow.db.
14. Reconcile flake.lock deliberately after the next `--fix` that runs
    nix-flake-update (what did it bump? was it wanted?).
15. Extract the 2 jscpd duplications in scripts/ into scripts/lib (launch +
    wait-for-server helpers are the repeated blocks).
16. file-size-check: 13 files >350 lines — triage each: split
    internal/web/server.go (618) and internal/search/search.go (444) along
    handler/query seams; EXCLUDE `*_templ.go` (generated) from the check via
    config with rationale; tests (836/446 lines) — split or accept+document.
17. branching-flow: CallView at 17 fields (threshold 15) — judge whether the
    view genuinely decomposes or the threshold gets a documented bump.
18. art-dupl 98 findings — schedule the deferred dedup pass (the ledger says
    "revisit if a dedicated pass is scheduled"; this is the scheduling).
19. HARVEST this report + the 17:40 harvest-pass report into TODO_LIST.md
    (docs-health HARVEST mode).
20. Verify .github/dependabot.yml actually excludes templ (AGENTS.md says
    manual-only bumps because a dependabot templ PR mangles go.mod — the
    exclusion was asserted, never checked this session).
21. Confirm the 2027-01 Telnyx live re-verify window exists as a REAL
    calendar entry (AGENTS.md says "calendar entry, m27.9" — the repo cannot
    prove a calendar; see g.3).
22. Copy the .gitignore styles.css incident note's essence into
    docs/gotchas.md by cross-link (one line) so the war story is not entombed
    in a dotfile.
23. CI parity check: file-size/branching/dup checks exist only as buildflow
    advisories — decide whether ci.yml should carry any of them (or document
    BuildFlow-as-the-only-gate for those classes).
24. After the binary rebuild, record the new binary hash in the gate ledger
    so "binary built at X" preflight warnings are instantly triageable.
25. Add a scoped make-target-style flake app for the documented scoped rerun
    pair (`-s test-race` + `-s go-structure-linter`) used by this triage —
    it is now the de-facto post-incident verification ritual.
26. Review whether `--budget 5m` class commands should warn loudly when
    full mode is requested (the 17:08 run's 5m cap contradicts the ≥15m
    doctrine; a preflight check would have caught it).
27. Annotate the 2026-10-15 status reports (16-32 self-review, 17-40
    harvest-pass) once today's gate work settles — they reference a red gate
    that is now green.
28. Sweep for any OTHER committed artifact with dual-format risk like
    styles.css (generated-but-committed files: check .codegen-inputs.sha256
    consumers and templ-components-classes.txt staleness).
29. Consider `buildflow precommit install` health check — the sandboxed
    pre-commit check chain (treefmt → templ fmt) has bitten twice; verify the
    hook still matches current templ versions.
30. Keep AGENTS.md at ≤219 lines until item 3 lands; add a one-line reminder
    comment is NOT possible (line budget) — instead the convention lives in
    the gate ledger (done) and should be re-checked on every AGENTS.md edit.

## g) Questions I cannot figure out myself

1. **Who/what produced the unminified styles.css on disk before 17:08?** A
   manual `tailwindcss` run without `--minify` by you, or another agent
   session using a PATH tailwind? The .gitignore note says it was
   "sharpened by" today's incident — so someone diagnosed it concurrently.
   Knowing the source decides whether item 9 (single entrypoint) is a
   convenience or a hard requirement.
2. **Should the CSS gate stay format-strict?** Strict caught today's
   non-canonical artifact but produced confusing "lost the ds-brand bridge"
   messages for a semantically-correct file. Do you want it to stay a
   canonical-artifact pin (my current read), or become format-tolerant and
   move the canonical-pin elsewhere?
3. **Fleet policy for the two BuildFlow bugs (counter off-by-one, and the
   go-version dep-forced-floor handling):** fix directly in the BuildFlow
   repo as a dev task there, or file issues to your own repo first? I can
   prepare either; BuildFlow development is out of scope for pbx-lens
   sessions by the skill's own rule, so this is your call.

---

## Verification appendix (what was actually run)

| Check                                                                 | Result                           |
| --------------------------------------------------------------------- | -------------------------------- |
| `GOEXPERIMENT=jsonv2 go test -race -shuffle=on ./...` (devShell)      | 14/14 packages ok                |
| `buildflow -s test-race` (scoped, no result cache)                    | exit 0, all packages ok          |
| `buildflow -s go-structure-linter` (scoped, twice: before/after trim) | exit 1 → exit 0                  |
| `go mod tidy -diff`                                                   | clean                            |
| `nix build` + `./result/bin/pbx-lens version`                         | exit 0, `pbx-lens 0.2.0`         |
| `nix-instantiate --parse flake.nix`                                   | ok                               |
| `dprint fmt` on all edited files                                      | clean                            |
| AGENTS.md line count                                                  | 219 (`wc -l`), tool-visible ≤220 |

**Files touched this session:** AGENTS.md, docs/gotchas.md,
docs/buildflow-dev-gate.md, flake.nix (comments/docs only — zero Go/Nix
semantic changes; the gate failure was healed by the concurrent 17:11 regen,
this session verified and documented it).
