<!-- Status report — point-in-time snapshot (2026-10-11 04:58 CEST).
     Session scope: adopting the go-output house pattern (flake, devShell,
     CI lane, repo hygiene) into pbx-lens. Written .md per the operator's
     explicit instruction (status-report skill is HTML-canonical; the
     explicit .md path demand wins and is flagged in the closing message). -->

# Status Report — go-output pattern adoption into pbx-lens

**Date:** 2026-10-11 04:58 CEST
**Repo:** pbx-lens @ main, working tree clean (auto-daemon swept everything; session-start commit `fa01331` → report-time HEAD `0d386bb`)
**Session task:** "Adopt: /home/lars/projects/go-output/**" — bring pbx-lens's repo infrastructure up to go-output's current house pattern (flake.nix, devShell, pre-commit, CI lane, dependabot, repo hygiene files), adapted to a single-module BuildFlow-covered app.

## TL;DR

| Category                                | Count                                |
| --------------------------------------- | ------------------------------------ |
| Fully done (verified)                   | 9 work items                         |
| Partially done                          | 4                                    |
| Not started (session-relevant)          | 6                                    |
| Totally fucked up (this session, by me) | 1 incident (repaired, but real)      |
| Observed breakage NOT from this session | 1 (parallel WIP test, deterministic) |
| Next tasks brainstormed                 | 50                                   |

The headline outcome: **`nix build` ships a real binary now** (was: empty result — the top High-impact TODO), **cqrs-lint lives in the flake devShell** (kills the /tmp binary), and **the CI story is decided and wired** (GitHub Actions lane). Three TODO_LIST rows retired.

---

## What I forgot, what I could have done better, what I can still improve

**Forgot:**

- FEATURES.md was never updated — the adoption added genuine capabilities (CI lane, hermetic devShell, `update-vendor-hash` app, vendored cqrs-lint) that belong in the feature inventory. Only AGENTS/TODO_LIST/CHANGELOG got touched.
- Never attempted a fully green `buildflow --build-mode dev` gate run — the one gate run failed only on the parallel WIP `internal/cli` test, and I verified every dimension around it, but "green gate at end of session" was never achieved and I should state that as an open item rather than let "everything else passed" stand in for it.
- Did not document the `.golangci.yml` timeout deviation (pbx-lens 10m vs go-output 5m) in AGENTS.md — the go-version drift got fixed, the timeout divergence got silently kept.
- Did not notice until writing this report that `cmd/pbx-lens`'s `version` subcommand hard-codes "pbx-lens 0.1.0" — go-standard's ldflags inject `-X main.version=<rev>` into every nix build, and there is no `main.version` var to receive it. The injection is a no-op; nix-built binaries still print the hard-coded string.
- Did not pin or plan for pin-refresh of the vendored cqrs-lint (pinned to go-cqrs-lite master `afaffa19`, which is NEWER than the v4.x library versions pbx-lens's go.mod actually pins — a lint binary/library version skew I noticed and documented as a gotcha but did not resolve).

**Could have done better:**

- The TODO_LIST edit. I corrupted the table mid-edit (details in section d). Root cause: I wrote edits that paired a deletion with a substitution of a DIFFERENT row, instead of pure single-row deletions. The repair was correct and verified, but the incident was avoidable with one-row-per-edit discipline.
- Verification before substitution: I had the full TODO_LIST content in context; a mechanical "delete exactly these 3 lines" (3 separate `edit` calls) could not have produced collateral damage.

**Can still improve (process):**

- After any docs-table edit, diff against git HEAD and assert the net change equals the intended change — I did this, but only AFTER noticing the failed edit count. Make it unconditional.
- For adopted patterns: write down deliberate deviations (BuildFlow-owns-it skips, timeout) at adoption time, not in a later session's report.

---

## a) FULLY DONE (verified this session)

1. **Empty `nix build` fixed — the package now targets the real binary.**
   Evidence: `result/bin/pbx-lens` exists; `nix run . -- version` → `pbx-lens 0.1.0` (before: `result/` empty, `nix run` errored — reproduced at session start).
   Scope: `flake.nix` (`subPackages = ["./cmd/pbx-lens"]`; go-standard module default `[\".\"]` built the package-less repo root). Retires the High-impact TODO row.

2. **cqrs-lint vendored into the flake devShell — /tmp binary dependency killed.**
   Evidence: `nix develop -c cqrs-lint .` → exit 0 (0 errors, policy-consistent findings); tool present at `cqrs-lint-0.0.0-unstable-2026-10-11/bin/cqrs-lint` in the shell. Pinned to go-cqrs-lite master `afaffa19105b2422ef0722f156bca6d92169c529` (2026-10-11 04:00 CEST), `sourceRoot = "source/cmd/cqrs-lint"`, `env.GOWORK = "off"` (monorepo go.work broke the modules FOD in workspace mode), `doCheck = false` (its test fixtures fetch modules over the network — impossible hermetically). Two hash iterations paid (src `sha256-2NQB…`, vendor `sha256-sGol…`). Retires the Medium TODO row.

3. **GitHub Actions CI lane wired (CI-story TODO decided).**
   Evidence: `.github/workflows/ci.yml` — actionlint zero findings, YAML parse OK; build + test(-race, shuffle-tolerant) + coverage floor (50%, measured 52.2% total; packages 76–94%, cmd/wire drag) + go mod tidy check + golangci-lint (v2.12) + govulncheck + warning-only art-dupl (t=4). Pinned SHAs copied verbatim from go-output's working workflows (checkout v7, setup-go v7, golangci-lint-action v9). Cannot verify an actual remote run without pushing — noted in (b).

4. **flake.nix house settings adopted (go-output parity, BuildFlow-aware).**
   Evidence: `nix flake check` → "all checks passed!" (treefmt-check incl. nixfmt/deadnix/statix over the flake, vendor-hash check, pre-commit-run check, build check). `enableGofumpt`/`enableGoimports` = false (BuildFlow owns Go formatting — no split brain), `enableNixfmt` = true, treefmt = deadnix + statix, `projectRootFile = "go.mod"`.

5. **Hermetic devShell with the full toolchain.**
   Evidence: `nix develop -c sh -c '…'` → go 1.27.2, golangci-lint 2.14.0, gopls 0.23.0, govulncheck 1.8.0, cqrs-lint; env `GOEXPERIMENT=jsonv2 GOTOOLCHAIN=local GOWORK=off`. git-hooks flakeModule + treefmt hook wired (pattern parity; see (b) for the install caveat).

6. **`nix run .#update-vendor-hash` repair app.**
   Evidence: app present in `apps.` output, script verbatim from go-output (fake-hash → capture got → rewrite → verify green build). Not exercised end-to-end this session (no vendorHash drift occurred) — wiring mirrors the battle-tested go-output original.

7. **Repo hygiene adoption.**
   Evidence (all parsed/validated): `.github/dependabot.yml` += github-actions ecosystem (weekly, `chore(ci)` prefix, grouped) alongside existing gomod entry; `.gitignore` += `.pre-commit-config.yaml` (git-hooks artifact; `coverage.out` already covered by `*.out`); `.gitattributes` += `docs/status/*.md linguist-documentation` (go-output pattern; skipped `reports/**` and binary rules — no such assets exist here, YAGNI); `git-town.toml` created (`main = "main"` — pbx-lens branches from `main`, unlike go-output's `master`; TOML parse OK).

8. **Config drift fix: `.golangci.yml` go 1.26.7 → 1.27.0** (matches go.mod floor and go-output's pin). Evidence: golangci-lint ran clean post-change — 0 errors, 96 warnings across 15 files, all pre-existing code, none in session-touched files (gate fails at ERROR severity only).

9. **Docs updated and internally consistent.**
   AGENTS.md: flake/CI/structure Stack-decision bullets rewritten to describe what actually ships now; two new paid-for gotchas (cqrs-lint vendoring traps; global `core.hooksPath=.githooks` makes git-hooks-managed hooks inert on this machine — go-output's installed hook is dead for the same reason); Commands block un-BROKEN'd (`nix build` line now says ships `result/bin/pbx-lens`) + devShell/update-vendor-hash/cqrs-lint invocations; cqrs-lint findings-policy Run line switched to the devShell invocation.
   TODO_LIST.md: exactly the 3 completed rows removed (nix-fix, cqrs-lint-flake, CI-story) — verified by grep counts (0/0/0 remaining; httptest ×1, resolveKey ×1, T7 ×1, zero duplicates).
   CHANGELOG.md: 2 Added bullets (CI lane; devShell+tooling) + 1 Fixed bullet (empty nix build), house style.

## b) PARTIALLY DONE

1. **CI lane — written, linted, but never executed remotely.**
   What works: actionlint-clean workflow, all job logic mirrors go-output's proven shape, every local equivalent passes (build/test/tidy/lint/govulncheck-tool present).
   What remains: first real push will exercise it; the coverage floor (50) and the golangci `v2.12` pin (local nixpkgs has 2.14) are unproven in anger.
   Blocker: none — just needs a push. Effort to finish: S.

2. **Full quality-gate green run.**
   What works: gate's format/lint/nix steps passed; all pre-existing packages green under race+shuffle.
   What remains: `buildflow --build-mode dev` end-to-end green — blocked only by the parallel-WIP `internal/cli` failure (section below). Effort: S once that test is fixed by its owner.

3. **git-hooks pre-commit wiring.**
   What works: flakeModule imported, treefmt hook configured, `pre-commit run --all-files` works inside the devShell (formatted 0-changed → treefmt-clean), first-entry config generation works.
   What remains: the hook never auto-installs on this machine — your GLOBAL git config sets `core.hooksPath=.githooks`, so pre-commit's installer refuses (and go-output's `.git/hooks/pre-commit` is equally inert — the global path intercepts). This is a machine-level policy fact, documented in AGENTS.md, not fixed here.
   Blocker: owner decision (see questions). Effort: S.

4. **cqrs-lint version pinning.**
   What works: hermetic, reproducible build from a pinned master rev.
   What remains: no refresh policy; the rev is already newer than the v4.x library versions in pbx-lens's go.mod (lint binary/library skew). AGENTS.md says "bump the rev deliberately; expect two hash iterations" — deliberate, but unowned. Effort: S–M.

## c) NOT STARTED (session-relevant, deliberately out of scope)

1. **Release train (release.yml + scripts/pre-tag-check.sh + tag-release.sh).** Skipped: pbx-lens has no tags and no release train yet; adopting go-output's multi-module release workflow verbatim would be wrong (it carries TAGGED_SUBMODULES machinery). Needs a single-module adaptation when v0.1.0 is cut. Still wanted: yes, at tag time. **→ ROADMAP theme 2 (at tag time)**
2. **`.config/metadata.yaml`** (go-output has it; go-nix-helpers does not — fleet policy unclear). Skipped as not clearly fleet-standard. Needs the owner's call. **→ fleet policy, undecided**
3. **go-output's `scripts/` set** (pre-deploy-check, pre-tag-check, setup-workspace, tag-release). Skipped: workspace/release-specific; none apply to a single-module app without a release train. **→ ROADMAP theme 2 (at tag time)**
4. ~~**uptime.yml / website.yml / website/ / package.json / .structure-linter.yml / AUTHORS / CODE_OF_CONDUCT / RELEASE.md.** Skipped: go-output-specific (website infrastructure) or not present in the go-nix-helpers template repo (not fleet-standard).~~ **Won't implement — go-output-specific; YAGNI for a single-module app without a website.**
5. ~~**Flake apps fan-out (test/test-race/build/lint/tidy/govulncheck per-module apps).** Deliberately NOT adopted: BuildFlow is the covered quality gate in this repo; duplicating those steps as flake apps would create the split brain the buildflow skill warns about. go-standard's generic `apps.test`/`apps.lint` remain as module-generated conveniences. Documented in AGENTS.md.~~ done (docs-health pass 2026-10-12)
6. ~~**Any `internal/cli` work.** New package landed mid-session from parallel work (daemon commits `eeb9f24`, `ead1617`); hands off — actively being iterated, including uncommitted edits while I worked.~~ done — landed the same day (04:59 session)

## d) TOTALLY FUCKED UP

1. **I corrupted TODO_LIST.md mid-edit — and the damage was briefly committed to history.**
   - What happened: retiring the 3 done rows, I wrote two multiedit operations that paired a row-deletion with substituting a DIFFERENT row's text. Net effect: the httptest row and the resolveKey row were each DUPLICATED, and the T7 (systemd unit) row — which is NOT done — was destroyed. The auto-commit daemon swept the damaged state into a commit before I noticed.
   - Severity: medium (docs only; no code or data). But history now contains a transient commit with duplicate rows and a missing T7 row.
   - Root cause: edit-shape error — pairing deletion with substitution instead of three pure single-row deletions; I compounded it by not diffing against HEAD until after the failed-edit count tipped me off.
   - Mitigation/repair: fully repaired in-session and verified — final state vs. HEAD shows exactly the intended 3 deletions; grep counts prove httptest ×1, resolveKey ×1, T7 ×1 restored, nix/cqrs-lint/CI rows ×0. Nothing else in the file changed.
   - Lesson applied: docs-table edits get one-row-per-edit + post-edit `git diff` assertion, unconditionally.

2. **Observed, NOT from this session (owner is on it): `internal/cli` convergence test fails deterministically.**
   - `TestIngestRecordsCorrectsAndRejects` (`internal/cli/ingest_test.go:69`): "projection for telnyx-conversation:conv-a did not converge in 5s". Reproduces with plain `go test ./internal/cli/` (no race/shuffle needed — not an order/load flake). Boot logs show the documented-advisory "unconsumed event type" WARNs for `call.corrected`/`call.enriched`/`call.observed`.
   - Severity: blocks the quality gate (the only red step in `buildflow --build-mode dev`).
   - Not mine: my session touched zero Go files; the package landed mid-session from parallel work and has live uncommitted edits (`system.go`, later `ingest_test.go`). I did not touch it, per the never-revert-others'-work rule.

## e) WHAT WE SHOULD IMPROVE

1. **Edit discipline for tables/lists:** one row per edit, then assert `git diff` equals intent. (Incident d.1.) A `docs-health`-adjacent convention worth writing into the global lessons file.
2. **"Gate green" as an explicit done-criterion:** a session that changes infra should end with either a green `buildflow --build-mode dev` or a named, owned blocker. This session ended at "green except a foreign WIP test" — correct, but it should be stated as a criterion, not discovered in the report.
3. **Version-injection honesty:** either add `var version = "dev"` to `cmd/pbx-lens` so the nix ldflags `-X main.version` injection actually lands, or remove the pretense (keep the hard-coded string and accept it). Currently the machinery silently no-ops.
4. **Vendored-tool pin policy:** cqrs-lint's rev needs an owner (manual bump cadence vs. tracking go-cqrs-lite releases vs. a tiny `nix flake update`-style automation). Two-hash iteration cost per bump should be scripted (extend `update-vendor-hash`-style app).
5. **CI/devShell tool-version alignment:** workflow pins golangci-lint v2.12 (copied from go-output), devShell ships 2.14. Align on one and note the other as the deliberate stragglers' lane.
6. **Fleet decision on machine-level git hooks:** global `core.hooksPath=.githooks` silently disables every git-hooks.nix hook fleet-wide (including go-output's). Either drop the global, adopt `.githooks/` as the fleet hook dir, or accept BuildFlow as the only firing gate — but decide once, not per-repo.

## f) 50 things to get done next (brainstorm — HARVEST input; `[S]` = spawned this session, `[B]` = existing TODO_LIST/ROADMAP backlog, `[P]` = policy/fleet)

| #      | Task                                                                                                                                                                                                                                                      | Impact       | Effort | Category          |
| ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ----------------- |
| ~~1~~  | ~~`[B]` Fix `TestIngestRecordsCorrectsAndRejects`: projection for `telnyx-conversation:conv-a` never converges (ingest_test.go:69) — the gate's only red step~~ done — root-caused the same night (StreamMarker prefix + one-shot drainer; 04:59 session) | ~~Critical~~ | ~~M~~  | ~~Bug~~           |
| ~~2~~  | ~~`[S]` Add `var version = "dev"` to `cmd/pbx-lens` so nix ldflags version injection actually lands; `version` subcommand prints it~~ **Won't implement — kept the hard-coded dev placeholder; no release train yet.**                                    | ~~High~~     | ~~S~~  | ~~Feature~~       |
| ~~3~~  | ~~`[B]` httptest end-to-end test for the ingest-telnyx command loop (now in `internal/cli`)~~ done (docs-health pass 2026-10-12)                                                                                                                          | ~~High~~     | ~~M~~  | ~~Quality~~       |
| 4      | `[B]` Raise CI coverage floor 50 → fleet 80 once cmd/wire have tests **→ ROADMAP theme 2**                                                                                                                                                                | High         | S      | Quality           |
| 5      | `[B]` T6 list page (table, AI badge, excerpt, filters) — the 51% slice **→ ROADMAP theme 2 (public presence)**                                                                                                                                            | High         | L      | Feature           |
| ~~6~~  | ~~`[B]` Adopt go-appkit core + templ-components frontend stack (plan §2–§4)~~ done (docs-health pass 2026-10-12)                                                                                                                                          | ~~High~~     | ~~L~~  | ~~Feature~~       |
| ~~7~~  | ~~`[B]` Transcript-excerpt order-dependence: pin semantics + test~~ done (docs-health pass 2026-10-12)                                                                                                                                                    | ~~High~~     | ~~S~~  | ~~Bug~~           |
| ~~8~~  | ~~`[B]` T5 webhook JSONL tailer + ingest command~~ done (docs-health pass 2026-10-12)                                                                                                                                                                     | ~~High~~     | ~~M~~  | ~~Feature~~       |
| ~~9~~  | ~~`[B]` T12 poller timers + freshness self-check~~ done (docs-health pass 2026-10-12)                                                                                                                                                                     | ~~High~~     | ~~M~~  | ~~Feature~~       |
| 10     | `[S]` Cut v0.1.0 and adapt go-output's release.yml + pre-tag-check to single-module **→ TODO_LIST blocked row (owner release gate)**                                                                                                                      | High         | M      | Feature           |
| 11     | `[S]` Update FEATURES.md with the new infra capabilities (CI lane, devShell, update-vendor-hash, cqrs-lint) **→ ROADMAP theme 2 (tooling polish)**                                                                                                        | Med          | S      | Documentation     |
| ~~12~~ | ~~`[S]` Harvest this report's (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST)~~ done (docs-health pass 2026-10-12)                                                                                                                               | ~~Med~~      | ~~S~~  | ~~Documentation~~ |
| ~~13~~ | ~~`[S]` Align golangci-lint CI pin (v2.12) with devShell (2.14) — one number, one place~~ done (docs-health pass 2026-10-12)                                                                                                                              | ~~Med~~      | ~~S~~  | ~~Cleanup~~       |
| ~~14~~ | ~~`[S]` Decide + document `.golangci.yml` timeout 10m vs go-output's 5m~~ done (docs-health pass 2026-10-12)                                                                                                                                              | ~~Med~~      | ~~S~~  | ~~Documentation~~ |
| ~~15~~ | ~~`[S]` cqrs-lint rev refresh policy (owner + cadence; consider an update app that automates the two-hash dance)~~ done (docs-health pass 2026-10-12)                                                                                                     | ~~Med~~      | ~~M~~  | ~~Cleanup~~       |
| ~~16~~ | ~~`[S]` Verify aarch64-linux/aarch64-darwin nix builds (`nix flake check --all-systems`) — CI only proves x86_64-linux~~ done (docs-health pass 2026-10-12)                                                                                               | ~~Med~~      | ~~S~~  | ~~Quality~~       |
| ~~17~~ | ~~`[S]` First real push to exercise ci.yml; fix whatever the runner exposes~~ done (docs-health pass 2026-10-12)                                                                                                                                          | ~~Med~~      | ~~S~~  | ~~Quality~~       |
| ~~18~~ | ~~`[S]` Run the never-run on-demand scans once: `buildflow -s gitleaks`, `-s codespell`, `-s markdown-lint` (covers this report + today's doc edits)~~ done (docs-health pass 2026-10-12)                                                                 | ~~Med~~      | ~~S~~  | ~~Quality~~       |
| 19     | `[P]` Fleet decision on pre-commit: BuildFlow `precommit install` vs `.githooks/` convention vs inert git-hooks hooks (global `core.hooksPath` makes the current wiring dead) **→ fleet decision; BuildFlow is the firing gate (documented)**             | Med          | S      | Policy            |
| ~~20~~ | ~~`[S]` Re-verify TODO_LIST evidence rows against the new `internal/cli` reality (several rows cite `cmd/pbx-lens/main.go` line numbers that the CLI-layer extraction moved)~~ done (docs-health pass 2026-10-12)                                         | ~~Med~~      | ~~S~~  | ~~Documentation~~ |
| ~~21~~ | ~~`[B]` Move `resolveKey`/`parseSince` into `internal/cli` + tests (evidence row needs the same path refresh)~~ done (docs-health pass 2026-10-12)                                                                                                        | ~~Med~~      | ~~S~~  | ~~Cleanup~~       |
| ~~22~~ | ~~`[B]` Fix smoke command's real-looking fake number (+14084757593 → 555 fake)~~ done (docs-health pass 2026-10-12)                                                                                                                                       | ~~Med~~      | ~~S~~  | ~~Bug~~           |
| 23     | `[B]` T8 per-call detail page + recording streaming proxy **→ ROADMAP theme 1 (export streaming)**                                                                                                                                                        | Med          | L      | Feature           |
| ~~24~~ | ~~`[B]` T10 FTS5 transcript search~~ done (docs-health pass 2026-10-12)                                                                                                                                                                                   | ~~Med~~      | ~~M~~  | ~~Feature~~       |
| 25     | `[B]` T9 agent-transcript join (+ revisit E003 flat-domain decision) **→ documented HOLD (AGENTS.md, docs/cqrs-lint-policy.md)**                                                                                                                          | Med          | M      | Feature           |
| ~~26~~ | ~~`[B]` T7 systemd unit + thin NixOS module for the sibling repo~~ done (docs-health pass 2026-10-12)                                                                                                                                                     | ~~Med~~      | ~~M~~  | ~~Feature~~       |
| ~~27~~ | ~~`[B]` T14 nightly backup-staging for the index DB~~ done (docs-health pass 2026-10-12)                                                                                                                                                                  | ~~Med~~      | ~~S~~  | ~~Feature~~       |
| ~~28~~ | ~~`[B]` T25 quality dashboard page~~ done (docs-health pass 2026-10-12)                                                                                                                                                                                   | ~~Med~~      | ~~L~~  | ~~Feature~~       |
| ~~29~~ | ~~`[B]` T24 htpasswd basic-auth for the lens UI~~ done (docs-health pass 2026-10-12)                                                                                                                                                                      | ~~Med~~      | ~~M~~  | ~~Feature~~       |
| ~~30~~ | ~~`[B]` httptest e2e suite for UI routes (list→filter→detail→search→auth→stale)~~ done (docs-health pass 2026-10-12)                                                                                                                                      | ~~Med~~      | ~~M~~  | ~~Quality~~       |
| ~~31~~ | ~~`[B]` README: drop the `go install @latest` apology until a tag exists (unblocks with #10)~~ done (docs-health pass 2026-10-12)                                                                                                                         | ~~Low~~      | ~~S~~  | ~~Documentation~~ |
| 32     | `[B]` T11 dialplan-path AI-vs-human badge for FreeSWITCH CDRs **→ ROADMAP theme 2 (public presence)**                                                                                                                                                     | Low          | S      | Feature           |
| ~~33~~ | ~~`[B]` T13 Telnyx conversation insights per AI call~~ done (docs-health pass 2026-10-12)                                                                                                                                                                 | ~~Low~~      | ~~M~~  | ~~Feature~~       |
| ~~34~~ | ~~`[B]` T15 cost column via Detail Records poller~~ done (docs-health pass 2026-10-12)                                                                                                                                                                    | ~~Low~~      | ~~M~~  | ~~Feature~~       |
| ~~35~~ | ~~`[B]` Export API: JSONL/CSV dump of the index~~ done (docs-health pass 2026-10-12)                                                                                                                                                                      | ~~Low~~      | ~~M~~  | ~~Feature~~       |
| ~~36~~ | ~~`[B]` File upstream: cqrs-lint C009 exemption excludes bare `must` (`len(name) > 4`) — verify at source, then file with verify-before-filing~~ done (docs-health pass 2026-10-12)                                                                       | ~~Low~~      | ~~S~~  | ~~Bug~~           |
| ~~37~~ | ~~`[S]` `nix flake update` (go-nix-helpers `ad423c8` → current) + rebuild; schedule a cadence~~ done (docs-health pass 2026-10-12)                                                                                                                        | ~~Med~~      | ~~S~~  | ~~Cleanup~~       |
| 38     | `[S]` Improve devShell first-entry UX: the one-time loud pre-commit "Cowardly refusing" message on hooksPath machines — consider README note or `enable...` guard **→ fleet decision; BuildFlow is the firing gate (documented)**                         | Low          | S      | Documentation     |
| 39     | `[S]` Verify README reflects the adopted reality (nix build/run, devShell, CI badge placeholder) **→ ROADMAP theme 2 (public presence)**                                                                                                                  | Med          | S      | Documentation     |
| 40     | `[P]` Decide `.config/metadata.yaml` fleet policy (go-output has it, go-nix-helpers doesn't) **→ fleet policy, undecided**                                                                                                                                | Low          | S      | Policy            |
| 41     | `[P]` Quarterly re-verify of the Telnyx live-API truths cluster (AGENTS.md mandate; next due ~2027-01) **→ TODO_LIST blocked row (needs a real API key)**                                                                                                 | Med          | M      | Quality           |
| 42     | `[S]` Coverage reporting per-package (total 52.2% hides cmd/wire at 0%) — consider per-package floors in ci.yml **→ ROADMAP theme 2**                                                                                                                     | Low          | S      | Quality           |
| 43     | `[S]` CI: add a concurrency group to cancel superseded runs (fleet-level idea; go-output lacks it too) **→ ROADMAP theme 2 (CI polish)**                                                                                                                  | Low          | S      | Quality           |
| 44     | `[S]` Extend `update-vendor-hash` app to also handle the vendored cqrs-lint hashes (one command, all hashes) **→ ROADMAP theme 2 (tooling polish)**                                                                                                       | Low          | M      | Cleanup           |
| 45     | `[P]` Decide `git-town.toml` rollout across the fleet (added here because both reference repos carry it; schema-valid TOML, but git-town usage itself is unconfirmed for this repo) **→ fleet policy**                                                    | Low          | S      | Policy            |
| ~~46~~ | ~~`[B]` FTS/index maintenance story (vacuum/analyze cadence) once ingest volume is real~~ done (docs-health pass 2026-10-12)                                                                                                                              | ~~Low~~      | ~~M~~  | ~~Quality~~       |
| ~~47~~ | ~~`[S]` Confirm `nix develop` works on a fresh clone (pre-commit config generation + tool paths) — local machine has warm caches~~ done (docs-health pass 2026-10-12)                                                                                     | ~~Med~~      | ~~S~~  | ~~Quality~~       |
| ~~48~~ | ~~`[S]` Dependabot: expect first github-actions PRs; review whether gomod dependabot should stay alongside `buildflow update` (go-output dropped gomod)~~ done (docs-health pass 2026-10-12)                                                              | ~~Low~~      | ~~S~~  | ~~Policy~~        |
| 49     | `[S]` Document the adopted-pattern deviations in one place (BuildFlow-owns-test/lint/tidy; no release.yml yet; hooksPath caveat) — currently spread across AGENTS.md + this report **→ fleet decision; BuildFlow is the firing gate (documented)**        | Low          | S      | Documentation     |
| 50     | `[P]` Genuinely consider the E003 flat-`internal/calls` split once `internal/cli` + agent transcripts settle (owner question already open in AGENTS.md) **→ documented HOLD (AGENTS.md, docs/cqrs-lint-policy.md)**                                       | Low          | M      | Architecture      |

## g) Three questions I cannot answer myself

1. **Coverage floor policy:** I set the CI gate at 50% (measured floor; packages 76–94%, cmd/wire at 0%). Do you want it at 50 until cmd/wire are tested (current), or go-output's 80 immediately with cmd/wire exempted like go-output's test-support modules — i.e., should the gate be green-now or honest-pain-now? **→ ROADMAP theme 2**
2. ~~**cqrs-lint pin:** the devShell ships cqrs-lint built from go-cqrs-lite master (`afaffa19`), which is newer than the v4.x library tags pbx-lens's go.mod pins. Is lint-binary-ahead-of-library acceptable to you (my assumption), or should the vendored rev track the same versions the repo consumes — accepting that per-version cqrs-lint binaries may not exist as tags?~~ done — accepted and documented (AGENTS.md vendored cqrs-lint note)
3. **Pre-commit end state on this machine:** your global `core.hooksPath=.githooks` silently disables every git-hooks.nix hook fleet-wide (including go-output's). Do you want (a) status quo — BuildFlow stays the only firing gate, the nix wiring stays decorative; (b) `buildflow precommit install` in covered repos; or (c) a fleet migration to `.githooks/` as the hook dir? **→ fleet decision; BuildFlow is the firing gate (documented)**

---

_Point-in-time snapshot; will go stale. Section (f) is the primary input for docs-health HARVEST into TODO_LIST.md / ROADMAP.md (not yet harvested — operator asked to wait for instructions)._
