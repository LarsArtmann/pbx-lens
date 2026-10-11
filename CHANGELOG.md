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
