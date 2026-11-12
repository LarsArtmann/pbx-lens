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
  - **Module-boundary decision (2026-10-11): HOLD single go.mod** — small
    single-domain project, zero external Go consumers. Revisit triggers: a
    second aggregate IN THE LENS DOMAIN, an external Go consumer, or
    measurable CI cost. The 2026-11-12 identity verdict (NO TRIP — usermgmt's
    aggregates live upstream on their own DB) is recorded in
    docs/cqrs-lint-policy.md; the E003 hold + pre-agreed write/read seam too.
  - sqliteengine (blank import registers the "sqlite" driver), pinned to the
    vetted set from go-cqrs-lite `example/getting-started/go.mod` (versions
    live in go.mod — never re-enumerate). ADR-0123: system.New is the only
    composition root; never hand-wire event/command/decider/storage.
- **Engines:** sqlite (`EngineConfig{Driver: "sqlite", DSN}`) via pure-Go
  modernc.org/sqlite, memory for tests; `app.New(ctx, dsn)` with dsn == ""
  selecting memory is the one deploy knob.
- **Language level:** go 1.27.3 in go.mod + `GOEXPERIMENT=jsonv2` (fleet
  house style; becomes load-bearing when a surface needs it).
- **Dependencies:** stdlib + go-cqrs-lite + go-error-family + the UI trio
  - `templ-components/htmx`. No testify (banned; stdlib table-driven tests),
    no cobra, no ORM. Promotion precedent: a module ALREADY in the graph rides
    free. server_timing's mount is appkit's default since v0.12 (pbx-lens wires
    no explicit timing middleware; `MeasureServerTiming` call sites are the
    used surface); `entitytag` stays indirect (`go-etag/server` re-exports).
  - `coreos/go-oidc/v3` v3.21.0 + `golang.org/x/oauth2` v0.37.0 (2026-11-12,
    the OIDC lane — fleet pins, x/oauth2 held below v0.38.0): code + PKCE-S256,
    lazy discovery, RS256 verify + nonce, userinfo fallback for claim-less
    access tokens. Routes auto-mount at `/auth/oauth/<provider>/*`; the
    registered callback is `https://<host>/auth/oauth/pocketid/callback`.
- **Test-only modules (operator-approved 2026-10-24):** systemscenario/v4 +
  scenario/v4 + rapid (+ go-snaps). Tests boot through
  `app.DomainConfig()`/`app.DeploymentConfig(dsn)`; committed rapid `.fail`
  seeds are regression pins (docs/gotchas.md).
- **cqrs-htmx (ADOPTED 2026-11-11):** usermgmt + loginpage + webauthn own the
  identity context, PATH B manual wiring (NO setup bundle — it collides with
  /dashboard and drags adminui in). `internal/web/userauth.go` composes
  session → CSRF → gate; `/login` `/auth/*` `/app.css` `/robots.txt` public;
  browsers 303 to /login, APIs 401. Identity persistence = ONE sqlite file
  (its checkpoint schema SQL is applied BY HAND — no self-migration).
  `MaxUsers: 1` = the first /login visitor becomes the operator, registration
  closes after. identity.db is NOT disposable — the NixOS module backs it up
  beside the index. WebAuthn refuses an IP RPID (`deriveAuthDefaults` maps
  127.0.0.1 → `localhost`); passkeys need HTTPS or localhost; the gate's CSRF
  dance in tests needs Host+Origin+Referer like a browser.
- **Upstream watchlist:** docs/gotchas.md owns it (go-cqrs-lite v5 rename
  sweep budgeted; versions.json lies — verify with `git tag --contains`).
- **UI trio (coordinated bump 2026-11-11 → tc v1.27.0 + templ v0.3.1100 —
  ONE bump, manual templ only: a dependabot templ PR mangles this go.mod;
  v1.27's tcValidity bridge needs `Nonce` on EVERY forms.Form):**
  `go-appkit` v0.12.0 (NewService → svc.Mux; svc.Run owns signals + drain;
  v0.10+ mounts Server-Timing as the default outermost — the serve.go wiring
  is DELETED; `RegisterClassifications()` at boot), `templ-components`
  v1.27.0 (display/forms/feedback/layout/navigation), `a-h/templ` v0.3.1100
  (codegen). Codegen behavior changes (icon→text separator space, duplicate
  `</script>` after singleton scripts) are in the adoption map. Stay on the
  PUBLISHED templ-components — no local replace (breaks hermetic nix).
- **UI theme:** app.css overrides the tc-primary/-hover/-info trio to teal in
  an @theme block placed AFTER the templ-components-theme.css import (later
  @theme wins); heatmap is re-bridged via --ds-brand-rgb/--ds-brand. NEVER
  hand-write `bg-blue-*` in pbx-lens .templ files. Mechanics + guards:
  docs/templ-components-adoption.md.
- **templ-components adoption map:** per-component truth for THIS repo lives
  in docs/templ-components-adoption.md (check it before importing anything
  new; it carries the bump checklist + watchlist). **utils/format ADOPTED**
  — never hand-roll duration/percent/dash formatting; keep `mosGaugePercent`.
- **Flake:** go-standard flakeModule: `subPackages = ["./cmd/pbx-lens"]` (the
  default `["."]` shipped an empty result); treefmt = nixfmt + deadnix +
  statix (BuildFlow owns Go formatting); DevShell sets GOTOOLCHAIN=local +
  GOEXPERIMENT=jsonv2. UI codegen is a package preBuild — it MUST run
  `go tool templ generate` (module-pinned), never bare `templ` (nixpkgs
  drifts ahead; the 2026-11-11 `templ.ResolveAttributeValue` compile break).
  styles.css is COMMITTED (rebuild ONLY via `nix run .#update-styles`);
  *_templ.go stays gitignored. `vendorHash.nix` holds the landed hash:
  repair via `nix run .#update-vendor-hash` on a CLEAN tree (never
  nix-hash-fix, BuildFlow#55) and verify ANY touch against root AND devShell
  (`nix develop -c echo ok`) — `nix flake check` never builds devShells.
  cqrs-lint's source hash + its OWN vendorHash live in flake.nix, NOT
  vendorHash.nix — the build order is source FOD → vendor FOD → devshell;
  repair either with the surfaced `got:` value (2026-11-12: the gate's
  hash-fix step cross-overwrote both — restored from got). The flake
  hardening contract (unfree license, lock guards, one goAttr):
  docs/gotchas.md § "The flake hardening contract".
  `scripts/cut-release.sh` guards version-const parity (v0.3.0 shipped
  printing 0.2.0). KNOWN: checks.pre-commit false-positives on
  import-dependent templ alignment — the CI lane is truth, NEVER commit its
  drift (docs/gotchas.md).
- **CI:** GitHub Actions (go-output's pattern): build+test(-race)+coverage+
  tidy, golangci-lint, govulncheck, and the art-dupl ZERO-GROUP GATE
  (`--min-lines 2` vs `scripts/art-dupl-baseline.json`; the default report
  hides a suppressed bucket — audit with `--show-suppressed` after pin
  bumps; invoke only via `scripts/check-dedup-gate.sh`). Every Go-compiling
  job runs `go tool templ generate` first (`go run .../cmd/templ` FAILS —
  the CLI is a go.mod `tool`). The golangci-lint pin must be BUILT with go ≥
  the module's level; `.golangci.yml` enables `mnd` — run
  `golangci-lint run ./...` before pushing. BuildFlow stays the LOCAL gate;
  Dependabot watches github-actions + gomod.
- **Structure:** the CLI command layer (smoke/ingest-cdr/ingest-telnyx/
  version) lives in `internal/cli`, wired from `cmd/pbx-lens` (`catalog` is
  the DB-free exception: renders the eventcatalog tree + `-architecture`
  HTML for the hub). Flag validation is shared (`requireDB`/`requireOneArg`).
  The `internal/ingest` ROOT package hosts the shared parser scaffolding —
  `Result` and `NewScanner`/`EachLine`; `cdr.Parse`/`webhook.Parse` return
  `ingest.Result` directly, `agent.Result` keeps its own shape. Support:
  `internal/wire` (FlexTime), `internal/backup` (VACUUM INTO + verify),
  `internal/retention` (Policy/OldViews/Retire behind `purge`;
  `Policy.Admits` also gates serve-poller `-max-age`). Read-model rebuilds
  go through `cli.ResetReadModel` (runbook: docs/runbooks/reset-rebuild.md;
  proof: `TestHostResetRebuildsReadModel`), never a hand-picked projection
  name. Projection record keys are BARE (`calls.RecordKey` = `StreamID.Get()`)
  — never hand-build one; read through `calls.LookupView` (its
  marker-prefixed fallback keeps pre-flip rows reachable; docs/gotchas.md).
  `calls.DefaultExcerptMax` (200 runes) caps every excerpt; source mappers
  select via `calls.Excerpt`. Auth facts (passkey-only, lockout 5/15min,
  cookie Secure follows origins' scheme, the e2e browser lane): the
  cqrs-htmx bullet above + docs/runbooks/identity-first-deploy.md +
  docs/runbooks/lockout-recovery.md.
- **serve/UI:** `serve` composes appkit around `web.Register`:
  ServiceConfig{Addr, `Version` → `/version`, `Metrics` → `/metrics`
  (explicit `AllowUnauthenticated` opt-in; both beside the appkit
  `/health{,/live,/ready}` routes), `WriteTimeout: NoTimeout` — exports and
  the recording audio stream are unbounded by contract} + `systemd.Install`
  - the identity session gate when `-auth-db` is set (empty = unauthenticated
    localhost dev). The ExtraMiddlewares chain + ordering/ETag/no-cache
    doctrine lives in docs/http-stack.md — read it before touching caching,
    timing, or the recording route. Every UI route is a query or a proxy,
    never a write. Exports and the dashboard STREAM via keyset pagination
    (`StreamExportCalls`, 5k-row pages, memory-bounded); day buckets resolve
    in Europe/Berlin (bare date filters too). htmx polling (R9):
    `/frag/dashboard` 30 s, `/frag/staleness` 60 s, runtime self-hosted —
    no CDN. `/api/ops` (R15) serves per-source + identity + age counters and
    answers 503 with `scan_failures` when a scan fails (never a silent
    partial 200). Detail page = READ-TIME join (rules in
    docs/DOMAIN_LANGUAGE.md § "Read-time call join"). List + dashboard
    collapse duplicate source rows onto the representative row (CDR first;
    `internal/web/representative.go`), export stays per-source. Analytics:
    spend, "who calls what" segments (`?number=` drill-through), "how calls
    end" rates — every figure carries its own denominator.

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
  ContentHash). Full rules: docs/DOMAIN_LANGUAGE.md § "Transcript truth".

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
                                      # EVERY docs batch, not only pre-push (scripts/README.md)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkConcurrentPoll -benchmem ./cmd/pbx-lens/  # poll ceiling ≈82 µs/op 304 / ≈101 µs full-200 (docs/http-stack.md)
GOEXPERIMENT=jsonv2 go test -bench BenchmarkIngestCDR -benchmem ./internal/cli/  # ingest per-row (docs/http-stack.md has both generations)
node scripts/screenshot-probe.js --only dashboard --diff /tmp/pd   # alert-only pixel diff (scripts/README.md)
nix develop -c cqrs-lint .            # findings policy: docs/cqrs-lint-policy.md
nix run .#update-vendor-hash          # after go.mod/go.sum; CLEAN tree only — never nix-hash-fix (BuildFlow#55)
```

Receipt-grade commits: `git add … && git commit …` in ONE call; amend only
UNPUSHED; a guard proof asserts the guard's MESSAGE, not its exit code.
Concurrent sessions + daemon: docs/concurrency.md.

## Deep context (docs/) — read the full doc BEFORE touching its area

- **docs/gotchas.md** — war stories: record keys, the one-shot drain,
  full-recompute folds, FTS5, tombstones, drain-before-enrich, `Exists`,
  the flake hardening contract.
- **docs/status/archived/** (69 reports) + **docs/planning/archived/** (10
  plans) — closed history, every executed plan (manifests in each README):
  never re-opened.
- **docs/telnyx-live-api.md** — live-verified API truths; re-verify quarterly (recordings 2026-10-12: MP3-ONLY; enrichment spec-verified only; window 2027-01, m27.9).
- **docs/cqrs-lint-policy.md** — every cqrs-lint finding as a decision
  (incl. the E003 hold + the 2026-11-12 identity verdict); the census there
  covers the TWO-BINARY drift (devShell's source-pinned engine reports 27
  incl. E005/F026; buildflow's bundled newer engine reports 26 with C044/
  E023 instead — never reconcile one against the other);
  **docs/buildflow-dev-gate.md** — gate traps + the zero-findings receipt rule.
- **docs/upstream-filings.md** — every filing ledger and its upstream state.
- **docs/http-stack.md**, **docs/templ-components-adoption.md** (middleware
  chain, component map), **docs/research/README.md** (the deep-dive index).
- **docs/review/libs/** — per-lib reports + index.html for ALL 22 first-party
  libs (2026-11-04 fleet audit; re-verified 2026-11-11/12, cqrs-htmx flipped
  to ADOPTED). Scores live in the index; refresh a report on its pin bump or
  when its flip-trigger fires. Audit sessions default to report-then-wait
  and execute only on the operator's explicit order.
