<!-- docs/status/README.md — the live-set index (05-01 §f44): what is live
     vs archived, at a glance. Update the live table whenever a report lands
     or is archived; the archive manifest lives in archived/README.md. -->

# Status reports — live set

Point-in-time snapshots live here until every item they carry is resolved
inline (`done at` / `Won't implement` / `→ routed` markers), then they move
to [archived/](archived/README.md) with a manifest row. **Do not harvest from
archived/** — the living backlog is [TODO_LIST.md](../../TODO_LIST.md).

## Live reports

| Report                                                                        | State                                                                                                                                                                                                                                                           |
| ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `2026-10-24_17-20_dedup-pass2-brutal-self-review-full-status.md`          | live — pass-2 brutal self-review + full status: a–g inventory (4 extractions + zero-group gate verified; pin parity still unproven; daemon likely committed a broken intermediate); §f 26 ranked next items; §g three owner questions                                                     |
| `2026-10-24_13-30_dedup-pass2-zero-groups.md`                                 | live — pass 2: audited the SUPPRESSED bucket (the 12:33 gate was blind to it), extracted four production clone groups, recalibrated the gate to a ZERO-group baseline (`--min-lines 2`); §e resolves the 12:33 §g questions                                            |
| `2026-10-24_12-33_dedup-to-zero-baseline-gate.md`                             | live — dedup-to-zero run: 20-group t=5 audit + 23-group t=4 baseline gate wired into CI (replaces warning-only txt lane); §b holds the version-parity receipt (first CI run), §g three owner questions (gate-vs-warning, devShell pin, suppressed-bucket depth) |
| `2026-10-23_16-21_buildflow-44-of-72-triage-probe-dedup-gate-recovery.md`     | live — gate triage + probe-bootstrap dedup receipts; §f is a brainstorm (not commitments); §g holds three owner questions; its bounded items were harvested into TODO_LIST on 2026-10-23                                                                        |
| `2026-10-23_16-47_docs-health-audit-analytics-reframe-and-corpus-closeout.md` | live — this pass's own report: corpus close-out, analytics reframe, the six living docs re-verified; §f is the next-work inventory, §g the three open owner questions                                                                                           |
| `2026-10-23_17-10_pareto-planning-truth-first-analytics.md`                   | live — the Pareto planning pass: 27 tasks / 130 steps over all 42 rows; errata corrected two stale rows and reframed the prod re-ingest; §g repeats the three owner questions                                                                                   |

The 2026-10-23 docs-health pass annotated and archived the five 2026-10-18
reports (manifest in archived/README.md) and harvested their surviving queue
into TODO_LIST (stamp churn, buildflow filing decision, release decision,
poll-exec wrapper, probe residue, prettier devShell) and ROADMAP. The
2026-10-23 report above is the current single harvest source.

## Archive

59 closed reports under [archived/](archived/README.md) — annotated inline,
with per-file manifests for the 2026-10-12, 2026-10-17 (corpus), 2026-10-17
(live-set), and 2026-10-23 (this sweep) passes.
