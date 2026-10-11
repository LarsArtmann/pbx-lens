<!-- docs/status/2026-10-11_23-41_gate-closure-verification-status.md
     Status report for the 2026-10-11 ~20:40-23:41 dispatch: the single
     open item from the 20:26 dispatch-3 report — the full buildflow dev
     gate rerun — plus the two failures paid for on the way to green.
     Format .md per this repo's recorded dispatch-report convention
     (user-named path). Scope: THIS session only; no new research.
     All numbers verified in-session against command output and logs. -->

# Status: Gate Closure Verification — 2026-10-11 23:41

**Session scope:** rebuild the stale todo list, re-verify suite state, and close the ONE open verification item from dispatch 3 (full buildflow dev gate rerun). Report written at owner instruction; session then WAITS.

**Headline: GATE CLOSED GREEN.** `buildflow --build-mode dev` = **73 success, 0 failed, 0 skipped** (23:30, exit 0, 18.4 s wall; nix-build ✔ 737 ms on a warm FOD). It took three full-gate attempts and two failures paid for — both now documented gotchas, neither a code defect.

| a) Fully done | b) Partially done | c) Not started | d) Fucked up |
| ------------- | ----------------- | -------------- | ------------ |
| 8             | 4                 | 7              | 3            |

---

## a) FULLY DONE

1. **Todo list rebuilt and closed (10/10).** The stale visible list from dispatch 3 was replaced with true statuses; every P3-P8/P11 item plus the throttle/CI-lane/age-gate and annotation-sweep rows are `completed`. Evidence: `todos` tool final state, 0 pending.
2. **Suite re-verified before the gate (20:41).** `GOEXPERIMENT=jsonv2 go build ./...` clean; `go test ./...` = **13/13 packages ok** (129 test functions inherited from dispatch 2-3). Evidence: command output this session.
3. **Full buildflow dev gate GREEN (23:30).** `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev` → exit 0; `73 success, 0 failed, 0 skipped (+3 via config)`; nix-build ✔ 737 ms. Remaining ✗ entries are exactly the documented advisories (cqrs-lint 17 warnings/infos per policy, go-auto-upgrade 13, structure-linter 1 known, version-auto-configure 1 dep-forced, statix 15, nix-checker 6). Evidence: `/tmp/gate-run3.log` summary; recorded in `docs/buildflow-dev-gate.md`.
4. **dprint-format failure fixed.** Run 1's only failure was `dprint-format` on `TODO_LIST.md`: a hand-padded markdown table whose cells sat at 491-byte rows while the widest cell defines 547. Fixed with `nix develop -c dprint fmt TODO_LIST.md` — **whitespace-only** (15/15 lines re-padded, byte-compared line 36 old-vs-new: content identical). Committed by auto-daemon as 0327a54.
5. **vendorHash re-verified green post-templ-bump.** A concurrent session landed `a-h/templ v0.3.1020 → v0.3.1070` (40e1a49, 10:00). Ran `nix run .#update-vendor-hash`: hash unchanged (`sha256-Rucg9d…`), **verified with a green full nix build** — which also pre-validated the gate's nix-build FOD. The preflight "stale FOD hash" warning is an mtime-heuristic false positive.
6. **go.mod tidy-ness verified.** `go mod tidy` after the gate = zero diff; module is tidy. The go-generate failure (see d-2) was not a real module inconsistency.
7. **Two new paid-for gotchas + gate-closure record documented.** `docs/buildflow-dev-gate.md` gained: (i) dprint table-padding failure class + fix, (ii) go-generate transient + "rerun before diagnosing" doctrine, (iii) the green closure record with numbers. dprint check over the repo passes after my edits.
8. **Docs-state consistency check.** Both dispatch status reports present; TODO_LIST carries 11 `DONE` markers; no orphaned claims from this session's work.

## b) PARTIALLY DONE

1. **Gate reliability = single green sample.** One green run closes dispatch 3's verification item, but tonight's ledger is 2 failures / 1 success. What's missing: a stated reliability bar (N consecutive greens?) and a scheduling policy. Blocker: owner question ②. Effort: S once decided.
2. **go-generate transient root cause.** What works: step passes standalone and in the final gate; module provably tidy. What's open: the actual mechanism — theory is a race on the SHARED `/mnt/buildcache/go-mod` GOMODCACHE under concurrent sessions; unproven (I did not capture the exact `go` invocation under `--verbose` before it passed). Effort: M with a repro attempt.
3. **AGENTS.md templ-pin currency.** Stack decisions still say "templ-components v1.21.0 … a-h/templ v0.3.1020" while go.mod now requires v0.3.1070 (40e1a49). Noticed this session; NOT edited because the owner ordered report-then-wait. What remains: confirm the bump is intended, sync the pin note + adoption-map caveat. Effort: S.
4. **§f harvest into TODO_LIST.** The 50-item list below is the HARVEST input, but the owner's instruction was report-then-wait, so TODO_LIST was not touched this session. Effort: S via docs-health HARVEST on owner go.

## c) NOT STARTED

1. **Owner ① — go-atomic-write `Commit` primitive filing.** Draft ready at `docs/drafts/go-atomic-write-commit-primitive.md`; posting is blocked on owner go (P9 policy: no upstream filings without it).
2. **Owner ② — gate scheduling decision** (idle-only, serial mode, or load-gated wrapper). Tonight's 75-minute wait for load < 20 is the cost of not deciding.
3. **Owner ③ — production `-max-age` window + v0.1.0 release.** Blocked on the sibling NixOS module's real ingest cadence; release is preconditioned on it.
4. **v0.1.0 tag itself.** Everything else green; no tag cut (owner-gated).
5. **Idle/load-gate tooling.** No `scripts/` helper exists for "wait for load < N, then run gate"; tonight used an ad-hoc shell loop (which had its own bug, see d-1).
6. **docs-health ANNOTATE sweep of the 19 status reports.** Older reports accumulate unverified-forever claims; not touched this session.
7. **D6 upstream filing monitoring** (go-cqrs-lite#62/#63/#64, go-nix-helpers#7, treefmt-nix#545) — standing, not started this session.

## d) TOTALLY FUCKED UP

1. **My own wait-loop was broken for its first 10 minutes (severity: medium, self-inflicted).** The first load-poll used `bc`, which is not installed on this machine — every comparison failed silently, the loop no-op'd through the exact window where load exploded 31 → 157, and I burned the one low-load window of the evening. Root cause: I scripted around a tool I never verified exists. Workaround: awk-based rewrite (which worked, 74 min later). Lesson recorded mentally; belongs in e-1.
2. **go-generate transient is undiagnosed (severity: medium if it recurs).** One gate run failed with `go: updates to go.mod needed` while `go build` and `go mod tidy` were simultaneously clean; rerun passed with zero changes. If this fires randomly on shared-cache contention, gate trust erodes — the exact failure mode that produced tonight's three-attempt saga. Root cause: UNKNOWN (shared-GOMODCACHE race is the working theory, unproven). Mitigation in place: rerun-before-diagnosing doctrine in `docs/buildflow-dev-gate.md`.
3. **Gate historical failure stats are alarming and unowned (severity: medium, environmental).** Buildflow's own preflight reports `nix-hash-fix failed 55/64 (86%)` and `nix-build-verify failed 6/6 (100%)` historically — on THIS repo, on THIS machine. Tonight added a dprint failure and a go-generate failure. Individually explained; collectively the gate's signal-to-noise is poor and the scheduling decision (owner ②) has been open across three sessions.

## e) WHAT WE SHOULD IMPROVE

1. **Verify tooling before scripting around it** — `command -v` any binary a loop depends on. Impact: tonight cost a 10-minute dead window + a wasted low-load slot. Fix: one-line guard at loop start; adopt into any future wait-script.
2. **Gate scheduling economics** — 3 runs, ~75 min idle-waiting, 1 green. Impact: hours per session under load. Fix: owner ② decision, then implement (load-gated wrapper or serial mode).
3. **Root-cause discipline for transient gate failures** — rerun-passed is a mitigation, not a diagnosis. Impact: unknown unknowns. Fix: next occurrence, capture `buildflow -s go-generate --verbose` output to a file BEFORE rerunning; diff go.mod/go.sum mtimes against concurrent-session commits.
4. **Shared-cache awareness** — GOMODCACHE is `/mnt/buildcache/go-mod`, shared across 40+ sessions. Impact: unprovable-but-plausible race class. Fix: consider session-local GOMODCACHE for gate runs, or fleet-document the race pattern.
5. **Sample-size bar for "gate green"** — one green ≠ reliable. Fix: define the bar (e.g. 2 consecutive greens with BUILDFLOW_NO_RESULT_CACHE=1) and record it next to the gate command in AGENTS.md.
6. **Markdown table edits must end in `dprint fmt`** — hand-padded tables are a repeatable gate-blocker class (blocks 7 downstream steps via the DAG). Fix: recorded in gate doc; make it muscle memory after any TODO_LIST table edit.
7. **Cross-session change awareness** — a concurrent session bumped templ, rebuilt buildflow, and edited TODO_LIST mid-flight. Impact: two of tonight's surprises were their (legitimate) work. Fix: pre-flight `git log --since='2 hours ago' --oneline` before gate runs; treat preflight warnings as first-class triage input, not noise.
8. **Preflight-warning backlog** — tonight's runs surfaced: 9 tools unavailable (health check), prettier/vulnix not in devShell (nix-run fallback), interrogate missing, lychee token-vs-exclude policy undecided. Individually advisory; collectively they dilute preflight signal. Fix: one devShell round-up pass (see f-12/13/14).

## f) TOP 50 NEXT TASKS (brainstorm — most are ROADMAP fuel; HARVEST with routing rigor)

Ranked by impact; Effort S <30 min, M 30 min-2 h, L >2 h.

| #  | Task                                                                                                                                         | Impact | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Sync AGENTS.md templ pin note to v0.3.1070 + confirm the bump is owner-approved (d-3)                                                        | High   | S      | Documentation |
| 2  | Owner ② decision: gate scheduling (idle-only vs serial vs load-gate)                                                                         | High   | S      | Decision      |
| 3  | Owner ① decision: file go-atomic-write `Commit` upstream (draft exists)                                                                      | High   | S      | Decision      |
| 4  | Owner ③ decision: production `-max-age` for sibling NixOS module                                                                             | High   | S      | Decision      |
| 5  | Cut v0.1.0 tag once ③ lands (precondition recorded)                                                                                          | High   | S      | Release       |
| 6  | Harvest this §f list into TODO_LIST/ROADMAP (docs-health HARVEST)                                                                            | High   | S      | Documentation |
| 7  | Root-cause go-generate transient: verbose capture + cache-race repro                                                                         | High   | M      | Bug           |
| 8  | Second consecutive green gate run (sample-size bar, e-5)                                                                                     | High   | S      | Quality       |
| 9  | Load-gate wrapper script for buildflow (scripts/await-low-load + gate)                                                                       | High   | S      | Quality       |
| 10 | CI run check on templ bump (go.mod changed after last known-green CI)                                                                        | High   | S      | Quality       |
| 11 | Dependabot templ handling: durable pin/ignore so bumps don't "mangle go.mod" (40e1a49 msg)                                                   | Medium | S      | Cleanup       |
| 12 | Add prettier to devShell (kills nix-run-without-deps fallback warning)                                                                       | Medium | S      | Quality       |
| 13 | Add vulnix to devShell (same warning class)                                                                                                  | Medium | S      | Quality       |
| 14 | Triage "9 tools unavailable (health check failed)" — name them, install or configure out                                                     | Medium | M      | Quality       |
| 15 | Fix 2 standing staticcheck QF1008 nits in `internal/wire/flextime_test.go`                                                                   | Medium | S      | Cleanup       |
| 16 | Run on-demand gate steps once for a baseline: markdown-lint, gitleaks, codespell                                                             | Medium | M      | Quality       |
| 17 | Decide lychee policy: GITHUB_TOKEN vs exclude-pattern (fleet question, recorded warning)                                                     | Medium | S      | Decision      |
| 18 | Triage statix 15 findings: fix or document as accepted                                                                                       | Medium | M      | Cleanup       |
| 19 | Record gate failure-rate baseline (55/64, 6/6, tonight's 2) with dates in gate doc for trend tracking                                        | Medium | S      | Documentation |
| 20 | Session-local GOMODCACHE experiment for gate runs (e-4)                                                                                      | Medium | M      | Quality       |
| 21 | Warm/cold nix-build timing note in gate doc (737 ms warm tonight) for starvation triage                                                      | Low    | S      | Documentation |
| 22 | D6 filings: check responses on go-cqrs-lite#62/#63/#64, go-nix-helpers#7, treefmt-nix#545                                                    | Medium | S      | Upstream      |
| 23 | File cqrs-lint C009 bare-`must` exemption upstream (verified in-repo; TODO_LIST row)                                                         | Medium | S      | Upstream      |
| 24 | Periodic revisit: go-auto-upgrade 13 findings (json v2 waits on T4)                                                                          | Low    | S      | Quality       |
| 25 | nix-checker 6 hash-bookkeeping infos: cleanup pass or accepted-note                                                                          | Low    | S      | Cleanup       |
| 26 | SegmentBar adoption when templ-components publishes it (dashboard hand-rolled bar)                                                           | Low    | S      | Feature       |
| 27 | Verify sibling-repo state: module landed per TODO_LIST 22:31 note — confirm `modules/telephony/lens.nix` matches the draft's intent          | Medium | M      | Verification  |
| 28 | Live-verify Telnyx Detail Records + insights DTOs against real API (TODO_LIST row)                                                           | Medium | M      | Feature       |
| 29 | Quarterly re-verify Telnyx live-API cluster (standing cadence; last 2026-10-10)                                                              | Low    | M      | Quality       |
| 30 | PII policy: make `E164`/`ParticipantHash` load-bearing now that retention+purge exist (T23 tail)                                             | Medium | M      | Feature       |
| 31 | Search UX: substring number search hint (FTS5 tokenizes `+1728…` as one token)                                                               | Low    | S      | Feature       |
| 32 | Backup restore drill: end-to-end restore test, not just `Verify`                                                                             | Medium | M      | Quality       |
| 33 | Wire TestLiveServerEndToEnd into the CI lane (currently a local checked-in probe)                                                            | Medium | S      | Quality       |
| 34 | Measure coverage post-129-tests; set a target                                                                                                | Low    | S      | Quality       |
| 35 | Dashboard idle-window UX revisit (day window ends "today UTC" → idle index drifts out of chart)                                              | Low    | S      | Feature       |
| 36 | CHANGELOG entry for tonight: gate closure + dprint/go-generate gotchas                                                                       | Low    | S      | Documentation |
| 37 | docs-health ANNOTATE sweep of 19 status reports (stale-claim risk)                                                                           | Low    | M      | Documentation |
| 38 | `pbx-lens version`: include git-dirty marker for deploy tracing                                                                              | Low    | S      | Feature       |
| 39 | Gate runbook entry: "how to rerun safely at low load" next to the command in AGENTS.md                                                       | Low    | S      | Documentation |
| 40 | Vendor-hash drift check in CI (catch before local gate does)                                                                                 | Medium | S      | Quality       |
| 41 | Evaluate `go-version-auto-configure` finding noise: dep-forced floor note is documented — can the checker learn it? (upstream BuildFlow ask) | Low    | M      | Upstream      |
| 42 | buildflow binary freshness: either pin the installed rev in flake or automate `nix run .#reinstall` cadence (advisory fires every run)       | Low    | M      | Cleanup       |
| 43 | interrogate tool: install in devShell or add to skip_steps config                                                                            | Low    | S      | Quality       |
| 44 | Pre-gate `git log --since` check as a habit/script (e-7)                                                                                     | Low    | S      | Quality       |
| 45 | E003 flat-domain split revisit trigger: define "when T9 lands" concretely in cqrs-lint-policy doc                                            | Low    | S      | Documentation |
| 46 | ContentHash length-prefixed encoding design note (schema v2 scheduled — capture the design while fresh)                                      | Low    | M      | Documentation |
| 47 | Consider per-report HTML-canonical migration policy for docs/status (mixed .md/.html today)                                                  | Low    | S      | Documentation |
| 48 | Machine hygiene flag to owner: 40+ users, load peaks 157 on shared box — affects every gate                                                  | Medium | S      | Decision      |
| 49 | AGENTS.md adoption-map: verify table still accurate against templ-components v1.21.0 post-bump (spot-check 2 components)                     | Low    | S      | Verification  |
| 50 | Nightly-gate experiment: scheduled idle-hours run + failure-rate dashboard (owns the 86%/100% stats)                                         | Low    | L      | Quality       |

## g) QUESTIONS ONLY THE OWNER CAN ANSWER

1. **Is the concurrent session's templ bump (v0.3.1020 → v0.3.1070, 40e1a49) owner-approved?** I verified the build/hash are green with it, but I cannot know the other session's mandate — if it was exploratory, AGENTS.md's pin doctrine ("stay on PUBLISHED versions") needs a deliberate re-pin decision, not just a doc sync.
2. **Gate scheduling (owner ②, now with data):** tonight cost ~75 min of idle-waiting and 2 of 3 runs failed environmentally, and buildflow's own history shows 86%/100% step-failure rates on this machine. Idle-only gates, serial mode, or accept the noise? I can implement any of the three; I cannot pick the policy.
3. **What is the reliability bar for "gate closed"?** One green run satisfied dispatch 3's literal verification item. Before v0.1.0 tagging, do you want N consecutive `BUILDFLOW_NO_RESULT_CACHE=1` greens (my suggestion: 2), or is one sufficient?

---

_Point-in-time snapshot; goes stale. §f is HARVEST input (docs-health), §d-1/§e-1 are fleet-lesson candidates. Session WAITS for instructions per owner order._
