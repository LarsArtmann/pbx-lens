# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
v0.1.0 is the first tagged release (2026-10-12); new work lands under Unreleased.

## [Unreleased]

### Fixed — 2026-10-25 (gate truth: the first calm-window FULL-gate receipt)

- **Four erraudit error-severity findings fixed** — the scenario-test
  rejection sentinels had concrete type `*errorfamily.Error`; declared as
  the `error` interface (erraudit's prescribed fix). This was the only
  thing between the repo and its first green FULL-gate receipt.
- **First calm-window FULL-gate receipt captured**: `poll-gate.sh` verified
  START LOAD 15m=6.06 (policy < 40), `BUILDFLOW_NO_RESULT_CACHE=1
  buildflow --build-mode full` → 60 success / 0 failed, exit 0, frozen
  tree. The three starved attempts of the busy morning are superseded.
- **Quiet-box bench re-measure** (docs/http-stack.md carries both
  generations): ingest parse 16–26 → **8.7 µs/row**, fresh-stream dispatch
  10–12 → **6.1 µs/row**; poll 304 ≈ 88 µs / full-200 ≈ 94 µs (unchanged
  within noise). The load-affected numbers were ceilings, as suspected.
- **TODO_LIST repaired**: the blank line splitting the Blocked table is
  gone, two mis-filed 🔴 rows moved to Unblocked (versions.json,
  Server-Timing), two done rows deleted (bench re-measure, gate receipt),
  and three stale rows deleted with evidence — probe-residue bundle
  (`assertFirstTryRevalidation` + `--baseline` parse order + seed-clone
  note all landed 2026-10-25), prettier-devShell (the devShell ships
  prettier; the bare-`buildflow` PATH residue belongs to crush-config, not
  this repo), and the go-appkit#26 half of the fold-in row (testkit
  adopted at all three stack-test sites 2026-10-25; go-etag#2 + quarterly
  re-verify remain).

### Added — 2026-10-25 (go-cqrs-lite wave: systemscenario testing depth, catalog read surface)

- **The eight go-cqrs-lite modules rode the 2026-10-24 disk-truth tail
  wave** (catalog v4.10.0, system v4.15.4, metaengine v4.20.0,
  sqliteengine v4.6.0, event v4.15.2, id v4.7.3, schema v4.7.1,
  projectionadapter v4.6.2) — unlocking the catalog architecture exporter
  and the sticky-stream-key + typed-decoder fixes. Audit correction: the
  layout-stamp drift WARN never covered call_views (the collection
  registers no layout plan, so there is nothing to stamp —
  docs/gotchas.md); the `WithLayoutStampsDisabled` opt-out is not needed.
- **The systemscenario harness replaced the hand-rolled test rig.**
  `app.DomainConfig()`/`app.DeploymentConfig(dsn)` are exported and feed
  both production and tests, so scenario boots exercise the exact
  production wiring; the 20 ms-poll lifecycle code is deleted. New depth:
  idempotent re-ingest and retire-tombstone system specs, an OCC
  stale-writer spec, a 2 ms-latency chaos race over sqlite, a rapid
  property oracle (projected row == pure fold, journals strictly
  ordered), and a payload-hash golden for the record scenario.
- **Retiring a call is now terminal for enrichment and correction**: enriching
  a retired call rejected nothing before — a late enrichment re-created the
  tombstoned view row and defeated the purge. The decide rejects with
  `call.enrich.after_retired`, and a grown snapshot on a retired stream now
  rejects symmetrically with `call.corrected.after_retired` (an unchanged
  re-poll stays the idempotent no-op), so a full re-ingest wave cannot
  resurrect purged calls whose sources grew after the purge. The scenario/v4
  decide matrix pins all fourteen record/enrich/retire paths.
- **The federation catalog declares the read surface**: `call.search`,
  `call.dashboard`, `call.detail` query messages with ids mirroring the
  web read functions; every domain field carries `doc:` descriptions into
  the exported schemas; the service card links the repo; and
  `catalog -architecture` writes the self-contained architecture
  reference HTML. `-skip-bootstrap` and always-on bootstrap skipping for
  `-plain` lint exports round out the CLI.
- **Ingest per-row receipts**: parse ≈16-26 µs/row, fresh-stream dispatch
  ≈10-12 µs/row (docs/http-stack.md, concurrent-load caveat).

### Added — 2026-10-25 (the analytics surface: spend, segments, quality — on true counts)

- **One call is one row on the list and the dashboard.** A CDR and an agent
  transcript describe the same call under one uuid, so an AI call used to
  appear twice and the dashboard counted its volume twice. The operator-facing
  surfaces now collapse duplicates onto the REPRESENTATIVE row (CDR first,
  transcript only as fallback) and let the representative absorb the facts only
  its sibling recorded (the opening line). The export deliberately stays
  per-source: a forensic dump carries every row under its own identity.
  `internal/web/representative.go`; the list walk keeps page size honest by
  scanning ahead and fails loudly on an unwalkable timestamp group
  (`list_cursor_stalled`, the export's documented failure mode applied to the
  page).
- **CDR direction derivation** (`internal/ingest/cdr/cdr.go`): the dialplan
  transfers every inbound DID into `default`, so the a-leg's `context` column
  said nothing and inbound AI calls indexed as OUTBOUND (live example
  4084757593 → 9100). Precedence is now: a template-declared `call_direction`
  column (read by VALUE, so the deployment can add columns without re-ordering
  the MOS field) → `context == "public"` → the transfer heuristic (external
  caller ringing an internal extension = inbound). Re-ingesting the CDR file
  heals existing rows through one `call.corrected` per stream.
- **Dashboard "Spend" card** (north-star slice 1): window spend, cost per
  billed minute, week-over-week delta, and a daily spend sparkline, built on
  `Enrichment.CostMicros`. Every figure names its denominator ("N of M calls
  priced"); no priced call renders an honest empty state, and a mixed-currency
  window says "mixed" instead of summing units it cannot add.
- **"Who calls what" segmentation** (slice 2): the top five of OUR numbers and
  the top five counterparties by volume, each with calls, AI share, average
  duration and spend — and each row drills through into the filtered call list
  via a new `?number=` filter that matches either side of a call.
- **"How calls end" quality rollup** (slice 3): hangup-cause distribution plus
  standing rates for `no user turn`, `self-dial` and the recording tail, each
  over its OWN population (transcripts, calls with both parties, timed
  transcripts) so a quiet window cannot fake a 0% rate.
- **Four non-browser gates**, each with a `--self-test` that seeds the failure
  it exists to catch: `scripts/check-js-lint.sh` (oxlint `no-undef` over the
  probe scripts — the `bin is not defined` class that shipped 2026-10-23),
  `scripts/check-docs-counts.sh` (corpus counts re-derived from the tree),
  `scripts/check-doc-links.sh` (relative links across living docs), and
  `scripts/poll-gate.sh` (load poll that `exec`s the full gate so the
  receipt's START LOAD is the load the loop verified). The first three also
  run in the pre-commit hook; all four run at the head of `verify-all.sh`.

### Changed — 2026-10-25 (templ-components v1.24.0, one formatting dialect, lint at zero)

- **templ-components bumped v1.23.0 → v1.24.0** through the six-step
  watchlist: ARIA fixes land for the two components this repo renders
  (Heatmap, CopyButton gains a `data-tc-copy-status` role="status"
  announcement), scrollable tables emit `role="region"`, and the detail
  page's Split column renders `<div>` instead of `<aside>`. Goldens
  reviewed; the class corpus regenerates machine-independently now (the
  script resolves the module cache through `go env`/`go list -m` after the
  v1.21.0-era hardcoded paths silently scanned nothing and shipped an
  empty corpus) and refuses to write one.
- **One duration dialect.** `utils/format` is adopted wholesale: the four
  hand-rolled twins (`clockDuration`, `compactDuration`, `stringOrDash`,
  `percent`) are deleted (−46 lines) in favor of the library's
  `ClockDuration`/`CompactDuration`/`StringOrDash`/`Percent` — the
  staleness banner copy changes accordingly ("1m00s" → "1m"), NaN/Inf
  percentages degrade to an em dash instead of rendering "NaN%".
  `mosGaugePercent` stays (MOS-to-gauge mapping is domain math).
- **The ops page says "never"** for a poller that has not completed a
  pass, instead of rendering the zero time as "Jan 1, 0001"
  (RelativeTime has no empty escape upstream — commented on
  templ-components#52 with the use case; the local guard retires when an
  EmptyText prop ships).
- **Lint findings at zero:** the four standing complexity findings are
  fixed by extraction, not suppressed — `ParseWithOptions` (row→template
  classification into `mapRow`), the dashboard accumulator (`viewMOS`/
  `viewCost`/`viewSegments`), `ListCalls` (the keyset walk's row loop and
  newer-page probe into `fillListPage`/`newerPageCursor`), and the
  catalog test's message assertions. Behavior pinned by the existing
  suites; `golangci-lint run ./...` exits clean.

### Changed — 2026-10-25 (honesty pass: docs corrected, findings triaged, gate de-noised)

- **Static assets must revalidate on the FIRST replay.** `assertConditionalGET`
  tolerates a re-tag because a rendered page embeds a timestamp; a static asset
  is byte-identical, so `/static/htmx.min.js` and `/app.css` now go through
  `assertFirstTryRevalidation` — a 200 on the first conditional GET is a
  failure, not a retry.
- **`verify-all.sh` parses its own flags before running anything**, so a
  typo'd `--baseline` fails in a second instead of after four browser suites.
- **`docs/backup-rotation-opinion.md` corrected:** the app publishes ONE
  fixed-path snapshot (`index.db`, atomically replaced and verified), so
  "rotate the app's staging files" was a no-op; versioned retention lives
  entirely with the host backup tool.
- **`docs/did-direction-design.md` closed and archived:** both halves shipped
  (`-ai-did` 2026-10-12, CDR derivation 2026-10-25); its "the CDR side stays
  untouched" claim was disproven live and is annotated inline.
- **`docs/process-prep-2026-10-15.md`:** the scanner ceiling is 64 KiB → 1 MiB
  (`scannerMaxLine`), and the ContentHash-v2 runbook names `refresh` for the
  Telnyx leg (local sources re-parse for free).
- **`docs/followons-2026-10-15.md`:** F56's cost-panel sketch is marked
  EXECUTED — the Detail-Records enrichment, not the MOS window, supplied the
  data.
- **Findings triage closed in writing:** `docs/buildflow-dev-gate.md` now
  dispositions the detect-only lanes (art-dupl 101 / branching-flow 116 /
  file-size 16) and names all ten "tools unavailable" entries;
  `docs/cqrs-lint-policy.md` gained F021/F026/F031 plus the time-typed-field
  entries. Two real fixes fell out: a named-return lint finding in the CDR
  mapper, and AGENTS.md's stale go-cqrs-lite version reference (which the
  linter reads as drift).
- **`prettier` and `oxlint` joined the devShell** — the prettier-format step
  no longer falls back to `nix run nixpkgs#prettier` (~2.7 s + a warning per
  run), and the JS gate finds its tool in the shell it documents.

### Added — 2026-10-24 (catalog infra truth: containers, channel, read/write edges)

- **The pbx.artmann.tech lineage renders in the federated catalog**:
  FreeSWITCH and Telnyx are declared as external-system containers
  (`AccessMode read-only`; FreeSWITCH `Authoritative` — it is the system
  of record for which calls happened), the service wires
  `readsFrom(freeswitch, telnyx)` + `writesTo(call-index)` so the
  visualiser draws the data-source edges, and the receiver-webhook JSONL
  feed is the first channel (`pbx-receiver-webhook`, http). No infra
  repo exports anything — the lens describes its sources. Tests cover
  the three containers, the channel, and both edge lists; export diff
  verified (the `source_events` schema drift in the diff was
  pre-existing domain drift against a stale baseline). Repo hygiene in
  the same pass: `/work/` gitignored and the daemon-committed binary +
  lint trees untracked (kills the standing go-structure-linter
  "binary tracked in git" error).

### Changed — 2026-10-24 (dedup to zero, pass 2: hidden production clones extracted, gate recalibrated to zero groups)

- **Audited the suppressed bucket and extracted four real production clone
  groups the default report (and therefore the gate) could not see.** In
  `internal/web`: the `serveStyles`/`serveHtmx`/`serveFavicon` trio collapsed
  into a `staticAsset` table (path/contentType/missingMsg) with one `serve`
  method; `list`+`stalenessFragment` now share `queryListStaleness` (typed
  `listStaleness` result, so the two trailing bools cannot transpose);
  `dashboard`+`dashboardFragment` share `queryDashboardStats`; the export
  format preamble moved into `beginExportResponse(w, r, contentType,
  disposition)`. Behavior identical — error strings, timing labels, header
  order preserved; full test suite green.
- **Duplication gate recalibrated from "23 pinned groups" to "zero groups".**
  `--min-lines 2` drops the 1-line signature/type-expression matches
  (parameter lists like `(v string)` — shape coincidence, not duplication),
  so `scripts/art-dupl-baseline.json` now records an EMPTY accepted set: any
  new multi-line clone group fails CI. Both directions re-proven (clean tree
  exit 0; seeded 12-statement cross-package clone exit 2). Remaining hidden
  groups re-audited: test scaffolding only (self-contained tests by design).
  docs/art-dupl-baseline.md rewritten for the zero-group contract, including
  the suppressed-bucket audit duty after pin bumps.

### Changed — 2026-10-24 (dedup to zero: the accepted clone verdict is now a CI gate, not prose)

- **art-dupl lane upgraded from warning-only text diff to a hash baseline
  gate.** `scripts/art-dupl-baseline.json` (23 groups, generated at t=4 in
  directory mode so `.templ` source is scanned) replaces
  `scripts/art-dupl-baseline.txt`; CI runs `art-dupl check` and FAILS on any
  clone group whose hash is not pinned — the 2026-10-24 audit's "a body-level
  clone appearing" trigger, enforced instead of remembered. Acceptance table
  and refresh procedure: docs/art-dupl-baseline.md; pin moved to v0.9.0 to
  match the baseline generator.
- **Verified zero harmful clones at both thresholds.** At the t=4 fleet gate
  the only shown group is the `beginExportWithPII` call pair (helper already
  extracted, per-format header values differ); at t=5 type-aware all 20
  actionable groups are signature/expression matches between .templ helpers
  with distinct business rules. Gate proven both directions with live
  probes: clean tree exits 0, a seeded cross-package clone exits red.

### Added — 2026-10-23 (additive fields get a mechanism: schema version + `refresh` backfill)

- **`ObservationSchemaVersion` / `EnrichmentSchemaVersion` ride the hash
  pre-image.** A field that is persisted but deliberately unhashed
  (`SourceEvents`) reached history only by luck — the decider answers an
  equal `ContentHash` with a no-op. Bumping the version forces exactly one
  correction per re-ingested stream; v1 → v2 is that deliberate bump.
- **`ContentHash` is now length-prefixed and named.** The retired
  newline-joined pre-image could collide when a byte moved across a field
  boundary (`"ab"+"c"` vs `"a"+"bc"`), so two different observations could
  claim one identity. Pinned by `TestContentHashSeparatesFieldBoundaries`;
  `TestContentHashSchemaVersionMovesTheHash` pins that a bump really moves it.
- **`pbx-lens refresh`** classifies every fetched stream against the read
  model WITHOUT writing (new / stale / additive-only / unchanged /
  unbuildable) and, with `-apply`, re-ingests and rebuilds the search index.
  `additive-only` is the alarm for "hashes match but the stored row is
  missing source data", the class that otherwise never heals. Unbounded
  look-back by default; tested end to end against a stub API.
- **Measured correction wave (live, 2026-10-23, the whole account history):**
  **25 of 25** Telnyx conversations carry wrong turn times under
  `created_at` — every one of them rewrites once — plus 6 zero-user-turn
  conversations and 37 source markers to persist, over 247 spoken turns. A
  `-tags live` test (`TestCorrectionWave`) re-runs the measurement before any
  future deploy: read-only, no index opened.
- **Signal thresholds documented in one place**
  (`docs/DOMAIN_LANGUAGE.md` § Signal thresholds): the 20 s recording-tail
  bar, the `no user turn` / `self-dial` / `source markers` badge conditions,
  and the bench's `expectsUserTurns` tri-state — so nobody has to read `templ`
  or the scorer to learn what a bar means.
### Changed — 2026-10-23 (gate triage: probe bootstrap deduplicated, tooling residuals resolved)

- **Probe bootstrap deduplicated**: the five single-serve browser probes
  (verify-ui-contract, verify-style-contract, verify-staleness-live,
  verify-ops-verdict, csp-sweep) boot through a shared
  `lib/ui-probe.js launchProbe()` (codegen → puppeteer → binary → seed →
  serve → 1280×900 page; `seedOpts`/`serveArgs`/`serveEnv`/`resetDb`
  variants replace each probe's private fork of the sequence), and the
  per-script `arg()` parsers delegate to `probe.arg`. Clears all three
  jscpd clone pairs; screenshot-probe keeps its bespoke multi-phase flow.
  All lanes verified green (verify-all).
- **vendorHash advisory dispositioned without a repair**: `nix build` is
  green with the landed hash — the 2026-10-23 go.mod change was the go
  DIRECTIVE only (tidy pinning go-appkit's 1.27.2 floor), so the
  doctor's FOD-hash-freshness mtime heuristic misfired; no
  update-vendor-hash run needed (TODO row closed).
- **DevShell gains vulnix** (the BuildFlow fallback ran it via
  `nix run nixpkgs#vulnix` without project deps, and a stale `./result`
  from a dirty tree crashed it; `nix build .# -o result` refreshes the
  scan target). The stale BuildFlow binary (ed9ce48) was rebuilt and
  reinstalled (3509b30) per its own preflight remediation, clearing the
  toolchain-skew failures in go-fix/license-check. The govulncheck
  GO-2026-6809 report is a toolchain-version artifact (fixed in go1.27.3;
  nix go_1_27 and CI setup-go already ship it). War stories recorded in
  docs/gotchas.md.

### Changed — 2026-10-23 (docs-health audit: corpus closed, living docs re-verified)

- **The 2026-10-18 status live set is closed**: all five reports annotated
  inline (394 verdicts — executed, routed to a specific TODO row, or NOT-DO
  with a reason; every 04-25 §f queue row and every 06-55 §f row now carries
  its own disposition) and archived with a per-file manifest. The executed
  01-01 Pareto plan moved to `docs/planning/archived/` with an EXECUTED
  header and per-row verdicts on all 24 task rows, 96 fine-breakdown rows,
  and the §6 landing-place ledger. check-rows ran COMPLETE on every file
  before the move.
- **Harvested into the live backlog**: TODO_LIST rebuilt to verified-open
  rows only (new: stamp-churn doctrine, buildflow filing decision,
  patch-vs-minor release decision, poll-exec gate wrapper, probe residue,
  prettier devShell, plus the three analytics slices, the probe-sweep re-run
  and the detect-only findings triage harvested from the 2026-10-23 gate
  report); ROADMAP gained theme 0, the flake-record convention, the
  test-complexity policy, and client-side stamp rendering. The vendor-hash
  row was opened and closed the same day (the advisory was a heuristic
  misfire; `nix build` is green). Upstream state re-verified with `gh`:
  go-appkit#26 CLOSED, go-etag#2 / go-nix-helpers#7 / treefmt-nix#545 / templ-components
  #48/#49/#50/#51/#55 still OPEN.
- **The projection-key gotcha was corrected, not just re-worded**: the
  2026-10-11 "keys carry the `StreamMarker:` prefix" entry now states the
  post-v4.6.0 truth (bare `calls.RecordKey`, `LegacyRecordKey` fallback via
  `LookupView`) — the old text described exactly the read path the 2026-10-23
  repair had to remove.
- **Living-doc drift fixed against code**: AGENTS.md now says go 1.27.2 (the
  go.mod floor), documents bare record keys, the receipt-commit discipline
  (add+commit in one call; amend only unpushed; a guard proof asserts the
  guard's message), and 59 archived reports — at 219/219 budget. README
  gained the `catalog` usage section, `purge`/`catalog` in the CLI layout,
  and the missing ingest-telnyx flags (`-ai-did`, `-max-age`). CONTRIBUTING
  lost the stale "nix build is BROKEN" claim and gained the hooksPath +
  commit-message norms. FEATURES lists the serve flags and the now-seven-job
  CI lane. The gate doc re-verified the tooling counts (65 passed /
  3 warnings / 30 missing-tool failures, all dispositioned), the HTTP-stack
  doc carries the recording-timing test pointer, and the templ-components
  bump watchlist gained the art-dupl baseline step.
- **The product framing is now the analytics platform**: README leads with
  the operator questions the index answers (who/when/how long/AI or human,
  cost, quality, transcript) instead of "call-history index"; FEATURES gained an honest
  Insights & analytics maturity table (13 capabilities, including the gaps:
  cost analytics, segmentation, quality rollup, saved views, anomaly
  detection are PLANNED); ROADMAP gained theme 0 — the north star plus
  cost/segmentation/quality/anomaly/delivery raw ideas; TODO_LIST carries
  the first three bounded slices (dashboard cost card, top-DID/counterpart
  segmentation, quality rollup), all read-only over data already ingested.
- **Extracted, not appended** (the AGENTS 219-line rule): the transcript-truth
  and read-time call-join invariants moved from AGENTS.md to
  `docs/DOMAIN_LANGUAGE.md` (both sections carry the full rules; AGENTS keeps
  one-line pointers), and `work/**` joined the dprint excludes so generated
  catalog output stops reporting as unformatted.

### Added — 2026-10-19 (event/command catalog export)

- **`catalog` subcommand**: `pbx-lens catalog -o <dir> [-plain]` renders the
  calls aggregate's command/event catalog as an EventCatalog project tree —
  no database, no network. Message schemas are derived from the Go structs by
  reflection (`internal/calls.BuildCatalog`), so the catalog tracks the
  domain instead of a hand-maintained copy; the service, domain, data store,
  and the call-lifecycle flow are declared as first-class entries.
- **`.eventcatalog-hub.json`** wires the repo into the architecture-catalog
  hub: the export entry renders the tree, the lint entry renders the
  `-plain` variant for the hub's frontmatter-ID governance checks.
- **`docs/nolint-ledger.md`** gained the catalog command's single
  suppression (`erraudit` on the CLI status-line write, a non-actionable
  output stream), keeping the ledger-completeness check green.

### Fixed — 2026-10-23 (a call's detail page now shows its transcript and its recording)

- **Cross-source detail join**: `CallDetailPage` (internal/web/queries.go)
  completes the requested source row from the sibling row that observed the
  same call. FreeSWITCH's call uuid is shared by the CDR and the voice
  agent's transcript, so a CDR page now renders the transcript and a
  transcript page now renders the parties/duration even when the ingest-time
  join missed (transcript written before the CDR landed). The requested row
  stays the authority: its own facts, identity, ContentHash, enrichment and
  recording reference are never overwritten, and the joined source is named
  on the page ("Also indexed as"). Telnyx and webhook rows are NOT joined —
  their native ids live in different key spaces.
- **Local FreeSWITCH recordings resolve again**: the dialplan owns the WAV
  stem (`{uuid}_{destination}`), and two of its namings matched neither
  candidate the proxy tried: the agent route writes `{uuid}_ai_agent.wav`
  and outbound PSTN writes `{uuid}_pstn.wav`. The resolver keeps the cheap
  stat ladder and adds a `{uuid}_*.wav` sweep, so a new dialplan destination
  cannot silently orphan audio again.
- **The recording card keys on what exists**: a FreeSWITCH row indexes no
  `RecordingRef` at all, so the card used to claim "no recording" for a call
  whose WAV was on disk (and whose recording route served it). It now
  renders the player whenever the server resolved a local file (naming it
  under the player) and explains honestly when no file exists.
- Proofs: `detail_join_test.go` pins the join in both directions, the
  `{uuid}_ai_agent` / `{uuid}_pstn` stems, the unknown-destination sweep, the
  honest no-file state, and the merge precedence (primary facts, enrichment,
  identity); `go test ./...` and `go test -race ./internal/web` green,
  `nix build .#pbx-lens` green (sandboxed checkPhase included),
  golangci-lint 0 findings, nolint ledger green. Also proven against the real
  binary with a fixture corpus: the page renders the turns, the player, the
  resolved filename and "Also indexed as", and both recording routes answer
  200 with the file bytes.

### Fixed — 2026-10-23 (read-model row keys follow the projection adapter's bare form)

- **Every read-model row key now flows through `internal/calls.RecordKey`**
  (the bare `StreamID.Get()` value). go-cqrs-lite's `projectionadapter`
  v4.6.0 flipped `EventWithID.ID` from the marker-prefixed `String()` form
  to the bare value, so the index's own readers, ingest supersede check,
  ops inspector, enrichment seeds and tests were all asking for a key no
  fold writes any more: detail pages 404'd and every projection lookup
  missed. `projection.go` needed no change (the folds always used the
  adapter's ID); the drift was entirely on the read side.
- **`calls.LookupView` keeps historical rows readable**: it tries the
  canonical bare key, then the marker-prefixed key that rows written under
  v4.5.x still carry. Retiring a legacy row still cannot remove it (the
  library's remove fold derives the bare key from the record) — the
  one-time read-model rebuild in TODO_LIST closes that.
- Proofs: `go test ./...` green, the lint/ledger step green, and a nix
  build of `packages.pbx-lens` (sandboxed `checkPhase` included).

### Added — 2026-10-18 (pareto plan execution: gate hardening, probes, rehearsals)

- **statix no-op tripwire** in `scripts/verify-all.sh`: the lane fails
  loudly when statix prints its "config error" fail-open signature (exit 0,
  lints nothing — the five-day October outage class); lint verdicts stay
  BuildFlow's. Proven both ways with a fake fail-open binary (red) and real
  statix (green).
- **CI nix lane** (`nix-lint` job): statix (with the no-op tripwire) +
  `deadnix --fail` + `nix fmt --ci`, every tool resolved through the
  flake's locked nixpkgs rev. Each lane proven red on a seeded failure in a
  throwaway clone before landing; green on remote CI.
- **`ingest-telnyx -dry-run` parity** with ingest-cdr: report-only fetch
  (conversations fetched / would-record / age-refused / fetch-rejected,
  nothing written, no index), plus an `-api-base` override for stub-server
  rehearsals. Pinned by a stub-API test that proves no index is created.
- **`serve -wired-ops` dev fixture + fourth lane probe**
  (`verify-ops-verdict.js`): a canned STALE poller snapshot exercises the
  /ops stale-verdict alert and the /api/ops payload end-to-end — the
  poller-wired composition the probes could never reach before.
- **Recording-proxy timing-hop asserts** through the full served chain:
  metadata/headers appear in Server-Timing within the upstream budget, no
  body metric, and a stalling upstream fails fast at 502 without
  stretching the header budget.
- **FTS ranking corpus test**: a 106-view synthetic corpus pins bm25
  ordering (term frequency; both-terms beats single-term heavyweight) and
  the digit-shadow corners (runs inside longer numbers, overlapping
  prefixes, FTS+digit dedupe).
- **art-dupl verdict baseline** (`scripts/art-dupl-baseline.txt`) wired
  into the CI duplication job: a diff against the accepted verdict warns
  with the refresh procedure (34 groups / 2 shown at pin 0.9.0); a seeded
  clone proves a new clone stands out. `verify-all` gains a
  baseline-freshness warning (markup newer than the pixel baseline).
- **`verify-all --baseline <dir>`** — the LENS_DIFF_BASELINE pixel-diff
  hook as a lane flag; the print probe now asserts computed
  `break-inside:avoid` on every `.lens-ops-card` (verified live: 3/3).

### Changed — 2026-10-18 (pareto plan execution: docs, wiring, triage)

- **AGENTS.md recording-proxy extraction** (218/219): the two-hop upstream
  budget details moved to docs/http-stack.md (verified against
  `internal/web/recording.go`: metadata context deadline +
  `ResponseHeaderTimeout`, body unbounded).
- **Pre-commit hook chain**: `.githooks/pre-commit` runs the AGENTS budget
  gate, then delegates to any flake-installed hook (treefmt);
  `core.hooksPath .githooks` is SET in the primary clone, routing the
  auto-commit daemon through the gate. The CI budget job stays gated on
  the lint-policy answer.
- **Four upstream filing drafts staged** (docs/drafts/, filing owner-gated
  per the verify-before-filing doctrine): CopyButton's empty-nonce
  unconditioned singleton script (cross-refs #63/#24), Card's h3 TitleTag
  default (12/12 call sites override it), statix's exit-0-on-config-error
  footgun (upstream identified as molybdenumsoftware/statix; dupe search
  clean), and the go-appkit testkit startTimeout saturation flake.
- **TestCacheSemanticsHardening "flake" root-caused and FIXED** (the
  2026-10-16 CI one-off): the rendered bodies embed a 1-second-granularity
  stamp (the PolledRegion "Updated HH:MM:SS" footer, RelativeTime chips),
  so a conditional replay crossing a second boundary legitimately answers
  200 with a fresh tag — the gate reproduced it live under load
  (`gzip conditional replay status = 200, want 304`, 5.04s duration). The
  conditional legs now go through a `conditionalReplay` helper that
  re-tags until the representation is stable (a 200 carrying an identical
  body still fails). The session's first triage (upstream testkit
  startTimeout saturation) was WRONG — the draft was retracted before
  filing and the test comment corrected. 20x -race loop green.
- **cqrs-lint F009 suppression question resolved as documentation**:
  v4.17.0 has no inline suppression and no per-path exclude; the policy
  doc records the gap and the repo-wide-disable tradeoff.
- **Golden scrubber contract test**: `TestGoldenScrubberCoversFooterStamp`
  fails on footer-copy drift that would silently un-scrub the goldens;
  the duplicated scripts/README codegen paragraphs merged (hash-primary
  wins).
- **Buildflow tooling triage re-verified** (29 doctor marks, all in the
  documented dispositions; no new relevant tool) and the statix invocation
  contract pinned in `.buildflow.yml` (bare `statix check`; re-audit on
  nixpkgs bumps).
- **The literal full-gate receipt exists** (T2, closes the v0.2.0-era
  BuildFlow#56 workaround): full-mode gate from the clean checkout —
  57 success / 0 failed / 0 skipped (+5 via config), exit 0, at load
  14.41/35.00 (run `20261018-041735-4709f40d`; a second green run 6 min
  earlier), test-race/coverage/nix-build/flake-check/license-check all ✔
  and go-generate/govalid green IN-GATE on the rebuilt binary — go and
  govalid are no longer verified individually. The gate doc gains the
  receipt and the lint-parity doctrine: buildflow reports golangci
  findings as ADVISORY (the ca31221 gate-green/CI-red incident, solved by
  run evidence — the gate saw the same trio and passed the step), so a
  receipt is CI-equivalent only with ZERO golangci findings; this one has
  them. The generated styles.css joined `*.min.js` in the .buildflow.yml
  exclude (prettier flagged it; foreign-formatter rewrite would re-red the
  CSS gate).
- **cut-release.sh rehearsed end-to-end** (T13): full body green on
  b4a9970 — nix build, dev gate, all four probe suites (42 PNGs), and the
  real remote-CI-green check — with abort-before-push proven by a
  pre-created scratch v0.0.0 and every quiesce guard proven to fire
  (dirty tree, unpushed HEAD, over-policy load at 34.80/61.08, and a
  stubbed-gh red CI verdict). The `go list -m` retry loop stays
  unimplemented: the tag guard makes the proxy tail unreachable in a
  rehearsal, and no real release has shown the lag (YAGNI; the FAIL
  message already instructs the retry). Receipt in scripts/README.

### Changed — 2026-10-17 (docs-health pass: live-set close-out + planning archive)

- The four 2026-10-17 status reports were annotated inline (247 verdicts:
  every §b/§c/§f/§g item carries a done / routed / Won't verdict verified
  against code, CHANGELOG, and the upstream ledgers) and archived — the
  `docs/status/` live set is empty, and a `docs/status/README.md` now
  indexes live-vs-archived at a glance.
- The eight fully-executed planning docs (00-09, 01-32, 19-05, 03-43,
  12-47, 13-32, 17-31, 05-07) moved to `docs/planning/archived/` with a
  per-file manifest, resolving the question the corpus sweep staged
  (annotate-in-place vs archive). The three plans with owner-gated or
  2027-01 residue stay live. Living-doc citations updated; CHANGELOG
  historical entries keep their original paths (append-only).
- The upstream filings ledger and the drafts README were re-synced with
  the 2026-10-17 gh-verified states (BuildFlow#45/#50/#55/#56,
  go-cqrs-lite#62-64, and go-appkit#26 CLOSED; go-etag#2, go-nix-helpers#7,
  treefmt-nix#545, templ-components#48/#49/#50/#51/#55 open), and the two
  2026-10-15 drafts gained their ledger rows and FILED banners.
- Harvest: the close-out report's §f 19+ queue triaged into six new
  bounded TODO rows (gate hardening, upstream filings, probe polish,
  release rehearsals, lint-policy intent, hooks wiring), the CSP
  enforce-mode decision staged as answer-sheet §11, ten ROADMAP raw
  ideas, and the README ingest section now mirrors `ingest-cdr -dry-run`;
  AGENTS.md holds the 219-line budget.

### Fixed — 2026-10-17 (styles.css foreign rebuild)

- The committed `styles.css` had been replaced by a foreign UNMINIFIED
  Tailwind build (daemon commit f91f3f4, after the morning's green
  verification), re-breaking `TestCSSGateBridgesHeatmapBrandToTeal` —
  the same class as the 2026-10-15 incident. Repaired via the sanctioned
  `nix run .#update-styles` (pinned, minified, bridge-verified); the
  styles-freshness CI job would have red-flagged it on push.

### Added — 2026-10-17 (Pareto-plan quality tail)

- Security-headers middleware in the serve chain (`X-Content-Type-Options:
  nosniff`, `Referrer-Policy: no-referrer`, `X-Robots-Tag: noindex`), wired
  second (after auth) per the docs/http-stack.md ordering doctrine and
  contract-tested across page, 304, static, and export surfaces; the CSP
  branch ships with a report-only sweep instrument
  (`scripts/csp-sweep.js`, six surfaces, nonce + violation events) whose
  run found and fixed the real nonce gaps (table row-hrefs, CopyButton
  scripts).
- Export-lane consumer tests: a real `csv.Reader`/JSON decoder must parse
  the stalled-stream artifacts end to end (EXPORT_INCOMPLETE marker
  contract from the consumer side) plus a 10k-row streaming benchmark
  (~268ms/op, memory-bounded — the R7 stream-don't-materialize evidence).
- Recording-proxy negative paths answer honestly: upstream 500, signed-URL
  expiry (403 between metadata and audio), and empty-URL recordings are
  fixture-covered (`TestRecordingNegativePathsAnswerHonestly`).
- Browser-lane scale and polish: `verify-all --fast` (contracts without
  the 41-PNG sweep), KNOWN_SURFACES drift guard, volatile-pixel budget in
  the screenshot probe, pixel baseline v2 saved post-h2-change (0 alerts),
  and `/ops` joined the authed e2e lane as the fifth surface.
- CI: `styles-freshness` job (pinned Tailwind regenerates and byte-diffs
  the committed styles.css), coverage floor raised 50→70 (measured 73.0%
  after the new cmd wiring tests took cmd/pbx-lens 9.2%→27.6%).
- `ingest-cdr -dry-run`: parse-and-age-gate rehearsal that reports
  would-record counts without touching any index (no -db required) — the
  F63 production re-ingest can rehearse first.
- `scripts/cut-release.sh` (the release runbook: quiesce → CHANGELOG
  sweep → build → gate → probes → CI-green → tag → proxy check) and
  `scripts/check-agents-budget.sh` (+ `.githooks/pre-commit`) for the
  AGENTS.md 220-line budget at commit time.
- `docs/telnyx-quarterly-checklist.md`: the 2027-01 re-verify bundle
  (commands, expected shapes, the calendar-confirm bullet) so the next
  live window runs from a file, not from memory.

### Changed — 2026-10-17 (status-corpus archive)

- The whole 2026-10-11→15 status-report corpus (27 reports) was annotated
  inline — every open item carries a verified done/routed/won't verdict —
  and archived under `docs/status/archived/` with a per-file manifest; only
  the newest report stays live as a harvest source. The living docs were
  rebuilt around it: TODO_LIST collapsed to open work only (211 → 90 lines,
  pass notes moved to the pass reports), two new bounded rows entered from
  the sweep (buildflow tooling triage, recordings lookback audit), and the
  version truths were refreshed (v0.2.0 in README/FEATURES, the archived
  count in AGENTS.md, and the hash-repair line now warns off the
  BuildFlow#55-corrupting `nix-hash-fix` step).

### Changed — 2026-10-17 (Pareto-plan UX and docs)

- All twelve page-level Cards now render `h2` titles (library default was
  `h3`, skipping a level under the page `h1` — the a11y sweep catch);
  goldens regenerated, heading order asserted in the UI contract probe,
  and the ops cards gained `break-inside: avoid` print protection with a
  print-ops probe capture.
- The search hint teaches the multi-word AND semantics with a worked
  example (`thermostat broken`), matching the newly pinned
  `TestMultiWordQuerySemantics`; RelativeTime's exact-timestamp tooltip
  and day-precision old-age fallback are pinned by contract test.
- docs/DOMAIN_LANGUAGE.md gained the MOS and CostMicros/CostCurrency
  vocabulary rows (zero-means-unreported is now written law); the search
  hint, README alerting recipe, and the Telnyx lookback-bound contract
  paragraph landed with them.

### Added — 2026-10-15 (test hardening)

- The `internal/web` package pins its clock: a `TestMain` sets `time.Local`
  to Europe/Berlin (zones embedded via `time/tzdata`), so golden snapshots
  and staleness fixtures render the operator zone on every machine — the
  v0.2.0 CI red (a golden diffing 09:01 UTC vs 11:01 Berlin) was the class
  this removes from the environment for good.
- The full-chain revalidation contract now covers `/ops` too: the page
  answers 200 with ETag + `Cache-Control: no-cache` and revalidates to an
  empty 304, pinned beside the dashboard fragment in the e2e suite.

### Fixed — 2026-10-17 (lint gate revival)

- The root `statix.toml` used `disabled = [20]` — an integer where statix
  expects rule-name strings, i.e. a TOML parse error. statix exits 0 on a
  config parse error, so the BuildFlow statix step (and every manual
  `statix check`) had silently linted nothing since 2026-10-12. Replaced
  with a file-level `ignore` of the sealed design record
  `deploy/nixos/module-draft.nix`, which keeps the repeated-keys rule
  active everywhere else.
- The `.golangci.yml` enabled `gohumanize`, a linter that does not exist
  in any golangci-lint release (added 2026-10-16; both the devShell and
  CI pin v2.14, which rejects it as unknown), so every lint run aborted
  with "unknown linters". Removed the phantom entry; lint runs again.
- Repo sync closed: remote main is green again (the 01:53 daemon push
  carried the statix + phantom-linter fixes; CI run 38992729791 success),
  and the docs-corpus tail (27-report archive, living-doc rebuild, the
  Pareto plan) is pushed on top of it — the "local ahead of a red remote"
  state is gone, so the push-tail TODO row is resolved by events.

Nothing else yet.

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
