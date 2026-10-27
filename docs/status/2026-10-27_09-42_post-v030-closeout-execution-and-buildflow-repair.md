# Post-v0.3.0 closeout execution + the BuildFlow binary repair — status (2026-10-27 09:42 CET)

Session window 2026-10-27 00:14 → 09:42 CET, over the resume point `b20ce19`
(the 23-43/23-48 plan-execution + self-review pair). This is the execution
session that consumed the self-review's §b remainder list: the three
one-sentence debt fixes, the golangci pin decision, the sibling T6
verification, the parked BuildFlow binary rebuild, and a fresh banked gate
receipt. Ending state: pbx-lens `main` at `d6e4265`, tree clean, **26
commits ahead of origin — NOT PUSHED**; BuildFlow `master` at `49ced3e00`,
**2 commits ahead of its origin — NOT PUSHED**. All four §g questions from
23-48 are now either resolved (golangci, BuildFlow checkout) or sitting one
word away (the push).

## a) FULLY DONE

1. **Resume reconciliation and state verification.** Todo list rebuilt to
   the true state (T1–T19 done; remaining items re-derived from the 23-48
   §b/§g lists). Verified before acting: pbx-lens clean at `b20ce19`, 22
   ahead; machine uptime (the ~23:10 reboot confirmed real, background
   jobs dead); the BuildFlow checkout still dirty (`go.mod`, `flake.lock`)
   — which drove the whole worktree detour below.
2. **Triage-doc §3 drift fix (self-review §b item, "one sentence owed").**
   `docs/drafts/2026-10-26_advisory-triage-jsonv2-inventory.md` §3 said the
   15-09 audit's "39 findings" "counted types, not sites"; the receipt
   (19-53 row 17, 19-54 row 26) proved the tool counts FILES. Corrected,
   now consistent with both annotations. Commit `00d6577`.
3. **T10's receipt recorded beside R16 (the other "one sentence owed").**
   R16 resolved to the restore drill, documented in README's backup
   section. Re-ran `TestRestoreDrillSnapshotToFreshIndex` fresh for a real
   receipt: **PASS, 0.02s, current schema (2026-10-27)** — and wrote that
   receipt into the README paragraph that documents the drill, so the
   proven claim now carries its latest proof. Commit `00d6577`.
4. **AGENTS solo-gate pointer (budget-neutral).** The Commands block's
   poll-gate comment now says run it SOLO with the starvation mechanism
   named ("parallel jobs starve test-coverage → exit 75"). Line count
   verified unchanged by the gate: 219/219. Commit `00d6577`.
5. **golangci-lint pin alignment — DECIDED AND LANDED (23-48 §g2).** Chose
   **CI-up**: `ci.yml` `GOLANGCI_LINT_VERSION: v2.14` → `v2.15`, because
   the devShell's locked nixpkgs already ships v2.15.0 (local runs = 0
   issues), the parity probe proved v2.14 also = 0, and one CI line beats
   pinning the flake against nixpkgs drift. Triage §4 marked RESOLVED with
   the re-parity rule (next deliberate flake.lock bump re-checks); the
   TODO lint-skew row removed; CHANGELOG `Changed` block added; ci.yml
   YAML-validated. Commit `a6feeae`. The owner can flip this with one line
   if flake-down was preferred — verdict-identical either way.
6. **T6's sibling verification debt paid.** The session-end "sibling flake
   check still running" died with the reboot. Re-proved by evaluating both
   hosts' toplevel derivations in `nix-international-telephony`
   (`nix eval .#nixosConfigurations.{pbx,pbx-prod}.config.system.build.toplevel.drvPath`
   — both resolve, no eval errors), which validates T6's `04c9d61`
   lens.nix/DOMAIN_LANGUAGE changes. No build needed for a wording change.
7. **The BuildFlow binary rebuild — T17's parked half, DONE with root
   cause.** The dirt on `~/projects/BuildFlow` was a dead session's
   mid-flight dep reconcile; a clean HEAD worktree build then failed with
   a go-modules fixed-output hash mismatch. Root cause: BuildFlow's own
   daemon commits (`24dff8181`, later `0b3c1d302`) landed
   execution/go.mod + go.sum + tools/go.mod dependency changes WITHOUT the
   vendorHash refresh, so every `nix build` of that flake failed. Repaired
   per their book: `nix run .#update-vendor-hash` on a clean tree
   (9y4NB… → ffjf0h…, rebuild green), `nix run .#reinstall` →
   **REINSTALL-OK**, `buildflow doctor`: **env/binary-freshness ✓ binary
   is current (built at HEAD 49ced3e)**. The stale-binary advisory that
   has haunted the gate doc since 2026-10-23 is retired. TODO row 120
   removed; receipt + trap recorded in docs/buildflow-dev-gate.md. Commit
   `b76d9dd`.
8. **The worktree trap discovered and documented.** BuildFlow's root
   go.mod carries a RELATIVE `../go-finding/toolsdk` replace while
   tools/go.mod carries the absolute path — identical in the main
   checkout, CONFLICTING from any `git worktree` (the relative path
   resolves against the worktree's parent). Their pre-commit build gate
   therefore can never run from a worktree. Recorded in the gate doc so
   the next repair doesn't pay for the diagnosis.
9. **BuildFlow master repaired forward.** Mid-session the daemon swept the
   dead session's dep bump onto master (`0b3c1d302`); the main tree went
   clean, the sanctioned in-place repair ran there (same hash value,
   rebuild green), and the fix is on master as `49ced3e00`. Master is
   buildable again.
10. **Fresh gate receipt banked at the final tree.** Solo
    `poll-gate.sh -- env BUILDFLOW_NO_RESULT_CACHE=1 buildflow
    --build-mode dev`: START LOAD **34.38/29.13/34.68 (policy 15m < 40)**,
    **51 success / 0 failed / 0 skipped (+5 via config)**, exit 0, wall
    59.5s, run `20261027-050740-2cbac547`, "passed with warnings 47/66,
    337 findings" — the documented detect-only set. Recorded in the gate
    doc with full provenance of the two failed/lost attempts, and in the
    CHANGELOG poll-gate block. Commit `d6e4265`.
11. **Four receipt-grade commits, each gate-checked.** `00d6577` (the
    three debt fixes), `a6feeae` (pin alignment), `b76d9dd` (BuildFlow
    receipt + row closure), `d6e4265` (gate receipt). Pre-commit lanes
    green at every commit; doc gates re-verified independently: links
    20 checked / 0 broken, test claims 79 / 0 phantom, counts 69 archived
    / 10 plans / 18 live (19 after this report registers), budget 219/219,
    art-dupl 0 groups, dprint clean.

## b) PARTIALLY DONE

1. **THE PUSH (23-48 §g1).** 26 pbx-lens commits + 2 BuildFlow commits
   ahead of their origins, everything green, everything receipted — held.
   The resume rule and my hard rule agree: no push without an explicit
   ask, and "get the whole list done" named the list, not the push. One
   command each when you say go.
2. **BuildFlow master content health.** The vendorHash is fixed and the
   binary is fresh, but the CONTENT the daemon swept onto master
   (`0b3c1d302` — the dead session's dep bump) was never proven green by
   BuildFlow's own full pipeline AFTER the fix: the pre-fix pipeline run
   showed 11 failed steps, the post-fix hook run completed (444s) without
   a failure banner in the output I captured, but I never banked a
   full-output green proof, never identified what the 11 steps were, and
   never ran their devShell-verify analog of our vendorHash rule. The
   binary works for OUR gate (receipt proves it); master's own gate state
   is unproven.
3. **The golangci alignment's CI half.** Config changed and locally
   proven, but CI cannot run until the push, and the dead gh token means
   no remote watching — the v2.15 lane's first real verdict arrives
   whenever you push.
4. **Commit-message provenance on the BuildFlow fix.** The vendorHash fix
   CONTENT is on master, but under the daemon's heuristic message — I
   lost the commit race twice. The receipt-grade message exists only in
   pbx-lens's gate doc. Unpushed, so amendable if you care.

## c) NOT STARTED (owner-gated or cycle-gated, unchanged by this session)

- Release object for v0.3.0 (draft paste-ready in drafts/) and the
  v0.3.1-vs-wait call.
- Next docs-health cycle: archive pass over the live set (this report
  makes it 19 live), TODO_LIST re-harvest.
- T18's stamp-churn prototype half + AFTER numbers (needs a live serve and
  your doctrine call; harness `scripts/poll-304-probe.js` + BEFORE numbers
  landed 2026-10-26).
- TODO row 113 (human-browser htmx revalidation check), row 117
  (dependabot weekly cycle confirmation), the upstream filings queue
  (per-filing go-ahead), row 119 (mobile card-list taste call).
- Push-watching infra (gh token dead; all remote claims cite
  last-verified dates).

## d) TOTALLY FUCKED UP

1. **`buildflow receipts list` — an unverified subcommand, invoked
   blind.** Unknown subcommands appear to fall through toward a run; it
   errored out fast and `pgrep` confirmed nothing started, but I gambled
   the exact starvation-free window the whole solo-gate discipline exists
   to protect. `--help` first, always.
2. **I destroyed my own banked receipt.** The first fully green gate run's
   receipt line was cut by my `tail -12` in the capture pipe; proving the
   run cost a complete re-run plus hours more load-window waiting. The
   fix is embarrassing in its simplicity: full output to a file FIRST
   (which is what the re-run did).
3. **Two-trees, zero mental model.** I ran `git add vendorHash.nix` in the
   main BuildFlow checkout while the fix lived only in the worktree copy —
   a no-op stage, a wasted hook run, and a cycle of confusion about where
   the change was. The worktree detour was right; keeping its state
   straight in my head was not.
4. **Lost the commit-message race to the daemon twice.** On BuildFlow the
   fix landed under `chore: auto-commit 1 changed file(s) (heuristic)` —
   for a fix whose WHY (daemon commits without hash refresh) is exactly
   what the message should carry. Atomic change+stage+commit in one call
   was the play; I staged and committed as separate steps.
5. **Armed the gate at 15m=36.38 on a volatile box.** Barely under
   policy; the machine spiked mid-run and one step starved (the wrapper
   caught it correctly, exit-and-rerun). Patience for headroom below the
   line, not a limbo pass under it.
6. **Unverifiable remote claims shipped in config.** The ci.yml bump to
   v2.15 assumes the action fetches that tag cleanly — locally proven,
   remotely unproven (dead token), and I flagged it nowhere in the commit
   beyond the CHANGELOG text. Should have carried a "verify on first CI
   run" note in the watch checklist.

## e) WHAT WE SHOULD IMPROVE

1. **Capture-then-summarize, every time**: `> file 2>&1` before any tail;
   receipts are written once, read many.
2. **Atomic mutations under a live daemon**: edit + stage + commit in ONE
   shell call, or the daemon's heuristic message eats the provenance.
3. **Verify CLI surface before invoking** (`--help`); "reads like a
   subcommand" is not a subcommand.
4. **Re-check external-tree state at resume AND mid-session** — BuildFlow
   flipped dirty → daemon-swept-clean under me; resume context ages in
   minutes on this machine.
5. **After any vendorHash repair, run the repo's devShell-verify norm
   too** (ours: `nix develop -c echo ok`; BuildFlow's analog unchecked
   this time).
6. **Poll-gate launches need headroom** (15m well under 40), not a
   just-under-the-line arm on a box that swings 9 → 140.
7. **A daemon-coordination convention** (staged-marker or lock) so
   receipt-grade messages stop losing races — process-level, your call.
8. **Resolve resume-context references BEFORE editing** ("beside R16"
   cost several searches before it resolved to the README drill section).
9. **Upstream candidates noticed this session**: BuildFlow post-commit
   hook could run `update-vendor-hash` automatically (their own comment
   invites it), and the relative-replace worktree break is a real filing
   candidate — both go through verify-before-filing first.
10. **Machine hygiene noticed**: stale `result`/`result-1`/`result-2`
    symlinks in nix-international-telephony; BuildFlow's
    `workspace/gomod-freshness` doctor scan OOMs under load (≈140 spikes
    killed it) — environmental, but worth a look on a quiet day.

## f) Up to 50 things to get done next

_Push/release lane (mostly your call):_

1. Push pbx-lens (26 commits; CI runs the new concurrency group + v2.15 lane).
2. Push BuildFlow master (2 commits; `49ced3e00` + the daemon's dep-bump sweep).
3. Watch the first CI run of the concurrency group + golangci v2.15 (needs a live token or your eyes).
4. Decide the fate of `49ced3e00`'s heuristic message (amend before push, or leave).
5. Cut the v0.3.0 release object from the paste-ready draft.
6. Decide v0.3.1-vs-wait (one-pager defaults to wait).
7. Run the tag-CI watch checklist when a `v*` tag first pushes.

_BuildFlow follow-ups:_

8. Bank a full-output green proof of BuildFlow's own pipeline on post-fix master.
9. Identify what the pre-fix "11 step(s) failed" were; confirm they're all hash-staleness fallout.
10. Run BuildFlow's devShell-verify norm after the hash repair (their AGENTS.md analog of ours).
11. Decide the dead session's dep bump: finish it deliberately (reconcile + prove) or revert `0b3c1d302`.
12. File the post-commit-hook-runs-update-vendor-hash suggestion upstream (their comment invites it).
13. File the relative-replace worktree break upstream (go.mod replaces vs hooks).
14. Investigate the gomod-freshness doctor OOM (scan killed under load).

_Docs/health:_

15. Register + then next-cycle archive pass (19 live reports now).
16. Verify TODO row 118 against T6's landed lens.nix wording — likely closable.
17. TODO_LIST re-harvest after the push lands.
18. Annotate the 23-43/23-48 pair + this report as their items resolve.
19. Quarterly re-verifications on the calendar: Telnyx recordings API (window 2027-01), versions.json (109/112), go-cqrs-lite v5 watchlist.

_Code/quality leftovers (standing rows):_

20. Stamp-churn AFTER numbers via `scripts/poll-304-probe.js` on a live serve.
21. Stamp-churn prototype half (your doctrine call).
22. jsonv2 flip standing order: P3 → P2 first, P0 last behind a stored-hash proof.
23. GO-2026-6809: toolchain bump to go1.27.3 at the fleet's next go pin.
24. LICENSE files upstream for the 17 go-licenses error lines (your repos).
25. cqrs-lint 19 findings: continue the policy pass (B010 row landed).
26. Nolint re-audit with `--show-suppressed` after the next pin bump.
27. templ-components next bump: run the adoption-map corpus sweep (the v1.23.0 lesson).
28. Row 114: fold in go-etag#2 (Flush observability) when it lands.
29. Row 115: templ-components #48–#63 folds when they land.
30. Row 116: marathon-transcripts buffering policy.
31. Row 121–122: lint-policy intent answer + `golangci-lint config verify` surface.
32. Row 125: AGENTS-budget CI job proof (220-line seed rejection).
33. Row 126: the upstream filings queue, per-filing go-ahead.
34. Golden re-pin audit stays on the checklist for any ops/quality UI change.

_Machine/process hygiene:_

35. Recurring cache.db VACUUM (it re-grows; 194 MB → 33 MB last time).
36. Sibling stale `result*` symlink cleanup.
37. Load-shaping: identify the box's heavy jobs (21 users) before gate windows.
38. Evaluate `--max-concurrency` for busy-machine days (doctrine change, your call).
39. A daemon/commit coordination convention (see §e7).
40. Restore gh token access for remote watching.

_Smaller but real:_

41. Prove the ci.yml v2.15 fetch works on first push (watchlist item 3 dup — keep).
42. Consider a `make`-free flake app that tails the latest gate receipt from the log file convention (§e1 institutionalized).
43. Backup/restore drill: schedule the next scheduled re-run cadence (this one was ad hoc).
44. 304 probe: wire into verify-all as an opt-in lane or leave standalone (your call).
45. The `34.38` START LOAD receipt suggests raising the fleet's quiet-hour guidance — measure first.

_(Capped at ~45 concrete items; the remainder of the 50 slots are the
owner-gated duplicates noted inline above.)_

## g) Three questions I cannot answer myself

1. **Push authorization**: pbx-lens is 26 commits ahead and BuildFlow 2 —
   push both now, one, or neither? (Everything is green and receipted;
   this is the only external action left.)
2. **golangci pin direction**: I landed CI-up (v2.15, `a6feeae`) — keep,
   or flip to flake-down? (Probe says verdict-identical; one line to
   change either way.)
3. **The dead BuildFlow session's dep bump** (now on master as
   `0b3c1d302`): is that dependency sweep still wanted — should a session
   finish it deliberately and prove BuildFlow's pipeline green — or
   revert master to the pre-bump state? (The tree can't tell me the dead
   session's intent.)
