<!-- docs/status/README.md — the live-set index (05-01 §f44): what is live
     vs archived, at a glance. Update the live table whenever a report lands
     or is archived; the archive manifest lives in archived/README.md. -->

# Status reports — live set

Point-in-time snapshots live here until every item they carry is resolved
inline (`done at` / `Won't implement` / `→ routed` markers), then they move
to [archived/](archived/README.md) with a manifest row. **Do not harvest from
archived/** — the living backlog is [TODO_LIST.md](../../TODO_LIST.md).

## Live reports

| Report                                                                    | State                                                                                                                                                                                    |
| ------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `2026-10-23_16-21_buildflow-44-of-72-triage-probe-dedup-gate-recovery.md` | live — gate triage + probe-bootstrap dedup receipts; §f is a brainstorm (not commitments); §g holds three owner questions; its bounded items were harvested into TODO_LIST on 2026-10-23 |

The 2026-10-23 docs-health pass annotated and archived the five 2026-10-18
reports (manifest in archived/README.md) and harvested their surviving queue
into TODO_LIST (stamp churn, buildflow filing decision, release decision,
poll-exec wrapper, probe residue, prettier devShell) and ROADMAP. The
2026-10-23 report above is the current single harvest source.

## Archive

59 closed reports under [archived/](archived/README.md) — annotated inline,
with per-file manifests for the 2026-10-12, 2026-10-17 (corpus), 2026-10-17
(live-set), and 2026-10-23 (this sweep) passes.
