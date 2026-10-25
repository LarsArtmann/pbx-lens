<!-- docs/planning/archived/README.md — manifest for the executed-plan archive
     (created by the 2026-10-17 second docs-health pass, which resolved the
     question the 05-01 corpus sweep staged as its §g2: the dispatch directive
     "archive FULLY done and UPDATED files" decided it). -->

# Archived execution plans

10 closed plans. Fully executed and annotated; each carries an EXECUTED header pointing
at its archived execution reports and the CHANGELOG receipts — the plan text
itself is a historical snapshot, never rewritten. Open work lives in
[TODO_LIST.md](../../../TODO_LIST.md). The three plans still in
`docs/planning/` hold owner-gated or window-gated residue (02-24: D1-D6
answers; 05-40: owner-gated M-items; mos-poller-design: the 2027-01 window).

| Plan (old path: `docs/planning/<file>`)                                 | Classification | Deciding reason                                                                                                                                                                  |
| ----------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-11_00-09_SUPERB-pareto-remaining-work.md                        | ARCHIVED       | EXECUTED header 2026-10-17 (planning-doc closure); execution record 19-37 + CHANGELOG R-series                                                                                   |
| 2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md               | ARCHIVED       | EXECUTED header 2026-10-17; the frontend era shipped 10-11→12 (09-35 record + UI-trio CHANGELOG)                                                                                 |
| 2026-10-11_19-05_SUPERB-pareto-todo-plan.md                             | ARCHIVED       | EXECUTED header 2026-10-17; tiers executed 10-11→12 (19-37 record)                                                                                                               |
| 2026-10-12_did-direction-design.md (was `docs/did-direction-design.md`) | CLOSED         | design note: BOTH halves shipped (`-ai-did` resolver 2026-10-12; CDR direction derivation 2026-10-25); the "CDR side already true" claim was disproven live and annotated inline |
| 2026-10-12_03-43_SUPERB-pareto-open-work-plan.md                        | ARCHIVED       | EXECUTED header 2026-10-12 (docs-health ANNOTATE); superseded plans all archived                                                                                                 |
| 2026-10-12_12-47_SUPERB-httputil-go-etag-go-appkit-max-adoption.md      | ARCHIVED       | Status line EXECUTED (shipped as fd0e082 + follow-ups); the NOT-adopted doctrine lives in AGENTS.md                                                                              |
| 2026-10-12_13-32_SUPERB-templ-ui-pareto-plan.md                         | ARCHIVED       | EXECUTED table 2026-10-12 16:12 (all 27 tasks); execution record 15-20                                                                                                           |
| 2026-10-12_17-31_SUPERB-verification-closure-and-ops-hardening.md       | ARCHIVED       | EXECUTED 2026-10-12 ~20:30 (T1-T17 done, T18 routed to TODO rows); execution record 20-30                                                                                        |
| 2026-10-17_05-07_SUPERB-pareto-plan-post-corpus-archive.md              | ARCHIVED       | EXECUTED header 2026-10-17 (08:02 + 10:28 legs; only the load-gated receipt remains, as a TODO row)                                                                              |
| 2026-10-18_01-01_SUPERB-pareto-plan-post-liveset-closeout.md            | ARCHIVED       | EXECUTED 2026-10-18 (T2-T23); per-row verdicts + §6 ledger (2026-10-23 docs-health pass)                                                                                         |

Historical citations: reports archived before this move cite the old
`docs/planning/...` paths — `git log --follow <file>` resolves them; living
docs were updated to the archived paths by the same pass.
