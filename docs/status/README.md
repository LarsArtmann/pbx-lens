<!-- docs/status/README.md — the live-set index (05-01 §f44): what is live
     vs archived, at a glance. Update the live table whenever a report lands
     or is archived; the archive manifest lives in archived/README.md. -->

# Status reports — live set

Point-in-time snapshots live here until every item they carry is resolved
inline (`done at` / `Won't implement` / `→ routed` markers), then they move
to [archived/](archived/README.md) with a manifest row. **Do not harvest from
archived/** — the living backlog is [TODO_LIST.md](../../TODO_LIST.md).

## Live reports

| Report                                                                        | State                                                                |
| ----------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `2026-10-18_02-40_pareto-plan-execution-gate-hardening-probes-rehearsals.md`  | newest — pareto plan execution receipt; §e carries 3 owner questions |
| `2026-10-18_00-55_docs-health-pass-self-review-and-remote-ci-verdict.md`      | self-review + remote-CI verdict (§g questions still open)            |
| `2026-10-18_00-18_docs-health-pass-live-set-closeout-and-planning-archive.md` | same-pass companion — the close-out ledger itself                    |

(A single live report is the steady state after the 2026-10-17 second pass:
the previous sweep emptied the queue, so each new pass starts from one
report, not twenty-eight.)

## Archive

54 closed reports under [archived/](archived/README.md) — annotated inline,
per-file manifests for the 2026-10-12, 2026-10-17 (corpus), and 2026-10-17
(live-set) sweeps.
