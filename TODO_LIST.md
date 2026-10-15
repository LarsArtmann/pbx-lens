<!-- TODO_LIST.md — short-term, actionable, bounded work; open items only.
     Living document: delete items when done (they move to CHANGELOG.md).
     Never convert to HTML; tools upsert and grep it. See docs-health skill. -->

# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Long-term vision and unrefined ideas live in ROADMAP.md; shipped features in
> FEATURES.md. Items are ranked by impact. Status is verified, not assumed.
>
> Refreshed 2026-10-12 (fourth pass, docs-health AUDIT): the recording-proxy upstream timeout
> knob, the file-size split of the three >350-line files (calls domain, CLI
> main, telnyx client), the flake app `meta.description`, and the upstream
> loop (five D6 filings re-verified OPEN; the C009 bare-`must` gap confirmed
> covered by go-cqrs-lite#64; the BuildFlow compile-gate misattribution filed
> as BuildFlow#45) are DONE and recorded in FEATURES.md/CHANGELOG.md.
> Fourth pass added the owner-gated rows below (prod re-ingest, first release
> tag, sibling module values) harvested from the 2026-10-11/12 status
> reports before their archive, plus two bounded Low rows. What remains is
> blocked or small bounded work only (below).
>
> Fifth pass 2026-10-12 (SUPERB plan execution): the owner decisions arrived
> (timezone Berlin, M20.1/M20.2 deliver, polling, release go, sibling
> -max-age forever, raw-at-rest doctrine, R24 filing permission), so the
> release and sibling rows unblocked; the DID-direction design note
> (docs/did-direction-design.md) spawned the bounded implementation row
> below. The timeout-test hardening row shrank to the injected-deadline
> refactor only (budgets are load-safe already).
>
> Sixth pass 2026-10-12 (release + post-release, harvest of the 10:33
> report §f): v0.1.0 tagged on a CI-green commit and published; the local
> gate closed green (75/75) and the evidence-based load policy is recorded
> in docs/buildflow-dev-gate.md; DID-direction shipped (mapper
> DirectionResolver + `-ai-did` on ingest-telnyx/serve + fixture tests);
> the sibling module gained `services.telephony.pbx-lens.aiDids` (flake
> check green, pushed). The old "expose -max-age/-recording-timeout"
> sibling row resolved against code: both options existed and were wired
> already — only the host deploy remains (blocked on the deploy path).
>
> Seventh pass 2026-10-12 (harvest of the 13-49 SUPERB wrap §f, 50 items):
> go-etag cache-hook wiring, e2e cache-semantics hardening, README ops
> notes, the templ-flush gotchas entry, coverage + hash-cost rows, and
> SECURITY.md landed below; browser-level 304 check, upstream fold-in
> (go-etag#2/go-appkit#26), v0.2.0 cut, >1MiB transcript policy, and the
> dependabot confirmation landed as blocked rows; the deferred-trigger
> ideas and the multi-agent-protocol question routed to ROADMAP; two
> cross-project lessons routed to crush-config. The 13-49 §g questions:
> Q1 answered by events (concurrent session landed; disjoint-file work
> proved fine), Q2 = the blocked browser row + `Cache-Control: no-cache`
> shipped, Q3 = ROADMAP open question.
>
> Eighth pass 2026-10-12 (templ-UI Pareto plan close-out, plan
> docs/planning/2026-10-12_13-32): the whole 27-task plan is EXECUTED —
> T1-T19+T22 (UI adoption, theming, a11y, goldens) landed in the earlier
> session, T20/T21 (copy buttons, search-jump highlight) in this one,
> and T23-T27 produced decisions/docs instead of code. Four TODO rows
> closed as done (staleness filter carry, dashboard presets, drafts
> README, nolint ledger); three upstream filings (templ-components#48/
> #49/#50) joined the fold-in watch as blocked rows; the adoption map
> gained the bump watchlist. Plan annotation + evidence pointers live
> in the plan file; the three 13-32 §g questions stay with the owner.

> Ninth pass 2026-10-12 (cache observability + drill lane, same day): the
> backup restore drill became a named CI lane; `/api/ops` gained the
> go-etag cache section (hit ratio + overflow warn); the README ops
> surfaces, SECURITY.md `/metrics` posture (its stale "no tagged
> releases" claim fixed on sight), and the templ-flush gotcha landed; and
> the golden suite's wall-clock bombs (staleness banner presence,
> dashboard bucket dates) were scrubbed, goldens regenerated.

> Eleventh pass 2026-10-15 (UI-contract hardening plan, M1-M9 harvest of
> the 02-24 SUPERB plan): contract probes shipped (M1 style contracts +
> CSS gate, M3 unseen-states captures + staleness-live watch, M4 probe
> tooling: free-port default, codegen bootstrap, golden-pretty-diff), the
> keyboard/print a11y bundle landed (M5), and the empty-index chart
> honesty ladder fixed the collapsed-0px-bars eyeball finding (M3). The
> owner decision pack (docs/decisions/2026-10-15_ui-owner-decision-pack.md,
> D1-D5) gates the date-render change and two filings; the big tail rows
> below (MOS poller, PII enforcement, mobile cards) come from the same
> plan and stay blocked/low until their window opens.
>
> Fifteenth pass 2026-10-15 (HARVEST of the two 16-32 status reports, ~100
> candidate items): v0.2.0 verified cut AND pushed (7d15c6f, tag on origin) —
> the release question closed by events; the recording-throttle 429 test
> verified present (`TestRecordingHandlerAnswers429WhenThrottled`); the stale
> flake.lock/AGENTS tree question resolved (a3ebda1, fdea674); the
> perf_test lint residue gone (CLI gate 0 issues). The trailing-slash
> canonical-form call and the pixel-diff baseline strategy routed to ROADMAP
> open questions; CSP report-uri, corpus webp, i18n, and the templ-mtime
> upstream evaluation routed to ROADMAP raw ideas. Three small tests executed
> inline this pass (web TestMain TZ pin, /ops 304 assert, version-footer
> agreement) — they live in CHANGELOG, not here. Everything else bounded
> landed as the Low rows below; declines cite reasons in the pass ledger
> (docs/status/2026-10-15_17-40_harvest-pass.md).
>
> Fourteenth pass 2026-10-15 (05-40 release + hardening road): v0.2.0 CUT
> (CHANGELOG sweep, version const, tag; receipt: build/test/lint/nix green
> individually — the buildflow gate's go/govalid steps trail the new
> 1.27.2 toolchain floor, BuildFlow#56); BuildFlow#55 filed (nix-hash-fix
> wrong-attribute write, the a61ba8c incident) + #56 (stale gate-tool
> go); M15 security bundle landed (robots.txt, noindex everywhere,
> CSP_REPORT_ONLY dry-run mode, recording-stream rate limit, loopback
> exempt); M16 rails (10k benchmark 36 ms/op, 500-turn render, malformed
> JSONL property test, ETag hash cost 22/104 µs); docs batch (MOS poller
> design, PII one-pager, process prep + follow-on sketches); probe fixes
> (staleness DB reset, phase-server port race, inputHash static/ path).
> Owner-gated rows below stay: D1-D5 answers, PII boundary (one-pager
> ready), mobile card taste, MOS window, deploy paths.

> Thirteenth pass 2026-10-15 (05-40 M9-M11 closure, verified against code):
> rows DELETED as done — the `/api/ops` stale-pass warning (verdicts ride
> `SnapshotAny`), the `/ops` human page (cards + goldens), retention
> surfacing (live would-retire card), the coverage pass (caching.go,
> buildServiceConfig, nonceCSP all 100%), and the hash-cost probe
> (BenchmarkConcurrentPoll: 304 revalidation ≈82 µs/op parallel vs ≈101 µs
> full-200 on the dev box — thousands of viewers per core at the 30 s poll
> cadence). The cache-hardening row SHRANK to the recording timing-hop
> asserts only; the trailing-slash policy is deliberately unpinned (05-40
> M11) until the canonical-redirect-vs-exact-404 call settles.

> Twelfth pass 2026-10-15 (sweep of the unblocked low rows): four rows
> DELETED as already done by the 05-18 session per its report §a.11/12 —
> the F59-F61 simplification row (evaluated: hasFilters/aiValue already
> minimal, TZ table landed, long-excerpt wraps cleanly), T23 evidence PNG
> (committed), the single-command dark sweep (`--themes` flag), and the
> SectionHeading trial (rejected as D6; hover aligned 100/800). This pass
> executes the remaining unblocked rows (stale-pass warning, /api/ops
> human surface, retention surfacing, cache-semantics hardening, coverage
> pass, hash-cost probe, golden normalization audit) and the small
> hygiene batch from the 05-18 NEXT-50 list.

> Tenth pass 2026-10-12 evening (verification-closure plan, 17-31): seven
> rows DELETED as done — ingest.go extraction (gateAndFinalize,
> `83fe566`), export stall UI + memory-budget test (`ba6ec8f`), the
> HTTP-stack research report (`399fd50`, docs/research/), the /metrics
> sibling scrape + smoke (serve_test.go text-format assertions), the
> BuildFlow spawn-starvation filing (**BuildFlow#50**, `a061fbb`), and
> the status-report ANNOTATE+archive pass (63 inline markers, 2 files
> archived; residue = 13 live reports triaged in the archive README).
> The recording-proxy row downgraded to BLOCKED: its two timeout tests
> already follow the context-blocking pattern (verified race-clean);
> only the injected-deadline REFACTOR for future timing tests remains.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## Unblocked (owner answered, work remains)

None right now — remaining work is blocked or low-impact (below).

## Blocked (owner / external gates)

| Task                                                                                                                                                                                                                                                                                                                                                                                                                        | Status       | Impact | Effort | Evidence                                                                                                                                                         |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production re-ingest for ContentHash v2 (one-time; the live DB still hashes v1 fields)                                                                                                                                                                                                                                                                                                                                      | 🔵 `BLOCKED` | High   | 60min  | needs the prod deploy path (owner); CHANGELOG domain-redesign entry; docs/status/archived 2026-10-11_18-21 §c.1; docs/status/2026-10-12_02-34 §c                 |
| Deploy both hosts with the current lens module options (`aiDids` newly available; `-max-age`/`-recording-timeout` verified already exposed and wired)                                                                                                                                                                                                                                                                       | 🔵 `BLOCKED` | High   | 30min  | needs the host deploy path (owner); `modules/telephony/lens.nix` (nix-international-telephony e7c0f8b); sibling flake check green 2026-10-12                     |
| Browser-level check: htmx fragment polls really revalidate to 304 + DOM-churn on cached-body swaps                                                                                                                                                                                                                                                                                                                          | 🔵 `BLOCKED` | Low    | 15min  | owner manual check (no browser in the agent env); server-side 304s + no-cache proven in `cmd/pbx-lens/serve_test.go`; docs/status/2026-10-12_13-49 §f.7/39, §g.2 |
| Fold in go-etag#2 (Flush observability) and go-appkit#26 (testkit drain cost) when upstream lands them; re-verify the ledger quarterly                                                                                                                                                                                                                                                                                      | 🔵 `BLOCKED` | Low    | 30min  | both OPEN 2026-10-12, zero comments; `docs/upstream-filings.md`; docs/status/2026-10-12_13-49 §f.24/48                                                           |
| Fold in templ-components#48 (vertical BarChart 0px bars), #49 (TableRow focus ring), #50 (skill version stamping), #51 (ScrollbackLine id/class hook — retires the hand-rolled ?q= match-log classes), #55 (EnsureID random-id default — retire the pinned nav id + determinism gate only if upstream makes ids deterministic) when they land; the bump watchlist in docs/templ-components-adoption.md is the re-check list | 🔵 `BLOCKED` | Low    | 30min  | all three OPEN, filed 2026-10-12; `docs/upstream-filings.md`; local workarounds (`!items-stretch`, focus-visible CSS) retire on fold-in                          |
| >1MiB marathon transcripts: accept untagged streaming (log via `OnBufferOverflow`) or raise `MaxBufferSize` — policy decision                                                                                                                                                                                                                                                                                               | 🔵 `BLOCKED` | Low    | 15min  | owner policy; `internal/web/caching.go` buffer config; docs/status/2026-10-12_13-49 §f.28                                                                        |
| Confirm the next weekly dependabot run proposes bumps for the newly-direct deps (`httputil`, `server_timing`, `go-etag/server`)                                                                                                                                                                                                                                                                                             | 🔵 `BLOCKED` | Low    | 10min  | waits on the weekly dependabot cycle; `.github/dependabot.yml` gomod; docs/status/2026-10-12_13-49 §f.29/49                                                      |
| Owner decision pack D1-D5 (date/number vocabulary; dashboard control rows; quiet-week volume-chart filing; export-note collapse; AI-share purple tone) — D1 gates a one-decision all-surfaces date change                                                                                                                                                                                                                   | 🔵 `BLOCKED` | High   | 30min  | docs/decisions/2026-10-15_ui-owner-decision-pack.md; 02-24 SUPERB plan M2; outcomes recorded here + AGENTS.md on answer (plan F7)                                |
| T15 MOS poller (UI ready, data never arrives): fetch CDR Detail-Records, extract scores, idempotent wire into serve/ingest, fixtures + live-verify + dashboard end-to-end                                                                                                                                                                                                                                                   | 🔵 `BLOCKED` | High   | XL     | 02-24 SUPERB plan M12 (F39-F45); drain-before-enrich gotcha in docs/gotchas.md; needs a Telnyx API window (next opens 2027-01)                                   |
| PII enforcement: E164 normalization + ParticipantHash at the chosen boundary (ingest vs read), policy flag + docs                                                                                                                                                                                                                                                                                                           | 🔵 `BLOCKED` | Medium | L      | 02-24 SUPERB plan M13 (F46-F49); `calls.E164`/`ParticipantHash` exist unenforced; enforcement-point decision = owner                                                                                                             |
| Re-run the full buildflow gate from a clean checkout for the literal 70/70 receipt once BuildFlow#56 lands (gate tools trail the 1.27.2 toolchain floor); close the workaround note in the v0.2.0 receipt                                                                                                                                                                                                                     | 🔵 `BLOCKED` | Low    | 30min  | BuildFlow#56; self-review 16-32 §f.3; CHANGELOG v0.2.0 receipt                                                                                                                                                                   |
| Sibling repo: fold the backup-rotation opinion (app produces verified snapshots and never deletes; the host owns retention/shipping) into `modules/telephony/lens.nix` timer descriptions/comments                                                                                                                                                                                                                            | 🔵 `BLOCKED` | Low    | 15min  | docs/backup-rotation-opinion.md; multi-session 16-32 §f.31; nix-international-telephony                                                                                                                                          |

## Low impact

| Task                                                                                                                                                                                                                                                                                                                  | Status       | Impact | Effort | Evidence                                                                                                                                  |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec)                                                                                                                                                                    | 🔵 `BLOCKED` | Low    | 30min  | needs a real API key; `internal/ingest/telnyx/dto.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                       |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)                                                                                                                                                                         | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate |
| Harden the recording-proxy timeout tests: injected deadlines instead of budget+sleep pairs (expected `-test.shuffle` flake candidates under load; the context-blocking pattern already covers the two timeout tests — verified race-clean 2026-10-12; the injected-deadline refactor remains for any NEW timing test) | 🔵 `BLOCKED` | Low    | 45min  | `internal/web/recording_test.go`; verified state 2026-10-12; docs/buildflow-dev-gate.md                                                   |
| Watch dependabot-core#16538 and answer maintainer questions; owner decides gomod pinning afterward                                                                                                                                                                                                                    | 🔵 `BLOCKED` | Low    | 10min  | owner call pending (#16538 context); docs/status/2026-10-12_10-33 §f.29/§f.30                                                             |
| R27.x advisory residue pass: art-dupl, branching-flow, nix-checker cleanup, README screenshots, vulnix re-triage (all documented advisories)                                                                                                                                                                          | 🔵 `BLOCKED` | Low    | 60min  | docs/buildflow-dev-gate.md advisory list; docs/status/2026-10-12_10-33 §f.40                                                              |
| Recording-proxy timing-hop budget asserts: Server-Timing hops must not perturb the upstream timeout budgets (cache-semantics hardening landed: Vary, Content-Length, HEAD, query-string; trailing-slash deliberately unpinned, 05-40 M11)                                                                                                                                             | 🔴 `TODO`    | Low    | 45min  | `cmd/pbx-lens/serve_test.go`; `internal/web/recording_test.go`; docs/status/2026-10-12_13-49 §f.15/16/25/46                               |
| Security headers middleware: `Referrer-Policy: no-referrer` + `X-Content-Type-Options: nosniff` on every response (one small middleware in the ExtraMiddlewares chain; ordering per docs/http-stack.md)                                                                                                                                                                                                     | 🔴 `TODO`    | Low    | 30min  | self-review 16-32 §f.38; docs/http-stack.md chain doctrine                                                                                |
| /ops page hardening leftovers: auth-gated e2e lane membership (the named lane hits four surfaces; ops is the fifth page), stale-verdict browser probe, a11y sweep (heading order, focus path, verdict-chip contrast)                                                                                                                                                                                       | 🔴 `TODO`    | Low    | 45min  | the /ops 304 assert landed this pass (CHANGELOG); multi-session 16-32 §f.36 + self-review 16-32 §f.28/37                                  |
| verify-all `--fast` mode (contracts only, skip the 41-PNG sweep) + screenshot-probe polish: KNOWN_SURFACES/pages single-source drift guard, skip detail discovery when no detail-* surface is selected                                                                                                                                                                                                     | 🔴 `TODO`    | Low    | 45min  | multi-session 16-32 §e.5/§f.20/34/35; scripts/verify-all.sh; scripts/screenshot-probe.js                                                  |
| Pixel-diff practice: save the first baseline corpus, mask the dashboard footer stamp (~354-668 volatile px), evaluate tightening the 0.5% default, wire `--diff` into verify-all as alert-only; baseline STRATEGY itself = the ROADMAP owner question                                                                                                                                                      | 🔴 `TODO`    | Low    | 45min  | multi-session 16-32 §f.12/18 + self-review 16-32 §f.12; scripts/screenshot-probe.js                                                       |
| Export lane tests: consumer-side parse of a generated CSV/JSONL asserting the EXPORT_INCOMPLETE marker contract end-to-end + streaming memory-bound benchmark at 10k rows (M16 covered list+transcript)                                                                                                                                                                                                   | 🔴 `TODO`    | Low    | 45min  | self-review 16-32 §f.23; multi-session 16-32 §f.38; `internal/web/export` streaming walk                                                  |
| Recording-proxy negative paths: upstream 403 (signed-URL expiry) must surface an honest error, not a hang + expose the recording throttle limit/window as serve flags only if ops asks for tuning (generous defaults hardcoded today)                                                                                                                                                                     | 🔴 `TODO`    | Low    | 45min  | self-review 16-32 §f.24; multi-session 16-32 §f.42; `internal/web/recording.go`                                                           |
| Ops/retention visibility: README alerting recipe for `last_pass_age_seconds` + surface the deployment's `-max-age`/purge-timer state and last-purge age on the /ops card (deployment config visibility, not just counters)                                                                                                                                                                                 | 🔴 `TODO`    | Low    | 45min  | self-review 16-32 §f.14/15/36; `/api/ops` payload; internal/cli/poller.go                                                                 |
| CSP report-only dry run: actually collect browser violations (probe the surfaces under `CSP_REPORT_ONLY=1`) and make the enforce/adjust call — the branch ships (v0.2.0) but the sweep never ran                                                                                                                 | 🔴 `TODO`    | Low    | 45min  | multi-session 16-32 §f.14; `cmd/pbx-lens/serve.go` CSP_REPORT_ONLY branch; FEATURES CSP row                               |
| Print page-breaks for the ops page (it renders cards; print was specced for the detail page only — cards can split across printed pages)                                                                                                                                                                                                  | 🔴 `TODO`    | Low    | 15min  | self-review 16-32 §f.26; app.css `@media print` block                                                                                     |
| FTS quality pass: multi-word queries + ranking spot checks (untouched since the digits-shadow work)                                                                                                                                                                                                                                                                                                       | 🔴 `TODO`    | Low    | 60min  | self-review 16-32 §f.21; internal/search                                                                                                  |
| Search-page hint: add one example query to the hint copy (usability)                                                                                                                                                                                                                                                                                                                                      | 🔴 `TODO`    | Low    | 5min   | self-review 16-32 §f.30; search_templ hint copy                                                                                           |
| `X-Robots-Tag: noindex` response header alongside the meta (covers non-HTML endpoints uniformly)                                                                                                                                                                                                                                                                                                          | 🔴 `TODO`    | Low    | 15min  | self-review 16-32 §f.46; robots route + `SEOMeta` NoIndex landed M15                                                                      |
| RelativeTime tooltip: day-precision fallback text when relative age exceeds a week (pre-existing UX nit)                                                                                                                                                                                                                                                                                                  | 🔴 `TODO`    | Low    | 15min  | self-review 16-32 §f.29                                                                                                                   |
| Release runbook as a script (`scripts/cut-release.sh`: dirty-tree quiesce check → build → test → probes → CHANGELOG completeness sweep → tag → `go list -m` proxy check); encodes the 2026-10-15 tag-red-then-fix lesson; add the never-on-a-dirty-tree check for BuildFlow#55 until it folds                                                                                                               | 🔴 `TODO`    | Low    | 60min  | self-review 16-32 §f.4/19/32; release commit 7d15c6f receipt                                                                              |
| styles.css freshness guard: CI regenerates with a pinned Tailwind and `git diff --exit-code` the committed copy — prove nixpkgs-pin vs npx byte-determinism FIRST (the 2026-10-15 foreign-build incident shipped a committed copy the pinned builder never produced)                                                                                                                                        | 🔴 `TODO`    | Low    | 45min  | incident fix dd50513/2b0bcb2; self-review 16-32 §f.5; .gitignore policy note                                                              |
| Telnyx quarterly re-verify bundle as a checklist file (date + commands, next window 2027-01) + confirm the calendar entry exists                                                                                                                                                                                                                                                                          | 🔴 `TODO`    | Low    | 15min  | AGENTS quarterly cluster (recordings verified 2026-10-12); self-review 16-32 §f.35; multi-session 16-32 §f.28                             |
| docs/DOMAIN_LANGUAGE.md: MOS/cost vocabulary rows so the 2027-01 window starts with shared language                                                                                                                                                                                                                                                                                                       | 🔴 `TODO`    | Low    | 15min  | self-review 16-32 §f.49; docs/DOMAIN_LANGUAGE.md                                                                                          |
| Ingest `-dry-run` flag on ingest-cdr (parse report without writes) for the F63 re-ingest rehearsal                                                                                                                                                                                                                                                                                                        | 🔴 `TODO`    | Low    | 30min  | self-review 16-32 §f.48; docs/process-prep-2026-10-15.md F63                                                                              |
| Pre-commit hook: AGENTS.md 220-line budget check at commit time (enforced at gate time today)                                                                                                                                                                                                                                                                                                             | 🔴 `TODO`    | Low    | 15min  | self-review 16-32 §f.47; fdea674                                                                                                          |
| True clean-clone rehearsal of ensureCodegen in a scratch worktree (`git clean -xfd` path proven end-to-end; approximated by deleting one generated file)                                                                                                                                                                                                                                                  | 🔴 `TODO`    | Low    | 30min  | multi-session 16-32 §f.23; scripts/lib/ui-probe.js self-heal path                                                                         |
| Dark-theme eyeball pass over the fresh 41-PNG corpus (M15/M17 changed visuals; capture ≠ eyeball)                                                                                                                                                                                                                                                                                                         | 🔴 `TODO`    | Low    | 30min  | multi-session 16-32 §f.16; scripts/screenshot-probe.js output dir                                                                         |
| Coverage floor 80% in CI + cmd/pbx-lens httptest wiring test (pre-existing ci.yml TODO)                                                                                                                                                                                                                                                                                                                   | 🔴 `TODO`    | Low    | 60min  | self-review 16-32 §f.16/17; .github/workflows/ci.yml                                                                                      |
| Mobile card-list for the calls table (375px hides "Opening line" behind horizontal scroll): sketch vs scroll table, implement, verify                                                                                                                                                                                 | 🔵 `BLOCKED` | Low    | L      | 02-24 SUPERB plan M17 (F56-F58); status f-14; owner taste call on card vs scroll first                                                    |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
