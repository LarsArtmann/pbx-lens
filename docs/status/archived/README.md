# Archived status reports

Closed history: every open item in these reports is resolved INLINE (each file
carries `done` / `Won't implement` / `→ routed` strikethrough markers; the
living backlog is [TODO_LIST.md](../../../TODO_LIST.md)). Do not harvest from
here — terminal state. Archived 2026-10-12 by the docs-health AUDIT pass.

## Archive manifest (bulk sweep 2026-10-12)

| File                                                                   | Classification      | Deciding reason                                                                                        |
| ---------------------------------------------------------------------- | ------------------- | ------------------------------------------------------------------------------------------------------ |
| 2026-10-10_23-56_docs-health-audit-first-doc-set.md                    | ANNOTATE → ARCHIVED | First doc set built + repaired; every §f/§c/§g item resolved inline (nix fix, CI, C009 → #64)          |
| 2026-10-11_01-05_session-qa-templ-components-cli-surface.md            | ANNOTATE → ARCHIVED | Q&A session; the UI it proposed shipped the same day (UI trio); §f 1-40 resolved inline                |
| 2026-10-11_03-55_SUPERB-execution-domain-redesign-and-gate-truth.md    | ANNOTATE → ARCHIVED | Domain redesign landed; its §f 35 items all shipped or routed (owner threads tracked)                  |
| 2026-10-11_04-58_go-output-pattern-adoption-status.md                  | ANNOTATE → ARCHIVED | Infra adoption landed (nix fix, CI lane, devShell); §b/c items resolved or fleet-routed                |
| 2026-10-11_04-59_SUPERB-session-status-what-did-you-forget.md          | ANNOTATE → ARCHIVED | Superseded by 05-21/06-29; Next-40 list fully executed (lint zero, cli package, B1/B4/B5)              |
| 2026-10-11_05-21_SUPERB-session-status-where-are-we.md                 | ANNOTATE → ARCHIVED | Superseded by 06-29 (B3 completed); its next-5 tasks all landed                                        |
| 2026-10-11_05-41_dedup-session-status.md                               | ANNOTATE → ARCHIVED | Dedup pass landed; gate root cause resolved (toolchain unified); §f 1-50 resolved inline               |
| 2026-10-11_06-29_SUPERB-comprehensive-status-full-truth.md             | ANNOTATE → ARCHIVED | B3/warts/MOS-data landed; §f items shipped, filed (#62/#63/#7/#545), or routed                         |
| 2026-10-11_09-35_session-frontend-completion-dashboard-systemd-docs.md | ANNOTATE → ARCHIVED | Frontend plan complete (dashboard/systemd/docs); §c items all shipped (T13/T15/D2/C3/D6)               |
| 2026-10-11_09-39_backend-hardening-c2-c4-d5-mos.md                     | ANNOTATE → ARCHIVED | C2/C4/D5/MOS landed; §f 50 items executed across the 11:31-20:26 dispatches                            |
| 2026-10-11_10-28_github-launch-public-repo-ci-green-templ-bump.md      | ANNOTATE → ARCHIVED | Launch session (public repo, CI green); vendorHash repaired; hardening items routed to ROADMAP         |
| 2026-10-11_11-31_SUPERB-session-status-telnyx-enrichment-todo-sweep.md | ANNOTATE → ARCHIVED | Enrichment (T13/T15) landed; §c/§f resolved (age gate = P2; filings posted)                            |
| 2026-10-11_11-48_todo-list-completion-d2-c3-d6.md                      | ANNOTATE → ARCHIVED | D3/D2/C3/D6 completion; open items routed inline (age gate landed, filings tracked)                    |
| 2026-10-11_18-21_full-status-and-self-review-todo-complete.md          | ANNOTATE → ARCHIVED | TODO list complete (12/12); full-gate residue closed by the 2026-10-12 green gates                     |
| 2026-10-11_18-50_naming-review-session-status.md                       | ANNOTATE → ARCHIVED | 9 renames landed; the mosAvgLabel remnant closed in the 2026-10-12 pass; §g answered                   |
| 2026-10-11_18-57_four-lens-review-and-brutal-self-check.md             | ANNOTATE → ARCHIVED | Four-lens fixes all shipped (P2-P11 execution); D6 filings posted; timeout knob landed                 |
| 2026-10-11_19-37_tier1-2-pareto-execution-status.md                    | ANNOTATE → ARCHIVED | P1-P11 executed; §c P9/P10/P15/P16/P17 resolved (filings, rebuild, knob, interface)                    |
| 2026-10-11_20-26_dispatch3-reuse-types-and-gate-status.md              | ANNOTATE → ARCHIVED | Reuse-first round landed (AgeGate, typed models, throttle, e2e); gate closed 23:30                     |
| 2026-10-11_23-35_gate-closure-lint-residuals-test-dedup.md             | ANNOTATE → ARCHIVED | Superseded (inline note) by 00-08; lint residuals + helper extraction landed; split done 02:34         |
| 2026-10-11_23-41_gate-closure-verification-status.md                   | ANNOTATE → ARCHIVED | Gate closed green (73/0/0); templ-pin + harvest §b items closed by 00-08 + this pass                   |
| 2026-10-12_00-08_session-status-gate-closure-and-lint-dedup.md         | ANNOTATE → ARCHIVED | Step-level closure superseded by the 01:16 full-mode green; §f misclassification filed as BuildFlow#45 |

Kept ACTIVE (not archived — they hold open work):

- `2026-10-12_01-16_full-mode-gate-green-and-lint-debt-surfaced.md` — resolved
  items annotated inline; remaining §f items live in TODO_LIST/ROADMAP.
- `2026-10-12_02-34_third-pass-todo-closure-upstream-filings-and-file-splits.md`
  — the newest snapshot; §b/§c residuals closed or routed by this pass, the
  rest tracked in TODO_LIST.md.
