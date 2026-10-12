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

| Task                                                                                                                                                                                                           | Status       | Impact | Effort | Evidence                                                                                                                                                         |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production re-ingest for ContentHash v2 (one-time; the live DB still hashes v1 fields)                                                                                                                         | 🔵 `BLOCKED` | High   | 60min  | needs the prod deploy path (owner); CHANGELOG domain-redesign entry; docs/status/archived 2026-10-11_18-21 §c.1; docs/status/2026-10-12_02-34 §c                 |
| Deploy both hosts with the current lens module options (`aiDids` newly available; `-max-age`/`-recording-timeout` verified already exposed and wired)                                                          | 🔵 `BLOCKED` | High   | 30min  | needs the host deploy path (owner); `modules/telephony/lens.nix` (nix-international-telephony e7c0f8b); sibling flake check green 2026-10-12                     |
| Cut v0.2.0: the CHANGELOG Unreleased section carries the whole HTTP-stack adoption set                                                                                                                         | 🔵 `BLOCKED` | Medium | 30min  | owner release cadence; CHANGELOG.md Unreleased; docs/status/2026-10-12_13-49 §f.41                                                                               |
| Browser-level check: htmx fragment polls really revalidate to 304 + DOM-churn on cached-body swaps                                                                                                             | 🔵 `BLOCKED` | Low    | 15min  | owner manual check (no browser in the agent env); server-side 304s + no-cache proven in `cmd/pbx-lens/serve_test.go`; docs/status/2026-10-12_13-49 §f.7/39, §g.2 |
| Fold in go-etag#2 (Flush observability) and go-appkit#26 (testkit drain cost) when upstream lands them; re-verify the ledger quarterly                                                                         | 🔵 `BLOCKED` | Low    | 30min  | both OPEN 2026-10-12, zero comments; `docs/upstream-filings.md`; docs/status/2026-10-12_13-49 §f.24/48                                                           |
| Fold in templ-components#48 (vertical BarChart 0px bars), #49 (TableRow focus ring), #50 (skill version stamping) when they land; the bump watchlist in docs/templ-components-adoption.md is the re-check list | 🔵 `BLOCKED` | Low    | 30min  | all three OPEN, filed 2026-10-12; `docs/upstream-filings.md`; local workarounds (`!items-stretch`, focus-visible CSS) retire on fold-in                          |
| >1MiB marathon transcripts: accept untagged streaming (log via `OnBufferOverflow`) or raise `MaxBufferSize` — policy decision                                                                                  | 🔵 `BLOCKED` | Low    | 15min  | owner policy; `internal/web/caching.go` buffer config; docs/status/2026-10-12_13-49 §f.28                                                                        |
| Confirm the next weekly dependabot run proposes bumps for the newly-direct deps (`httputil`, `server_timing`, `go-etag/server`)                                                                                | 🔵 `BLOCKED` | Low    | 10min  | waits on the weekly dependabot cycle; `.github/dependabot.yml` gomod; docs/status/2026-10-12_13-49 §f.29/49                                                      |

## Low impact

| Task                                                                                                                                                                                             | Status       | Impact | Effort | Evidence                                                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------ | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec)                                               | 🔵 `BLOCKED` | Low    | 30min  | needs a real API key; `internal/ingest/telnyx/dto.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                                                                |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)                                                    | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate                                          |
| Harden the recording-proxy timeout tests: injected deadlines instead of budget+sleep pairs (expected `-test.shuffle` flake candidates under load)                                                | 🔴 `TODO`    | Low    | 45min  | `internal/web/recording_test.go`; budgets raised to load-safe 50 ms/250 ms 2026-10-12 (the 1 ms flake is gone); the injected-deadline refactor remains; docs/buildflow-dev-gate.md |
| `cmd/pbx-lens/ingest.go` (269 lines): extract the shared open-system → age-gate → drain → rebuild sequence before it re-approaches 350                                                           | 🔴 `TODO`    | Low    | 30min  | `cmd/pbx-lens/ingest.go`; docs/status 2026-10-12_02-34 §f.14                                                                                                                       |
| Surface `export_cursor_stalled` in the export UI (the loud-fail path is server-side only today)                                                                                                  | 🔴 `TODO`    | Low    | 30min  | `internal/web/export.go` `advanceCursor`; docs/status/2026-10-12_10-33 §f.16                                                                                                       |
| Export memory-budget test: pin `exportBatchSize` page behavior under a bounded allocation (streaming contract guard)                                                                             | 🔴 `TODO`    | Low    | 30min  | `internal/web/queries.go` `exportBatchSize`; docs/status/2026-10-12_10-33 §f.22                                                                                                    |
| `/api/ops`: warn when the reported poller pass is stale (age the lastPass timestamp)                                                                                                             | 🔴 `TODO`    | Low    | 20min  | `internal/web` ops endpoint + `SourceCounters`; docs/status/2026-10-12_10-33 §f.35                                                                                                 |
| Render the 2026-10-12 HTTP-stack audit as the library-deep-dive HTML research report (substance already landed in the SUPERB plan + AGENTS.md)                                                   | 🔴 `TODO`    | Low    | 30min  | docs/planning/2026-10-12_12-47_SUPERB-httputil-go-etag-go-appkit-max-adoption.md; library-deep-dive skill deliverable `docs/research/`                                             |
| Optional: scrape the new `/metrics` surface from the sibling NixOS module and smoke-test with a real scraper (Prometheus/Gatus; endpoint is htpasswd-gated or loopback-only)                     | 🔴 `TODO`    | Low    | 30min  | `cmd/pbx-lens/serve.go` `buildServiceConfig` Metrics wiring; nix-international-telephony `modules/telephony/lens.nix`; docs/status/2026-10-12_13-49 §f.13/14                       |
| File the BuildFlow spawn-starvation failure mode upstream (tools killed at spawn under load; propose pre-flight load check or spawn retry)                                                       | 🔴 `TODO`    | Low    | 30min  | docs/buildflow-dev-gate.md starvation entries (runs #1/#2); sibling filing BuildFlow#45; docs/status/2026-10-12_10-33 §f.6                                                         |
| ANNOTATE + archive the 2026-10-11/12 status reports once their items are resolved; re-annotate the SUPERB plan post-release                                                                      | 🔴 `TODO`    | Low    | 45min  | docs/status/2026-10-12_10-33 §f.38/§f.25; docs-health ANNOTATE mode                                                                                                                |
| Watch dependabot-core#16538 and answer maintainer questions; owner decides gomod pinning afterward                                                                                               | 🔵 `BLOCKED` | Low    | 10min  | owner call pending (#16538 context); docs/status/2026-10-12_10-33 §f.29/§f.30                                                                                                      |
| R27.x advisory residue pass: art-dupl, branching-flow, nix-checker cleanup, README screenshots, vulnix re-triage (all documented advisories)                                                     | 🔵 `BLOCKED` | Low    | 60min  | docs/buildflow-dev-gate.md advisory list; docs/status/2026-10-12_10-33 §f.40                                                                                                       |
| E2E cache-semantics hardening: `Vary: Accept-Encoding` on 304s, `Content-Length` on buffered renders, HEAD/query-string/trailing-slash skip-policy variants, recording timing-hop budget asserts | 🔴 `TODO`    | Low    | 45min  | `cmd/pbx-lens/serve_test.go`; `internal/web/recording_test.go`; docs/status/2026-10-12_13-49 §f.15/16/25/46                                                                        |
| Coverage pass on `caching_test.go` + `serve_test.go` (assert the hot path — fragment conditional GET — not static stand-ins)                                                                     | 🔴 `TODO`    | Low    | 30min  | `internal/web/caching_test.go`; `cmd/pbx-lens/serve_test.go`; docs/status/2026-10-12_13-49 §f.43                                                                                   |
| Concurrent-poll hash-cost sanity probe: overlapping 30s/60s fragment polls under one ETag middleware (CPU ceiling check)                                                                         | 🔴 `TODO`    | Low    | 30min  | `internal/web` poller wiring; docs/status/2026-10-12_13-49 §f.26                                                                                                                   |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
