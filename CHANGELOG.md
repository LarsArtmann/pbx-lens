# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
v0.1.0 is the first tagged release (2026-10-12); new work lands under Unreleased.

## [Unreleased]

Nothing yet.

## [v0.2.0] — 2026-10-15

### Added — 2026-10-15 (/ops surface)

- New `/ops` page: the ingest-and-cache health answer at a glance — poller
  pass freshness with the staleness verdict up top, per-source ingest
  counters, what retention retired and when it last ran, and the response
  cache stats (ETag hit ratio, buffer overflows). Read-only like every
  page; the machine view stays at `/api/ops`. The API also gained
  `last_pass_age_seconds` so monitors can alert on a silently stopped
  poller.

### Fixed — 2026-10-15 (chunked HTML pages)

- Every HTML response now carries an explicit `Content-Length`. Pages are
  rendered into one buffered write, but at 25k+ bytes they overshoot Go's
  2 KiB implicit-length buffer and shipped chunked; the exact length is
  now stated upfront (the compression layer rewrites it when it actually
  encodes).

### Changed — 2026-10-15 (UI trio bump)

- `go-appkit` v0.8.0 → v0.9.0 and `templ-components` v1.21.0 → v1.23.0
  (go.mod floor moves to go ≥ 1.27.2 with appkit's requirement; templ
  stays v0.3.1070). Bump watchlist run; goldens regenerated; no
  user-visible regression.

### Fixed — 2026-10-15 (nav identity broke the 304 contract)

- Every page's navigation shell carried a random `id` attribute (the UI
  library mints one whenever none is set), so two fetches of the same page
  never had identical bytes — the `Cache-Control: no-cache` + ETag
  revalidation contract could never answer 304, and every visit and poll
  silently re-downloaded the full page. The shell now pins a
  deterministic id: pages and poll fragments revalidate with real 304s
  (proven live — three consecutive fetches produced three different
  ETags before the fix, one stable 304 chain after). A regression test
  rejects any future random-id markup.

### Fixed — 2026-10-15 (dashboard honesty when the window is empty)

- An empty (or all-zero) window rendered two zero-height chart boxes and
  an all-flat trend line pretending to be data. Volume/duration bars and
  the trend card now show their honest empty states ("No calls in the
  window yet.") exactly when there is nothing to chart — matching the
  heatmap's existing behavior.

### Changed — 2026-10-15 (dark-mode + polish)

- The dark-theme page background uses the palette's dark surface
  (gray-900) instead of the UI library's indigo-tinted default — the one
  spot the library color leaked past the teal bridge. Search hit-card
  hover aligns with the calls-table hover shade.

### Added — 2026-10-15 (browser-contract tooling + probe fixtures)

- `scripts/verify-style-contract.js`: browser-level checks that the teal
  signal palette actually resolves (`--ds-brand` light/dark triplets,
  heatmap cells bridged off violet), the Go twin pins the embedded CSS.
- `scripts/verify-staleness-live.js`: the staleness banner's live
  lifecycle — appears when the index ages out, survives a forced poll,
  disappears after fresh ingest without a reload.
- `scripts/golden-pretty-diff.sh`: readable word-diff when a golden
  changes, instead of eyeballing wall-of-HTML.
- Screenshot probe grew to 39 pages including the 404 page, a
  recording-audio detail card, print rendering, the empty-index ladder
  (own seeded DB), a `--themes` flag, and an optional `--port` (free OS
  port when omitted).
- `smoke` gained `-age <duration>` (backdate the index for staleness
  states) and `-with-recording <ref>` (attach a recording reference) —
  additive probe fixtures.

### Added

- HTTP-stack adoption pass (audit: docs/planning/2026-10-12_12-47_SUPERB-
  httputil-go-etag-go-appkit-max-adoption.md): conditional GET everywhere —
  go-etag/server middleware answers unchanged pages, htmx poll fragments
  (the 30s dashboard region), and static assets with 304 Not Modified,
  with a Skip policy that keeps the unbounded export/recording streams and
  the ever-changing staleness/freshness bodies out of the buffer;
  Server-Timing instrumentation on every route (per-route named metrics:
  query/stats/fts/count plus the recording proxy's metadata/headers hops)
  via httputil/server_timing; gzip wire compression for HTML/JSON/CSS/JS/
  CSV, ordered outer of ETag so tags hash identity bodies; `GET /version`
  and a Prometheus `/metrics` surface from previously unused appkit
  ServiceConfig knobs; and the stream-contract fix — `WriteTimeout:
  NoTimeout` removes the appkit default 30s request deadline that
  truncated no-row-cap exports and long recording downloads. `web.render`
  now buffers templ pages into one atomic write (a flushed body can never
  be hashed into a 304, and a mid-render failure is a real 500 instead of
  an error appended to a half-written page). `httputil` v1.4.1,
  `httputil/server_timing` v1.0.2, and `go-etag/server` v0.6.1 promoted to
  direct dependencies — all three were already in the module graph via
  go-appkit (same precedent as x/crypto and x/time).
- UI pass over the serve web app using templ-components v1.21.0 to the hilt:
  library `forms.Form` (inline layout) + `display.Button` replace every
  hand-rolled filter/search form control; `layout.Container`/`Split`/`Stack`
  replace hand-rolled page wrappers; `display.Scrollback` renders transcripts
  as a terminal-style voice log; the AI/human mix becomes a `display.PieChart`
  donut with the exact counts in its legend (supersedes the blocked
  SegmentBar swap); `feedback.CircularProgress` gauges the MOS mean;
  StatCards and EmptyStates gain icons; the page header surfaces the existing
  CSV export route; a footer anchors the shell; and the duplicate nested
  `<main>` landmark is fixed.
- Signal-palette theming: `app.css` overrides the library's semantic tokens
  (`--color-tc-primary`/`-hover`/`--color-tc-info`) with teal after the theme
  import, re-skinning every library blue (buttons, badges, charts, focus
  rings, nav active state) from one block.

- DID-interception direction auto-detect (R22, post-release slice of the
  design note): `-ai-did` (comma-separated AI-facing DIDs) on both
  `ingest-telnyx` and `serve`; conversations touching a known AI DID
  index as inbound regardless of the declared `-poll-direction`/
  `-direction`, via a mapper-side `DirectionResolver` — no migration,
  ContentHash and idempotent re-ingest unchanged. Empty flag keeps
  exactly the old behavior. The NixOS module option lands in
  nix-international-telephony.

- Dashboard density + honesty pass (templ-UI Pareto plan T13–T19, plan:
  docs/planning/2026-10-12_13-32_SUPERB-templ-ui-pareto-plan.md):
  weekday×hour call-density heatmap (Mon-first rows in the dashboard
  zone, peak highlight, honest empty state), 7/30-day window presets
  with a `windowDays`-parameterized stats query (bare date filters and
  chart labels follow the window), StatCard change/trend deltas,
  volume/duration bars fixed at a real height (the library's vertical
  BarChart computes 0px bars without a CSS-value Height AND
  `!items-stretch` — filed upstream as templ-components#48 with
  measurements), htmx ErrorHandlingConfig + runtime self-hosting, and
  the density card verified light/dark × desktop/mobile by screenshot.

- Detail-page operator affordances (T20/T21 + T15/T18): CopyButton on
  the native id (PageHeader action) and both phone numbers (facts rows,
  button variant only); the search deep link now carries `?q=` into the
  detail page, marks matching transcript turns (amber), and redirects
  once to the first match's anchor so the click lands at the evidence
  without JS (`?q=…&at=N#turn-N`, at-breaks-the-redirect-loop);
  breadcrumbs on the detail header; the transcript is screen-reader
  reachable (`AriaLabel` — the library defaults the block to
  aria-hidden); keyboard-focusable table rows got the missing
  focus-visible ring (filed upstream as templ-components#49); and the
  page prints facts + transcript cleanly (`@media print` drops nav,
  footer, buttons).

- List/export honesty (T8/T11/T12): the pager is two honest numberless
  buttons (a TotalPages pager would fabricate totals over keyset
  cursors — rationale lives in `pagerRow`), the export endpoints emit
  an explicit `EXPORT_INCOMPLETE` terminal marker when the cursor
  stalls (CSV row / JSONL object, loud on every consumer), and the
  EmptyState offers a "Clear filters" way back. The contract is also
  visible text now: a note under the calls table names the marker, the
  `lens.web.export_cursor_stalled` rejection id, and the no-row-cap
  property (a memory-budget test pins the streaming walk past the old
  cap).

- Golden-HTML regression net (T22): the six operator pages are pinned
  by `.golden` files (`go test ./internal/web/ -run TestGoldenPages`,
  refresh via `-update` then review the diff), a palette gate bans raw
  blue classes in pbx-lens templates (the theme bridge owns blue), and
  a templ-generate sync test keeps codegen honest. Contrast-audited
  palette tokens and the extended blue→teal bridge landed in `app.css`.

- `/api/ops` cache section (R26): go-etag's outcome hooks are wired —
  `etag_generated` / `not_modified` counters with the derived `hit_ratio`
  (how many conditional-capable responses skipped their body) and
  `buffer_overflows`; an overflow also logs an immediate warning naming
  the limit exceeded. The endpoint answers when either counter source is
  wired (the etag middleware runs in every serve composition, poller or
  not); a 503 with an explanation remains for neither.

- Backup restore drill as a named CI lane: the R16 end-to-end proof
  (VACUUM INTO snapshot → integrity verify → restore into a fresh DSN →
  serve-startup FTS rebuild) runs as its own visible Build & Test step on
  every push, matching the e2e server probe's precedent, instead of an
  implicit byproduct of `./...`.

- README operational surfaces notes (version/metrics/health, the
  ETag→304 + Server-Timing + `Cache-Control: no-cache` contract), a
  SECURITY.md entry documenting the `/metrics` unauthenticated opt-in as
  deliberate (perimeter = bind address/firewall; appkit mounts the
  endpoint outside the htpasswd chain), and a gotchas entry: templ's
  streaming flush silently disables go-etag — buffered `render` is what
  makes the 304 contract work.

Nothing else yet.

### Fixed — 2026-10-12 (recording proxy vs MP3-only AI-assistant recordings)

- Every AI-assistant recording 404'd in the player: live-verified
  (2026-10-12, first key-in-hand pass) that Telnyx stores these calls
  MP3-ONLY — all nine `status=completed` recordings on the assistant
  connection carry an empty `download_urls.wav` and a populated
  `download_urls.mp3` — while the proxy hard-required WAV. Fix: WAV stays
  preferred, MP3 is the fallback rendition (the content-type default
  follows the chosen rendition), and a recording with neither answers 404
  naming both. The shape is recorded in docs/telnyx-live-api.md.

- The golden-HTML suite was a time bomb: `page_calls` pinned whichever
  staleness-banner state the wall clock produced (the seed sits at a
  fixed 09:00 UTC, so the golden silently captured the under-6h state
  and every run after 17:00 local failed), and `page_dashboard` pinned
  bare ISO bucket labels that shift at midnight Europe/Berlin. The
  scrubber now reduces both runtime-coupled structures to placeholders —
  the staleness region's attributes stay pinned (banner behavior keeps
  its dedicated handler tests) and fixed-seed `datetime="…T…"` row
  attributes remain exact — goldens regenerated and verified stable.

### Added — 2026-10-12 (deterministic browser revalidation)

- `Cache-Control: no-cache` on every HTML response (pages and htmx
  fragments): browsers and polls must revalidate — ETag → 304 when
  unchanged — instead of silently serving stale HTML. The header survives
  into 304 responses and is asserted end-to-end on the dashboard fragment
  (200 + 304) and the ETag-less staleness fragment.

### Added — 2026-10-15 (security hardening bundle)

- `robots.txt` answers the one honest policy for a private call index
  (`User-agent: * / Disallow: /`), intentionally public and cached a day —
  a robots file only bites if fetchable. Every page also carries a
  `noindex` meta (the UI library's `SEOMeta{NoIndex}`), and the route sits
  outside the auth chain by design.
- The recording stream now rate-limits per client address: over the limit
  answers `429` with `Retry-After` instead of multiplying upstream Telnyx
  fetches; loopback is exempt (tests and local use throttle nothing).
- `CSP_REPORT_ONLY=1` switches the Content-Security-Policy to its
  report-only twin for a dry run: the page loads exactly as today while
  the console surfaces what enforcement would block — the collect step of
  the enforce/adjust decision (the enforcement flip itself is still
  pending that sweep).

### Fixed — 2026-10-15 (version footer)

- The footer rendered `vv0.1.1-0.2026…`: Go's `debug.Main.Version` is
  already semver, and the footer prepended a second `v` on top. The
  version formatting is extracted and regression-tested (`v`-prefixed,
  bare-semver, and devel inputs all render exactly once).

### Changed — 2026-10-15 (dark favicon + print)

- `favicon.svg` adapts to dark browser chrome via an in-SVG
  `prefers-color-scheme` media query (teal-700 → teal-400) — no second
  file, no layout prop.
- Printed transcript turns no longer split across page boundaries
  (`break-inside: avoid` on each log line; the print block already drops
  nav/footer/buttons).

### Added — 2026-10-15 (browser-contract lane)

- `scripts/verify-all.sh`: one command runs all four browser probe suites
  (UI contracts, style contracts, staleness-live, the 41-PNG sweep) —
  sequential on free ports, first failure fails the lane, extra arguments
  pass through to the screenshot probe.
- The screenshot probe gained `--only <surfaces>` (capture a subset,
  unknown names rejected before the slow bootstrap) and an alert-only
  pixel diff (`--diff`/`--save-baseline`/`--tolerance`, pixelmatch via
  scratch-installed deps): visual regressions are REPORTED, never
  red — the dashboard's live footer stamp (~0.04% of pixels) is absorbed
  by the default tolerance by design.

## [v0.1.0] — 2026-10-12

### Fixed — 2026-10-12 (cdr ingest vs vanilla mod_cdr_csv output)

- CDR polling recorded NOTHING from a real FreeSWITCH: vanilla
  mod_cdr_csv writes every Master.csv line with `,"", ""` in the
  compiled-in default template's tail (comma-space before the final
  empty fields = a bare quote in a non-quoted field), and Go's strict
  CSV rejected the whole line. The nix-international-telephony
  telephony-lens VM suite caught it live (`poll cdr:
  [corruption:cdr.parse.read]`, 0 offered/0 recorded at a 2s cadence
  against a real placed call). Fix: `csv.Reader.LazyQuotes = true`
  (normalize already strips the residue) + a regression arm pinning
  the exact vanilla bytes. Author note: the code landed via auto-commit
  `e1d9ff7` in a heuristic bundle; this entry is the attribution.

### Added — 2026-10-12 (SUPERB pareto plan execution)

- Dashboard day buckets follow Europe/Berlin (R2, owner decision): the
  "Calls today" window, `todayHref`, and bare YYYY-MM-DD filters all resolve
  in the operator's zone; `time/tzdata` is embedded so the hermetic binary
  resolves the zone anywhere. UTC remains available (zone is injectable,
  tested).
- Partial phone-number search (R6): a digits shadow table
  (`lens_digits`) rebuilt alongside FTS answers any 4+ digit run — prefix,
  suffix, or mid-number — merged after the ranked FTS hits; the search-page
  hint now tells the truth about it.
- Streaming exports (R7): `/api/export.jsonl` and `/api/export.csv` walk the
  index keyset-paginated (5k-row pages, bounded memory, no row cap); views
  sharing a cursor timestamp dedupe across pages and a nanosecond-collision
  stall fails loudly instead of truncating.
- PII export honesty (R3): both export endpoints answer with an explicit
  `X-PII-Mode` header, and FEATURES.md gains the field-coverage matrix with
  the raw-free-text disclosure; raw-at-rest is now explicit owner doctrine
  (R21) — the export gate is the PII boundary.
- Dashboard completion (R8, owner deliver): date-range picker (M20.1) and a
  call-duration histogram (M20.2) join the stat cards; week-over-week delta
  and a Sparkline aria label round the cards out (m27.6).
- Live auto-refresh (R9, owner decision): dashboard cards poll
  `/frag/dashboard` every 30 s and the staleness banner re-checks
  `/frag/staleness` every 60 s via htmx `PolledRegion`; the htmx 2.0.10
  runtime is self-hosted (no CDN) and the server-rendered page is the
  no-JS fallback.
- Auth hardening (R11/R13): SIGHUP reloads the htpasswd file without a
  restart (a broken file keeps the running table and logs why), and failed
  logins throttle per client address (bounded FIFO bucket map, shared
  fallback for hostless peers; X-Forwarded-For deliberately not honored).
- Ops visibility (R14/R15): poller and serve logs are structured key-value
  records through `log/slog`, and `GET /api/ops` serves per-source
  offered/recorded/corrected/rejected counters from the latest pass
  (auth-gated; 503 with an explanation when no poller runs).
- Restore drill (R16): snapshot → verify → restore → rows + FTS hits match
  the source, as a checked-in test (`internal/backup/restore_drill_test.go`).
- `purge -json` (m27.8): machine-readable reports for both `-dry-run` and an
  applied purge; the purge logic is extracted behind a testable core.
- A named end-to-end server probe step in CI (R17): the checked-in
  `TestLiveServerEndToEnd` loopback lane is now a visible job step.
- DID-interception direction design note (R22,
  `docs/did-direction-design.md`) with the bounded implementation row in
  TODO_LIST.md.

### Changed — 2026-10-12 (SUPERB pareto plan execution)

- Naming closure (R18): cdr's `shape` local reads `byColumn`; the
  `CostEnrichment` tuple became `CostOf(rec) (RecordCost, bool)`; the
  `Client.Recording` fetcher is `FetchRecording`. `ReasonRetention` is a
  recorded permanent non-fix (stable audit label).
- The dashboard aggregates stream instead of slicing; `ExportCalls` remains
  as the collecting convenience (tests, retention).
- Drafts ledger (R12, owner: annotate-and-keep): six filed drafts carry
  FILED banners naming their upstream issues; go-atomic-write stays the one
  live draft.

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
- Shared `newTestServerWithSearch` test helper (search.Open + Rebuild over
  the seeded views); the one-flow test and the live-server e2e test both
  collapsed onto it (was two 15-line copies of the index wiring).
- Recording-proxy upstream timeout knob: `web.DefaultUpstreamTimeout` (30 s)
  bounds both upstream hops of the recording route — the Telnyx metadata
  fetch runs under a hard deadline (504 on expiry) and the WAV stream must
  deliver response headers within the budget (502), while the audio body
  itself streams unbounded. Deployment-tunable via `serve
  -recording-timeout` and `web.RecordingOptions.UpstreamTimeout`; the
  previously unbounded `http.DefaultClient` ride is gone. Production value
  still awaits real traffic data.
- File-size split of the three >350-line files the structure linter kept
  flagging: `internal/calls/domain.go` (570) into domain/events/commands/
  decider, `cmd/pbx-lens/main.go` (677) into main/ingest/serve/ops, and
  `internal/ingest/telnyx/telnyx.go` (379) into client + `dto.go` — zero
  API or behavior change; every non-test file is now well under the limit.
- Internal naming pass (2026-10-11): 9 renames for verb consistency and
  split-brain closure (`ListMessages`/`ListDetailRecords`/
  `ListConversationInsights`, `agent.MapTranscript`, `stats.AvgMOS`,
  `cli.RebuildOpenIndex`, `EnrichmentSeed`, `enrichConversation`,
  `streamLabel`, `hasStart`) — internal Go identifiers only; no JSON tags,
  wire fields, CLI flags, or DB columns changed.
- Documentation pass (2026-10-12 docs-health AUDIT): the 21 superseded
  status reports (2026-10-10 → 2026-10-12) annotated inline (every open item
  carries a done/routed/won't verdict) and archived to `docs/status/archived/`
  with a manifest; ROADMAP rebuilt (shipped ideas graduated to FEATURES,
  owner questions collected); TODO_LIST fourth pass (prod re-ingest, v0.1.0,
  and sibling-module values added as owner-blocked rows); FEATURES evidence
  refreshed against the post-split file layout; the `mosAvgLabel` →
  `avgMOSLabel` split-brain remnant renamed.

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
- Two golangci-lint residuals (err113 dynamic test error replaced by the
  static `errClientVanished`; a 132-column flag help line wrapped under the
  120 limit).
- AGENTS.md templ version corrected to the landed v0.3.1070 (manual-bump-only
  dependency since dependabot mangles this go.mod).
- `apps.update-vendor-hash` in flake.nix carries `meta.description` (and
  uses `lib.getExe` instead of `pkgs.lib.getExe`), so `nix flake check`
  stops warning on every gate run.
- Upstream loop closed for the 2026-10-11/12 findings: all five D6 filings
  re-verified OPEN (go-cqrs-lite#62/#63/#64, go-nix-helpers#7,
  treefmt-nix#545); the cqrs-lint C009 bare-`must` gap source-verified as
  covered by go-cqrs-lite#64; the load-starved compile-gate misattribution
  in BuildFlow (tool.timeout kill reported as "dependency bump introduced
  compile errors" + silent go.mod/go.sum revert) filed as BuildFlow#45.
- flake.nix cqrs-lint pin hashes repaired (an earlier session had pasted
  the pbx-lens module vendor hash into both pins, leaving the devShell
  unbuildable on any cold cache) and a stranded `vendorHash.nix` AAAA…
  placeholder restored via `nix run .#update-vendor-hash` (2026-10-11
  naming-review session).
