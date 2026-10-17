# Status Report — statix silent-no-op repair, phantom linter removal, gate revival

**Session:** 2026-10-17 ~01:05–01:45 CEST · **Author:** Crush
**Trigger:** user's `statix check .` printed "config error: unable to parse
config file … invalid type: integer `20`, expected a string" and exited 0.
**Scope:** this session only — the statix config repair it demanded, the doc
trail it exposed, and the two pre-existing gate breaks the verification gate
surfaced (styles.css, `.golangci.yml`). The full dev gate is **green again**
(exit 0, 58 success / 0 failed). Nothing was pushed; all work landed via the
auto-commit daemon (db18c22, f48b722, d403cd2, 0f99144).

---

## a) FULLY DONE

1. **Root cause fixed — `statix.toml`** (`disabled = [20]` → file-level
   `ignore = ["deploy/nixos/module-draft.nix"]`). The integer was a TOML
   parse error; statix wants rule-NAME strings. File-level ignore is
   NARROWER than the original 2026-10-12 intent (repo-wide rule kill): W20
   `repeated_keys` stays live everywhere except the sealed design record —
   closing that report's own regret b3/c25 ("suppression is broader than the
   problem"). Evidence: `statix check .` and bare `statix check` (BuildFlow's
   exact form) exit 0 with no config error; direct-file inspection still
   honestly shows W20. Commit db18c22.
2. **The silent no-op discovered and documented.** statix **exits 0 on a
   config parse error and lints nothing** — the repo's statix lane was
   green-but-dead from 2026-10-12 (4b9fac5) until this session, ~5 days.
   The 2026-10-12_19-20 report's claim that "string and `[files] ignore`
   schemas were empirically tested and do NOT work" is falsified — that
   session's test was almost certainly fooled by this same no-op. Contract
   recorded in `docs/gotchas.md` (new entry), the gate-doc statix entry
   rewritten (`docs/buildflow-dev-gate.md:190` — it previously described the
   broken integer mechanism as working), and the false claim annotated
   inline in the 2026-10-12 report. Commit f48b722.
3. **BuildFlow statix step genuinely revived.** Cache-bypassed
   (`BUILDFLOW_NO_RESULT_CACHE=1`) drill shows `exec: statix check` →
   green, 0 findings, repair skipped. Before this, the step "passed" by
   never linting.
4. **treefmt statix lane mapped (not changed — nothing to change).**
   Discovery: treefmt-nix's statix formatter wraps a FROZEN STORE config
   (`statix fix --config /nix/store/…`) per file — the repo `statix.toml`
   was never its input. `nix fmt -- --ci` green (9 files, 0 changed; W20 has
   no auto-fix, so the sealed record can't be silently rewritten).
5. **CSS gate test revived (pre-existing red, found by the full gate).**
   `TestCSSGateBridgesHeatmapBrandToTeal` failed because the committed
   `styles.css` was a foreign UNMINIFIED rebuild (Oct 15 19:50, daemon
   commit 450a75e — spaces after colons broke the pinned token substrings).
   Repaired via the sanctioned `nix run .#update-styles` (pinned tailwind,
   `--minify`, bridge-verified by the app itself). Test green. Commit d403cd2.
6. **Phantom `gohumanize` linter removed from `.golangci.yml` (second
   pre-existing red).** Added by daemon commit 3d93fdf (2026-10-16 04:40);
   it does not exist in ANY golangci-lint release — verified three ways:
   devShell v2.14.0 `help linters` (absent), sourcegraph over
   `golangci/golangci-lint` (zero hits), global search (only
   `dustin/go-humanize` import aliases). Both the devShell and CI pin v2.14,
   so every lint run aborted with "unknown linters". Removed; lint now
   actually runs: 0 issues in 77 s. Commit 0f99144.
7. **CHANGELOG** — Unreleased "Fixed — 2026-10-17 (lint gate revival)" with
   both repairs (statix config + phantom linter). Commits f48b722, 0f99144.
8. **Full dev gate green:** `nix develop -c buildflow --build-mode dev` →
   exit 0, **58 success, 0 failed, 0 skipped** (+3 via config). Two
   consecutive runs (01:31, 01:32) after both repairs.

## b) PARTIALLY DONE

~~1. **Remote CI state unverified.** All fixes are local-only (nothing pushed~~ done — the postscript mapped all 6 red runs; pushed at d78291a with remote CI success (close-out report §a16)
   — harness rule). The last pushed commits (Oct 16) carry `gohumanize`, so
   GitHub CI's lint job is almost certainly red on remote main right now;
   I never ran `gh run list` to confirm, and pushing is the user's call.
   Remaining: check CI, then push. S.
~~2. **The 3d93fdf sibling edits in `.golangci.yml` left standing,~~ routed → TODO_LIST Low row (confirm-or-formalize the lint-policy edits; this pass)
   intent unknown.** In the same commit that added gohumanize, a session
   also removed `lll` (line-length policy now unenforced repo-wide),
   relaxed `run.go: 1.27.1` → `"1.27"` (vs the go.mod 1.27.1 floor), and
   changed tagliatelle case rules. v2.14 accepts all of them (lint passes),
   but I only reverted the provably-broken entry. Needs an intent
   confirmation (question g1). S.
~~3. **LSP still shows stale `gohumanize` diagnostics** in tool output at~~ done — postscript §3: cleared after a full LSP restart
   report time — the golangci-lint language server hasn't re-read the fixed
   config; the real binary reports 0 issues. Needs an LSP restart to
   confirm cleared. S.
~~4. **HARVEST of section (f) into TODO_LIST.md NOT run** — report-first per~~ done — harvested by the 05-01 sweep (§a8) and re-routed by this pass
   the user's "wait for instructions". The 2026-10-12 regret that caused
   this whole outage (c25) was NEVER harvested into TODO_LIST either — that
   process gap is why it survived 5 days. Awaiting go (question g3). S.
~~5. **No clean-tree confirmation gate run.** The final green gate ran while~~ done — postscript §6: gate re-ran clean on the settled tree (58/0)
   the daemon committed mid-run (same class the 2026-10-12 report flagged as
   its b6). A post-commit re-run on the settled tree was not done. S.

## c) NOT STARTED

~~1. **CI has no statix/nix lane at all** — CI runs golangci-lint, govulncheck,~~ routed → TODO gate-hardening bundle (CI nix lane; this pass)
   art-dupl; statix and deadnix exist only in local BuildFlow. The
   silent-no-op class would be invisible on CI by construction.
~~2. **No tripwire for the statix no-op class** (stderr says "config error",~~ routed → TODO gate-hardening bundle (statix no-op tripwire; this pass)
   exit code 0 = red). Documentation only; nothing automated prevents a
   recurrence.
~~3. **Repo `statix.toml` vs treefmt's frozen store config split brain** —~~ routed → ROADMAP raw idea (revisit if treefmt-nix grows a statix config option)
   documented, not unified (investigate a treefmt-nix statix config option).
~~4. **Upstream statix filing** (exit 0 on config parse error is a footgun) —~~ routed → TODO upstream-filings row (verify-before-filing first; this pass)
   D6-pattern candidate; existing-issue search not done (verify-before-filing).
~~5. **`nix flake check`** full lane — the dev gate covered the module lanes;~~ done — postscript §5: all checks passed on the settled tree
   the flake-check lane was not run this session.
~~6. **docs-health VERIFY sweep** over other "empirically tested" claims in~~ done — the statix claim was the only falsified one (corrected in this file; doctrine §e1)
   status docs — the class that produced the false schema claim.

## d) TOTALLY FUCKED UP

1. **Five days of zero statix linting while every gate read green.**
   Chain of failure: an unsupported `disabled = [20]` (TOML parse error) +
   statix's fail-open design (exit 0, check skipped) + a false "empirically
   tested, strings do NOT work" claim in a status report that made the
   broken form look deliberate + nobody reading stderr. The only signal
   ever printed was one "config error" line on manual `statix check .`.
   Fixed — but the CLASS is unguarded (see c2).
2. **A ~29-hour window in which the repo gate was red and nobody knew.**
   Two independent breaks (foreign styles.css rebuild Oct 15 19:50; phantom
   linter Oct 16 04:40) sat while the daemon kept committing. No session ran
   the gate between Oct 15 19:51 and this one (01:13). Gate cadence is
   effectively "whenever someone remembers".
3. **The auto-commit daemon launders breakage.** Both breaks entered history
   as "chore: auto-commit N changed file(s) (heuristic)" — zero validation
   between a concurrent session's edit and permanent history. This is the
   second status report in a row whose root-cause section is daemon-shaped
   (see the 2026-10-12 hash-collision story).
4. **Remote main is probably red right now** (unverified — b1): local green,
   remote carrying gohumanize. The public CI face may be failing while we
   celebrate a local 58/0.
5. **My own process mistakes this session (minor but real):**
   - I ran the full gate LAST, after all doc edits — gate-first (right after
     the statix fix) would have surfaced both pre-existing reds at the start
     and framed the session honestly instead of surprising me at the end.
   - Two broken probe loops: a missing `/tmp` dir silently broke the first
     glob-robustness test, and `PIPESTATUS` isn't supported in mvdan/sh (two
     "exit:" lines printed empty). Sloppy shell cost investigation cycles.
   - My closing claim "verified every consumer" overstated: I verified
     BuildFlow's CURRENT bare invocation, not a contract — buildflow feeds
     deadnix explicit file lists; if it ever does that for statix, the
     per-file `ignore` stops matching and W20 re-fires as findings. The
     gotchas entry records this; the claim should have said "current
     invocation forms".

## e) WHAT WE SHOULD IMPROVE

1. **Treat "empirically tested" in old docs as unverified until re-run.**
   A wrong claim with test-shaped language blocked the correct fix for five
   days. When a config decision cites an empirical test, re-verify the test
   before trusting the config (docs-health VERIFY exists for exactly this).
2. **Gate cadence needs an owner.** "Run `buildflow --build-mode dev` at
   session end (or lightning mode mid-session) after touching ANY config"
   should be doctrine; consider a daemon-side or pre-push guard so
   un-gated daemon commits can't pile up for 29 hours again.
3. **Fail-open tools need tripwires.** statix exits 0 with an error on
   stderr; the class (exit 0 + stderr output = red) is cheap to grep in
   scripts/verify-all.sh and CI. golangci-lint fails LOUDLY on config
   errors — that asymmetry is why gohumanize was caught in one gate run
   while the statix outage lasted 5 days.
4. **The statix knowledge now lives in four places** (statix.toml comment,
   gotchas entry, gate-doc entry, CHANGELOG) — deliberate audience split,
   but drift-prone. The gotchas entry is the canonical contract; the others
   point at it rather than restate (mostly true today; keep it that way).
5. **Pinning invocation contracts.** The `ignore` glob only matches
   walk-form targets; buildflow's bare `statix check` (cwd = repo root,
   default target `.`) is an implicit contract nobody pinned. A comment in
   `.buildflow.yml` or a scripts/README note would keep a future buildflow
   update from silently breaking the suppression.
6. **`golangci-lint config verify` exists** (upstream provides it) and would
   have caught the phantom linter at commit time — wire it into pre-commit
   or CI. statix has no equivalent, which is why (3) matters more there.

## f) Top 25 things to get done next

| #  | Task                                                                                     | Impact                                                                   | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ------ | ------------- |
| ~~1~~  | ~~Check `gh run list`; confirm remote CI red; push when the user says go~~ done at `d78291a — pushed with remote CI success (close-out report §a16)` | ~~High~~ | ~~S~~ | ~~Bug~~ |
| ~~2~~  | ~~Confirm or revert the 3d93fdf sibling edits (`lll` removal, `run.go` relax, tagliatelle)~~ done — → routed TODO_LIST Low row (lint-policy intent confirm; this pass) | ~~High~~ | ~~S~~ | ~~Quality~~ |
| ~~3~~  | ~~Add a statix no-op tripwire (fail on "config error" in stderr) to scripts/verify-all.sh~~ done — → routed TODO gate-hardening bundle (this pass) | ~~High~~ | ~~S~~ | ~~Quality~~ |
| ~~4~~  | ~~Add a CI nix lane (statix + deadnix + `nix fmt --ci`) so nix lint isn't local-only~~ done — → routed TODO gate-hardening bundle (this pass) | ~~High~~ | ~~M~~ | ~~Quality~~ |
| ~~5~~ done — the 05-01 corpus sweep (its §a8): HARVEST executed | ~~HARVEST this report's (f) into TODO_LIST.md (docs-health)~~ | ~~High~~ | ~~S~~ |
| ~~6~~ done — the 05-01 corpus sweep (its §a8): un-harvested regrets swept; c25 closed by the statix repair | ~~Sweep other un-harvested report regrets into TODO_LIST (c25 cost 5 days)~~ | ~~High~~ | ~~M~~ |
| ~~7~~  | ~~Clean-tree confirmation gate run on the settled tree~~ done — → routed the gate-receipt TODO row (BuildFlow#56 CLOSED; load policy) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~8~~  | ~~Wire `golangci-lint config verify` into CI or pre-commit~~ done — → routed TODO gate-hardening bundle (golangci config verify) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~9~~  | ~~CSS gate test: also assert minified output (foreign-build tripwire)~~ done — superseded by the styles-freshness CI byte-diff (stronger) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~10~~ | ~~Unify repo statix.toml with treefmt's store statix config (kill the split)~~ done — → routed ROADMAP raw idea | ~~Medium~~ | ~~M~~ | ~~Quality~~ |
| ~~11~~ | ~~Triage buildflow "9 tools unavailable (health check failed)" with -v~~ done — P1-T8 dispositioned (08:02 §a); remainder = the tooling-triage TODO row | ~~Medium~~ | ~~M~~ | ~~Cleanup~~ |
| ~~12~~ | ~~Triage nix-hash-fix 82% / nix-build-verify 100% historical failure rates~~ done — P1-T8 (nix-build-verify skip rationalized; hash-fix failures environmental) | ~~Medium~~ | ~~M~~ | ~~Quality~~ |
| ~~13~~ | ~~Triage or accept the 24 vulnix advisories (bash 2.05b, binutils CVEs)~~ done — → routed ROADMAP (rides the next flake bump) | ~~Medium~~ | ~~M~~ | ~~Security~~ |
| ~~14~~ | ~~docs-health VERIFY sweep over "empirically tested" claims in status docs~~ done — the statix claim was the only falsified one; corrected in this file | ~~Medium~~ | ~~M~~ | ~~Documentation~~ |
| ~~15~~ | ~~Run `nix flake check` on the settled tree~~ done — postscript §5 (all checks passed) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~16~~ | ~~Restart LSP; confirm gohumanize diagnostics cleared~~ **Won't implement — obsolete — machine-local state; cleared per postscript §3.** | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~17~~ | ~~File upstream statix issue re: exit 0 on config parse error (search dupes first)~~ done — → routed TODO upstream-filings row (verify-before-filing first) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~18~~ | ~~Record the statix exit-0 contract in crush-config references/lessons.md (fleet)~~ done — → routed fleet (crush-config lessons.md by commit; out of this repo) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~19~~ | ~~Propose a BuildFlow "lint-config validity" step (fleet value)~~ done — → routed ROADMAP fleet proposal | ~~Low~~ | ~~M~~ | ~~Process~~ |
| ~~20~~ | ~~Pin the statix invocation contract (comment in .buildflow.yml or scripts/README)~~ done — → routed TODO rehearsal/tooling row (.buildflow.yml contract comment) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~21~~ | ~~Daemon-commits-breakage: propose a fleet-level guard (BuildFlow repo)~~ done — → routed ROADMAP fleet proposal | ~~Medium~~ | ~~L~~ | ~~Process~~ |
| ~~22~~ | ~~AGENTS.md headroom is 219/220 — plan a trim before the next addition~~ done — budget enforced (check-agents-budget.sh + pre-commit; 219/219) | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~23~~ | ~~Re-test the statix ignore invocation matrix whenever nixpkgs bumps statix~~ done — → routed ROADMAP (statix bump watch) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~24~~ | ~~Note the rejected c25 alternative (restyle module-draft.nix) in the gate doc~~ done — rejected-alternative note added to docs/buildflow-dev-gate.md this pass | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~25~~ | ~~Investigate why buildflow's statix step passes no target (cwd-fragile) upstream~~ done — → routed ROADMAP fleet (rides the BuildFlow filings) | ~~Low~~ | ~~S~~ | ~~Process~~ |

## g) Questions I cannot answer myself

~~1. **The Oct 16 04:40 `.golangci.yml` edits — yours or another agent's?**~~ routed → TODO_LIST Low row (lint-policy intent confirm; this pass)
   I removed only the provably-nonexistent `gohumanize` and left its
   siblings standing: the `lll` removal (line-length checks now gone
   repo-wide), `run.go` 1.27.1→"1.27", and tagliatelle case changes. If
   those were deliberate curation, say so and they stay; if not, I should
   review them properly.
~~2. **Should lint coverage be enforced in CI (statix/deadnix lane + config~~ routed → TODO gate-hardening bundle — answered: enforce in CI (this pass)
   tripwires), or is local-BuildFlow-only a deliberate doctrine?** CI
   already runs golangci-lint; nix linting has no remote lane at all, which
   is exactly how a 5-day local-only outage stayed invisible.
~~3. **Run docs-health HARVEST now** to pull this report's (f) into~~ done — harvested by the 05-01 sweep and re-routed by this pass; the report can archive
   TODO_LIST.md, or do you want to triage the list yourself first?

---

## Postscript — remote CI investigated (read-only, same session ~01:50)

The follow-up pass closed b1/b3 and corrected this report's own CI claim,
which was UNDERSTATED:

1. **Remote main is red on 6 consecutive CI runs — since 18:11 CEST
   2026-10-15**, not "probably red since gohumanize" (b1/d4). Last green
   run: 38765780762 (18:03 CEST). Nothing was pushed since; every local fix
   (d403cd2 styles.css, 0f99144 phantom linter) is still unpushed.
2. **Failure causes, mapped per run (`gh run view --log-failed`):**
   - 18:11 / 18:13 / 18:17 CEST — `TestCSSGateBridgesHeatmapBrandToTeal`:
     styles.css was mid **flip-flop**. A concurrent session alternated the
     file FIVE times on Oct 15 (17:05 unmin → 17:12 min → 18:04 unmin →
     18:22 min → 19:51 unmin; commits f1fbaaa/2b0bcb2/577b6cb/ad10482/
     450a75e) between the 1-line minified form and a 5413-line UNMINIFIED
     build. Every unminified window redden CI. Section (a)5's "a foreign
     unminified rebuild (Oct 15 19:50)" was the LAST wrong landing of that
     fight, not the whole story; the interleaved minified commit (ad10482)
     carried the correct tokens.
   - 19:48 CEST — `TestCacheSemanticsHardening` (5.06 s): a DIFFERENT,
     flaky-looking failure on a CSS-clean tree (no cache-code change since;
     the suite is green locally twice tonight). **New finding: candidate
     flaky test** — real-time-dependent timing worth a look (→ task 26).
   - 20:14 CEST — TestCSSGate again (450a75e unminified — the state this
     session repaired via `nix run .#update-styles`).
   - Oct 16 04:44 CEST — Build & Test (CSS) + **Lint (gohumanize)**.
3. **b3 closed:** the stale `gohumanize` LSP diagnostics cleared after a
   full LSP restart (single-server restart did NOT suffice).
~~4. Task 26 added to (f): triage `TestCacheSemanticsHardening` flakiness~~ claim was FALSE — the table still holds 25 rows (the edit never landed); flake triage routed to the TODO gate-hardening bundle; test green in every CI run since 2026-10-16
   (CI-only reproduction so far).
5. **c5 closed:** `nix flake check` → all checks passed (treefmt, pre-commit,
   build, vendor-hash) on the settled tree.
6. **b5 closed, with one honest casualty:** the first clean-tree gate run hit
   the `nix-build` step's 47 s execution deadline — the daemon had committed
   this report's postscript mid-pass (new source hash → cold nix rebuilds),
   the documented daemon-race class, not a build failure. After warming the
   derivations with plain `nix build`, the gate re-ran clean: **exit 0,
   58 success / 0 failed / 0 skipped (+3 via config)**.

---

_Format note: written as Markdown at the operator's explicit path/format
demand; the status-report skill's canonical HTML output and the
brutal-self-review skill's `docs/reviews/` HTML convention were folded into
this single file accordingly._
