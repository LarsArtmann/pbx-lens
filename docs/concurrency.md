# Concurrent-session norm — sharing this tree with other agents and the daemon

Status: RATIFIED by owner gate §g3 (2026-10-25): **squash-while-unpushed
is the codified daemon norm** — the repair in rule 3 stays legal, pushing
immediately after every receipt-grade commit is the doctrine that keeps it
rare, and pushed daemon commits stay in history (no force-push, no
rebase-of-record). Everything below was paid for during the
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
3. **Expect the daemon to sweep your work — and repair only while
   unpushed.** Uncommitted changes can be committed within seconds under
   a `chore: auto-commit … (heuristic)` message. Therefore: when the
   commit MESSAGE matters, `git add … && git commit …` in the SAME tool
   call as the final edit, and PUSH immediately after — the push is what
   shrinks the race window. If the daemon split your files across
   several chore commits anyway: `git fetch origin` FIRST, then
   `git reset --soft <last-real-commit>` while everything is still
   UNPUSHED, re-commit receipt-grade in one call, verify `git log`, push.
   Never repair pushed history (no force-push, no rewriting landed
   daemon commits — their hygiene loss is accepted; the content was
   verified complete via `git diff <last-real>..HEAD --stat` before any
   squash). Never leave a half-edited pinned hash (`vendorHash`, FOD
   hashes) on disk — one atomic edit with the derived value, or the
   daemon commits the placeholder (paid for 2026-10-25).
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
   this way. The same audit applies to TOOL BINARIES: the dedup gate ran
   for days against a stale user-local art-dupl whose verdicts disagreed
   with CI's pin in both directions (docs/art-dupl-baseline.md).
6. **Proofs assert the message, not just the exit.** A guard is proven
   by the refusal it prints (assert the string), never by a non-zero
   exit alone — pipelines hide exit codes (`git commit … | tail` faked a
   pass in the first hook proof).
7. **Re-stat before any create/overwrite (added 2026-11-12, paid for
   twice the same morning).** The plan said "draft report X" and another
   session had created X minutes earlier — twice. The rule: before
   WRITING any file (report, doc, new module), `ls`/`git status` the
   exact path FIRST; if it exists, READ it and reconcile (a better
   foreign version is kept and credited, not overwritten). Drafting
   from a stale mental map is never cheaper than one `ls`. The paired
   rule: re-check `git log --oneline -3` before a receipt commit — the
   daemon or the interleaved session may have landed work that changes
   what the commit should say.

## Where this is enforced

- `.gitignore` carries `.claims/`.
- `.githooks/pre-commit` gates every commit (including the daemon's) once
  `core.hooksPath` points at `.githooks` (rule 5: per clone).
- Status reports record concurrent-session interference under their
  own-failures sections; that is the audit trail for rules 2–4.

## Rule 3 receipts (2026-10-25, the ratification day)

- Four daemon barrages intercepted feature commits; the first was
  squashed cleanly while unpushed (fetch-first, soft-reset, one
  receipt-grade commit). Later barrages hit already-pushed commits —
  their `chore:` entries stay in history per the ratified norm.
- The repair was executed twice more the same evening, both clean:
  `afae51e` (dedup extraction, 3 chore commits → 1) and `315587e` (gate
  wiring, 2 chore commits → 1), each pushed within a minute of landing.
- Find the squash point fast: `git log --oneline origin/main..HEAD`
  after a `git fetch origin` — everything above the last non-`chore:`
  commit is yours to squash.

## Norm upgrades from the 2026-11-12 double-dispatch (plan A17)

Tonight ran TWO sessions on one repo through a whole plan execution
again — one report rewrite wasted, one gate receipt voided by a
mid-flight edit, one shared file nearly mangled. Three rules tighten:

1. **The claims roster: `.claims/ACTIVE.md`.** Every session opens its
   working stint by writing `.claims/ACTIVE.md` (overwrite freely —
   last writer wins, it is a roster not a lock) with: session name,
   start time, the FILE SCOPE it intends to touch, and the plan/report
   it is executing. Before claiming an individual file, READ the
   roster: if another session's scope covers your target, switch to a
   non-overlapping task instead of "claiming harder". Individual
   `.claims/<file>.<session>.claim` files stay the per-file signal;
   the roster is the at-a-glance scope map that makes collisions
   visible BEFORE the first wasted write. Delete your roster entry on
   exit (stale entries are notes, not locks — same as file claims).
2. **Leave HEAD green, or tag the commit `[red]`.** A session that must
   step away mid-task commits its work in a state it has verified
   (build + the touched package's tests, minimum). If a commit lands
   KNOWN-red (mid-refactor, half a migration), the message's first
   line ends with the literal tag ` [red]` and the body names the
   failing lane. Every other session (and the gate) can then trust
   untagged HEADs and treat `[red]` heads as expected-construction,
   not a regression to bisect. Never push a `[red]` head to origin
   without the tag; never leave `[red]` standing for more than one
   working stint.
3. **Dual-session protocol (STUB — pending the owner's Q1 answer).**
   The open question: when the owner dispatches two sessions, do they
   get DISJOINT plan halves (session A takes A1-A12, session B takes
   A13-A24) or live CLAIM-based arbitration on the full plan? Tonight
   ran full-plan arbitration and paid ~1 report + 1 gate receipt in
   collisions. Until the owner answers: sessions SHOULD partition by
   plan-task range when a shared plan file exists, and both sessions
   record the partition in their ACTIVE.md roster entries. This stub
   becomes a ratified rule or gets deleted on the answer.

### Receipts (2026-11-12)

- One gate receipt voided by a concurrent userauth.go edit mid-run
  (rule 2 applied, re-run queued) — the void is recorded in the gate
  log, not silently re-rolled.
- One wasted report rewrite (the 2026-11-12 audit session, §d.1 of its
  own report) — the collision the roster would have prevented.
