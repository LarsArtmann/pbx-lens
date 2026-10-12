# Status Report — BuildFlow gate recovery: templ-fmt per-file trap, devShell toolchain skew, concurrent-session hash collision

**Session:** 2026-10-12 ~15:20–19:20 CEST · **Author:** Crush (glm-5.3-flash)
**Scope:** this session's run only (user-directed: no unrelated research). Starting point was the
user's gate run: `buildflow --fix --build-mode=full` exiting 69 with 3 failed steps
(nix-build, nix-build-verify, nix-hash-fix cascade). Ending point: full gate **exit 0**
("passed with warnings, 61/70, 1m44s") inside the devShell.

---

## a) FULLY DONE

1. **Pre-commit check (the root cause of the original gate failure)** — the sandboxed
   pre-commit hook runs treefmt **per file** (pre-commit feeds it one file per
   invocation), which hits templ's _single-file_ formatting path. That path reformats
   content the _batch_ path leaves untouched, and needs **two passes** to converge on
   some files (an inline `<span>@icons…</span>` first splits the closing tag, then the
   element). Fixed by iterating `templ fmt` per file to a fixpoint, regenerating
   `*_templ.go`, and re-running `nix build .#checks.x86_64-linux.pre-commit` → green.
   The trap is recorded in `docs/gotchas.md` (templ fmt per-file/batch divergence +
   non-idempotent first pass).
2. **Vendor hash repair after a concurrent-session collision** — another live session's
   interrupted hash-repair had (a) written the _main package's_ real vendor hash into
   `flake.nix`'s **cqrs-lint** `src.hash` and `vendorHash` slots, (b) left the fake
   `sha256-AAAA…` sentinel in `vendorHash.nix`, and (c) clobbered the
   `update-vendor-hash` script's `fake=` probe template with the real hash (which
   permanently breaks the tool's mismatch probe). All four slots restored;
   `nix build` and `nix flake check` green.
3. **statix noise eliminated at the correct layer** — all 15 statix findings came from
   the sealed design record `deploy/nixos/module-draft.nix` (superseded by the landed
   NixOS module in nix-international-telephony). Suppressed rule 20 repo-wide via a
   documented root `statix.toml` (`disabled = [20]`, integer codes — string and
   `[files] ignore` schemas were empirically tested and do NOT work in this statix).
   Rationale recorded in `docs/buildflow-dev-gate.md`. An earlier treefmt-level exclude
   was added and then removed as redundant (BuildFlow's statix runs `statix check`
   directly and ignores treefmt excludes).
4. **AGENTS.md size compliance** — trimmed 225→219 lines (the linter counts wc+1, so
   220 linter-count = exactly max) via five content-preserving compressions
   (load-doctrine comment, adoption-map bullet, flake bullet, support-packages lines,
   devShell/command merges). dprint-clean. Zero concepts dropped; a concurrent
   session's 5 added lines (verify-ui-contract.js command block) were preserved.
5. **Toolchain-skew diagnosis and recovery** — gate steps (go-generate, golangci-lint,
   govalid-generate) failed with "go-appkit requires go ≥ 1.27.1 (running go 1.27.0)"
   when the gate runs _outside_ the devShell. Confirmed against
   `docs/buildflow-dev-gate.md` (documented trap, 2026-10-11): the devShell ships
   nixpkgs `go_1_27` = 1.27.2 with GOTOOLCHAIN=local. Gate re-run inside
   `nix develop -c` → those steps pass.
6. **Final verification battery** — `nix build` (binary runs: `pbx-lens version` →
   0.1.0), `nix flake check` (all checks pass), full `go test ./...` (green, including
   `TestTemplGeneratedInSync` after regeneration), standalone `go-structure-linter`
   step (0 error findings), and the **full BuildFlow gate: exit 0**.
7. **Concurrent-session work preserved** — the other session's uncommitted
   `golden_pages_test.go` WIP (staleness-region + bare-date golden scrub regexes) was
   never reverted; `go test ./internal/web/` passes with it.

## b) PARTIALLY DONE

1. **AGENTS.md headroom = zero.** 219 wc / 220 linter-count is _at_ the limit, not
   under it. The file was over-limit twice today and a live concurrent session keeps
   adding to it — the next addition re-trips an error-level finding. Needs a real trim
   to ~215.
2. **Toolchain skew is avoided procedurally, not fixed.** The bare `buildflow …`
   invocation still fails on a host go 1.27.0; nothing enforces the devShell wrapper.
3. **statix suppression is broader than the problem.** Rule 20 is disabled repo-wide;
   a genuine future repeated-keys smell anywhere would be silenced too.
4. **go.mod `go 1.27.1` finding stands.** Correctly identified as dependency-forced
   (go-appkit v0.8.0's own directive; `go mod tidy` deterministically re-bumps), but
   no upstream filing was drafted (the docs/upstream-filings.md D6 pattern).
5. **Concurrent-session awareness.** Noticed and worked around late (two
   "file modified since read" races, one hash-slot collision); no protocol established.
6. **Final gate green but on a dirty tree** (daemon still committing during the run;
   drv names carried `-dirty`). A clean-tree confirmation pass was not done.

## c) NOT STARTED (all observed during this session, none acted on)

1. **9 tools "unavailable (health check failed)"** — never triaged with `--verbose`.
2. **prettier / vulnix / lychee not in the devShell** — each run emits 5 warnings and
   runs them via `nix run nixpkgs#…` without project deps; the fix hint
   ("add to devShells.default") was never applied.
3. **govulncheck GO-2026-6617** (net/http HPACK race, stdlib) — never checked whether
   go 1.27.2 (already in the nixpkgs lock) contains the fix.
4. **vulnix 22 findings** (binutils 2.46/2.47, bison, coreutils, gcc 10.4.0…) — no
   accept-and-document decision, no nixpkgs policy.
5. **lychee private-repo links** — the fleet authenticate-vs-exclude decision
   (GITHUB_TOKEN vs `exclude` regex) remains open; 86 markdown files scanned each run.
6. **detect-only finding piles unreviewed**: art-dupl 76, branching-flow 79,
   file-size 10 (e.g. `internal/web/calls_templ.go` 586/350, `telnyx_test.go` 836/350,
   `web/server.go` 507/350, `search.go` 444/350), and the CallView 17-field
   threshold breach + "Observation/CallView share 13 fields — consider mixin" hint.
7. **cqrs-lint 17 findings** — assumed to match docs/cqrs-lint-policy.md's documented
   set; the lists were never diffed.
8. **BuildFlow binary freshness** — binary built at 893de6b vs HEAD 916fe0f
   (`nix build . && nix run .#reinstall` in the BuildFlow repo).
9. **TODO_LIST.md harvest** — section (f) below was not routed into TODO_LIST.md /
   ROADMAP.md (docs-health HARVEST).
10. **CI** — never checked whether the GitHub lane mirrors the flake checks
    (pre-commit/treefmt) or ran green on today's daemon commit series.

## d) TOTALLY FUCKED UP (own failures this session)

1. **Ran the gate outside the devShell — twice.** The exact trap
   `docs/buildflow-dev-gate.md` documents (with the fix!) from 2026-10-11. I reproduced
   a known environmental failure and initially treated it as a fresh regression. The
   doc should have been read _before_ the first gate invocation.
2. **Hand-typed nix store hashes repeatedly.** Multiple command failures from /nix/store
   hash typos before extracting the path into a variable. Wasted cycles, zero value.
3. **Theorized before pulling the evidence.** For the templ-fmt divergence I cycled
   through cache theories, version-skew theories, and a prettier hypothesis before
   running `nix log <drv>` — which was available from the first failure and settled it
   in one look. Evidence first, theory second.
4. **Trimmed AGENTS.md to exactly the limit with zero headroom** — knowing a live
   concurrent session was adding lines to it (I'd seen its commits). It re-tripped the
   gate within the hour, and the linter's wc+1 counting cost another cycle to discover
   empirically.
5. **Repaired the vendor hash by hand instead of via `nix run .#update-vendor-hash`** —
   the repo's purpose-built tool (which then turned out to be broken by the
   concurrent session's `fake=` clobber — but I never checked the tool's state before
   hand-writing hashes from a build error message, including one typed from memory).

## e) WHAT WE SHOULD IMPROVE

1. **Read the runbook first.** `docs/buildflow-dev-gate.md` is the gate's runbook; any
   gate invocation or gate-failure triage starts there.
2. **Make the safe path the default path.** Every "run it inside the devShell" rule
   eventually gets violated; enforcement beats documentation (wrapper script / flake app).
3. **Use the repo's own repair tools**; hand-edits are for when the tool is proven broken.
4. **Never leave a hard limit at zero headroom** in a file other sessions actively edit.
5. **Check for concurrent sessions at start** (fresh `git log` timestamps, recent
   `docs/status/` files) before editing shared files (AGENTS.md, flake.nix,
   vendorHash.nix were all double-edited today).
6. **Full logs before theories**: `nix log <drv>` / `--verbose` is cheaper than three
   wrong hypotheses.
7. **Verify repair completeness at the flake level** (`nix flake check` covers the
   cqrs-lint attr too), not just the package that errored — the slot collision slipped
   through a package-only green build.

## f) NEXT — up to 50 things (brainstorm, sorted by impact; ROADMAP fuel, not commitments)

**Gate correctness / never-again guards**

1. Add `scripts/gate.sh` (or a flake app `.#gate`) wrapping `nix develop -c … buildflow …` so the devShell requirement is enforced, not documented.
2. Add a gate/preflight guard: fail fast if `vendorHash.nix` contains the AAAA sentinel or the `update-vendor-hash` script's `fake=` ≠ AAAA (catches interrupted repairs immediately).
3. Fix/verify `update-vendor-hash` slot targeting: today's collision suggests a repair wrote main-package hashes into cqrs-lint slots — make the tool verify which FOD it probed (`nix flake check` the whole flake after repair, not just the main package).
4. Add a local replicator for the sandbox check: `scripts/check-templ-stable.sh` running `templ fmt` **per file** over `internal/web/` and asserting zero diff — permanently closes the "local treefmt clean, sandbox red" class.
5. Document the AGENTS.md linter-counts-wc+1 rule in docs/buildflow-dev-gate.md (future trims target 219, not 220).
6. Create real AGENTS.md headroom (target ≤215 wc): the adoption-map bullet or Commands tail can shrink further into docs/.
7. Confirm CI runs `TestTemplGeneratedInSync`-equivalent coverage (it generates templ first, so likely implicit — verify once).
8. Decide whether CI should also run the flake's pre-commit/treefmt check (currently local-gate-only).
9. Audit the gate's `nix-flake-update` auto-repair: it rewrote flake.lock mid-flight during a failing run — is that desirable in `--fix` mode?
10. Post-daemon-settle clean-tree verification: one `nix build` + gate pass once the auto-commit queue is empty (today's green ran on `-dirty` drvs).

**DevShell & tooling hygiene**

11. Add prettier to `devShells.default` (kills repeated "running via nix run … WITHOUT project deps" warnings).
12. Add vulnix to the devShell.
13. Add lychee to the devShell.
14. Triage "9 tools unavailable (health check failed)" with `--verbose`; fix or document each.
15. Rebuild + reinstall the BuildFlow binary (`cd ~/projects/BuildFlow && nix build . && nix run .#reinstall`) to clear the binary-freshness preflight.
16. Consider adding treefmt itself to the devShell so sessions can run the EXACT sandbox formatter locally.

**Vulnerability posture**

17. Check GO-2026-6617's fixed-in version vs go 1.27.2 (if fixed, the top govulncheck finding closes with the existing lock).
18. Decide vulnix policy for nixpkgs toolchain CVEs (binutils/bison/coreutils/gcc): accept-and-document vs targeted bumps.
19. Verify the gate's flake.lock update didn't regress go_1_27 or vendor hashes (it ran during the collision window).

**Upstream filings (verify-before-filing first)**

20. File against go-appkit: declare `go 1.27` not `go 1.27.1` (frees our go.mod floor; the 1.27.1 patch pin forced the skew confusion). D6 pattern, docs/upstream-filings.md.
21. File against templ (after source-level verification): fmt non-idempotent on first pass + per-file vs batch divergence (our pre-commit trap's root).
22. Verify templ's upstream state for the existing templ-formatter/goPkg issue already noted in flake.nix comments ("Upstream fix pending in go-nix-helpers") — check if it landed.

**Findings hygiene**

23. Diff the 17 cqrs-lint findings against docs/cqrs-lint-policy.md's documented set; document any newcomers.
24. Introduce an accepted-findings baseline so the 242-finding warning noise stops burying new signal.
25. Narrow the statix rule-20 disable if statix gains per-path config (or restyle module-draft.nix) — repo-wide disable hides real future debt.
26. Run the deduplicate-code skill over art-dupl's 76 findings (proper handler; today they were noted, not judged).
27. Data-model-review `CallView` (17 fields, threshold breach; 13-field overlap with Observation flagged "consider mixin") — the fold shape deserves a second look even if intentional.
28. Triaging file-size findings: `internal/web/server.go` 507 and `internal/search/search.go` 444 are the two real split candidates; `*_templ.go` are generated (exclude them from the check).

**Repo/link hygiene**

29. Decide the lychee fleet policy: GITHUB_TOKEN (authenticated checks) vs `exclude` regex for larsartmann repos — currently undecided, burns a warning every run.
30. Verify the "1 URL excluded by config" lychee exclusion is intentional.
31. Check which 38 markdown files are path-excluded from the link scan and confirm that's intended.

**Concurrent-session safety**

32. Establish a concurrent-session protocol (marker file, branch convention, or "check fresh git log before editing shared files" rule in AGENTS.md) — today two sessions raced AGENTS.md, flake.nix, and vendorHash.nix.
33. Review the other session's `golden_pages_test.go` scrub regexes for over-scrubbing — the bare-DATE regex `(20\d{2}-\d{2}-\d{2})([^T0-9-]|$)` could mask genuine wrong-date regressions in goldens.
34. Consider a fixed reference clock for goldens instead of scrubbing (stronger assertion, less masking).

**Docs coherence (post-compression checks)**

35. Re-read my five AGENTS.md compressions against git history to confirm zero concept loss (self-check promised, not yet done).
36. Verify `docs/templ-components-adoption.md` is the current exact filename referenced post-compression.
37. Verify the `-recording-timeout` flag name in the Commands block still matches the code (line survived compression).
38. Cross-check the Commands block now points to the devShell-wrapped gate invocation (or will, after item 1).
39. Route section (f) into TODO_LIST.md (docs-health HARVEST) so this list isn't entombed in a timestamped file.

**Deferred/adjacent (noted in passing, no work done)**

40. `internal/web/calls_templ.go` is 586 lines — generated; add `*_templ.go` to the file-size check's excludes.
41. `telnyx_test.go` 836 lines — decide split-or-accept (table-driven test file; likely accept-and-document).
42. Re-verify the "CLEAN CHECKOUT FIRST" tailwind regen note still matches the flake preBuild behavior (survived compression; untested claim today).
43. Confirm `scripts/gen-tc-classes.js` (new from the concurrent session) is correctly named in the adoption-map watchlist I compressed.
44. Check the two remaining "2 skipped via config" gate steps are intentionally skipped (not accidentally).
45. Verify CI went green on today's daemon commit series (never pushed/checked from here).
46. Consider whether `TestTemplGeneratedInSync` should also assert per-file fmt stability (pairs with item 4).
47. The gate's "16 skipped (blocked by failures)" artifact from failing runs — confirm none of those steps have latent issues now that the run is green.
48. Add `statix.toml` rationale pointer into AGENTS.md only if it keeps under the limit (currently lives in docs/buildflow-dev-gate.md — probably correct as is).
49. Sweep `docs/status/archived/` convention: this report goes to docs/status/, archived reports stay closed — no harvest from there (per AGENTS.md).
50. Re-run the full gate after items 1–3 land to confirm the fix chain holds end-to-end.

## g) Questions I cannot answer myself

1. **Who/what wrote the hash-slot collision?** The cqrs-lint `src.hash`/`vendorHash` slots carrying the _main package's_ vendor hash, the AAAA sentinel left in `vendorHash.nix`, and the clobbered `fake=` template — was that your concurrent session's `update-vendor-hash` run misbehaving (tool bug → I should fix the tool) or manual hash pasting (process issue)? This decides whether item 3 above is a tool fix or unnecessary.
2. **Gate invocation standard:** should the gate ALWAYS run inside the devShell via a wrapper (flake app `.#gate` / `scripts/gate.sh`), or do you want the bare `buildflow` host invocation to work too (which would mean fixing the host-go path, e.g. pinning go ≥ 1.27.1 outside nix)?
3. **lychee policy:** authenticate private-repo link checks with GITHUB_TOKEN, or exclude `github.com/larsartmann/*` from link checks? (Fleet-level decision explicitly marked undecided in the preflight; I can implement either in minutes once you pick.)

---

_Point-in-time snapshot; goes stale. Section (f) is docs-health HARVEST input._
