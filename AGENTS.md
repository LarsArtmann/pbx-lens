# AGENTS.md — pbx-lens

The call insights and analytics platform for pbx.artmann.tech — north star:
the superb call-insights platform (the SUPERB plan:
`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).
Read-only over every source: FreeSWITCH CDRs, Telnyx conversations/recordings
APIs, the receiver webhook JSONL. The index DB is disposable and always
re-derivable; never add a write-back path to any source. The sibling repo
(nix-international-telephony) gets only a thin NixOS deploy module (LANDED:
`modules/telephony/lens.nix`, backup + purge timers); webphone gets a
read-only deep link, never a data dependency.

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
- **Language level:** go 1.27.3 in go.mod (the go-appkit v0.11.0 floor) +
  `GOEXPERIMENT=jsonv2` (fleet house style; nothing imports encoding/json/v2
  yet — the env var becomes load-bearing when a surface needs it).
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family + the UI trio
  (below) + `templ-components/htmx` (rides the root pin). No testify (banned;
  stdlib table-driven tests), no cobra, no ORM. Promotion precedent (bcrypt,
  x/time rate, templ-components/htmx, the 2026-10-12 HTTP-stack trio
  httputil/server_timing/go-etag): a module ALREADY in the graph rides free.
  server_timing ships `ServerTimingMiddleware()`/`...When(pred)` since v1.0.1
  (the serve wiring uses it; doctrine in docs/http-stack.md); `entitytag` stays
  indirect (`go-etag/server` re-exports everything).
- **Test-only modules (operator-approved 2026-10-24):** systemscenario/v4 +
  scenario/v4 + rapid (+ go-snaps, rides systemscenario). Tests boot through
  `app.DomainConfig()`/`app.DeploymentConfig(dsn)`, never hand-rolled harness bookkeeping; committed rapid `.fail` seeds are regression pins (docs/gotchas.md).
- **cqrs-htmx (ADOPTED 2026-11-11, the flip-trigger fired):** usermgmt
  v4.20.0 + loginpage v4.18.0 + usermgmt/webauthn v4.13.0 own the identity
  context (PATH B manual wiring — NO setup bundle: it would collide with
  pbx-lens's own /dashboard and drag adminui/dashboardui in). `internal/web/
  userauth.go` composes session → CSRF → gate; `/login` + `/auth/*` +
  `/app.css` + `/robots.txt` public; browsers 303 to /login, APIs 401.
  Identity persistence = ONE sqlite file: `OptimizeSQLiteDB` +
  `NewSQLEventStore` + `NewSQLSessionStore` + `ReadModelDB`+dialect +
  `storage.NewSQLiteCheckpointStore` (its schema SQL is applied BY HAND —
  no self-migration; `NewSQLiteEventSourcedSetup` is a //go:build ignore
  template, not compiled). `MaxUsers: 1` = the first /login visitor becomes
  the operator, registration closes after (the library's single-operator
  posture); session = authority (no roles/Casbin gating). Sessions survive
  restarts (SQL store); identity.db is NOT disposable — the sibling NixOS
  module stages its backup beside the index's. WebAuthn refuses an IP RPID
  at construction (`deriveAuthDefaults` maps 127.0.0.1 → `localhost`);
  passkeys need HTTPS or localhost. The gate's CSRF dance in tests needs
  Host+Origin+Referer like a browser (nosurf same-origin attestation).
- **Upstream watchlist:** docs/gotchas.md owns it (go-cqrs-lite v5 rename
  sweep budgeted; versions.json lies — verify with `git tag --contains`).
- **UI trio (2026-10-11, F3/F4; coordinated bump 2026-11-11 → tc v1.27.0 +
  templ v0.3.1100, checklist step 8 landed; v1.27's tcValidity bridge needs
  `Nonce` on EVERY forms.Form — the three filter/search/range forms thread
  it): `go-appkit` v0.12.0 (HTTP service shell: NewService → svc.Mux, svc.Run
  owns SIGINT/SIGTERM + graceful drain; bumped 2026-11-12 from v0.9.0 — the
  v0.10+ Server-Timing-default watch EXECUTED: the serve.go wiring is deleted,
  the header is appkit's default outermost; `RegisterClassifications()` wired
  at boot, docs/review/libs/go-appkit.md), `templ-components` v1.27.0
  (display/forms/feedback/layout/navigation), `a-h/templ` v0.3.1100 (codegen;
  manual bumps only — a dependabot templ PR mangles this go.mod; ≥v1.27
  templ-components needs templ ≥v0.3.1100 — ONE coordinated bump). templ
  ≥v0.3.1100 codegen emits the icon→text separator space inside anchors (the
  old `>Text</a>` test matches broke) and a duplicate `</script>` after
  singleton scripts (orphan close tag — browsers ignore it; upstream-filing
  candidate). Stay on the PUBLISHED templ-components version — no local
  replace (breaks hermetic nix); hand-roll missing thin pieces.
- **UI theme — signal palette (2026-10-12; heatmap bridge 2026-10-15):**
  app.css overrides the `--color-tc-primary`/`-hover`/`--color-tc-info` trio to
  the teal family in an @theme block placed AFTER the
  `templ-components-theme.css` import (later @theme wins; the theme binds
  `--color-blue-600: var(--color-tc-primary)`, so one override re-skins every
  library blue). The one component OFF the token chain is display.Heatmap: it
  fills via rgba(var(--ds-brand-rgb), α) and rings its peak cell with
  var(--ds-brand), both defaulting to violet-600 — app.css re-bridges the pair
  in a :root/.dark block after the imports (light/dark teals; the rgb triplets
  must stay comma-separated — legacy rgba contract). Never hand-write
  `bg-blue-*` in pbx-lens .templ files.
- **templ-components adoption map:** per-component truth for THIS repo lives
  in docs/templ-components-adoption.md (skill docs drift ahead of the pinned
  tag; check the map before importing anything new; it carries the version-bump
  watchlist + corpus-script mechanics to run on every bump — the v1.23.0 sweep
  skipped it once and paid for it). **utils/format is ADOPTED (2026-10-24)** —
  never hand-roll duration/percent/dash formatting (dialect + the
  `-run 'TestGolden'` goldens gotcha: docs/templ-components-adoption.md);
  keep `mosGaugePercent` (domain math).
- **Flake:** go-standard flakeModule (go-nix-helpers; go-output's pattern):
  `subPackages = ["./cmd/pbx-lens"]` wires the package to the real binary (the
  default `["."]` shipped an empty result); treefmt = nixfmt + deadnix + statix
  (BuildFlow owns Go formatting); DevShell sets GOTOOLCHAIN=local +
  GOEXPERIMENT=jsonv2 (package list: the `nix develop` command line). UI codegen
  is a package preBuild (`templGenerationPolicy = "preBuild"`), and styles.css
  is COMMITTED (CI embeds it); *_templ.go stays gitignored. THE PREBUILD HOOK
  MUST RUN `go tool templ generate` (module-pinned), never the bare `templ`
  go-standard injects: pkgs.templ drifts ahead of the go.mod pin and its
  codegen can emit calls the pinned runtime lacks (the 2026-11-11
  `templ.ResolveAttributeValue` compile break — the same doctrine the treefmt
  comment already carried). `vendorHash.nix`
  holds the landed hash: repair via `nix run .#update-vendor-hash` on a CLEAN
  tree (never nix-hash-fix, BuildFlow#55) and verify ANY touch against root AND
  devShell (`nix develop -c echo ok`) — `nix flake check` never builds
  devShells, and its green masked the 2026-10-25 devshell break. cqrs-lint's
  fetchFromGitHub source hash + its OWN vendorHash (both in flake.nix, NOT
  vendorHash.nix) went stale simultaneously with that break's fix wave and
  were re-landed 2026-11-11 (BH3IGPq… source / jTwZn/T… modules) — the
  devShell build order is: cqrs-lint source FOD → cqrs-lint vendor FOD →
  devshell; a mismatch at either surfaces as `hash mismatch … -source.drv` /
  `-go-modules.drv` and is repaired with the surfaced `got:` value.
  `scripts/cut-release.sh` guards version-const parity (v0.3.0 shipped printing
  0.2.0). Hardening (2026-10-28 nix-review): extraMeta.license = unfree (this
  repo is PROPRIETARY; extraMeta merges OVER the module meta), allowUnfree pkgs
  re-instantiated per-system from the LOCKED nixpkgs via `_module.args.pkgs`
  (hashes unchanged), enableLockGuards pins the follows pairs + nixpkgs lock
  node, ONE `goAttr` feeds both goPkgAttr and the perSystem `go` binding (else
  sandbox and devShell toolchains drift), update-vendor-hash declares
  coreutils+gnugrep, cqrs-lint carries meta. KNOWN pre-existing:
  checks.pre-commit false-positives on import-dependent templ alignment — the
  sandbox formats without module resolution, the CI lane is truth, NEVER commit
  its drift (docs/gotchas.md).
- **CI:** GitHub Actions (`.github/workflows/ci.yml`, go-output's pattern):
  build+test(-race)+coverage+tidy, golangci-lint, govulncheck, and the art-dupl
  ZERO-GROUP GATE (`art-dupl check -t 4 --min-lines 2 .` vs
  `scripts/art-dupl-baseline.json`; `--min-lines 2` drops 1-line shape
  coincidences, and the default report hides a suppressed bucket that CAN hold
  real clones — audit with `--show-suppressed` after pin bumps; invoke only via
  `scripts/check-dedup-gate.sh`). Every Go-compiling job runs `go tool templ
  generate` first (`*_templ.go` gitignored; the CLI is a go.mod `tool`, so `go run .../cmd/templ` FAILS).
  The golangci-lint pin must be BUILT with go ≥ the module's level, and
  `.golangci.yml` enables `mnd` — run `golangci-lint run ./...` before pushing.
  BuildFlow stays the LOCAL gate (the flake's apps.test/apps.lint are conveniences, not a second gate); Dependabot watches github-actions + gomod.
- **Structure:** the CLI command layer (smoke/ingest-cdr/ingest-telnyx/version)
  lives in `internal/cli`, wired from `cmd/pbx-lens` (`catalog` is the DB-free
  exception: `cli.RenderCatalog`/`cli.RenderArchitecture` render the
  eventcatalog tree + `-architecture` HTML for the hub; `.eventcatalog-hub.json`
  — `-plain` skips generated bootstrap files, `-skip-bootstrap` opts out, the
  merging hub owns site title/pins/deps). Flag validation is shared
  (`requireDB`/`requireOneArg`; messages derive from `fs.Name()`). The
  `internal/ingest` ROOT package hosts the shared parser scaffolding — `Result`
  and `NewScanner`/`EachLine` (64 KiB→1 MiB bounded scan, trims + skips blanks);
  `cdr.Parse`/`webhook.Parse` return `ingest.Result` directly (no per-package
  alias — one type, no split brain), `agent.Result` keeps its own shape, CDR
  keeps its semicolon-CSV loop through `ingest.NewScanner`. Support:
  `internal/wire` (FlexTime + decode helpers), `internal/backup` (VACUUM INTO +
  integrity verify), `internal/retention` (Policy/OldViews/Retire behind
  `purge`; `Policy.Admits` also gates serve-poller `-max-age`, 0 = keep
  everything). Read-model rebuilds go through `cli.ResetReadModel` (stop →
  `ResetProjection` → drain; runbook: docs/runbooks/reset-rebuild.md; proof:
  `TestHostResetRebuildsReadModel`), never a hand-picked projection name.
  Projection record keys are BARE (`calls.RecordKey` = `StreamID.Get()`) —
  never hand-build one, and read through `calls.LookupView`
  (its marker-prefixed fallback keeps pre-flip rows reachable; docs/gotchas.md
  has the incident). `calls.DefaultExcerptMax` (200 runes) caps every excerpt;
  source mappers select via `calls.Excerpt`. Auth (2026-11-11, corrected
  2026-11-12): the identity session gate (`internal/web/userauth.go`)
  replaced htpasswd — the bcrypt timing pad, SIGHUP reload, and per-IP
  throttle blocks are GONE from the tree; login is PASSKEY-ONLY (usermgmt
  ships no password login — over plain LAN http NOBODY can log in until
  TLS fronts the lens), brute-force defense is usermgmt's lockout (5
  failed WebAuthn ceremonies → 15 min) + the per-IP ceremony rate limits
  in `Identity.Register`, and the session cookie rides `Secure` exactly
  when the RPOrigin fronts TLS (`allOriginsHTTPS` — upstream's
  always-true default would break non-localhost http boots). Browser-lane
  receipts: `scripts/verify-identity-e2e.js` (gate/bootstrap/ceremony/
  lockout contracts) + screenshot-probe's `login` surface; deploy
  runbook: docs/runbooks/identity-first-deploy.md.
- **serve/UI:** `serve` composes appkit around `web.Register`:
  ServiceConfig{Addr, `Version` → `/version`, `Metrics` → `/metrics` (explicit
  `AllowUnauthenticated` opt-in; both beside the appkit `/health{,/live,/ready}`
  routes), `WriteTimeout: NoTimeout` — exports and the recording audio stream
  (WAV, else MP3 fallback) are unbounded and the former 30 s deadline truncated
  both} + `systemd.Install` (no-op without `NOTIFY_SOCKET`) + the identity
  session gate when `-auth-db` is set (empty = unauthenticated localhost
  dev; `-auth-rpid`/`-auth-origin` override the derived WebAuthn values —
  an explicit origin seeds the RPID from its host, IP/localhost binds
  derive `localhost`; `AuthEnabled` in web.Options gates the nav logout
  form).
  The ExtraMiddlewares chain (auth → SecurityHeaders → Server-Timing →
  Compression → ETag; ordering doctrine, ETag skip policy, render-buffering
  trap, `Cache-Control: no-cache` revalidation contract) lives in
  docs/http-stack.md — read it before touching caching, timing, or the recording
  route (it owns the two-hop budget). Every UI route is a query or a proxy,
  never a write. Exports and the dashboard STREAM via keyset pagination
  (`StreamExportCalls`, 5k-row pages, memory-bounded, no row cap); day buckets
  resolve in Europe/Berlin (bare date filters too), an idle index drifts out of
  the chart. htmx polling (R9): `/frag/dashboard` 30 s, `/frag/staleness` 60 s,
  runtime self-hosted at `/static/htmx.min.js` — no CDN. `/favicon.svg`
  (embedded lens mark) satisfies layout.Base's favicon link; `/api/ops` (R15)
  serves per-source counters behind the same auth. Detail page = READ-TIME join
  (`CallDetailPage`; rules in docs/DOMAIN_LANGUAGE.md § "Read-time call join").
  List + dashboard collapse duplicate source rows onto the representative row
  (CDR first; `internal/web/representative.go`), export stays per-source.
  Analytics: spend, "who calls what" segments (`?number=` drill-through), "how
  calls end" rates — every figure carries its own denominator.

## Domain

- `Observation` = one source row. `SourceKind`: freeswitch-cdr |
  telnyx-conversation | webhook-log | agent-transcript. `Direction`: inbound |
  outbound; a Telnyx conversation touching a configured AI DID (`-ai-did`)
  indexes inbound regardless of the declared direction. Numbers stay raw;
  `E164`/`ParticipantHash` exist for the planned PII policy, NOT enforced yet.
- Stream id = `<kind>:<native-id>` — deterministic: re-ingest is the same
  stream; `call.record` is idempotent (same ContentHash replays to zero events;
  a changed hash on a live stream = parsing bug — CDR rows are immutable).
- Read model: `call_views`, one `CallView` per stream (`metaengine.OnRecordTyped` folds).
- **Transcript truth:** the lens performs NO speech-to-text — `Transcript` is
  the source's own turns, verbatim; non-speech rows are `SourceEvents` (outside
  ContentHash). Full rules (idle markers, turn times, call-fact signals, the
  no-second-ASR rule): docs/DOMAIN_LANGUAGE.md § "Transcript truth".

## Commands

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
nix develop -c bash scripts/poll-gate.sh -- env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev
                                      # THE gate entry: run SOLO (parallel jobs starve test-coverage → exit 75); polls 15-min load<40, execs in-process (docs/buildflow-dev-gate.md)
nix build                             # ships result/bin/pbx-lens; nix run . -- version runs it
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens refresh -db /tmp/lens.db [-apply]  # additive-field backfill: classify, then -apply; see docs/DOMAIN_LANGUAGE.md
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens.db -addr 127.0.0.1:8300
                                      # UI e2e via a Go http probe (curl banned); app.css rebuilds ONLY
                                      # via `nix run .#update-styles` (docs/gotchas.md: MIME trap)
nix develop                           # go, golangci-lint, gopls, govulncheck, dprint, templ,
                                      # tailwindcss_4, cqrs-lint, vulnix, oxlint, prettier, shellcheck
bash scripts/verify-all.sh            # THE browser lane: five probe suites, sequential, first-red-wins;
                                      # every gate prints `verify-all: PASS <gate>` (greppable receipts)
bash scripts/check-docs-counts.sh && bash scripts/check-doc-links.sh
                                      # docs gates IN THE WRITING LOOP (2026-11-12 convention): run after
                                      # EVERY docs batch, not only pre-push — a drifted count or dead link
                                      # is fixed while the batch is still open (scripts/README.md)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkConcurrentPoll -benchmem ./cmd/pbx-lens/  # poll ceiling ≈82 µs/op 304 / ≈101 µs full-200 (docs/http-stack.md)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkIngestCDR -benchmem ./internal/cli/  # ingest per-row, quiet box 2026-10-25: parse ≈8.7 µs, fresh-stream dispatch ≈6.1 µs (docs/http-stack.md has both generations)
node scripts/screenshot-probe.js --only dashboard --diff /tmp/pd   # alert-only pixel diff (scripts/README.md)
nix develop -c cqrs-lint .            # findings policy: docs/cqrs-lint-policy.md
nix run .#update-vendor-hash          # after go.mod/go.sum; CLEAN tree only — never nix-hash-fix (BuildFlow#55)
```

Receipt-grade commits: `git add … && git commit …` in ONE call; amend only
UNPUSHED; a guard proof asserts the guard's MESSAGE, not its exit code.
Concurrent sessions + daemon: docs/concurrency.md.

## Deep context (docs/) — read the full doc BEFORE touching its area

- **docs/gotchas.md** — war stories: record keys, the one-shot drain,
  full-recompute folds, FTS5, tombstones, drain-before-enrich, `Exists`.
- **docs/status/archived/** + **docs/planning/archived/** — 69 reports, every
  executed plan (manifests in each README): CLOSED history, never re-opened.
- **docs/telnyx-live-api.md** — live-verified API truths; re-verify quarterly (recordings 2026-10-12: MP3-ONLY; enrichment spec-verified only; window 2027-01, m27.9).
- **docs/cqrs-lint-policy.md** — every cqrs-lint finding as a decision;
  **docs/buildflow-dev-gate.md** — gate traps + the zero-findings receipt rule.
- **docs/upstream-filings.md** — every filing ledger and its upstream state.
- **docs/http-stack.md**, **docs/templ-components-adoption.md** (middleware
  chain, component map), **docs/research/README.md** (the deep-dive index).
- **docs/review/libs/** (2026-11-04 fleet utilization audit; every finding
  closed the same day; REFRESHED 2026-11-11 when the cqrs-htmx flip-trigger
  fired and the usermgmt adoption landed; RE-VERIFIED 2026-11-12 — every
  report re-checked against fetched tags, cqrs-htmx.md rewritten to ADOPTED,
  the missing cmdguard.md + httputil-endpoints.md landed, index re-shot) —
  per-lib reports + index.html for
  ALL 22 first-party libs. Post-refresh scores: cqrs-lite core 92 (pooled
  ≈81), templ-components 95, go-etag 95, go-error-family 82, httputil 80,
  appkit 73, server_timing 67, cqrs-htmx 78 (ADOPTED, usermgmt/loginpage/
  webauthn leaf modules; root stays indirect);
  14 not imported directly (4 of them indirect-only riders with zero pbx-lens
  imports: codec/sse/idempotency/retry), each with a documented reason or a
  named flip-trigger (REJECT ×11, DEFER ×2: go-retry/go-atomic-write,
  PATTERN-ONLY ×1: go-output). Refresh a report on its pin bump or when its flip-trigger
  fires. Audit sessions default to report-then-wait and execute only on the
  operator's explicit order (2026-11-04, 2026-11-11, and again 2026-11-12).
