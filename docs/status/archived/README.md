# Archived status reports

Closed history: every open item in these reports is resolved INLINE (each file
carries `done` / `Won't implement` / `→ routed` strikethrough markers; the
living backlog is [TODO_LIST.md](../../../TODO_LIST.md)). Do not harvest from
here — terminal state. Archived 2026-10-12 by the docs-health AUDIT pass,
2026-10-17 by the corpus-closing sweep, 2026-10-17 (second pass) by the
live-set close-out, 2026-10-23 by the post-receipt sweep, and 2026-10-26 by
the v0.3.0-state sweep (manifests below).

## Archive manifest (bulk sweep 2026-10-26 — the v0.3.0-state close-out)

The whole 2026-10-23/24 live set annotated inline against the shipped v0.3.0
state (the analytics surface, the go-cqrs-lite wave, the dedup-to-zero gate
lineage, the CI pin enforcement) and the 2026-10-25/26 receipts; check-rows
ran COMPLETE on every file BEFORE the move. Surviving open work was routed to
its live TODO_LIST row (filings queue, detect-only advisory triage, answer
sheet, quarterly windows) — nothing was dropped.

| File                                                                        | Classification      | Deciding reason                                                                                                                                           |
| --------------------------------------------------------------------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-23_16-21_buildflow-44-of-72-triage-probe-dedup-gate-recovery.md     | ANNOTATE → ARCHIVED | all 15 receipts verified standing; the §f 50-item brainstorm fully dispositioned (done/routed/NOT-DO); §g routed (filings queue, ROADMAP vulnix question) |
| 2026-10-23_16-47_docs-health-audit-analytics-reframe-and-corpus-closeout.md | ANNOTATE → ARCHIVED | the pass's own corpus ledger; §f 50 all done (the three analytics slices shipped 10-25) or routed; its f50 cadence item executed BY this sweep            |
| 2026-10-23_17-10_pareto-planning-truth-first-analytics.md                   | ANNOTATE → ARCHIVED | the planning snapshot superseded by execution; T1-T27 verdicts land in the 10-25 reports; §g3 moot (all three slices shipped)                             |
| 2026-10-24_11-37_dedup-audit-self-review-serve-test-bootstrap.md            | ANNOTATE → ARCHIVED | the dedup lineage closed by the zero-group gate + pin enforcement; §f 26 dispositioned; §g answered (gate policy, quiet-box, teardown order)              |
| 2026-10-24_12-33_dedup-to-zero-baseline-gate.md                             | ANNOTATE → ARCHIVED | parity proven (CI green from run one); suppressed bucket audited same day (13-30); the policy change ratified by outcome                                  |
| 2026-10-24_13-30_dedup-pass2-zero-groups.md                                 | ANNOTATE → ARCHIVED | the recalibration receipts verified standing; the three 12:33 questions answered in §e                                                                    |
| 2026-10-24_17-20_dedup-pass2-brutal-self-review-full-status.md              | ANNOTATE → ARCHIVED | §f 26 done/routed/NOT-DO against the 315587e pin doctrine and the 2026-10-25/26 receipts; §g3 answered by the ratified concurrency norm                   |
| 2026-10-24_18-30_gocqrs-lite-module-audit-session.md                        | ANNOTATE → ARCHIVED | the §f 50-item plan executed by the 2026-10-25 wave (20/21 + the P-tail); §g answered in the header (owner, same day)                                     |
| 2026-10-24_18-59_dedup-to-zero-third-pass.md                                | ANNOTATE → ARCHIVED | superseded verdicts corrected inline (the utils/format postscript); §f 50 standing-contract items verified or routed                                      |
| 2026-10-24_19-30_templ-components-deep-dive-audit.md                        | ANNOTATE → ARCHIVED | actions ①–④ executed 2026-10-25 (utils/format, v1.24.0, the "never" guard, the map repair); §g answered by execution                                      |

## Archive manifest (sweep 2026-10-23 — post-receipt live set)

The five 2026-10-18 reports were annotated inline against the day's receipts
(gate 57/0/0, cut-release rehearsal, lint-parity doctrine) and against the
follow-on work through 2026-10-23 (catalog export, read-model key repair,
upstream re-verify); every row carries its own verdict. check-rows ran
COMPLETE on every file BEFORE the move. The surviving queue was harvested
into TODO_LIST (stamp churn, buildflow filing decision, release decision,
poll-exec wrapper, probe residue, prettier devShell) and ROADMAP (flake-record
convention, test-complexity policy, client-side stamp rendering).

| File                                                                        | Classification      | Deciding reason                                                                                                                             |
| --------------------------------------------------------------------------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-18_00-18_docs-health-pass-live-set-closeout-and-planning-archive.md | ANNOTATE → ARCHIVED | its own pass ledger; every §a/§b/§c/§d/§e/§f row dispositioned (receipt banked 06:17; dprint policy routed to ROADMAP)                      |
| 2026-10-18_00-55_docs-health-pass-self-review-and-remote-ci-verdict.md      | ANNOTATE → ARCHIVED | self-review + remote-CI verdict; all §a executed, §f 1-33 landed or routed (rows), §g 1/3 → TODO rows, §g 2 → ROADMAP                       |
| 2026-10-18_02-40_pareto-plan-execution-gate-hardening-probes-rehearsals.md  | ANNOTATE → ARCHIVED | execution snapshot superseded by the 06-55 receipts; every shipped bullet carries done, every gate an arrow to its TODO row                 |
| 2026-10-18_04-25_pareto-execution-flake-fix-and-gate-hardening.md           | ANNOTATE → ARCHIVED | the 50-item queue triaged row by row (executed, routed, or NOT-DO with a reason); its §e4 audit shipped as the gate-doc doctrine at b4a9970 |
| 2026-10-18_06-55_t2-receipt-t13-rehearsal-lint-parity-doctrine.md           | ANNOTATE → ARCHIVED | newest report; every §f row routed to a live TODO row, §b watches routed, §c owner gates unchanged                                          |

The same pass archived the executed plan
[docs/planning/archived/2026-10-18_01-01_SUPERB-pareto-plan-post-liveset-closeout.md](../../planning/archived/2026-10-18_01-01_SUPERB-pareto-plan-post-liveset-closeout.md)
— its T2-T23 rows shipped 2026-10-18 and every other row carries a per-row
verdict (routed) or the §6 landing-place ledger.

## Archive manifest (bulk sweep 2026-10-17, second docs-health pass — the live-set close-out)

The four 2026-10-17 live reports were annotated inline against the day's
executions (the 08:02→10:28 Pareto legs plus this pass) and archived; the
live set is now empty, and the pass report of THIS sweep is the single
harvest source going forward.

| File                                                                        | Classification      | Deciding reason                                                                                                                      |
| --------------------------------------------------------------------------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-10-17_01-45_statix-silent-noop-repair-and-gate-revival.md              | ANNOTATE → ARCHIVED | §b/§c/§f/§g fully dispositioned (postscript closures cited; CI-nix-lane/tripwire/config-verify routed to the new gate-hardening row) |
| 2026-10-17_05-01_docs-health-corpus-sweep-archive-and-living-doc-rebuild.md | ANNOTATE → ARCHIVED | every §b/§c/§f/§g item done (P1 legs) or routed; §g2 answered BY THIS PASS (planning archive); §g3 → answer sheet §9                 |
| 2026-10-17_08-02_pareto-plan-execution-p0p2-and-red-domain-test.md          | ANNOTATE → ARCHIVED | §b/§c/§f/§g closed by the 10:28 close-out receipts; the three §f specials (filings/csp-sweep/cut-release) routed to new TODO rows    |
| 2026-10-17_10-28_pareto-closeout-red-test-fix-and-tail-tasks-status.md      | ANNOTATE → ARCHIVED | §f 1-18 ride TODO rows, 19-50 triaged by this pass into TODO rows + ROADMAP; the three §g questions staged on the answer sheet       |

## Archive manifest (bulk sweep 2026-10-17 — the corpus close-out)

The whole 2026-10-11→15 live set annotated inline (verified against CHANGELOG,
code, and the fold-in ledgers) and archived. (Historical note: at sweep time
`2026-10-12_17-17` was folded in and `2026-10-17_01-45` stayed live as the
newest harvest source — both since archived by the manifests above.)

| File                                                                           | Classification      | Deciding reason                                                                                                               |
| ------------------------------------------------------------------------------ | ------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-12_01-16_full-mode-gate-green-and-lint-debt-surfaced.md                | ANNOTATE → ARCHIVED | §b/§c/§f/§g fully dispositioned (gate tools triage routed to the new TODO row; statix/DID/FTS done; jsonv2 = documented hold) |
| 2026-10-12_02-34_third-pass-todo-closure-upstream-filings-and-file-splits.md   | ANNOTATE → ARCHIVED | BuildFlow#45 fixed upstream; sibling flags verified exposed; residue routed (timeout row, fold-ins, schema-v2)                |
| 2026-10-12_03-39_docs-health-audit-annotate-archive-and-living-doc-rebuild.md  | ANNOTATE → ARCHIVED | every §f item done/routed (R2-R24 shipped or rowed); the §g questions resolved (R2, daemon norm, releases tagged)             |
| 2026-10-12_07-13_SUPERB-plan-execution-status.md                               | ANNOTATE → ARCHIVED | R1-R27 all done/routed (v0.1.0 tagged, DID shipped, sibling green); R27.x residue = the advisory row                          |
| 2026-10-12_07-56_gate-watch-lint-audit-status.md                               | ANNOTATE → ARCHIVED | continuation of 07-13; lint zero, gate green, release shipped; the mirror list resolved identically                           |
| 2026-10-12_10-33_lint-zero-gate-starvation-status.md                           | ANNOTATE → ARCHIVED | lint zero held; htmx culprit solved (oxfmt provider + exclude); gate policy recorded; §f fully dispositioned                  |
| 2026-10-12_12-23_release-did-sibling-gate-green-status.md                      | ANNOTATE → ARCHIVED | v0.1.0 released; oxfmt filing NOT-DO'd (repo-side exclude); host deploys = the deploy row                                     |
| 2026-10-12_13-27_templ-components-max-adoption-ui-pass-status.md               | ANNOTATE → ARCHIVED | the UI follow-ups shipped (heatmap, presets, print, a11y, goldens, probes); library leverage items routed to the bump watch   |
| 2026-10-12_13-28_superb-http-stack-adoption-status.md                          | ANNOTATE → ARCHIVED | appendix closures verified; §f 50 rows all done/routed (R26, research report, no-cache, Vary, Content-Length)                 |
| 2026-10-12_13-49_superb-adoption-final-wrap.md                                 | ANNOTATE → ARCHIVED | the 13-28 supersede completed: every ⬜ item resolved (v0.2.0 shipped the release-bound ones)                                 |
| 2026-10-12_14-16_css-mime-favicon-fix-session-status.md                        | ANNOTATE → ARCHIVED | styles.css committed policy closed the CI blind spot; T16 remainder landed; templ skew resolved by doctrine                   |
| 2026-10-12_15-07_followups-closure-and-self-review.md                          | ANNOTATE → ARCHIVED | 13-49 closure verified complete; gate green; every §f row landed/routed/declined                                              |
| 2026-10-12_15-10_telnyx-mp3-recording-fix-session-status.md                    | ANNOTATE → ARCHIVED | MP3 fallback shipped + documented; open tails routed to the 2027-01 window; lookback audit = the new TODO row                 |
| 2026-10-12_17-17_templ-ui-pareto-closeout-t20-t27-status.md                    | ANNOTATE → ARCHIVED | the five bare residuals dispositioned (stale-pass/cache-hardening/styles-trim/scratch-dir/quarterly); owner rows routed       |
| 2026-10-12_19-20_buildflow-gate-green-templ-fmt-and-hash-collision-recovery.md | ANNOTATE → ARCHIVED | statix narrowed (2026-10-17); hash collision = BuildFlow#55; every §f item done/routed/NOT-DO'd                               |
| 2026-10-12_19-20_cache-observability-drill-lane-gate-triage-status.md          | ANNOTATE → ARCHIVED | drill lane + cache section shipped; v0.2.0 cut; every §f row dispositioned                                                    |
| 2026-10-12_20-30_verification-closure-executed-t1-t17-status.md                | ANNOTATE → ARCHIVED | T1-T17 verified done; T18 roster = the TODO rows; §f 13-50 executed, routed, or declined with reasons                         |
| 2026-10-15_02-00_ui-design-audit-polish.md                                     | ANNOTATE → ARCHIVED | every actionable fix landed; taste calls = the decision pack (D1-D5); /ops + probe corpus closed the surfaced gaps            |
| 2026-10-15_04-45_vendorhash-and-flake-hash-repair-status.md                    | ANNOTATE → ARCHIVED | hashes repaired + verified; the guard class = BuildFlow#55 + the clean-tree rule; nine-tools = the triage row                 |
| 2026-10-15_05-18_superb-ui-contract-execution.md                               | ANNOTATE → ARCHIVED | M1-M11 executed; the queue landed (pixel-diff, /ops, retention, probes) or routed (owner pack, MOS, PII, mobile)              |
| 2026-10-15_06-19_superb-0540-execution-ops-retention-cache-armor.md            | ANNOTATE → ARCHIVED | M9/M10 landed, M11 completed by the thirteenth pass; purge telemetry declined-by-default (read-only doctrine)                 |
| 2026-10-15_16-30_superb-owner-unlock-execution.md                              | ANNOTATE → ARCHIVED | v0.2.0 receipt; BuildFlow#55/#56 filed; the owner-gated §g = the TODO rows                                                    |
| 2026-10-15_16-32_superb-0540-m11-m19-execution-multi-session.md                | ANNOTATE → ARCHIVED | every §f row carries its 17-40-ledger disposition inline; trailing-slash + baseline strategy = ROADMAP questions              |
| 2026-10-15_16-32_superb-execution-self-review.md                               | ANNOTATE → ARCHIVED | §b/§c/§f/§g fully dispositioned (f34's owner-read gate superseded by the 2026-10-17 sweep directive)                          |
| 2026-10-15_17-40_harvest-pass.md                                               | ANNOTATE → ARCHIVED | the pass artifact is complete; its three deliberate follow-ups closed by this sweep                                           |
| 2026-10-15_18-06_buildflow-full-gate-triage.md                                 | ANNOTATE → ARCHIVED | triage superseded by the 2026-10-17 green dev gate (58/0); §f done/routed (BuildFlow#56, tooling triage row)                  |
| 2026-10-15_18-45_css-gate-recurrence-fix-and-buildflow55-restore.md            | ANNOTATE → ARCHIVED | update-styles guard landed; the AGENTS ~192 fix DONE (2026-10-17); §f done/routed                                             |

## Archive manifest (bulk sweep 2026-10-12)

| File                                                                   | Classification      | Deciding reason                                                                                                       |
| ---------------------------------------------------------------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------- |
| 2026-10-10_23-56_docs-health-audit-first-doc-set.md                    | ANNOTATE → ARCHIVED | First doc set built + repaired; every §f/§c/§g item resolved inline (nix fix, CI, C009 → #64)                         |
| 2026-10-11_01-05_session-qa-templ-components-cli-surface.md            | ANNOTATE → ARCHIVED | Q&A session; the UI it proposed shipped the same day (UI trio); §f 1-40 resolved inline                               |
| 2026-10-11_03-55_SUPERB-execution-domain-redesign-and-gate-truth.md    | ANNOTATE → ARCHIVED | Domain redesign landed; its §f 35 items all shipped or routed (owner threads tracked)                                 |
| 2026-10-11_04-58_go-output-pattern-adoption-status.md                  | ANNOTATE → ARCHIVED | Infra adoption landed (nix fix, CI lane, devShell); §b/c items resolved or fleet-routed                               |
| 2026-10-11_04-59_SUPERB-session-status-what-did-you-forget.md          | ANNOTATE → ARCHIVED | Superseded by 05-21/06-29; Next-40 list fully executed (lint zero, cli package, B1/B4/B5)                             |
| 2026-10-11_05-21_SUPERB-session-status-where-are-we.md                 | ANNOTATE → ARCHIVED | Superseded by 06-29 (B3 completed); its next-5 tasks all landed                                                       |
| 2026-10-11_05-41_dedup-session-status.md                               | ANNOTATE → ARCHIVED | Dedup pass landed; gate root cause resolved (toolchain unified); §f 1-50 resolved inline                              |
| 2026-10-11_06-29_SUPERB-comprehensive-status-full-truth.md             | ANNOTATE → ARCHIVED | B3/warts/MOS-data landed; §f items shipped, filed (#62/#63/#7/#545), or routed                                        |
| 2026-10-11_09-35_session-frontend-completion-dashboard-systemd-docs.md | ANNOTATE → ARCHIVED | Frontend plan complete (dashboard/systemd/docs); §c items all shipped (T13/T15/D2/C3/D6)                              |
| 2026-10-11_09-39_backend-hardening-c2-c4-d5-mos.md                     | ANNOTATE → ARCHIVED | C2/C4/D5/MOS landed; §f 50 items executed across the 11:31-20:26 dispatches                                           |
| 2026-10-11_10-28_github-launch-public-repo-ci-green-templ-bump.md      | ANNOTATE → ARCHIVED | Launch session (public repo, CI green); vendorHash repaired; hardening items routed to ROADMAP                        |
| 2026-10-11_11-31_SUPERB-session-status-telnyx-enrichment-todo-sweep.md | ANNOTATE → ARCHIVED | Enrichment (T13/T15) landed; §c/§f resolved (age gate = P2; filings posted)                                           |
| 2026-10-11_11-48_todo-list-completion-d2-c3-d6.md                      | ANNOTATE → ARCHIVED | D3/D2/C3/D6 completion; open items routed inline (age gate landed, filings tracked)                                   |
| 2026-10-11_18-21_full-status-and-self-review-todo-complete.md          | ANNOTATE → ARCHIVED | TODO list complete (12/12); full-gate residue closed by the 2026-10-12 green gates                                    |
| 2026-10-11_18-50_naming-review-session-status.md                       | ANNOTATE → ARCHIVED | 9 renames landed; the mosAvgLabel remnant closed in the 2026-10-12 pass; §g answered                                  |
| 2026-10-11_18-57_four-lens-review-and-brutal-self-check.md             | ANNOTATE → ARCHIVED | Four-lens fixes all shipped (P2-P11 execution); D6 filings posted; timeout knob landed                                |
| 2026-10-11_19-37_tier1-2-pareto-execution-status.md                    | ANNOTATE → ARCHIVED | P1-P11 executed; §c P9/P10/P15/P16/P17 resolved (filings, rebuild, knob, interface)                                   |
| 2026-10-11_20-26_dispatch3-reuse-types-and-gate-status.md              | ANNOTATE → ARCHIVED | Reuse-first round landed (AgeGate, typed models, throttle, e2e); gate closed 23:30                                    |
| 2026-10-11_23-35_gate-closure-lint-residuals-test-dedup.md             | ANNOTATE → ARCHIVED | Superseded (inline note) by 00-08; lint residuals + helper extraction landed; split done 02:34                        |
| 2026-10-11_23-41_gate-closure-verification-status.md                   | ANNOTATE → ARCHIVED | Gate closed green (73/0/0); templ-pin + harvest §b items closed by 00-08 + this pass                                  |
| 2026-10-12_00-08_session-status-gate-closure-and-lint-dedup.md         | ANNOTATE → ARCHIVED | Step-level closure superseded by the 01:16 full-mode green; §f misclassification filed as BuildFlow#45                |
| 2026-10-12_14-30_superb-followups-harvest-wrap.md                      | SKIP → ARCHIVED     | Self-annotating wrap: §b gate outcome annotated inline same day, §c resolved, §d ledger carries per-item dispositions |
| 2026-10-12_15-20_templ-ui-pareto-execution-t1-t22-status.md            | ANNOTATE → ARCHIVED | Session two executed every §f/§b/§c item (13-32 EXECUTED); §g answered by owner defaults + #51 (22 inline markers)    |

## Triage of the 2026-10-12 live set (17-31 plan T16.3, 2026-10-12 evening)

Annotated this pass: **15-20 + 17-17** (41 + 22 inline markers — the templ-UI
lineage whose follow-ups the 17-31 plan executed). Archived: **15-20, 14-30**.

Keep-open triage for the rest (their forward items live in TODO_LIST /
ROADMAP / owner-gated rows; annotate+archive each in its own follow-up pass):

- `01-16`, `02-34`, `03-39` — already partially annotated by the earlier
  2026-10-12 AUDIT sweep; residue = owner-gated rows only.
- `07-13`, `07-56`, `10-33` — the gate-starvation line; resolved by the
  evening green gates; annotate when the next full-gate pass touches them.
- `12-23` — release + sibling session; candidate for the next archive pass.
- `13-27`, `13-28`, `13-49` — superseded in substance by the 12-47/13-32
  EXECUTED annotations + docs/research report; bulk-archive candidates.
- `14-16`, `15-07`, `15-10` — forward items routed to TODO_LIST; low-value
  to annotate until the owner-gated deploy row opens.
- `17-17` — annotated this pass; stays LIVE: §f 1-11 owner-gated rows +
  f17 (stale-poller warn) + f19 (cache e2e hardening) + f40 (styles trim) +
  f41 (scratch-dir convention) remain open.
- `19-20` (both) — the concurrent session's fresh reports; theirs to close.

Kept ACTIVE (not archived — they hold open work):

- `2026-10-12_01-16_full-mode-gate-green-and-lint-debt-surfaced.md` — resolved
  items annotated inline; remaining §f items live in TODO_LIST/ROADMAP.
- `2026-10-12_02-34_third-pass-todo-closure-upstream-filings-and-file-splits.md`
  — the newest snapshot; §b/§c residuals closed or routed by this pass, the
  rest tracked in TODO_LIST.md.
