# AGENTS.md — pbx-lens

The call insights and analytics platform for pbx.artmann.tech — north star:
the superb call-insights platform (the SUPERB plan:
`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).
Read-only over every source: FreeSWITCH CDRs, Telnyx conversations/recordings
APIs, the receiver webhook JSONL. The index DB is disposable and always
re-derivable; never add a write-back path to any source. The sibling repo
(nix-international-telephony) later gets only a thin NixOS deploy module;
webphone later gets a read-only deep link, never a data dependency.

## Stack decisions

- **Modules:** `go-cqrs-lite` `system/v4` + `metaengine/v4` + projectionadapter
  - **Module-boundary decision (2026-10-11 four-lens review): HOLD single
    go.mod** — small single-domain project, zero external Go consumers (the
    sibling repo gets a NixOS module, not an import): every candidate seam
    has zero composability payoff and the package DAG already enforces the
    layering. Revisit triggers: a second aggregate (E003: hold flat,
    write/read seam pre-agreed — docs/cqrs-lint-policy.md), an external Go
    consumer, or measurable CI cost.
  - sqliteengine (blank import registers the "sqlite" driver), pinned to the
    vetted set from go-cqrs-lite `example/getting-started/go.mod` (exact
    versions live in go.mod — never re-enumerate). ADR-0123: system.New is
    the only composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27.2 in go.mod (the go-appkit v0.9.0 floor) +
  `GOEXPERIMENT=jsonv2` (fleet house style; nothing imports encoding/json/v2
  yet — the env var becomes load-bearing when a surface needs it).
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family + the UI trio
  (below) + `templ-components/htmx` (rides the root pin). No testify
  (banned; stdlib table-driven tests), no cobra, no ORM. Promotion
  precedent (bcrypt, x/time rate, templ-components/htmx, the 2026-10-12
  HTTP-stack trio httputil/server_timing/go-etag): a module ALREADY in the
  graph rides free — promoting it to direct adds no new module.
  server_timing has NO middleware constructor in any release (wrap
  manually); `entitytag` stays indirect (`go-etag/server` re-exports
  everything). Considered and NOT adopted: the decision records live in
  docs/planning/archived/2026-10-12_12-47_* and docs/drafts/.
- **Test-only modules (operator-approved 2026-10-24):** systemscenario/v4 +
  scenario/v4 + rapid (+ go-snaps, rides systemscenario). Tests boot through
  `app.DomainConfig()`/`app.DeploymentConfig(dsn)` — never hand-roll harness
  bookkeeping (semantics: docs/gotchas.md 2026-10-25 entries).
- **Upstream watchlist:** docs/gotchas.md owns it (go-cqrs-lite v5 rename
  sweep budgeted; versions.json lies — verify with `git tag --contains`).
- **UI trio (2026-10-11, F3/F4; bumped by the 2026-10-15 sweep):**
  `go-appkit` v0.9.0 (HTTP service shell: NewService → svc.Mux, svc.Run
  owns SIGINT/SIGTERM + graceful drain), `templ-components` v1.23.0
  (display/forms/feedback/layout/navigation), `a-h/templ` v0.3.1070
  (codegen; manual bumps only — a dependabot templ PR mangles this go.mod).
  Stay on the PUBLISHED templ-components version — no local replace
  (breaks hermetic nix); hand-roll missing thin pieces.
- **UI theme — signal palette (2026-10-12; heatmap bridge 2026-10-15):**
  app.css overrides `--color-tc-primary` (teal-700 #0f766e), `-hover`
  (teal-800), `--color-tc-info` (teal-700) in an @theme block placed AFTER
  the `templ-components-theme.css` import (later @theme wins; the theme
  binds `--color-blue-600: var(--color-tc-primary)`, so one override
  re-skins every library blue). The one component OFF the token chain is
  display.Heatmap: it fills via rgba(var(--ds-brand-rgb), α) and rings its
  peak cell with var(--ds-brand), both defaulting to violet-600 — app.css
  re-bridges the pair in a :root/.dark block after the imports (teal-700
  light, teal-400 dark; the rgb triplets must stay comma-separated — legacy
  rgba contract). Never hand-write `bg-blue-*` in pbx-lens .templ files —
  the library's own classes re-skin through the token chain.
- **templ-components adoption map:** per-component truth for THIS
  repo lives in docs/templ-components-adoption.md (skill docs drift ahead
  of the pinned tag; check the map before importing anything new; it also
  carries the version-bump watchlist + corpus-script mechanics to run on
  every bump — the v1.23.0 sweep skipped it once and paid for it).
  Pin: v1.24.0. **utils/format is ADOPTED (2026-10-24)** — never hand-roll
  duration/percent/dash formatting (dialect + the `-run 'TestGolden'`
  goldens gotcha: docs/templ-components-adoption.md); keep
  `mosGaugePercent` (domain math).
- **Flake:** go-standard flakeModule from go-nix-helpers (go-output's
  pattern): `subPackages = ["./cmd/pbx-lens"]` wires the package to the
  real binary (the default `["."]` built the package-less root and shipped
  an empty result); treefmt runs nixfmt + deadnix + statix only (BuildFlow
  owns Go formatting). DevShell sets GOTOOLCHAIN=local + GOEXPERIMENT=jsonv2
  (package list: the `nix develop` command line). UI codegen is a package preBuild (`templGenerationPolicy = "preBuild"`:
  templ generate + tailwindcss → styles.css). styles.css is COMMITTED (CI
  embeds it; nix preBuild still regenerates; 2026-10-15 v0.2.0 CI fix);
  *_templ.go stays gitignored.
  `vendorHash.nix` carries the landed vendor hash; repair a go.mod/go.sum
  break via `nix run .#update-vendor-hash` on a CLEAN tree (never nix-hash-fix, BuildFlow#55).
- **CI:** GitHub Actions lane (`.github/workflows/ci.yml`, go-output's
  pattern, single-module): build+test(-race)+coverage+tidy, golangci-lint,
  govulncheck, art-dupl (ZERO-GROUP GATE: `art-dupl check -t 4 --min-lines 2 .`
  vs `scripts/art-dupl-baseline.json`, which records ZERO accepted groups —
  any new multi-line clone fails the lane; `--min-lines 2` drops 1-line
  signature/type matches, which are shape coincidence, not duplication.
  TRAP: the default report hides a suppressed bucket that CAN hold real
  clones (four were extracted 2026-10-24); audit with `--show-suppressed`
  after pin bumps. docs/art-dupl-baseline.md owns mechanics/history). Dependabot watches github-actions + gomod.
  BuildFlow stays the LOCAL gate; the flake's
  generic apps.test/apps.lint stay module-generated conveniences, not a second
  gate. Every Go-compiling CI job runs `go tool templ generate` first
  (`*_templ.go` are gitignored; the templ CLI is pinned by a go.mod `tool`
  directive — `go run github.com/a-h/templ/cmd/templ` FAILS: CLI-only deps
  not in go.sum). The golangci-lint pin must be BUILT with go ≥ the
  module's language level (v2.12 = go1.26 refuses the config; v2.14 =
  go1.27.1), and the repo .golangci.yml enables `mnd` (local buildflow's
  lint may not) — run `golangci-lint run ./...` before pushing.
- **Structure:** the CLI command layer (smoke/ingest-cdr/ingest-telnyx/version
  subcommands) lives in `internal/cli`, wired from `cmd/pbx-lens` (`catalog`
  is the DB-free exception: `cli.RenderCatalog`/`cli.RenderArchitecture`
  render the eventcatalog tree (+ `-architecture` HTML) for the hub; see
  `.eventcatalog-hub.json`. Hub bootstrap norm: `-plain` always skips the
  generated bootstrap files, `-skip-bootstrap` opts out of normal exports —
  the merging hub owns site title, core pin, and dependencies). Flag
  validation for the index-opening subcommands is shared: `requireDB` and
  `requireOneArg` in package main (messages derive from `fs.Name()`, so a new
  subcommand gets consistent rejections for free). The `internal/ingest` ROOT
  package hosts the shared parser scaffolding — `Result` (observations +
  skipped-row count) and `NewScanner`/`EachLine` (64 KiB→1 MiB bounded line
  scan; `EachLine` trims and skips blanks). `cdr.Parse`/`webhook.Parse` return
  `ingest.Result` directly (no per-package Result alias — one type, no split
  brain); `agent.Result` keeps its own shape (Transcripts, not Observations).
  CDR keeps its own scan loop (semicolon-trimming CSV semantics), but through
  `ingest.NewScanner`. Support: `internal/wire` (FlexTime + shared decode
  helpers; directly unit-tested), `internal/backup` (VACUUM INTO snapshot
  - integrity verify), `internal/retention` (Policy/OldViews/Retire behind
    `purge`; `Policy.Admits` also gates serve-poller `-max-age`, 0 = keep
    everything). Projection record keys are BARE (`calls.RecordKey` =
    `StreamID.Get()`; the projection adapter's bare ID) — never hand-build
    one, and read through `calls.LookupView` (its marker-prefixed fallback
    keeps pre-flip rows reachable; docs/gotchas.md has the incident).
    `calls.DefaultExcerptMax` (200 runes) caps every excerpt; source mappers
    select via `calls.Excerpt` (first user turn, fallback first turn). The serve htpasswd timing pad hashes at the
    LOADED entries' bcrypt cost (`padCost`), not MinCost — keep that property
    when touching auth.go; SIGHUP reloads the file (a broken reload keeps the
    running table) and auth-failure throttling is PER CLIENT ADDRESS (bounded
    FIFO bucket map, shared fallback for hostless peers; X-Forwarded-For is
    deliberately not honored).
- **serve/UI:** `serve` composes appkit around `web.Register`:
  ServiceConfig{Addr, `Version` → `/version`, `Metrics` → `/metrics`
  (explicit `AllowUnauthenticated` opt-in; both beside the appkit
  `/health{,/live,/ready}` routes), `WriteTimeout: NoTimeout` — exports and
  the recording audio stream (WAV, else MP3 fallback) are unbounded and the
  former 30 s deadline truncated both} + `systemd.Install` (no-op without `NOTIFY_SOCKET`;
  READY/STOPPING/WATCHDOG under the manager) + htpasswd auth. The
  ExtraMiddlewares chain (auth → SecurityHeaders → Server-Timing →
  Compression → ETag; the ordering doctrine, the ETag skip policy, the
  render-buffering trap, and the `Cache-Control: no-cache` revalidation contract) lives in
  docs/http-stack.md — read it before touching caching, timing, or the
  recording route (that doc owns the recording proxy's two-hop budget).
  Every UI route is a query or a proxy, never a write. Exports and the
  dashboard STREAM via keyset pagination (`StreamExportCalls`, 5k-row pages,
  memory-bounded, no row cap); day buckets resolve in Europe/Berlin (R2
  2026-10-12 — bare date filters too), an idle index drifts out of the
  chart. htmx polling (R9): `/frag/dashboard` 30 s, `/frag/staleness`
  60 s, runtime self-hosted at `/static/htmx.min.js` — no CDN.
  `/favicon.svg` (embedded lens mark) satisfies layout.Base's default favicon
  link; `/api/ops` (R15) serves the poller's per-source counters behind the same auth. The NixOS
  module LANDED in nix-international-telephony (`modules/telephony/lens.nix`,
  backup + purge timers).
  Detail page = READ-TIME join (`CallDetailPage`; rules in
  docs/DOMAIN_LANGUAGE.md § "Read-time call join"). List + dashboard collapse
  duplicate source rows onto the representative row (CDR first;
  `internal/web/representative.go`), export stays per-source. Analytics: spend,
  "who calls what" segments (`?number=` drill-through), "how calls end" rates —
  every figure carries its own denominator.

## Domain

- `Observation` = one source row. `SourceKind`: freeswitch-cdr |
  telnyx-conversation | webhook-log | agent-transcript. `Direction`:
  inbound | outbound; a Telnyx conversation touching a configured AI DID
  (`-ai-did`) indexes inbound regardless of the declared direction. Numbers
  stay raw; `E164`/`ParticipantHash` exist for the planned PII policy and are
  NOT enforced yet.
- Stream id = `<kind>:<native-id>` — deterministic: re-ingest is the same
  stream; `call.record` is idempotent (same ContentHash replays to zero
  events; a changed hash on a live stream = parsing bug — CDR rows are immutable).
- Read model: `call_views`, one `CallView` per stream (`metaengine.OnRecordTyped` folds).
- **Transcript truth:** the lens performs NO speech-to-text — `Transcript` is
  the source's own turns, verbatim; non-speech rows are `SourceEvents` (outside
  ContentHash). The full rules (idle markers, turn times, call-fact signals,
  the no-second-ASR rule): docs/DOMAIN_LANGUAGE.md § "Transcript truth".

## Commands

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
buildflow --build-mode dev            # the quality gate; FULL only when loadavg
                                      # 15-min < 40 (docs/buildflow-dev-gate.md)
nix build                             # ships result/bin/pbx-lens; nix run . -- version runs it
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens refresh -db /tmp/lens.db [-apply]  # additive-field backfill: classify, then -apply; see docs/DOMAIN_LANGUAGE.md
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens.db -addr 127.0.0.1:8300
                                      # UI e2e via a Go http probe (curl banned); app.css rebuilds ONLY
                                      # via `nix run .#update-styles` (docs/gotchas.md: MIME trap)
nix develop                           # go, golangci-lint, gopls, govulncheck, dprint, templ,
                                      # tailwindcss_4, cqrs-lint, vulnix, oxlint, prettier, shellcheck
bash scripts/verify-all.sh            # THE browser lane: four probe suites, sequential, first-red-wins
GOEXPERIMENT=jsonv2 go test -bench BenchmarkConcurrentPoll -benchmem ./cmd/pbx-lens/  # poll ceiling ≈82 µs/op 304 / ≈101 µs full-200 (docs/http-stack.md)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkIngestCDR -benchmem ./internal/cli/  # ingest per-row: parse ≈16-26 µs, fresh-stream dispatch ≈10-12 µs (docs/http-stack.md; both under concurrent-load caveat 2026-10-25)
node scripts/screenshot-probe.js --only dashboard --diff /tmp/pd   # alert-only pixel diff (scripts/README.md)
nix develop -c cqrs-lint .            # findings policy: docs/cqrs-lint-policy.md
nix run .#update-vendor-hash          # after go.mod/go.sum; CLEAN tree only — never nix-hash-fix (BuildFlow#55)
```

Receipt-grade commits: `git add … && git commit …` in ONE call; amend only
UNPUSHED; a guard proof asserts the guard's MESSAGE, not its exit code.

## Deep context (docs/) — read the full doc BEFORE touching its area

- **docs/gotchas.md** — war stories: record keys, the one-shot drain,
  full-recompute folds, FTS5, tombstones, drain-before-enrich, `Exists`.
- **docs/status/archived/** + **docs/planning/archived/** — 59 reports, every
  executed plan (manifests in each README): CLOSED history — never harvest
  or re-open (living backlog: TODO_LIST.md).
- **docs/telnyx-live-api.md** — live-verified API truths; re-verify quarterly (recordings 2026-10-12: MP3-ONLY; enrichment spec-verified only; window 2027-01, m27.9).
- **docs/cqrs-lint-policy.md** — every cqrs-lint finding as a decision.
- **docs/buildflow-dev-gate.md** — gate traps + the zero-findings receipt rule.
- **docs/upstream-filings.md** — every filing ledger and its upstream state.
- **docs/http-stack.md**, **docs/templ-components-adoption.md** (middleware chain, component map), **docs/research/README.md** (the deep-dive index).
