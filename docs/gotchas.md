# pbx-lens Gotchas (paid for)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit). Every
entry below is a live-verified behavior of this repo or its dependencies —
each one cost real debugging time. Telnyx live-API specifics live in
[telnyx-live-api.md](telnyx-live-api.md); the cqrs-lint findings policy in
[cqrs-lint-policy.md](cqrs-lint-policy.md); gate residuals in
[buildflow-dev-gate.md](buildflow-dev-gate.md). AGENTS.md keeps the
invariants summary; this file keeps the war stories.

- **Projection record keys carry the `StreamMarker:` prefix (2026-10-11):**
  `cmd.StreamID().String()` returns `"StreamMarker:<kind>:<native-id>"` and
  that marker-ful string IS the call_views record key (the folds key by the
  event record ID). Polling `reader.Get` with a hand-built
  `"telnyx-conversation:x"` key always misses — cost ~1h of debugging that
  looked like a flaky projection engine. Always derive keys via
  `obs.StreamID().String()`.
- **The projection host is a one-shot batch drainer (2026-10-11):** workers
  drain the journal from their checkpoint, then EXIT; `Start` refuses with
  `projectionhost.already_started` until `Stop` resets the host, and `Stop`
  CANCELS the run ctx (it can kill a drain before it processes anything).
  Fire-and-forget `go host.Start(ctx)` is a race (events appended after the
  first drain exits are never projected). The correct synchronous drain —
  `cli.DrainProjections`: idempotent `Stop` → `Start` → poll
  `host.Status()` until every worker is `WorkerStopped` (WorkerIdle right
  after Start means "not drained yet", NOT done) — is what every CLI
  command and the serve poller must use after each dispatch batch. Memory
  engine NEVER projects without a host drain (the sqlite e2e test passed
  only because its flow coincided with a drain).
- **metaengine fold selection:** for a full-recompute view ship
  ONLY the 2-arg create fold (`OnRecordTyped` returning `(key, view)`).
  Pairing it with a 3-arg update fold lets metaengine route the FIRST event
  through the update fold with a zero `prev`, storing a zero view. Two-fold
  patterns exist for INCREMENTAL folds (`prev.Field += ...`), not recompute.
- **The coeffect gate's consumed set ignores `system.RawQuery`**
  (projection_builder passthrough): declaring `DomainConfig.Events` with a
  RawQuery projection logs an advisory "unconsumed event type" WARN at boot.
  Harmless; keep the declaration (the hard-error direction still guards
  deciders). The fluent `Evolve` + explicit-fold path feeds the gate but
  generates ONLY an update fold and keys by the view's `ID` field, which the
  closure cannot populate from the stream id — it panicked in poisoned-
  collection here. Revisit only if the library grows an explicit-create fold.
- **Vendored cqrs-lint (flake devShell):** built from a pinned go-cqrs-lite
  master rev — the `cmd/cqrs-lint/v4` module has no stable tags to pin
  instead. Two traps paid for: the monorepo root carries a `go.work`, so the
  build needs `env.GOWORK = "off"` or the modules FOD dies in workspace mode;
  and its own test suite runs fixture projects through `go` (network module
  fetches), which is impossible in the hermetic sandbox — hence
  `doCheck = false` on the vendored build (the tool is exercised for real on
  every run against this repo). Bump the rev deliberately; expect two hash
  iterations (src + vendor). PAID FOR 2026-10-11: an earlier session pasted
  the pbx-lens module vendor hash (vendorHash.nix's value) into BOTH cqrs-lint
  pins — the devShell only kept working from nix's positive cache until a
  rebuild exposed the mismatch. The three hashes (cqrs-lint src, cqrs-lint
  vendor, vendorHash.nix) are independent; fix a devShell FOD failure by
  reading the `got:` hash from the error, not by copying from vendorHash.nix.
- **This machine's global git config sets `core.hooksPath=.githooks`**, so
  git-hooks.nix's treefmt hook can never auto-install here (first
  `nix develop` prints pre-commit's "Cowardly refusing" once, then skips
  silently; go-output's installed hook is inert for the same reason). The
  wiring stays for pattern parity and manual checks (`nix develop -c pre-commit
  run --all-files`); the firing quality gate remains BuildFlow.
- **`errorfamily.NewRejection/NewConflict` take exactly (code, message)** —
  no format args; wrap with fmt.Sprintf. StreamID is a struct (cbid), not a
  string: use .String() / id.ParseStreamID, never a cast.
- **metaengine time-typed filter values miscompare on sqlite (2026-10-11,
  paid for by the ?to= wart test)**: sqliteengine binds the raw `time.Time`
  as a SQL parameter; modernc renders it `2006-01-02 15:04:05 -0700 MST`
  while the stored JSON holds RFC3339 (`T` separator). Lexicographic compare
  then silently corrupts same-day bounds: `Le/Lt` leak the day, `Ge/Gt`
  drop it, `Eq` never matches. Fix in-repo: bind
  `t.UTC().Format(time.RFC3339Nano)` strings (`web.rfc3339Nano`) — the exact
  stored wire format, correct for eq/le/ge/lt and robust regardless of any
  future upstream fix. Upstream sqliteengine issue pending (D6).
- **The memory engine resolves filter columns by Go field name, not JSON
  tag (2026-10-11)**: its closure fallback EqualFolds
  `extractValueByName(item, col)` against struct FIELD names, so
  `WithFilter("started_at", ...)` reads nil → Le/Lt match EVERYTHING, Ge
  matches nothing (constant compare vs nil). sqlite's `json_extract` uses
  JSON tags and is correct. Consequence: every web-layer test uses a real
  temp sqlite file (`newTestServerWith`), never the memory engine — filter
  semantics must match production. Upstream divergence issue pending (D6).
- **FTS5 tokenizes `+17287289311` as one token** (unicode61; `+` is not a
  token char): substring queries like `87289311` find nothing — search the
  full number or a word from the transcript. Prefix queries CANNOT fix
  mid-number substrings (a prefix must match from the token start), so the
  search page carries an explicit hint instead (`search.templ`).
- **The enrich pass reads the read model — drain first (2026-10-11)**:
  `cli.EnrichTelnyxCalls` looks streams up via the projection, so callers
  MUST `DrainProjections` between `Ingest` and enrichment, or every
  just-recorded stream counts `Missing` and retries next pass (the domain
  rejects enrich-before-observed by design). `pollTelnyx` drains mid-pass
  for exactly this reason; `ingest-telnyx` drains twice (ingest → drain →
  enrich → drain). Second-order contract: enrichment is idempotent via
  `Enrichment.Hash()` — an unchanged enrichment replays to zero events,
  and already-final fields skip their fetch (cost set → no detail fetch;
  summary set → no insights fetch), so re-polling a settled window costs
  one lazy detail-list request, nothing per call.
- **`CallView.Observation()` is the only sanctioned view→observation
  rebuild**: `ViewFrom` is lossless, so the rebuild re-hashes to the view's
  stored `ContentHash` (pinned by
  `TestViewObservationRoundTripHashesIdentically`). The enrich pass uses
  it to address `call.enrich` — never hand-copy view fields into an
  Observation.
- **`calls.MicrosPerUnit` is the single cost-unit constant**: the ingest
  mapper and the web display must never re-declare it (a former
  `web.microsPerUnit` duplicate was folded into the domain 2026-10-11).

- **Retention deletion is tombstone-only (2026-10-11, D2)**: metaengine exposes NO public
  direct record delete (`MapDelete` is unexported on the Store's engine) — the sanctioned
  removal path is a registered remove fold: `metaengine.OnRecordTyped(EventRetired,
  CallRetired{}, metaengine.Remove[CallView]())`. Pairing it with the create folds is safe
  (the zero-prev hazard is an UPDATE-fold-only trap). `call.retired` tombstones keep the
  stream append-only; re-polling a purged call is a ContentHash no-op, so views stay purged
  (tested). A post-retirement CORRECTION would resurrect the view — convergence comes from
  the periodic purge, not from stream surgery.
- **The PATH `templ` can trail the module's templ (2026-10-11)**: the devShell pinned
  v0.3.1020 while go.mod moved to v0.3.1070 and `templ generate` then failed on valid
  syntax. Generate with the module version:
  `GOEXPERIMENT=jsonv2 go run github.com/a-h/templ/cmd/templ@v0.3.1070 generate ./internal/web/`.
- **`rg -r` means --replace (paid for, 2026-10-11)**: `rg -rn "pattern"` silently rewrote
  every match in the OUTPUT to "n" and sent the diagnosis down a wrong path for a minute.
  Use `rg -n` for line numbers.
- **Read-only NixOS options reject mkIf-gated second definitions (paid for, 2026-10-11)**:
  `services.telephony.state` in nix-international-telephony is read-only — contribute via
  the central list in `modules/telephony/default.nix`, never by defining the option again
  in a sibling module (even gated, the definition trips the read-only check).
- **go test green is NOT lint green (paid for, 2026-10-12)**: a full wave shipped six
  real golangci findings (errcheck on non-test `fmt.Fprintf(out, …)`, two golines
  > 120-col lines, one formatter diff) while "watch the gate" stood in for verification —
  > and the gate could not run (load), so nothing caught it until a scoped
  > `golangci-lint run <touched pkgs>` before the release. Rule: end every wave with a
  > scoped lint of the packages you touched.
- **The golangci LSP cache goes stale across sessions (2026-10-12)**: it reported
  findings for code that no longer exists (a `timePtr` ghost) and missed real ones
  until restart. Verify every LSP lint claim — dismissal AND action — against
  `golangci-lint run` on the package.
- **errcheck's blanket exclusion covers only `_test.go`** (`.golangci.yml` rules):
  non-test report writers must check `fmt.Fprintf` errors — a failed stdout/buffer
  write must not exit 0 with the report silently lost (`runPurge` now returns them).
- **A test DSN literal is a real file path (paid for, 2026-10-12)**:
  `runPurge(ctx, sys, "unused-dsn", …)` made `RebuildSearchIndex` open sqlite at the
  relative path `unused-dsn`, creating a 28 KiB database in `cmd/pbx-lens/` on every
  test run — which the auto-commit daemon then committed twice. Memory-mode tests
  pass `""` (the documented no-op contract), and stray artifacts must be
  `git rm --cached` + trashed, not left for the daemon.
- **`golangci-lint fmt --diff <pkg>` is the formatter oracle** (2026-10-12): when gci/
  golines/gofumpt say "File is not properly formatted", the diff shows the exact
  expected bytes (one-arg-per-line call wrapping, map-key realignment) — apply it
  verbatim instead of guessing at the style.
- **BuildFlow's oxfmt has no `*.min.js` exclude (paid for, 2026-10-12)**: the JS
  _profile_ defaults exclude minified files, but the oxfmt/oxlint _providers_ trigger
  on plain `**/*.js` — two gate runs pretty-printed the vendored htmx.min.js
  (1 → 3,415 lines) and oxlint then flagged it. Fix: global
  `exclude: ["*.min.js"]` in `.buildflow.yml` + restore pinned bytes via
  `git show <rev>:path`. Vendored assets are exclude-pinned, never reformatted.
- **templ fmt sorts Go imports inside `.templ` (paid for, 2026-10-12)**: the flake's
  pre-commit check runs templ fmt fail-on-change, so a hand-edited import block
  (feedback/forms out of order) failed the whole nix-build — deterministically, and
  looking like a code defect. Run `nix develop -c templ fmt ./internal/web/` after
  any `.templ` import edit; `nix build .#checks.x86_64-linux.pre-commit` verifies
  standalone in seconds before a 15-minute gate run.
- **Don't touch the tree during a gate run (paid for, 2026-10-12)**: a mid-run
  docs edit re-dirtied the nix `-dirty` derivations and started a second full
  build beside the running tests; nix-build got killed 8s in. Docs before or
  after, never during.
- **A bare `go sys.Start(ctx)` goroutine races `System.Close` (paid for,
  2026-10-12)**: `defer sys2.Close()` runs before `t.Cleanup(stop)`, so
  `Host.Stop()` overlapped the still-running `Start` — the race detector only
  caught it under package-wide shuffle + load, never with `-run` alone.
  Helpers that start background hosts must return a waitable stop and the
  caller must run it before every Close.
- **`go:embed static` snapshots at COMPILE time — an unbuilt stylesheet reads
  as a browser MIME error, not a 404 (paid for, 2026-10-12)**:
  `internal/web/static/styles.css` is a gitignored tailwind artifact; a
  checkout without it produces serve binaries whose `/app.css` answers the
  explanatory 404 (`http.Error` → `text/plain`), which Chrome reports as
  "Refused to apply style ... MIME type ('text/plain')" — no 404 in the
  console, so it looks like a broken Content-Type, not a missing artifact.
  BuildFlow's `tailwind-build` step is DELIBERATELY deferred in this repo
  (the CSS output is owned by the flake-pinned tailwindcss; a PATH tailwind
  would clobber the pinned bundle), so regenerate with the same pinned
  binary as the flake preBuild hook —
  `nix develop -c tailwindcss -i internal/web/static/app.css -o
  internal/web/static/styles.css --minify` — then REBUILD the binary
  (restarting an old one changes nothing: the embed already happened).
  `nix build` is immune (the package preBuild regenerates inside the
  sandbox). `TestStylesServed` locks both legitimate states: 200 +
  `text/css` when built, explanatory 404 when not.
- **Two templs disagree about struct-literal alignment (paid for, 2026-10-12)**:
  go.mod's tool-directive templ (v0.3.1070, what `go tool templ` and CI
  codegen use) formats `.templ` struct literals differently than the flake's
  `pkgs.templ` (v0.3.1020, what the devShell and the sandboxed pre-commit
  check enforce). A tree formatted by the newer templ fails
  `checks.x86_64-linux.pre-commit` with "unexpected changes detected" on
  alignment alone — no import disorder, no hand edits needed. The nix-side
  formatter is the arbiter: after any session that ran `go tool templ fmt`,
  finish with `nix develop -c templ fmt ./internal/web/`. Until the flake
  pins a templ matching go.mod (upstream fix belongs in go-nix-helpers'
  templ program), buildflow's `templ-fmt` accepts both layouts, so it will
  NOT catch the drift — only the nix check does.
- **templ's streaming flush silently disables go-etag (learned, 2026-10-12)**:
  templ's component writer flushes per section, and a response that has
  already flushed can never be hashed and revalidated — the ETag middleware
  stands down, no tag is emitted, and every poll pays full body bytes with
  nothing in the logs. The `render` helper (`internal/web/render.go`)
  renders into a `bytes.Buffer` and writes ONCE, which is what makes the
  304 contract work at all; keep every HTML handler on it and never switch
  to a direct `component.Render(w, ...)` streaming call for a cached page.
- **templ fmt is per-file only and not idempotent on first pass (2026-10-12):**
  the sandboxed pre-commit check runs treefmt PER FILE (pre-commit feeds it
  one file per invocation), which hits templ's single-file formatting path —
  that path reformats what the BATCH path (`templ fmt a.templ b.templ …`)
  leaves untouched. Running treefmt directly on the whole tree therefore
  reports "0 changed" while the check fails with alignment diffs. Worse,
  templ's single-file fmt needs TWO passes to converge on some files (e.g.
  an inline `<span>@icons…</span>` first splits the closing tag, then splits
  the element) — one local pass still leaves the check red. Verify with the
  real thing: `nix build .#checks.x86_64-linux.pre-commit`, not a local
  treefmt run.
