# AGENTS.md — pbx-lens

Unified call-history index for pbx.artmann.tech (the SUPERB plan:
`pbx-artmann/docs/planning/2026-10-09_20-28_SUPERB-pbx-lens-call-index-and-woolly-golive.md`).
Read-only over every source: FreeSWITCH CDRs, Telnyx conversations/recordings
APIs, the receiver webhook JSONL. The index DB is disposable and always
re-derivable; never add a write-back path to any source. The sibling repo
(nix-international-telephony) later gets only a thin NixOS deploy module;
webphone later gets a read-only deep link, never a data dependency.

## Stack decisions

- **Modules:** `go-cqrs-lite` `system/v4` + `metaengine/v4` + projectionadapter
  - **Module-boundary decision (2026-10-11 four-lens review): HOLD single
    go.mod.** Two High "do not modularize" signals (small single-domain
    project; zero external Go consumers — the sibling telephony repo gets a
    NixOS module, not an import) mean every candidate seam has zero
    composability payoff; the package DAG already enforces the layering.
    Revisit triggers: a second aggregate (E003 decided 2026-10-11: hold
    flat; write/read seam pre-agreed — see docs/cqrs-lint-policy.md), an
    external Go consumer, or measurable CI cost.
  - sqliteengine (blank import registers the "sqlite" driver), pinned to the
    vetted set from go-cqrs-lite `example/getting-started/go.mod` (exact
    versions live in go.mod — never re-enumerate them here). ADR-0123: system.New is
    the only composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27 + `GOEXPERIMENT=jsonv2` (fleet house style, see
  go-output's flake). Nothing imports encoding/json/v2 yet (plain
  `go test ./...` passes without the env var); it becomes load-bearing when a
  surface needs it.
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family + the UI trio
  (below) + `templ-components/htmx` v1.21.0 (direct since R9 2026-10-12 —
  it was ALREADY in the module graph as templ-components' submodule rider;
  promoting it added no new module, same precedent as `x/crypto`).
  No testify (banned; stdlib table-driven tests), no cobra, no ORM.
  `golang.org/x/crypto` v0.57.0 (bcrypt only) is direct since D1: it was
  ALREADY in the module graph via projectionhost/watermill — promoting it
  added no new module; htpasswd without bcrypt is not worth hand-rolling.
  Same precedent promoted `golang.org/x/time` (rate) direct 2026-10-11: the
  auth failed-attempt throttle rides a module that was already in the
  graph. The 2026-10-12 HTTP-stack audit promoted three more in-graph
  modules to direct (all were already pulled in via go-appkit → httputil):
  `httputil` v1.4.1 (Compression middleware — the ONLY direct use; its ETag
  support left for the sibling module in v1.1.0), `httputil/server_timing`
  v1.0.2 (Server-Timing instrumentation; the released surface has NO
  middleware constructor — wire via its own `WrapServerTiming`, and metrics
  recorded after the first body write are silently dropped from the
  header), and `go-etag/server` v0.6.1 (conditional GET; `entitytag` stays
  indirect — server re-exports everything). Considered and NOT adopted
  (2026-10-12 audit, docs/planning/2026-10-12_12-47_SUPERB-*.md):
  appkit `errorpages`/`security`/`realtime` (do not exist in v0.8.0 —
  HEAD-only), errorpages doubly blocked by templ-components `errorpage`
  being ABSENT in pinned v1.21.0, rate-limiting exports (auth throttle +
  bounded queries cover it), SSE-hub-instead-of-polling (rewrites shipped
  R9), go-etag client for Telnyx (upstream ETag emission unverified),
  handler-side data-fingerprint fast-304 (YAGNI at current index sizes).
  `go-atomic-write` was EVALUATED for the backup staging chain
  (2026-10-11) and NOT adopted: its write entry points take Go data, while
  the staging file is written by sqlite itself (`VACUUM INTO`); the
  hand-rolled chain mirrors its semantics and adopts its unique-staging
  discipline — revisit when it grows an exported commit/rename primitive
  (draft ask in docs/drafts/).
- **UI trio (2026-10-11, F3/F4):** `go-appkit` v0.8.0 (HTTP service shell:
  `NewService(ServiceConfig{Addr})` → `svc.Mux *http.ServeMux`, `svc.Run(ctx)`
  owns SIGINT/SIGTERM + graceful drain), `templ-components` v1.21.0 (display/
  forms/feedback/layout/navigation components), `a-h/templ` v0.3.1070 (codegen;
  manual bumps only — a dependabot templ PR mangles this go.mod).
  Stay on the PUBLISHED templ-components version — no local replace (breaks
  hermetic nix); hand-roll missing thin pieces instead.
- **UI theme — signal palette (2026-10-12):** app.css overrides the semantic
  tokens `--color-tc-primary` (teal-600), `-hover` (teal-700), `--color-tc-info`
  (teal-500) in an @theme block placed AFTER the `templ-components-theme.css`
  import (later @theme wins the merge — the theme file binds
  `--color-blue-600: var(--color-tc-primary)`, so one override re-skins every
  library blue: buttons, badges, focus rings, BarChart bars, StatCard tiles,
  nav active). Never hand-write `bg-blue-*` in pbx-lens .templ files; the
  library's own classes re-skin through the token chain.
- **templ-components v1.21.0 adoption map** (skill's consumer tip; the skill
  docs drift ahead of the pinned tag — this table is the truth for THIS repo):

  | Component                                                                                   | Status            | Note                                                                                                                          |
  | ------------------------------------------------------------------------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------- |
  | layout.Base/DefaultPageProps                                                                | used              | `.Title` set; CSSPath defaults to /app.css; Base EMITS the `<main>` landmark — never nest a second main in the shell            |
  | layout.Container / Split / Stack                                                            | used 2026-10-12   | Container replaces the hand-rolled main wrapper; Split(1-3)+Stack(LG) replaced hand-rolled grid/space-y in dashboard + detail  |
  | navigation.SimpleNav + ThemeToggle + Footer                                                 | used              | Footer (BrandText single-row) anchored to Base's Footer slot since 2026-10-12                                                  |
  | display.Table/TableRow.Href/TableCell                                                       | used              | row-click JS included                                                                                                         |
  | display.Badge, PageHeader, EmptyState, ListNote, RelativeTime, Card, DefinitionList, Button | used              | DefinitionList takes `Items []DefinitionItem{Term, Detail, DetailComponent}` — NOT children; PageHeader.Action = Export CSV     |
  | display.Scrollback (transcript)                                                             | used 2026-10-12   | replaced hand-rolled bubbles; `Stagger: false` (reading content, not replay); exact counts ride in legend Labels                |
  | display.PieChart Donut (AI/human mix)                                                       | used 2026-10-12   | Donut+CenterLabel+ShowLegend replaced the hand-rolled flex bar; Slice.Color is a text-* class (fill=currentColor)               |
  | display.Grid + StatCard (dashboard stat row)                                                | used              | icons (Phone/Calendar/Microphone/Clock) + Tones added 2026-10-12; StatCard.Value is a string; renders `>N </dd>` — assert so    |
  | display.BarChart (vertical) + Sparkline (dashboard volume/trend)                            | used              | MinBarWidth `min-w-1` + Gap `gap-px` for dense 30-day series; Tooltip carries the full date                                   |
  | forms.Select/DatePicker + Form(FormGet, FormLayoutInline)                                   | used 2026-10-12   | FormLayoutInline == the old hand-rolled `flex flex-wrap items-end gap-3` row; Form renders the `<form>`, Apply/Reset are Button |
  | forms.Input (InputSearch)                                                                   | used 2026-10-12   | replaced the raw search input on /search                                                                                       |
  | feedback.Alert + CircularProgress                                                           | used              | Alert: staleness + MOS placeholder; CircularProgress: MOS gauge ((MOS-1)/4*100, aria-valuenow asserted in tests)                |
  | icons                                                                                       | used 2026-10-12   | EmptyState.Icon takes `icons.Name`; Button.Icon takes a rendered `icons.Icon(name, class)`; ~100 names in icon_names.go         |
  | htmx.PolledRegion                                                                           | used              | dashboard cards 30s, staleness banner 60s                                                                                      |
  | display.SegmentBar / utils/format / errorpage / FilterBar / SegmentedControl                | ABSENT in v1.21.0 | feedback.ProgressBar EXISTS (contrary to earlier note) — SegmentBar alone is unreleased upstream; the rest stays unimported    |
- **Flake:** go-standard flakeModule from go-nix-helpers (go-output's pattern,
  adopted 2026-10-11): `subPackages = ["./cmd/pbx-lens"]` wires the package to
  the real binary (the module default `["."]` built the package-less root and
  shipped an empty result); treefmt runs nixfmt + deadnix + statix only —
  `enableGofumpt`/`enableGoimports` are OFF because BuildFlow owns Go
  formatting (go-output's house split). The devShell (mkShellNoCC) carries
  go/golangci-lint/gopls/govulncheck/dprint/templ/tailwindcss_4/cqrs-lint with
  GOTOOLCHAIN=local + GOEXPERIMENT=jsonv2. UI codegen is a package preBuild:
  `templGenerationPolicy = "preBuild"` + a preBuild hook running
  `templ generate ./internal/web/` and `tailwindcss -i internal/web/static/
  app.css -o internal/web/static/styles.css --minify` — `*_templ.go` and
  `styles.css` are gitignored build artifacts, regenerated inside nix.
  `vendorHash.nix` carries the landed vendor hash; repair
  a go.mod/go.sum break with `buildflow -s nix-hash-fix --fix` or
  `nix run .#update-vendor-hash` — never paste hashes by hand.
- **CI:** GitHub Actions lane (`.github/workflows/ci.yml`, go-output's pattern,
  adapted single-module): build+test(-race)+coverage+tidy, golangci-lint,
  govulncheck, art-dupl (warning-only). Dependabot watches github-actions +
  gomod. BuildFlow remains the LOCAL quality gate (`buildflow --build-mode
  dev`); the flake's generic apps.test/apps.lint stay module-generated
  conveniences, deliberately not overridden into a second gate. Every
  Go-compiling CI job runs `go tool templ generate` first — `*_templ.go`
  are gitignored, and the templ CLI is pinned by a go.mod `tool` directive
  (`go run github.com/a-h/templ/cmd/templ` FAILS: its CLI-only deps are not
  in go.sum). Paid for on the first real CI run (2026-10-11): the
  golangci-lint pin must be a version BUILT with go ≥ the module's language
  level (v2.12 binaries are go1.26 and refuse to load config; v2.14 is
  go1.27.1), and the repo .golangci.yml enables `mnd` — local buildflow's
  lint may not, so run `golangci-lint run ./...` before pushing.
- **Structure:** the CLI command layer (smoke/ingest-cdr/ingest-telnyx/version
  subcommands) lives in `internal/cli`, wired from `cmd/pbx-lens`. Flag
  validation for the index-opening subcommands is shared: `requireDB` and
  `requireOneArg` in package main (messages derive from `fs.Name()`, so a new
  subcommand gets consistent rejections for free). The `internal/ingest` ROOT
  package hosts the shared parser scaffolding — `Result` (observations +
  skipped-row count) and `NewScanner`/`EachLine` (64 KiB→1 MiB bounded line
  scan; `EachLine` trims and skips blanks). `cdr.Parse`/`webhook.Parse` return
  `ingest.Result` directly (no per-package Result alias — one type, no split
  brain); `agent.Result` keeps its own shape (Transcripts, not Observations).
  CDR keeps its own scan loop (semicolon-trimming CSV semantics), but through
  `ingest.NewScanner`. Support packages: `internal/wire` (FlexTime + shared
  wire-decoding helpers; directly unit-tested since 2026-10-11),
  `internal/backup` (VACUUM INTO snapshot + integrity verify), and
  `internal/retention` (Policy/OldViews/Retire behind the `purge` command;
  `Policy.Admits` also gates ingest-side age via the serve poller's
  `-max-age`, 0 = keep everything). The projection record key is ALWAYS
  derived via `obs.StreamID().String()` — the last hand-built
  `"StreamMarker:..."` literal (cli/agent.go CDR join) was found and removed
  by the 2026-10-11 four-lens review; do not re-introduce one.
  `calls.DefaultExcerptMax` (200 runes) is the domain-owned transcript-excerpt
  ceiling; the selection rule is `calls.Excerpt` (first user turn, fallback
  first turn) and every source mapper goes through it (telnyx passes its
  configurable cap, agent passes the default). The serve htpasswd timing pad hashes at the
  LOADED entries' bcrypt cost (`padCost`), not MinCost — keep that property
  when touching auth.go; SIGHUP reloads the file (a broken reload keeps the
  running table) and auth-failure throttling is PER CLIENT ADDRESS (bounded
  FIFO bucket map, shared fallback for hostless peers; X-Forwarded-For is
  deliberately not honored).
- **serve/UI:** `serve` composes appkit (`ServiceConfig` with `Addr`,
  `Version` → `GET /version` + `appkit_build_info`, `WriteTimeout:
  NoTimeout` — the Timeout middleware drops out because exports and the
  recording WAV stream UNBOUNDED by contract and a 30 s deadline truncated
  both, and `Metrics` with the explicit `AllowUnauthenticated` opt-in —
  auth is the htpasswd middleware or loopback-only dev — → `GET /metrics`
  Prometheus text) + `systemd.Install` (a no-op without `NOTIFY_SOCKET`,
  READY/STOPPING/WATCHDOG under the manager) around `web.Register`; every
  UI route is a query or a proxy, never a write. ExtraMiddlewares wire in
  strict order auth → `web.ServerTimingMiddleware` (W3C Server-Timing via
  httputil/server_timing's `WrapServerTiming`; handlers add named metrics
  with `MeasureServerTiming` BEFORE the first body write — records after
  the header flush are lost, which is why streaming routes only time their
  pre-first-write hops) → `httputil.Compression` (q-value negotiation,
  outer of ETag so tags hash identity bodies — go-etag's ordering
  doctrine) → `web.ETagMiddleware` (go-etag/server; `SkipIfPresent` on;
  Skip policy: `/frag/staleness` + `/api/freshness` embed an age that
  changes every response, `/api/export.*` + `*/recording` stream
  unbounded and must never be buffered). `web.render` buffers templ pages
  into one atomic write: templ's streaming writer flushes per section and
  a flushed body can never be hashed into a 304. `/version` and `/metrics`
  exist beside the appkit `/health{,/live,/ready}` routes. The recording
  proxy's two upstream hops are bounded by `web.DefaultUpstreamTimeout`
  (30 s; `serve -recording-timeout`): metadata fetch deadline + WAV
  response headers, body streams unbounded. Exports and the dashboard
  STREAM via keyset pagination (`StreamExportCalls`, 5k-row pages,
  memory-bounded, no row cap); the dashboard's day buckets resolve in
  Europe/Berlin (owner R2 2026-10-12 — bare date filters too), an idle
  index drifts out of the chart. htmx polling (R9): `/frag/dashboard`
  30 s, `/frag/staleness` 60 s, runtime self-hosted at
  `/static/htmx.min.js` — no CDN. `/api/ops` (R15) serves the poller's
  per-source counters behind the same auth. The systemd draft lives in
  `deploy/systemd/pbx-lens.service` (Type=notify, LoadCredential for the
  Telnyx key + htpasswd); the NixOS module LANDED in
  nix-international-telephony 2026-10-11 (`modules/telephony/lens.nix`,
  `services.telephony.pbx-lens.*` with backup + purge timers;
  `deploy/nixos/module-draft.nix` is a superseded design record).

## Domain

- `Observation` = one source row. `SourceKind`:
  freeswitch-cdr | telnyx-conversation | webhook-log | agent-transcript.
  `Direction`: inbound | outbound; a Telnyx conversation touching a
  configured AI DID (`-ai-did`, `telnyx.DIDDirectionResolver`) indexes
  inbound regardless of the declared direction — the DID set is deployment
  config, not state. Numbers are stored raw as observed (real
  CDRs carry unknown/empty parties); `E164` + `ParticipantHash` exist for the
  planned PII policy (see TODO_LIST.md) and are NOT enforced on stored fields
  yet.
- Stream id = `<kind>:<native-id>` — deterministic, so re-ingest is the same
  stream and `call.record` is idempotent (same ContentHash replays to zero
  events; a different hash on a live stream is a conflict — CDR rows are
  immutable, so mutation means a parsing bug).
- Read model: `call_views` collection, one `CallView` per stream, folded from
  `call.observed` via `metaengine.OnRecordTyped`.

## Commands

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
buildflow --build-mode dev            # the quality gate (lint+format+test)
                                      # LOAD DOCTRINE (m27.9): run the FULL gate
                                      # only when the 15-min load average is < 40
                                      # (`cat /proc/loadavg` third field); above
                                      # that, verify casualties individually
                                      # (docs/buildflow-dev-gate.md)
nix build                             # ships result/bin/pbx-lens (fixed 2026-10-11)
nix run . -- version                  # run the built binary
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens.db -addr 127.0.0.1:8300
                                      # UI e2e: then probe /, /dashboard, /search,
                                      # /api/freshness with a Go http probe (curl banned);
                                      # -recording-timeout bounds proxy upstream hops
                                      # (default web.DefaultUpstreamTimeout, 30 s)
nix develop                           # devShell: go, golangci-lint, gopls,
                                      # govulncheck, dprint, cqrs-lint
nix develop -c cqrs-lint .            # findings policy: docs/cqrs-lint-policy.md
nix run .#update-vendor-hash          # repair vendorHash.nix after go.mod/go.sum
                                      # changes (buildflow -s nix-hash-fix
                                      # --fix does the same)
./scripts/...                         # (none yet)
```

## Deep context (extracted to docs/ 2026-10-11 — 220-line limit)

Invariants summary — read the full doc BEFORE touching its area:

- **docs/gotchas.md** — every paid-for war story: `StreamMarker:`-prefixed
  projection keys, the projection host's one-shot drain contract,
  full-recompute fold selection, FTS5 tokenization, retention
  tombstone-only deletes, drain-before-enrich, view→observation rebuild.
- **docs/status/archived/** — 21 superseded session reports (≤2026-10-12)
  with every open item resolved inline; CLOSED history — never harvest or
  re-open items from there (the living backlog is TODO_LIST.md).
- **docs/telnyx-live-api.md** — live-verified API truths (field-as-param
  filters, unstable timestamp precision, client-side recording
  correlation, enrichment endpoint shapes); re-verify the cluster
  quarterly (last live verify blocked on a key 2026-10-12; next attempt
  window opens 2027-01 — calendar entry, m27.9).
- **docs/cqrs-lint-policy.md** — every surviving cqrs-lint warning/info as
  a documented decision (E003 flat-domain HOLD + pre-agreed seam).
- **docs/buildflow-dev-gate.md** — gate environment traps (run inside the
  devShell, load-starvation casualties) + advisory-findings rationale.
- **docs/upstream-filings.md** — D6 filings (go-cqrs-lite#62/#63/#64,
  go-nix-helpers#7, treefmt-nix#545) and their upstream state.
