# Concurrent-session norm — sharing this tree with other agents and the daemon

Status: DRAFT for owner confirmation (P2 of the
2026-10-25 Pareto plan). Everything below was paid for during the
2026-10-23 → 2026-10-25 week (three fake-red gates, one dormant hook
chain, one broken devShell, one truncated TODO_LIST) — this file exists so
the next session does not re-pay.

Two or more agent sessions routinely work this repository at the same
time, and an auto-commit daemon sweeps the tree (60+ commits in 12 hours
at peak). That is the operating condition, not an incident.

## The rules

1. **Claim files before editing shared surfaces.** Before editing a file
   another session may also touch (`AGENTS.md`, `TODO_LIST.md`,
   `flake.nix`, anything under `scripts/`, the gate docs), write a marker
   file `.claims/<file-path-with-slashes-replaced>.<session>.claim`
   containing the session name and a timestamp. Check for a foreign claim
   first: formatting and re-indexing another session's file is fine;
   changing its CONTENT while it is claimed is not. Claims are gitignored
   and expire when their owner's session ends — a stale claim is a note,
   not a lock.
2. **No receipt from a moving tree.** A verification receipt (a gate run,
   a benchmark number, a probe sweep) is only evidence if the tree stood
   still while it ran. If another session landed edits mid-run, the
   receipt is void: say so and re-run. A red gate during concurrent edits
   is evidence about the tree state, not a defect — the
   "concurrent session's dirty tree fakes red gates" gotcha is the
   canonical case.
3. **Expect the daemon to sweep your work.** Uncommitted changes can be
   committed within seconds under a `chore: auto-commit … (heuristic)`
   message. Therefore: when the commit MESSAGE matters, `git add … &&
   git commit …` in the SAME tool call as the final edit; if the daemon
   beat you anyway, amend the message (only ever while UNPUSHED). Never
   leave a half-edited pinned hash (`vendorHash`, FOD hashes) on disk —
   one atomic edit with the derived value, or the daemon commits the
   placeholder (paid for 2026-10-25).
4. **A flaky test is a finding, not a re-roll.** Never re-run until
   green; either diagnose (the `scenarioAwaitTimeout` fix) or record the
   flake with its failure signature in the report. A green re-roll
   without understanding is a falsified receipt.
5. **Per-clone state can silently disarm the gates.** Local git config
   (`core.hooksPath`), devShell health, and tool pins are clone state
   another session may have changed while the docs still claim
   otherwise. Verify the gate is ARMED (`git config --get
   core.hooksPath`, `nix develop -c true`) before relying on it — the
   2026-10-25 dormant-hook incident survived a week of "green" claims
   this way.
6. **Proofs assert the message, not just the exit.** A guard is proven
   by the refusal it prints (assert the string), never by a non-zero
   exit alone — pipelines hide exit codes (`git commit … | tail` faked a
   pass in the first hook proof).

## Where this is enforced

- `.gitignore` carries `.claims/`.
- `.githooks/pre-commit` gates every commit (including the daemon's) once
  `core.hooksPath` points at `.githooks` (rule 5: per clone).
- Status reports record concurrent-session interference under their
  own-failures sections; that is the audit trail for rules 2–4.
Proof seed: [broken](./this-file-does-not-exist.md) link.
