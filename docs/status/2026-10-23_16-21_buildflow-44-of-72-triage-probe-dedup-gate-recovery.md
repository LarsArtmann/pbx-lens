# Status Report — BuildFlow 44/72 Run Triage, Probe-Bootstrap Dedup, Gate Recovery

**Generated:** 2026-10-23 16:21 CEST
**Session scope:** triage of the 14:47 `buildflow --fix --build-mode=full` run (44/72, 4 step
failures, exit 69) through the recovered dev-gate state (56/66, sole failure = external NVD
network). Repo: pbx-lens @ 45649d6 (auto-commit daemon, clean tree at write time).
**Companion runs:** a concurrent session was landing web-layer commits and monitor365/telephony
builds saturated the machine (loadavg 148 → 56 over the session) — both shaped every finding.

Verdict line at session end: `✗ BuildFlow failed 56/66 (1 failed: vulnix — nvd.nist.gov read
timeout)`, every other step green including full Go suite (14 pkgs, plain + `-race`), nix build
(vendor FOD rebuilt), golangci-lint 0 issues, all 6 browser instruments, jscpd 0 clones.

---

## a) FULLY DONE

1. ~~**Root-cause triage of all 4 failed steps.** test-race, nix-build, and the vulnix crash all~~ done (docs-health pass 2026-10-26)
   ~~traced to the concurrent session's mid-refactor dirty tree (failing drv was literally~~
   ~~`pbx-lens-e1b0fe1-dirty`); go-fix + license-check traced to BuildFlow's stale binary~~
   ~~(ed9ce48) bundling go 1.27.0 vs go-appkit v0.9.0's 1.27.2 floor. No repo defect behind any~~
   ~~of the original four.~~
2. ~~**internal/web detail-join failures** — verified transient: full `go test ./...` green (14~~ done (docs-health pass 2026-10-26)
   ~~packages) both plain and with `-race` on the settled tree.~~
3. ~~**nix-build failure** — green again end-to-end: package build, vendor FOD rebuild, checks~~ done (docs-health pass 2026-10-26)
   ~~(`build`, `format`, `pre-commit`, `treefmt`, `vendor-hash`), and `nix run . -- version` →~~
   ~~`pbx-lens 0.2.0`.~~
4. ~~**vendorHash "stale" advisory dispositioned as a heuristic misfire** — the go.mod change was~~ done (docs-health pass 2026-10-26)
   ~~the go _directive_ only (no dep change); hash verified valid by green builds. TODO_LIST row~~
   ~~deleted per the "done → CHANGELOG" doctrine.~~
5. ~~**go.mod `go 1.27.2` confirmed canonical** — an attempted `go 1.27` + `toolchain go1.27.2`~~ done (docs-health pass 2026-10-26)
   ~~shape was folded back by `go mod tidy` (dependency-floor promotion, go-appkit v0.9.0).~~
   ~~BuildFlow's "major.minor only" warning is unfixable repo-side while go-appkit pins 1.27.2;~~
   ~~documented as accepted noise in docs/gotchas.md.~~
6. ~~**BuildFlow binary rebuilt + reinstalled** (ed9ce48 → 3509b30, its own preflight remediation)~~ done (docs-health pass 2026-10-26)
   ~~— binary-freshness warning cleared; go-fix and license-check toolchain-skew failures~~
   ~~eliminated.~~
7. ~~**golangci-lint → 0 issues** — fixed two findings left by the concurrent session's web work:~~ done (docs-health pass 2026-10-26)
   ~~`exhaustive` (siblingKinds missing Telnyx/webhook cases, queries.go:344 — explicit cases,~~
   ~~behavior-identical) and `golines` (silence_only_test.go:29, via `golangci-lint fmt`).~~
8. ~~**nix-fmt roundtrip failure fixed repo-side** — BuildFlow's detect copies only flake.nix +~~ done (docs-health pass 2026-10-26)
   ~~flake.lock + literal treefmt configs into a temp dir; this repo's treefmt config comes from~~
   ~~treefmt-nix with `projectRootFile = "go.mod"`, absent there. Switched to~~
   ~~`projectRootFile = lib.mkForce "flake.nix"` (flake.nix:114). Single-step ✓; full gate ✓.~~
9. ~~**jscpd: 3 clone pairs → 0 (verified `✓ BuildFlow passed 1/1`)** — extracted into~~ done (docs-health pass 2026-10-26)
   ~~scripts/lib/ui-probe.js: `arg()`, `fail()`, `probeSession()` (shared --port/--out/DB/BASE~~
   ~~preamble), `launchProbe()` (codegen → puppeteer → binary → seed → serve → 1280×900 page,~~
   ~~with seedOpts/serveArgs/serveEnv/resetDb variants). Rewired verify-ui-contract,~~
   ~~verify-style-contract, verify-staleness-live, verify-ops-verdict, csp-sweep;~~
   ~~screenshot-probe keeps its bespoke multi-phase flow (only its arg parser delegates).~~
10. ~~**All 6 browser instruments verified green post-refactor** — all four contract probes via~~ done (docs-health pass 2026-10-26)
    ~~`verify-all.sh --fast` (plus individual reruns), csp-sweep (zero violations, 6 surfaces),~~
    ~~screenshot-probe minimal run (--only dashboard, arg-helper path).~~
11. ~~**vulnix repo-side cause fixed** — the "unscannable store path" was the stale `./result`~~ done (docs-health pass 2026-10-26)
    ~~from a dirty build; `nix build .# -o result` refreshed the scan target; `vulnix` added to~~
    ~~devShells.default (flake.nix) so BuildFlow stops falling back to `nix run nixpkgs#vulnix`.~~
12. ~~**govulncheck dispositions** — 13 reported findings collapse to ONE affecting vuln:~~ done (docs-health pass 2026-10-26)
    ~~GO-2026-6809 (crypto/x509), found in go1.27.2, fixed in go1.27.3. The repo's environments~~
    ~~(nix locked go_1_27 = 1.27.3; CI setup-go latest-patch) already ship the fix — the report~~
    ~~is a stale-global-toolchain artifact. Documented in gotchas.md.~~
13. ~~**Formatter lanes clean** — oxfmt --fix + dprint-format --fix applied (including line-wrap~~ done (docs-health pass 2026-10-26)
    ~~of my own gotchas entry); dprint step `✓ 1/1`.~~
14. ~~**Docs updated in-place:** CHANGELOG (new 2026-10-23 gate-triage section), docs/gotchas.md~~ done (docs-health pass 2026-10-26)
    ~~(two new war stories: tidy-owns-the-go-directive; concurrent-dirty-tree fakes red gates),~~
    ~~scripts/README.md (probeSession/launchProbe ownership), AGENTS.md devShell tool list.~~
15. ~~**Nothing corrupted, nothing destroyed** — the 2026-10-15 vendorHash-corruption class did~~ done (docs-health pass 2026-10-26)
    ~~NOT recur; the hash-fix guard held; tree committed clean by the daemon throughout.~~

## b) PARTIALLY DONE

1. ~~**vulnix step** — repo-side cause fixed (fresh target, devShell binary), but the step is~~ done — NVD recovered; vulnix green in the 2026-10-25/26 FULL receipts (60/0/0) — 22 advisories are now the finding class (17-29 §f31)
   ~~still red on `nvd.nist.gov` read timeouts (verified twice via direct run). The original~~
   ~~run's vulnix failure loop (4 consecutive) is broken environmentally, not by a green scan.~~
2. ~~**Probe-suite verification** — all lanes green individually and via `--fast`; the FULL~~ done — 42-PNG full sweeps green repeatedly (2026-10-24 lane; 19-09 v5 baseline; inside the v0.3.0 release runbook)
   ~~41-PNG screenshot sweep + golden diff was NOT re-run (repo doctrine requires it before~~
   ~~tagging or after visual changes — no visual change was intended, but the bootstrap all~~
   ~~probes share was rewritten).~~
3. ~~**Upstream filings** — two new filing candidates have verified diagnoses and gotcha entries~~ done — routed → the upstream-filings queue row (TODO_LIST); dispositions recorded in docs/buildflow-dev-gate.md
   ~~(BuildFlow nix-fmt/treefmt projectRootFile roundtrip; go-version-auto-configure vs tidy~~
   ~~floor promotion) but are NOT filed — filings are owner-go-ahead-gated per~~
   ~~docs/upstream-filings.md.~~
4. ~~**This report's (f) list → TODO_LIST** — not harvested; you asked for report-then-wait, so~~ done — harvested by the 2026-10-23/24 passes; this pass re-dispositions the list
   ~~HARVEST awaits your instruction (docs-health HARVEST routing rules apply; most items below~~
   ~~are brainstorm, not commitments).~~
5. ~~**Timing baselines** — the "regression verdict" numbers (+55%…+533%) are load artifacts;~~ done — quiet-box re-measure landed 2026-10-25 (docs/http-stack.md both generations)
   ~~no re-baseline was taken now that load is dropping (15m: 79 at last gate).~~

## c) NOT STARTED (noticed this session, untouched)

1. ~~**art-dupl: 107 detect-only findings** — never reviewed (warning-only lane, appeared in the~~ done — the 2026-10-24 dedup passes audited the detect and suppressed buckets and recalibrated to the zero-group gate (12-33/13-30/18-59)
   ~~dev-gate findings list).~~
2. ~~**branching-flow: 100 detect-only findings** — never reviewed.~~ done — routed → the detect-only advisory triage row (TODO_LIST, added this pass; 17-29 f28 names the branching-flow hotspots)
3. ~~**cqrs-lint: 20 findings** — not re-checked against docs/cqrs-lint-policy.md (surviving set~~ done — re-audited 2026-10-26 (15-09 report: 19 infos, policy-decided); the CBOR warning is routed (17-29 f29)
   ~~may have drifted from the documented decisions).~~
4. ~~**file-size-check: 12 findings** — never reviewed.~~ done — routed → the detect-only advisory triage row (17-29 f26 names the file-size splits)
5. ~~**The 9 "tools unavailable (health check failed)"** — never identified (both the original~~ done — routed → the gate-doc tooling-triage entry (docs/buildflow-dev-gate.md; the BuildFlow#56 TODO row tracks the binary lane)
   ~~full run and every dev gate carry the warning; `--verbose` was never pulled).~~
6. ~~**Two BuildFlow upstream filings** (see b3) — not filed.~~ done — routed → the upstream-filings queue row (TODO_LIST)
7. ~~**BuildFlow nix-hash-fix 82% failure rate** (122/149, preflight flag) — investigation not~~ done — routed → the owner question nix-hash-fix (17-29 g2; 145/180 failures, two corruptions)
   ~~started; BuildFlow-side.~~
8. ~~**BuildFlow resume checkpoint SQLITE_BUSY under load** (the 14:48 warning storm) — not~~ **NOT-DO — no recurrence through the 2026-10-26 receipts; re-open only on a fresh SQLITE_BUSY storm.**
   ~~investigated/filed; BuildFlow-side.~~
9. ~~**Fleet sweep** — every other LarsArtmann Go flake using `projectRootFile = "go.mod"` will~~ done — routed → the upstream-filings queue row (fleet impact documented with the projectRootFile candidate)
   ~~hit the same nix-fmt roundtrip failure now that BuildFlow 3509b30 is installed fleet-wide.~~
10. ~~**A `siblingKinds` behavior test** — the exhaustive switch is pinned only by the lint rule,~~ done — the join pairs are pinned by a table test (detail_join_test.go:249)
    ~~not by a table test of the join pairs.~~
11. ~~**JS linting for scripts/** — probes have formatters (dprint/oxfmt) but no linter; the~~ done — scripts/check-js-lint.sh landed 2026-10-25 (oxlint no-undef; CI Guard Self-Tests job)
    ~~`bin is not defined` class of bug would have been caught by eslint/oxlint unused/no-undef.~~

## d) TOTALLY FUCKED UP (my own work this session — honest ledger)

1. ~~**launchProbe shipped without `bin`** — the dedup refactor broke verify-staleness-live~~ **NOT-DO — process note; the audit-before-execute rule is e2.**
   ~~(`ReferenceError: bin is not defined`), caught only because verify-all ran. I broke the~~
   ~~verifiers while deduplicating them; the orphan-reference audit that caught it was run AFTER~~
   ~~the failure — it should have run BEFORE the first lane execution.~~
2. ~~**Misread my own edit history** — in the probeSession round I wrote header old_strings in~~ **NOT-DO — process note.**
   ~~the delegation form for staleness/ops, but round 1 had only replaced their `fail` function.~~
   ~~The "1 of 2 edits failed" result sent me into forensics (git show, daemon/formatter~~
   ~~suspicion) — ~4 tool calls burned rediscovering my own changes. The commit at 7f6d537 was~~
   ~~exactly what I had made; nothing was reverted.~~
3. ~~**Fought `go mod tidy`** — edited go.mod to `go 1.27` + `toolchain go1.27.2` without first~~ **NOT-DO — process note; tidy-owns-the-directive is a gotcha.**
   ~~checking tidy's dependency-floor promotion; tidy folded it back within the same command~~
   ~~chain. One wasted edit-and-revert cycle.~~
4. ~~**`nix run … | head -3` SIGPIPE'd a nix build** — the verification run for `pbx-lens~~ **NOT-DO — process note.**
   ~~version` was killed by my own pipe truncation and had to be repeated.~~
5. ~~**Re-ran the entire ~2-minute dev gate just to read its failure list** instead of pulling~~ **NOT-DO — process note.**
   ~~the failure section from the already-captured output — wasteful under a loaded machine.~~
6. ~~**Grep-filtered gate outputs hid confirmation** — I declared jscpd zero from structural~~ **NOT-DO — process note; verify-your-own-gate adopted (e7).**
   ~~reasoning but the final gate's grep never showed its line; only a dedicated~~
   ~~`buildflow -s jscpd` after the fact proved `0 findings`. Verify-your-own-gate with full~~
   ~~output, not excerpts.~~

## e) WHAT WE SHOULD IMPROVE

1. ~~**Concurrent-session preflight:** before triaging ANY gate red, snapshot~~ done — adopted — the concurrency pre-flight norm (docs/concurrency.md; ratified f57cebe)
   ~~`git log --since <run-start>` and re-run failing steps against the settled tree. This one~~
   ~~habit would have collapsed the entire "4 failures" chase to minutes. (Now a gotcha; make it~~
   ~~a reflex.)~~
2. ~~**Audit-before-execute for mechanical refactors:** the removed-locals audit~~ done — adopted — mechanical refactors run the orphan audit before the first lane run
   ~~(`bin`/`puppeteer`/`args` references) belongs before the first lane run, not after the first~~
   ~~red.~~
3. ~~**Reason about tool invariants before hand-edits:** `go mod tidy` owns the go directive;~~ done (docs-health pass 2026-10-26)
   ~~any manual shape must survive tidy or it is not a fix.~~
4. ~~**Keep an edit-ledger in multi-round refactors** (file → which round changed what) —~~ **NOT-DO — superseded by immediate re-view discipline (the edit-tool mod-time-bounce rule, 19-09 d3).**
   ~~prevents old_string drift and the self-suspicion spiral.~~
5. ~~**Authoritative checker over ambient diagnostics:** vtsls reported errors at lines past EOF~~ done — adopted — authoritative checker over ambient diagnostics (go vet disproved the LSP, 19-09 d3)
   ~~for ~30 minutes of edits; after the first two `node --check` contradictions, default to the~~
   ~~authoritative checker immediately.~~
6. ~~**Read the tool's source early:** BuildFlow's roundtrip mechanism sat greppable in~~ done — adopted — scripts-first convention (15-09 d4)
   ~~/home/lars/projects/BuildFlow; "read the tool" should precede "re-run the step" when a~~
   ~~failure mentions a harness (`/tmp/buildflow-roundtrip-*`).~~
7. ~~**Capture full gate output for the final verdict** — no grep filters on the run you cite as~~ done — adopted — poll-gate prints the verified triple and the receipt is read against policy (5db1853)
   ~~proof.~~
8. ~~**Batch network-dependent steps** (vulnix, lychee, govulncheck feeds) behind a~~ done — superseded — poll-gate.sh gates the FULL run on the load policy (15m < 40)
   ~~network-quiet precondition instead of letting each fail separately under load.~~

## f) NEXT — up to 50 candidates (brainstorm, NOT commitments; sorted by impact)

1. ~~Re-run vulnix when NVD/network is quiet — rebuild `./result` first (it drifted again: it~~ done — green in every receipt since (2026-10-25/26 FULL 60/0/0)
   ~~points at the 8928aba-dirty closure; HEAD is 45649d6+). This is the last red gate step.~~
2. ~~File BuildFlow upstream: nix-fmt/treefmt round-trip detect should copy the project root~~ done — routed → the upstream-filings queue row (TODO_LIST; docs/buildflow-dev-gate.md carries the diagnosis)
   ~~file (or honor generated treefmt config) — repo workaround in place, fleet still exposed.~~
3. ~~File BuildFlow upstream: go-version-auto-configure "major.minor only" policy vs `go mod~~ done — routed → the upstream-filings queue row (the go-version entry lives in docs/buildflow-dev-gate.md)
   ~~tidy` dependency-floor promotion — request a waiver/detection knob (go-appkit makes it~~
   ~~permanent here).~~
4. ~~Fleet sweep: audit other LarsArtmann Go flakes for `projectRootFile = "go.mod"` + BuildFlow~~ done — routed → the upstream-filings queue row (fleet impact noted with the filing candidate)
   ~~3509b30 — same nix-fmt roundtrip failure is pre-landed for them (go-output, monitor365-adjacent, etc.).~~
5. ~~Identify the 9 "tools unavailable (health check failed)" via `buildflow --verbose` —~~ done — routed → the gate-doc tooling-triage entry (the BuildFlow#56 row tracks the binary lane)
   ~~unknown since the original run.~~
6. ~~Review art-dupl's 107 detect-only findings — dedupe or baseline (warning-only lane, but~~ done — the 2026-10-24 dedup passes replaced the warning lane with the zero-group gate
   ~~107 is a haystack).~~
7. ~~Review branching-flow's 100 findings.~~ done — routed → the detect-only advisory triage row (TODO_LIST, added this pass)
8. ~~Re-check cqrs-lint's 20 findings against docs/cqrs-lint-policy.md — the documented~~ done — 15-09 audit: 19 infos policy-decided; the CBOR residue is routed (17-29 f29)
   ~~surviving set may have grown.~~
9. ~~Review file-size-check's 12 findings.~~ done — routed → the detect-only advisory triage row (17-29 f26 names the splits)
10. ~~Investigate BuildFlow nix-hash-fix 82% failure rate (122/149 preflight warning).~~ done — routed → the owner question nix-hash-fix (17-29 g2)
11. ~~Investigate/file BuildFlow resume SQLITE_BUSY under load (checkpoint batch retry/backoff).~~ **NOT-DO — no recurrence through 2026-10-26; re-open on recurrence.**
12. ~~Add a table test pinning `siblingKinds` join pairs (the exhaustive lint is the only guard).~~ done — detail_join_test.go:249 pins the join pairs
13. ~~Run the FULL screenshot sweep + golden diff before the next tag (probe bootstrap was~~ done — 42-PNG sweeps green 2026-10-24/25/26 (v4/v5 baselines and the release runbook)
    ~~rewritten; doctrine requires it after tooling changes that touch captures).~~
14. ~~Harvest this report's list into TODO_LIST.md (docs-health HARVEST) — owner call on which~~ done — harvested 2026-10-23/24; re-dispositioned by this pass
    ~~rows are commitments.~~
15. ~~Add both new filing candidates to docs/upstream-filings.md queue (2, 3 above).~~ done — the candidates live in docs/buildflow-dev-gate.md plus the filings-queue row
16. ~~Record the NVD-timeout vulnix signature in docs/buildflow-dev-gate.md triage section.~~ done — docs/buildflow-dev-gate.md carries the vulnix-exits-1 network signature
17. ~~Consider a gate preflight `nix build .# -o result` so vulnix's scan target never goes~~ **NOT-DO — superseded — cut-release.sh runs its own nix build (a fresh result at release time).**
    ~~stale between sessions.~~
18. ~~Add JS linting (oxlint/eslint) to scripts/ — would have caught the `bin` undefined bug~~ done — scripts/check-js-lint.sh (2026-10-25)
    ~~pre-run.~~
19. ~~Reset BuildFlow timing baselines once load normalizes (current "regressions" are load~~ done — quiet-box re-measure 2026-10-25 (docs/http-stack.md both generations)
    ~~artifacts polluting the verdicts).~~
20. ~~Exercise `nix run .#update-vendor-hash` on a clean tree once, deliberately, so the repair~~ done — exercised for real 2026-10-26 (18382a2 re-pin: fake-probe, learn, verify-green)
    ~~path is rehearsed before it is next needed for real.~~
21. ~~Decide vulnix gate policy while NVD is flaky: keep red (honest signal — my lean) vs skip~~ done — routed → the ROADMAP open question (gate meaning with a red external step)
    ~~via config (green gate) — owner call.~~
22. ~~Document the daemon + concurrent-session hazard in AGENTS.md (currently gotchas-only) if~~ done — docs/concurrency.md landed; the norm is owner-ratified (f57cebe)
    ~~you want it session-mandatory rather than reference-only.~~
23. ~~Re-verify CI greenness after the queries.go/go.mod-touching commits land remotely (CI uses~~ done — CI green through 2026-10-26 (runs 40221643859, 40222247523, 40359076464)
    ~~setup-go; exhaustive/golines fixes are new code in the CI lane).~~
24. ~~Next templ-components bump window: run the watchlist (gen-tc-classes → map re-audit →~~ done — the v1.24.0 bump executed the watchlist (2026-10-25, 05-49 report)
    ~~hand-rolled re-check → filings re-read → golden -update) — untouched this session.~~
25. ~~Verify the pinned golangci-lint (v2.14) still satisfies the repo language level after~~ done — v2.14 pinned and verified against the 1.27 module level (AGENTS.md)
    ~~nixpkgs drift (the AGENTS gotcha about v2.12-refuses-config will resurface on bumps).~~
26. ~~Check whether any probe README/table rows still reference removed per-script helpers~~ **NOT-DO — no stale helper reference surfaced through the 2026-10-26 gates.**
    ~~(sweep `grep -n "ensureCodegen\|serveProbe" scripts/*.md`).~~
27. ~~Consider making `probeSession`/`launchProbe` the enforced path (a check-nolint-style~~ **NOT-DO — no clone re-growth through 2026-10-26 (art-dupl zero-group); revisit on recurrence.**
    ~~ledger script that greps scripts/ for re-implemented bootstrap blocks) — prevents clone~~
    ~~re-growth.~~
28. ~~Re-run `buildflow history` after a quiet-network vulnix pass to close the failure-loop~~ **NOT-DO — the failure-loop record is closed by the green receipts.**
    ~~record (the original run flagged "4 consecutive runs, same error").~~
29. ~~Sweep other probe scripts for the `--port` pinned-default reproducibility note (README~~ **NOT-DO — defaults verified stable in every probe run since.**
    ~~says defaults remain; verify they still do after probeSession).~~
30. ~~Add `result` staleness (vs flake.lock AND vs HEAD) to the cut-release.sh quiesce guards —~~ **NOT-DO — superseded — the release path builds fresh (the cut-release nix-build step).**
    ~~vulnix silently scans yesterday's closure today.~~
31. ~~Confirm go fix-lint fleet policy for `toolchain` lines (when a dep floor exceeds~~ **NOT-DO — cross-project policy — belongs in crush-config lessons, not this repo.**
    ~~major.minor, is a toolchain line wanted in every module, or is the bare directive the~~
    ~~fleet standard? — affects how the gotcha generalizes).~~
32. ~~Prune the auto-commit daemon's heuristic for mid-refactor commits (the e1b0fe1 commit is~~ **NOT-DO — cross-project (the daemon is fleet infrastructure).**
    ~~what made the failing drv; an opportunistic-commit quiesce window during active sessions~~
    ~~would shrink the dirty-tree hazard class) — BuildFlow/fleet-policy adjacent.~~
33. ~~Check `.golangci.yml`'s exhaustive settings: `default-signifies-exhaustive` would have~~ **NOT-DO — the explicit-case form kept deliberately (better documentation).**
    ~~made siblingKinds' original default-case form legal — decide whether the repo wants that~~
    ~~policy (current explicit-case form is arguably better documentation anyway).~~
34. ~~Sweep for other switches on calls.SourceKind lacking explicit cases (the lint found one;~~ done — the 2026-10-26 full review read every source; no missing-case switch surfaced
    ~~siblings may exist in read-model mappers).~~
35. ~~Re-run `bash scripts/verify-all.sh` (full, not --fast) on a quiet machine as a~~ done — verify-all.sh green in full twice on 2026-10-26
    ~~post-refactor receipt — pairs with 13.~~
36. ~~Confirm whether `nix run .#update-styles` still round-trips after the flake.nix edits~~ done — update-styles used for real (the 2026-10-25 P4-P6 styles regen)
    ~~(untouched paths, but cheap to prove).~~
37. ~~Re-read docs/templ-components-adoption.md version-bump watchlist trigger state (filings~~ done — the watchlist ran with the v1.24.0 sweep
    ~~re-read leg) — adjacent to 24.~~
38. ~~Add a `probeSession` unit smoke (node --test or a --help dry flag) so the preamble has a~~ **NOT-DO — superseded — the oxlint lane plus green probes in every verify-all run.**
    ~~no-browser test at all.~~
39. ~~Check whether BuildFlow's "unmapped treefmt program(s) templ" warning (6 formatters) is~~ done — routed → the buildflow advisory-divergence candidate (filings queue)
    ~~filing-worthy — it fires on every run in this repo.~~
40. ~~Evaluate whether the dev gate should hard-fail (not warn) when 15m load exceeds the~~ done — poll-gate.sh enforces the load policy (15m < 40; 5db1853)
    ~~documented 40 threshold for LOAD-sensitive steps — the starvation verdict is currently~~
    ~~advisory.~~
41. ~~Capture a `git log --since` snippet into future status reports as a standard section~~ done — adopted — concurrent-session notes are standard in the 10-25/26 reports
    ~~(concurrent-session context was the whole story this time).~~
42. ~~Long-tail: bump `golang.org/x/crypto` etc. only via the next dependabot sweep — no action~~ done — dependabot sweeps continued; sqlite v1.61.0 landed via the PR-3-close hand-roll
    ~~now, noting the 0.58.0 pin was untouched this session.~~
43. ~~Confirm `result/bin/pbx-lens` provenance convention post-refresh (`nix build .# -o result`~~ **NOT-DO — behavior verified identical; no doc change needed.**
    ~~vs `.#pbx-lens` — the AGENTS command uses `.#`; behavior verified identical this session).~~
44. ~~Archive this report once dispositioned (docs-health ANNOTATE, not rewrite).~~ done — this pass (2026-10-26 docs-health)
45. ~~Re-verify `nix flake check` after the mkForce (the gate's checks covered build/format/~~ done — flake check green repeatedly (the 2026-10-26 repair receipts)
    ~~pre-commit/treefmt/vendor-hash; a plain `nix flake check` receipt was not separately run).~~
46. ~~Consider recording the probe refactor in ROADMAP's test-complexity policy row (the lib~~ done — ROADMAP carries the test-complexity policy row (2026-10-23 harvest)
    ~~extraction is precedent for shared-instrument ownership).~~
47. ~~Sweep docs for "18107/18110/18116/18118" pinned-port claims after probeSession (defaults~~ **NOT-DO — no stale port claim surfaced through the doc gates.**
    ~~unchanged, but the docs' "when omitted" phrasing predates the helper).~~
48. ~~Ask whether BuildFlow should expose `--step-timeout` per-step config in .buildflow.yml so~~ done — routed → the buildflow advisory-divergence candidate (filings queue)
    ~~vulnix/coverage survive loaded machines without CLI flags.~~
49. ~~One-line gotcha cross-link: gotchas.md's tidy entry ↔ cqrs-lint-policy.md's E003 flat-HOLD~~ done — cross-link added to docs/gotchas.md this pass
    ~~(both are "library contract, not ours" patterns) — cheap doc cohesion.~~
50. ~~Sleep on it: re-read this list in 48h and delete every row that no longer earns its place~~ done — re-read by this pass; the dispositions above are the re-read
    ~~(docs-health HARVEST anti-pattern: timestamped-file entombment).~~

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. ~~**Should I file the two BuildFlow candidates now** (nix-fmt roundtrip projectRootFile;~~ done — routed → the upstream-filings queue row (owner go-ahead pending per filing)
   ~~go-version-policy-vs-tidy)? Both have verified diagnoses and fleet impact (4 above), but~~
   ~~filings in your repos are go-ahead-gated per docs/upstream-filings.md — batch or file?~~
2. ~~**Is the concurrent web-layer session finished?** My "suite green" receipts are~~ done — moot — the concurrent session closed 2026-10-23; every later receipt ran on a settled tree
   ~~point-in-time against its last commit; if more web-layer commits are coming, the detail-~~
   ~~join area should be re-verified after it settles. I cannot see that session's state.~~
3. ~~**vulnix gate policy while NVD misbehaves:** keep the step red as an honest signal (my~~ done — routed → the ROADMAP open question (gate meaning with a red external step)
   ~~lean), or skip it via config until the feed recovers so the gate reads green? This decides~~
   ~~what "gate green" means for the next few days.~~

---

_Point-in-time snapshot; companion material: CHANGELOG 2026-10-23 (gate-triage section),
docs/gotchas.md (two 2026-10-23 entries), scripts/README.md (probe ownership). Session had
NO commit-permission — everything was picked up by the auto-commit daemon (45649d6…)._
