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
    go.mod** — small single-domain project, zero external Go consumers (the
    sibling repo gets a NixOS module, not an import): every candidate seam
    has zero composability payoff and the package DAG already enforces the
    layering. Revisit triggers: a second aggregate (E003: hold flat,
    write/read seam pre-agreed — docs/cqrs-lint-policy.md), an external Go
    consumer, or measurable CI cost.
  - sqliteengine (blank import registers the "sqlite" driver), pinned to the
    vetted set from go-cqrs-lite `example/getting-started/go.mod` (exact
    versions live in go.mod — never re-enumerate them here). ADR-0123: system.New is
    the only composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27 + `GOEXPERIMENT=jsonv2` (fleet house style).
  Nothing imports encoding/json/v2 yet; the env var becomes load-bearing
  when a surface needs it.
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family + the UI trio
  (below) + `templ-components/htmx` (rides the root pin). No testify
  (banned; stdlib table-driven tests), no cobra, no ORM. Promotion
  precedent (bcrypt, x/time rate, templ-components/htmx, the 2026-10-12
  HTTP-stack trio httputil/server_timing/go-etag): a module ALREADY in the
  graph rides free — promoting it to direct adds no new module.
  server_timing has NO middleware constructor in any release (wrap
  manually); `entitytag` stays indirect (`go-etag/server` re-exports
  everything). Considered and NOT adopted (appkit
  errorpages/security/realtime, templ-components errorpage, export
  rate-limiting, SSE-hub, Telnyx etag client, data-fingerprint fast-304,
  go-atomic-write for backup staging): docs/planning/2026-10-12_12-47_*
  - docs/drafts/.
- **UI trio (2026-10-11, F3/F4; bumped by the 2026-10-15 sweep):**
  `go-appkit` v0.9.0 (HTTP service shell: NewService → svc.Mux, svc.Run
  owns SIGINT/SIGTERM + graceful drain; requires go ≥ 1.27.2 — the go.mod
  directive mirrors that floor), `templ-components` v1.23.0
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
  carries the version-bump watchlist: gen-tc-classes → map re-audit →
  hand-rolled re-check → filings re-read → golden `-update`, run in order
  on every bump — the 2026-10-15 v1.23.0 sweep initially skipped it and
  the map/corpus were reconciled after the fact).
- **Flake:** go-standard flakeModule from go-nix-helpers (go-output's
  pattern): `subPackages = ["./cmd/pbx-lens"]` wires the package to the
  real binary (the default `["."]` built the package-less root and shipped
  an empty result); treefmt runs nixfmt + deadnix + statix only (BuildFlow
  owns Go formatting). DevShell (mkShellNoCC): go/golangci-lint/gopls/
  govulncheck/dprint/templ/tailwindcss_4/cqrs-lint, GOTOOLCHAIN=local + GOEXPERIMENT=jsonv2.
  UI codegen is a package preBuild (`templGenerationPolicy = "preBuild"`:
  templ generate + tailwindcss → styles.css). styles.css is COMMITTED (CI
  embeds it; nix preBuild still regenerates; 2026-10-15 v0.2.0 CI fix);
  *_templ.go stays gitignored.
  `vendorHash.nix` carries the landed vendor hash; repair a go.mod/go.sum
  break with `buildflow -s nix-hash-fix --fix` — never paste hashes by hand.
- **CI:** GitHub Actions lane (`.github/workflows/ci.yml`, go-output's
  pattern, single-module): build+test(-race)+coverage+tidy, golangci-lint,
  govulncheck, art-dupl (warning-only). Dependabot watches github-actions +
  gomod. BuildFlow stays the LOCAL gate; the flake's generic
  apps.test/apps.lint stay module-generated conveniences, not a second
  gate. Every Go-compiling CI job runs `go tool templ generate` first
  (`*_templ.go` are gitignored; the templ CLI is pinned by a go.mod `tool`
  directive — `go run github.com/a-h/templ/cmd/templ` FAILS: CLI-only deps
  not in go.sum). The golangci-lint pin must be BUILT with go ≥ the
  module's language level (v2.12 = go1.26 refuses the config; v2.14 =
  go1.27.1), and the repo .golangci.yml enables `mnd` (local buildflow's
  lint may not) — run `golangci-lint run ./...` before pushing.
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
  `ingest.NewScanner`. Support: `internal/wire` (FlexTime + shared decode
  helpers; directly unit-tested), `internal/backup` (VACUUM INTO snapshot
  - integrity verify), `internal/retention` (Policy/OldViews/Retire behind
    `purge`; `Policy.Admits` also gates serve-poller `-max-age`, 0 = keep
    everything). The projection record key is ALWAYS
    derived via `obs.StreamID().String()` — the last hand-built
    `"StreamMarker:..."` literal (cli/agent.go CDR join) was found and removed
    by the 2026-10-11 four-lens review; do not re-introduce one.
    `calls.DefaultExcerptMax` (200 runes) is the domain-owned excerpt
    ceiling; every source mapper selects via `calls.Excerpt` (first user
    turn, fallback first turn). The serve htpasswd timing pad hashes at the
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
  ExtraMiddlewares chain (auth → Server-Timing → Compression → ETag; the
  ordering doctrine, the ETag skip policy, the render-buffering trap, and
  the `Cache-Control: no-cache` revalidation contract) lives in
  docs/http-stack.md — read it before touching caching or timing. Every UI
  route is a query or a proxy, never a write. The recording proxy's two
  upstream hops are bounded by `web.DefaultUpstreamTimeout` (30 s; `serve
  -recording-timeout`): metadata fetch deadline + WAV response headers,
  body streams unbounded. Exports and the dashboard STREAM via keyset
  pagination (`StreamExportCalls`, 5k-row pages, memory-bounded, no row
  cap); the dashboard's day buckets resolve in Europe/Berlin (owner R2
  2026-10-12 — bare date filters too), an idle index drifts out of the
  chart. htmx polling (R9): `/frag/dashboard` 30 s, `/frag/staleness`
  60 s, runtime self-hosted at `/static/htmx.min.js` — no CDN.
  `/favicon.svg` (embedded lens mark) satisfies layout.Base's default
  favicon link, which otherwise 404s on every page load. `/api/ops` (R15)
  serves the poller's per-source counters behind the same auth. systemd
  draft `deploy/systemd/pbx-lens.service` (Type=notify, LoadCredential);
  the NixOS module LANDED in nix-international-telephony 2026-10-11
  (`modules/telephony/lens.nix`, backup + purge timers;
  `deploy/nixos/module-draft.nix` is a superseded design record).

## Domain

- `Observation` = one source row. `SourceKind`: freeswitch-cdr |
  telnyx-conversation | webhook-log | agent-transcript. `Direction`:
  inbound | outbound; a Telnyx conversation touching a configured AI DID
  (`-ai-did`, `telnyx.DIDDirectionResolver`) indexes inbound regardless of
  the declared direction — the DID set is deployment config, not state.
  Numbers stay raw as observed; `E164` + `ParticipantHash` exist for the
  planned PII policy and are NOT enforced on stored fields yet.
- Stream id = `<kind>:<native-id>` — deterministic: re-ingest is the same
  stream, `call.record` is idempotent (same ContentHash replays to zero
  events; a changed hash on a live stream = parsing bug, CDR rows are
  immutable).
- Read model: `call_views` collection, one `CallView` per stream, folded from
  `call.observed` via `metaengine.OnRecordTyped`.

## Commands

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./...
buildflow --build-mode dev            # the quality gate (lint+format+test);
                                      # FULL only when loadavg 15-min < 40, else
                                      # verify casualties (docs/buildflow-dev-gate.md)
nix build                             # ships result/bin/pbx-lens; nix run . -- version runs it
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens.db -addr 127.0.0.1:8300
                                      # UI e2e: probe /, /dashboard, /search, /api/freshness with a Go
                                      # http probe (curl banned); styles.css is committed — after
                                      # app.css changes REBUILD it (go:embed is compile-time;
                                      # /app.css MIME trap: docs/gotchas.md); -recording-timeout bounds
                                      # proxy upstream hops (default web.DefaultUpstreamTimeout, 30 s)
nix develop                           # devShell: go, golangci-lint, gopls, govulncheck, dprint, cqrs-lint
bash scripts/verify-all.sh            # THE browser lane: all four probe suites
                                      # (sequential, free ports, first-red-wins)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkConcurrentPoll -benchmem ./cmd/pbx-lens/
                                      # poll ceiling ≈82 µs/op 304 / ≈101 µs full-200 (docs/http-stack.md)
node scripts/screenshot-probe.js --only dashboard --diff /tmp/pd --save-baseline /tmp/pd
                                      # alert-only pixel diff vs baseline; footer stamp ≈0.04% px absorbed
individual probes: verify-ui-contract.js (T20/T21), verify-style-contract.js (M1), verify-staleness-live.js, golden-pretty-diff.sh — scripts/README.md
nix develop -c cqrs-lint .            # findings policy: docs/cqrs-lint-policy.md
nix run .#update-vendor-hash          # repair vendorHash.nix after go.mod/go.sum
                                      # changes (buildflow -s nix-hash-fix --fix does the same)
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
  quarterly (recordings live-verified 2026-10-12 — assistant recordings
  are MP3-ONLY: empty `download_urls.wav`, populated mp3, see the doc;
  enrichment still spec-verified only; next window opens 2027-01 —
  calendar entry, m27.9).
- **docs/cqrs-lint-policy.md** — every surviving cqrs-lint warning/info as
  a documented decision (E003 flat-domain HOLD + pre-agreed seam).
- **docs/buildflow-dev-gate.md** — gate environment traps (run inside the
  devShell, load-starvation casualties) + advisory-findings rationale.
- **docs/upstream-filings.md** — D6 filings (go-cqrs-lite#62/#63/#64,
  go-nix-helpers#7, treefmt-nix#545) and their upstream state.
- **docs/http-stack.md**, **docs/templ-components-adoption.md** (living truth for the middleware chain and the component map), **docs/research/2026-10-12_http-stack-adoption-audit.html** (12-47 deep-dive snapshot + NOT-adopted doctrine table).
