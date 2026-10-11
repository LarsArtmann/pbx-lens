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
  (below). No testify (banned; stdlib table-driven tests), no cobra, no ORM.
  `golang.org/x/crypto` v0.57.0 (bcrypt only) is direct since D1: it was
  ALREADY in the module graph via projectionhost/watermill — promoting it
  added no new module; htpasswd without bcrypt is not worth hand-rolling.
- **UI trio (2026-10-11, F3/F4):** `go-appkit` v0.8.0 (HTTP service shell:
  `NewService(ServiceConfig{Addr})` → `svc.Mux *http.ServeMux`, `svc.Run(ctx)`
  owns SIGINT/SIGTERM + graceful drain), `templ-components` v1.21.0 (display/
  forms/feedback/layout/navigation components), `a-h/templ` v0.3.1020 (codegen).
  Stay on the PUBLISHED templ-components version — no local replace (breaks
  hermetic nix); hand-roll missing thin pieces instead.
- **templ-components v1.21.0 adoption map** (skill's consumer tip; the skill
  docs drift ahead of the pinned tag — this table is the truth for THIS repo):

  | Component                                                                                   | Status            | Note                                                                                                                          |
  | ------------------------------------------------------------------------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------- |
  | layout.Base/DefaultPageProps                                                                | used              | `.Title` set; CSSPath defaults to /app.css                                                                                    |
  | navigation.SimpleNav + ThemeToggle                                                          | used              |                                                                                                                               |
  | display.Table/TableRow.Href/TableCell                                                       | used              | row-click JS included                                                                                                         |
  | display.Badge, PageHeader, EmptyState, ListNote, RelativeTime, Card, DefinitionList, Button | used              | DefinitionList takes `Items []DefinitionItem{Term, Detail, DetailComponent}` — NOT children                                   |
  | forms.Select/DatePicker                                                                     | used              | selected-ness is per `SelectOption.Selected`, no props-level Selected                                                         |
  | feedback.Alert                                                                              | used              | staleness banner; dashboard MOS placeholder                                                                                   |
  | display.Grid + StatCard (dashboard stat row)                                                | used              | StatCard.Value is a string; the library renders `>N </dd>` with a trailing space — assert accordingly                         |
  | display.BarChart (vertical) + Sparkline (dashboard volume/trend)                            | used              | MinBarWidth `min-w-1` + Gap `gap-px` for dense 30-day series; Tooltip carries the full date                                   |
  | display.SegmentBar / ProgressBar / utils/format / errorpage / FilterBar / SegmentedControl  | ABSENT in v1.21.0 | SegmentBar exists unreleased upstream — the dashboard ships a hand-rolled flex bar until it publishes; do not import the rest |
- **Flake:** go-standard flakeModule from go-nix-helpers (go-output's pattern,
  adopted 2026-10-11): `subPackages = ["./cmd/pbx-lens"]` wires the package to
  the real binary (the module default `["."]` built the package-less root and
  shipped an empty result); treefmt runs nixfmt + deadnix + statix only —
  `enableGofumpt`/`enableGoimports` are OFF because BuildFlow owns Go
  formatting (go-output's house split). The devShell (mkShellNoCC) carries
  go/golangci-lint/gopls/govulncheck/templ/tailwindcss_4/cqrs-lint with
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
  ceiling; every source mapper caps with it (telnyx aliases it as
  `telnyx.DefaultExcerptMax`). The serve htpasswd timing pad hashes at the
  LOADED entries' bcrypt cost (`padCost`), not MinCost — keep that property
  when touching auth.go.
- **serve/UI:** `serve` composes appkit (`ServiceConfig{Addr}` +
  `systemd.Install` — a no-op without `NOTIFY_SOCKET`, READY/STOPPING/WATCHDOG
  under the manager) around `web.Register`; every UI route is a query or a
  proxy, never a write. The dashboard aggregates via the same bounded full
  scan the export uses (`maxExportRows` cap); its day window always ends
  "today" UTC so an idle index drifts out of the chart. The systemd draft lives in
  `deploy/systemd/pbx-lens.service` (Type=notify, LoadCredential for the
  Telnyx key + htpasswd); the NixOS module LANDED in
  nix-international-telephony 2026-10-11 (`modules/telephony/lens.nix`,
  `services.telephony.pbx-lens.*` with backup + purge timers;
  `deploy/nixos/module-draft.nix` is a superseded design record).

## Domain

- `Observation` = one source row. `SourceKind`:
  freeswitch-cdr | telnyx-conversation | webhook-log | agent-transcript.
  `Direction`: inbound | outbound. Numbers are stored raw as observed (real
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
nix build                             # ships result/bin/pbx-lens (fixed 2026-10-11)
nix run . -- version                  # run the built binary
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens.db -addr 127.0.0.1:8300
                                      # UI e2e: then probe /, /dashboard, /search,
                                      # /api/freshness with a Go http probe (curl banned)
nix develop                           # devShell: go, golangci-lint, gopls,
                                      # govulncheck, cqrs-lint
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
- **docs/telnyx-live-api.md** — live-verified API truths (field-as-param
  filters, unstable timestamp precision, client-side recording
  correlation, enrichment endpoint shapes); re-verify the cluster
  quarterly.
- **docs/cqrs-lint-policy.md** — every surviving cqrs-lint warning/info as
  a documented decision (E003 flat-domain HOLD + pre-agreed seam).
- **docs/buildflow-dev-gate.md** — gate environment traps (run inside the
  devShell, load-starvation casualties) + advisory-findings rationale.
- **docs/upstream-filings.md** — D6 filings (go-cqrs-lite#62/#63/#64,
  go-nix-helpers#7, treefmt-nix#545) and their upstream state.
