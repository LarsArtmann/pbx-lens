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

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |
| 🟢 `DONE`        | Completed. Remove from this list; log in CHANGELOG.md.  |

## Unblocked (owner answered, work remains)

| Task                                                                                                                                                            | Status    | Impact | Effort | Evidence                                                                                                                                                                      |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cut v0.1.0: annotated tag + GitHub release notes + install-path verification                                                                                    | 🔴 `TODO` | Medium | 30min  | owner ratified the release 2026-10-12 (R4); precondition = one clean full gate at load < 40; CHANGELOG current; docs/status/2026-10-12_01-16 §a.1 has the step-level evidence |
| Sibling NixOS module: expose `-max-age` (default: unset = keep everything, owner R5) + `-recording-timeout` in `modules/telephony/lens.nix`, re-eval both hosts | 🔴 `TODO` | High   | 30min  | owner set the doctrine 2026-10-12 (R5); module lives in nix-international-telephony; docs/status/archived/2026-10-11_19-37 §f.14                                              |
| Direction auto-detect: implement the DID-interception resolver from docs/did-direction-design.md (mapper option + `-ai-did` flag + fixtures)                    | 🔴 `TODO` | Medium | 90min  | design note 2026-10-12 (R22); plugs into `internal/ingest/telnyx/map.go`; NixOS module option is the follow-up                                                                |

## Blocked (owner / external gates)

| Task                                                                                   | Status       | Impact | Effort | Evidence                                                                                                                                         |
| -------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Production re-ingest for ContentHash v2 (one-time; the live DB still hashes v1 fields) | 🔵 `BLOCKED` | High   | 60min  | needs the prod deploy path (owner); CHANGELOG domain-redesign entry; docs/status/archived 2026-10-11_18-21 §c.1; docs/status/2026-10-12_02-34 §c |

## Low impact

| Task                                                                                                                                               | Status       | Impact | Effort | Evidence                                                                                                                                                                           |
| -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Swap the hand-rolled AI/human proportion bar for `display.SegmentBar` when templ-components publishes it                                           | 🔵 `BLOCKED` | Low    | 10min  | `internal/web/dashboard.templ` `aiHumanCard`; SegmentBar exists unreleased upstream, not in v1.21.0                                                                                |
| Live-verify the Telnyx Detail Records + insights DTOs against the real API (shapes decoded tolerantly; call-control record not in the public spec) | 🔵 `BLOCKED` | Low    | 30min  | needs a real API key; `internal/ingest/telnyx/dto.go` `DetailRecord` aliases; AGENTS.md quarterly re-verify cluster                                                                |
| At schema v2: length-prefixed ContentHash field encoding (newline-separated concatenation can alias crafted field pairs; format frozen at v1)      | 🔵 `BLOCKED` | Low    | 60min  | SCHEDULED at v2 (owner decision 2026-10-11); data-model review in docs/reviews; hash change = one-time correction wave, needs the v2 gate                                          |
| Harden the recording-proxy timeout tests: injected deadlines instead of budget+sleep pairs (expected `-test.shuffle` flake candidates under load)  | 🔴 `TODO`    | Low    | 45min  | `internal/web/recording_test.go`; budgets raised to load-safe 50 ms/250 ms 2026-10-12 (the 1 ms flake is gone); the injected-deadline refactor remains; docs/buildflow-dev-gate.md |
| `cmd/pbx-lens/ingest.go` (269 lines): extract the shared open-system → age-gate → drain → rebuild sequence before it re-approaches 350             | 🔴 `TODO`    | Low    | 30min  | `cmd/pbx-lens/ingest.go`; docs/status 2026-10-12_02-34 §f.14                                                                                                                       |

<!-- Harvest ledger for this list lives in the pass report (docs/status/),
     per docs-health harvest-guide: the ledger records the pass, it is not a
     backlog. -->
