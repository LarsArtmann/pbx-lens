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
| Cut v0.2.0: the CHANGELOG Unreleased section carries the whole HTTP-stack adoption set                                                                                                                                                                                                                                                                                                                                      | 🔵 `BLOCKED` | Medium | 30min  | owner release cadence; CHANGELOG.md Unreleased; docs/status/2026-10-12_13-49 §f.41                                                                               |
| Browser-level check: htmx fragment polls really revalidate to 304 + DOM-churn on cached-body swaps                                                                                                                                                                                                                                                                                                                          | 🔵 `BLOCKED` | Low    | 15min  | owner manual check (no browser in the agent env); server-side 304s + no-cache proven in `cmd/pbx-lens/serve_test.go`; docs/status/2026-10-12_13-49 §f.7/39, §g.2 |
| Fold in go-etag#2 (Flush observability) and go-appkit#26 (testkit drain cost) when upstream lands them; re-verify the ledger quarterly                                                                                                                                                                                                                                                                                      | 🔵 `BLOCKED` | Low    | 30min  | both OPEN 2026-10-12, zero comments; `docs/upstream-filings.md`; docs/status/2026-10-12_13-49 §f.24/48                                                           |
| Fold in templ-components#48 (vertical BarChart 0px bars), #49 (TableRow focus ring), #50 (skill version stamping), #51 (ScrollbackLine id/class hook — retires the hand-rolled ?q= match-log classes), #55 (EnsureID random-id default — retire the pinned nav id + determinism gate only if upstream makes ids deterministic) when they land; the bump watchlist in docs/templ-components-adoption.md is the re-check list | 🔵 `BLOCKED` | Low    | 30min  | all three OPEN, filed 2026-10-12; `docs/upstream-filings.md`; local workarounds (`!items-stretch`, focus-visible CSS) retire on fold-in                          |
| >1MiB marathon transcripts: accept untagged streaming (log via `OnBufferOverflow`) or raise `MaxBufferSize` — policy decision                                                                                                                                                                                                                                                                                               | 🔵 `BLOCKED` | Low    | 15min  | owner policy; `internal/web/caching.go` buffer config; docs/status/2026-10-12_13-49 §f.28                                                                        |
| Confirm the next weekly dependabot run proposes bumps for the newly-direct deps (`httputil`, `server_timing`, `go-etag/server`)                                                                                                                                                                                                                                                                                             | 🔵 `BLOCKED` | Low    | 10min  | waits on the weekly dependabot cycle; `.github/dependabot.yml` gomod; docs/status/2026-10-12_13-49 §f.29/49                                                      |
| Owner decision pack D1-D5 (date/number vocabulary; dashboard control rows; quiet-week volume-chart filing; export-note collapse; AI-share purple tone) — D1 gates a one-decision all-surfaces date change                                                                                                                                                                                                                   | 🔵 `BLOCKED` | High   | 30min  | docs/decisions/2026-10-15_ui-owner-decision-pack.md; 02-24 SUPERB plan M2; outcomes recorded here + AGENTS.md on answer (plan F7)                                |
| T15 MOS poller (UI ready, data never arrives): fetch CDR Detail-Records, extract scores, idempotent wire into serve/ingest, fixtures + live-verify + dashboard end-to-end                                                                                                                                                                                                                                                   | 🔵 `BLOCKED` | High   | XL     | 02-24 SUPERB plan M12 (F39-F45); drain-before-enrich gotcha in docs/gotchas.md; needs a Telnyx API window (next opens 2027-01)                                   |
| PII enforcement: E164 normalization + ParticipantHash at the chosen boundary (ingest vs read), policy flag + docs                                                                                                                                                                                                                                                                                                           | 🔵 `BLOCKED` | Medium | L      | 02-24 SUPERB plan M13 (F46-F49); `calls.E164`/`ParticipantHash` exist unenforced; enforcement-point decision = owner                                             |

## Low impact

| Task                                                                                                                                                                                                                                                                                                                  | Status       | Impact | Effort | Evidence                                                                                                                                  |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec)                                                                                                                                                                    | 🔵 `BLOCKED` | Low    | 30min  | needs a real API key; `internal/ingest/telnyx/dto.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                       |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)                                                                                                                                                                         | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate |
| Harden the recording-proxy timeout tests: injected deadlines instead of budget+sleep pairs (expected `-test.shuffle` flake candidates under load; the context-blocking pattern already covers the two timeout tests — verified race-clean 2026-10-12; the injected-deadline refactor remains for any NEW timing test) | 🔵 `BLOCKED` | Low    | 45min  | `internal/web/recording_test.go`; verified state 2026-10-12; docs/buildflow-dev-gate.md                                                   |
| Watch dependabot-core#16538 and answer maintainer questions; owner decides gomod pinning afterward                                                                                                                                                                                                                    | 🔵 `BLOCKED` | Low    | 10min  | owner call pending (#16538 context); docs/status/2026-10-12_10-33 §f.29/§f.30                                                             |
| R27.x advisory residue pass: art-dupl, branching-flow, nix-checker cleanup, README screenshots, vulnix re-triage (all documented advisories)                                                                                                                                                                          | 🔵 `BLOCKED` | Low    | 60min  | docs/buildflow-dev-gate.md advisory list; docs/status/2026-10-12_10-33 §f.40                                                              |
| Recording-proxy timing-hop budget asserts: Server-Timing hops must not perturb the upstream timeout budgets (cache-semantics hardening landed: Vary, Content-Length, HEAD, query-string; trailing-slash deliberately unpinned, 05-40 M11)                                                                                                                      | 🔴 `TODO`    | Low    | 45min  | `cmd/pbx-lens/serve_test.go`; `internal/web/recording_test.go`; docs/status/2026-10-12_13-49 §f.15/16/25/46                               |
| Mobile card-list for the calls table (375px hides "Opening line" behind horizontal scroll): sketch vs scroll table, implement, verify                                                                                                                                                                                 | 🔵 `BLOCKED` | Low    | L      | 02-24 SUPERB plan M17 (F56-F58); status f-14; owner taste call on card vs scroll first                                                    |
| Alert-only pixel-diff on the 39-PNG screenshot set (tolerance report, never a gate)                                                                                                                                                                                                                                   | 🔴 `TODO`    | Low    | M      | 02-24 SUPERB plan M14 (F50-F51); status f-29; screenshots are the corpus                                                                  |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
