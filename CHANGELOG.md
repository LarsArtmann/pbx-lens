# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
No version has been tagged yet; everything lives under Unreleased.

## [Unreleased]

### Added

- Project scaffold: Go module, Nix flake (go-standard flakeModule), BuildFlow
  config, CI-agnostic quality gate (SUPERB plan T1).
- Domain model: `Observation` with validation invariants, `SourceKind`
  (freeswitch-cdr, telnyx-conversation, webhook-log, agent-transcript),
  `Direction`, `E164`/`ParticipantHash` PII primitives (T2).
- FreeSWITCH Master.csv parser (13- and 18-column templates) and the
  `ingest-cdr` CLI command (T3).
- Telnyx client (conversations pager, per-conversation messages, recordings
  list/fetch, `FlexTime` wire decoder for precision-unstable timestamps) and
  the `ingest-telnyx` CLI command with client-side recording correlation (T4).
  Live-proven: 8 real conversations double-ingested with zero duplicate events.
- Content-hash idempotency: identical re-observation replays to zero events;
  a changed observation on a live stream is superseded by a `call.corrected`
  event (append-only audit trail; enrichment rides along so the projection
  never loses cost/insight data).
- `call_views` sqlite/memory read model via metaengine projection.
- `smoke` and `version` CLI commands.
- Project documentation set: README, FEATURES, TODO_LIST, ROADMAP,
  docs/DOMAIN_LANGUAGE (SUPERB plan T16).
- GitHub Actions CI lane (go-output's pattern, adapted single-module): build,
  test with race + coverage floor, go mod tidy check, golangci-lint,
  govulncheck, and a warning-only art-dupl gate; dependabot now also watches
  the github-actions ecosystem.
- `internal/cli` package: tested flag parsing (`ResolveKey`, `ParseSince`,
  `ParseDirection`), the shared dispatch loop `Ingest` (records/corrects/
  rejects with per-observation reasons), `FetchTelnyxObservations` (the
  fetch/map/correlate loop both CLI and the serve poller use), and
  `OpenSystem`/`DrainProjections` — the correct synchronous projection-drain
  pattern (Stop → Start → await WorkerStopped) for the one-shot-drainer
  projection host.
- Domain redesign: `TranscriptTurn` + full transcript on the Observation
  (chronology-normalized, whole-second truncated), `Enrichment` (cost,
  insight) with value-hash idempotency, `call.corrected`/`call.enriched`
  events, and the excerpt-semantics fix (first user turn by created_at,
  order-independent of API page order). ContentHash is now v2 — the
  production DB must be re-ingested once.
- Flake devShell and tooling (go-output's pattern): hermetic shell with go,
  golangci-lint, gopls, govulncheck, and vendored cqrs-lint (pinned
  go-cqrs-lite rev, replacing the /tmp binary), plus the
  `nix run .#update-vendor-hash` repair app.
- Telnyx enrichment pass (T15 + T13): `EnrichTelnyxCalls` correlates
  `/detail_records` (call-control CDRs, cost → `Enrichment.CostMicros`)
  and `/ai/conversations/{id}/conversations-insights` (newest completed
  row → `Enrichment.InsightSummary`) against already-observed streams and
  dispatches `call.enrich`. Wired into `ingest-telnyx` and the serve
  poller (a drain separates ingest from enrichment, and enrichment events
  drain before the pass ends). Already-final fields skip their fetch; the
  detail list is fetched lazily at most once per pass; an unchanged
  enrichment replays to zero events. Detail-record decoding is
  alias-tolerant (`id`/`uuid`, `call_control_id`/`telnyx_call_control_id`,
  cost as string or number) because the public OpenAPI spec publishes no
  call-control record shape — live-API verification still pending.
- `CallView.Observation()` rebuild with a hash-roundtrip invariant test —
  the enrich pass addresses `call.enrich` through the stored view without
  re-fetching sources.
- Nightly backup staging drafts (T14): `deploy/systemd/pbx-lens-backup.service`
  + `.timer` (02:15 nightly, Persistent) staging a verified snapshot into
  `/var/lib/pbx-lens/backups/`, and `backup.enable`/`backup.calendar`
  options in the NixOS module draft. Retention and off-host shipping stay
  with the host backup tool.
- One-flow UI e2e test (seed → list → filter → detail → search →
  freshness → auth gate) and an httptest-fake Telnyx poller test covering
  the full fetch → ingest → enrich → index-rebuild loop plus second-pass
  idempotency.
- Search-page hint that partial phone numbers never match (FTS5 keeps a
  number as one token), replacing silent zero-result confusion.
- Ingest-side retention gate (2026-10-11): `serve -max-age` refuses polled
  observations older than the window with a labeled `retention:age-cutoff`
  rejection instead of recording them — the ingest-side half of the
  retention doctrine (the purge half is `pbx-lens purge`). Telnyx
  enrichment seeds are dropped in tandem, so refused streams cost no
  enrichment fetches. Zero keeps everything (retention stays opt-in).
- The one-shot `ingest-cdr`/`ingest-telnyx`/`ingest-webhook`/`ingest-agent`
  commands carry the same `-max-age` gate (`cli.AgeGate`), so a CLI backfill
  cannot silently re-import rows the window refuses.
- Failed basic-auth attempts are throttled by a shared token bucket
  (`golang.org/x/time/rate`, ~10/minute sustained, burst 5): brute-force
  guesses cost wall-clock time, not just bcrypt CPU; successful logins
  never wait.
- Typed boundary models: `TurnRole` (transcript speaker role, hash-compatible),
  `calls.RetentionReason` (audit reason on the retire command — a typo'd
  reason no longer compiles), `web.PIIMode` (export pii mode), and the
  consumer-side `web.RecordingFetcher` interface (the UI no longer imports
  the concrete Telnyx client type).
- Backup durability: the staging snapshot is fsynced before the atomic
  rename AND the parent directory is flushed after it, so a crash can
  neither publish half-written pages nor lose the new name.
- Checked-in UI e2e probe: `TestLiveServerEndToEnd` drives real HTTP over a
  real listener against a real sqlite index on all four surfaces (the
  manual probe is now a regression test).
- Four-lens review session (2026-10-11: data-model, full-code, naming,
  modularize) with every file visited, 8 on-sight fixes, and HTML reports
  under `docs/reviews/`.

### Fixed

- cqrs-lint full pass to zero errors: total fold (unknown event types on a
  Call stream are classified Corruption, not silently ignored), `mustRegister`
  composition-root helper, errorfamily classification across the CLI, and
  `Volume(100_000)` declared on the projection. Remaining warnings are
  documented decisions in AGENTS.md.
- `nix build` shipped an empty result (no binary): the flake built the
  package-less repo root instead of the binary; it now targets
  `./cmd/pbx-lens`, so `result/bin/pbx-lens` exists and `nix run . -- version`
  works.
- golangci-lint gate driven to ZERO findings (was 382, then 124): real fixes
  (errcheck on hash writers and file closes, err113 sentinel in FlexTime,
  magic numbers to named constants, CSV templates de-globaled, `strconv.Atoi`,
  main.go de-cyclopped via the `internal/cli` extraction) plus curated policy
  in `.golangci.yml` (snake_case JSON tags per project convention, wrapcheck
  globs for own module and the cqrs stack, fleet-precedent test-file and
  cmd/ exclusions).
- Smoke demo uses an obviously fake +1555 number instead of a real-looking
  handset number.
- README: dropped the not-yet-usable `go install @latest` line and corrected
  the changed-row doctrine (supersession, not conflict).
- The web UI era (`serve` on go-appkit + templ-components): calls list with
  direction/AI/date filters and keyset pagination, per-call detail with
  role-split transcript and honest empty states, FTS5 full-text search with
  `<mark>` highlighting, recording playback through a server-side proxy
  (signed URLs never reach the browser, Range supported), bcrypt-htpasswd
  basic auth, `/api/freshness` + stale-index banner, JSONL/CSV export, and
  the quality dashboard (stat cards, 30-day volume chart, AI/human split,
  trend sparkline, MOS placeholder).
- Poll loop inside `serve` (`-poll-every`): per-source ingest, one projection
  drain and one FTS rebuild per pass; per-source failures are logged and
  skipped, never fatal.
- `pbx-lens backup` command (T14): live-safe index DB snapshot for the
  nightly staging (the timer itself is host-side).
- sd_notify integration via `go-appkit/systemd` (`Type=notify`, READY/STOPPING/
  WATCHDOG) plus the systemd unit and NixOS module drafts under `deploy/`
  (T7; the module moves to the sibling repo).
- sqlite FTS index with busy-timeout coexistence, rebuild commands for every
  ingest verb, and a documented full-token matching caveat for phone numbers.
- `usage()` now lists the `purge` subcommand (it existed but was hidden).
- Search `Hit` carries typed `Kind`/`NativeID` (was raw strings re-parsed at
  the web boundary).
- Auth timing pad hashes at the LOADED htpasswd entries' bcrypt cost
  (`padCost`), so the equalization claim holds for cost>10 files; was
  MinCost.
- Transcript excerpts are capped by the domain-owned ceiling
  (`calls.DefaultExcerptMax`, 200 runes) in every source mapper, not just
  the display layer.
- The last hand-built `"StreamMarker:..."` projection-key literal (CDR join
  in the agent transcript path) was removed; keys derive via
  `obs.StreamID().String()` exclusively.
- AGENTS.md slimmed under the 220-line structure-lint limit by extracting
  the paid-for war stories and policies into `docs/` (gotchas,
  telnyx-live-api, cqrs-lint-policy, buildflow-dev-gate,
  upstream-filings). Owner decisions recorded there: flat `internal/calls`
  domain HOLD with a pre-agreed write/read seam, single-module HOLD,
  ContentHash v2 length-prefixed encoding SCHEDULED at v2, recording-proxy
  timeout bound DEFERRED until traffic data.
