# pbx-lens Gotchas (paid for)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit). Every
entry below is a live-verified behavior of this repo or its dependencies —
each one cost real debugging time. Telnyx live-API specifics live in
[telnyx-live-api.md](telnyx-live-api.md); the cqrs-lint findings policy in
[cqrs-lint-policy.md](cqrs-lint-policy.md); gate residuals in
[buildflow-dev-gate.md](buildflow-dev-gate.md). AGENTS.md keeps the
invariants summary; this file keeps the war stories.

- **Projection record keys are BARE; only history carries the marker
  (2026-10-11, corrected 2026-10-23):** the folds key by the adapter's
  `EventWithID.ID`, which go-cqrs-lite **v4.6.0** flipped from
  `StreamID.String()` (`"StreamMarker:<kind>:<native-id>"`) to the bare
  `StreamID.Get()` value. The key form is a LIBRARY contract, not ours: a
  routine module bump silently moved every write while our readers kept
  asking for the marked form — detail pages 404'd and every projection
  lookup missed (the 2026-10-23 repair). Never hand-build a key: derive it
  via `calls.RecordKey(obs.StreamID())`, and read through
  `calls.LookupView`, which falls back to `LegacyRecordKey` so rows written
  before v4.6.0 stay reachable. Retiring a legacy row still cannot delete
  it (the remove fold derives the bare key from the record); the one-time
  read-model rebuild in TODO_LIST is what retires the fallback.
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
- **`go:embed static` snapshots at COMPILE time — a missing or mis-built
  stylesheet fails late and confusingly (paid for, 2026-10-12 + 2026-10-15)**:
  `internal/web/static/styles.css` is COMMITTED since 2026-10-15 (the CSS
  gate embeds it; the earlier gitignored state red every fresh clone). A
  checkout without it produces serve binaries whose `/app.css` answers the
  explanatory 404 (`http.Error` → `text/plain`), which Chrome reports as
  "Refused to apply style ... MIME type ('text/plain')" — no 404 in the
  console, so it looks like a broken Content-Type, not a missing artifact.
  An artifact built by anything BUT the flake-pinned tailwindcss --minify
  (unminified, different custom-property spacing) re-reds
  `TestCSSGateBridgesHeatmapBrandToTeal` with "lost the ds-brand bridge"
  even though every token is present — the gate pins the minified token
  format (`--ds-brand-rgb:15,` not `--ds-brand-rgb: 15,`), so the 17:08
  gate run failed on an unminified on-disk regen while the committed copy
  was already correct.
  BuildFlow's `tailwind-build` step is DELIBERATELY deferred in this repo
  (the CSS output is owned by the flake-pinned tailwindcss; a PATH tailwind
  would clobber the pinned bundle). The single sanctioned rebuild is the
  flake app `nix run .#update-styles` (added 2026-10-15 after TWO
  unminified clobbers within one hour re-redded the gate: go.mod-pinned
  templ codegen, then flake-pinned `tailwindcss -i
  internal/web/static/app.css -o internal/web/static/styles.css --minify`,
  then a bridge-token verify that fails the app itself on a broken
  artifact) — then REBUILD the binary
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
- **An HTTP redirect can never carry the #fragment (paid for, 2026-10-12,
  templ-UI T21)**: the search deep link `?q=` wants to land the browser on
  the first matching transcript turn, and the obvious one-shot
  `http.Redirect(..., "/calls/…?q=…#turn-N")` loops forever — `Location`
  URLs reach the client whole, but the browser strips the fragment before
  the NEXT request, so the server sees the same `?q=` every time and
  redirects again. The fix is a self-imposed one-jump rule: the redirect
  target carries `&at=N` (plus the fragment for the browser), and the
  handler redirects only when `at` is ABSENT — its presence means "already
  jumped", render instead. `at` is never parsed (presence-checked only),
  so tampered or garbage values (`at=999`, `at=abc`) fall through to a
  plain 200 render with marks — pinned by
  `TestDetailQueryTamperedAtRendersNoJump`. Do not "simplify" the param
  away and do not start parsing it.
- **Goldens must never pin wall-clock-coupled structure (paid for,
  2026-10-12)**: the golden suite captured whichever staleness-banner
  state 09:00-UTC seed data produced at capture time and silently broke
  for any run after 17:00 local (banner crosses the 6h staleAfter), and
  pinned the dashboard's bare ISO bucket labels that shift at midnight
  Berlin. The scrubber reduces runtime-coupled structure to placeholders
  (STALENESS_REGION, bare dates → DATE) while fixed-seed
  `datetime="…T…"` row attributes stay pinned; see
  `goldenVolatile` in golden_pages_test.go before adding any new golden.
- **A daemon auto-commit can snapshot a hash-fix run MID-PROBE and corrupt
  two hash sites at once (paid for, 2026-10-15)**: `update-vendor-hash` /
  `nix-hash-fix` write the fake `AAAA…` probe value before capturing the
  real `got:`; the auto-commit daemon snapped that intermediate state
  (vendorHash.nix AND flake.nix's inline cqrs-lint `vendorHash` = fake) and
  also pasted the main FOD's `got:` hash into the cqrs-lint SOURCE `hash`
  — the wrong derivation's hash, which `nix store prefetch-file --unpack
  --hash-type sha256 <codeload-url>` disproved independently. Result:
  fresh checkouts could build neither the main package nor the devShell.
  Repair: verify each hash against the derivation that OWNS it
  (`nix build` for vendorHash.nix, the devShell for the inline pair);
  never trust a `got:` line without knowing which FOD printed it. After
  any hash-fix run, confirm `nix build` + `nix build
  .#devShells.<system>.default` are green before the daemon can commit.

## UI todos are not done until the after-screenshot is eyeballed (paid for, 2026-10-15)

The 2026-10-15 audit's search-hint "improvement" (moving the prose hint into
`Input.HelpText`) looked right in the diff and made the layout visibly
WORSE — caught only because the after-screenshot was actually looked at, and
reverted the same session. Goldens pin markup, not pixels; the eye is the
review step. Rule: every visual change runs `node scripts/screenshot-probe.js`
(with `--port`; omitted = free port) and the relevant PNGs are OPENED and
eyeballed in both themes before the todo closes. The same session's
empty-dashboard finding (chart cards rendering as silent collapsed-0px-bar
boxes instead of honest empty messages) was found the same way — no test
asked for it because no test knew the page could look that broken.

## `go test -update ./pkg/` loses the package — put the flag LAST (paid for, 2026-10-26)

`go test -update ./internal/web/` fails with "no Go files in /…/pbx-lens"
(reproduced verbatim 2026-10-26): the golden `-update` flag is defined by
the TEST binary, not by `go test`, and placed before the package pattern
the go tool never registers that pattern — it falls back to the current
directory, and the repo root has no .go files (subPackages layout). The
working form is flag-last: `go test ./internal/web/ -update` (or
`go test ./internal/web/ -run TestGolden -update`). Diagnosed after the
19-53 session shrugged at it twice (§d3 there).

## statix exits 0 on a config parse error — the check silently no-ops (paid for, 2026-10-17)

`statix check .` printed "config error: unable to parse config file …" and
exited 0: a broken `statix.toml` does not fail the run, it SKIPS it. The
repo's config carried `disabled = [20]` (integer where statix wants
rule-NAME strings, e.g. `disabled = ["repeated_keys"]`) for five days, and
BuildFlow's statix step read green the whole time — the "config error" line
on stderr was the only clue that zero files were being linted. Treat ANY
"config error" output from statix as a red gate. Verified contract (statix
0.5.8-unstable-2026-07-17): `disabled` takes rule names; top-level `ignore`
takes path globs but only filters WALKED paths — a direct file target
(`statix check path/to.nix`) bypasses it (harmless here: W20 has no
auto-fix, and showing the finding on direct inspection is honest); and
treefmt's statix formatter never reads the repo `statix.toml` at all —
treefmt-nix wraps a frozen store copy (`statix fix --config /nix/store/…`)
per file. The repo config's only real consumer is BuildFlow's bare
`statix check` (cwd = repo root, default target `.`), which is exactly the
invocation the `ignore` glob matches.

## go mod tidy owns the go directive — the "major.minor only" warning is accepted noise (paid for, 2026-10-23)

(Cousin pattern of cqrs-lint E003's flat-HOLD: both are "library contract,
not ours" — see docs/cqrs-lint-policy.md E003.)

go-appkit v0.9.0 declares `go 1.27.2`, and `go mod tidy` (go ≥1.23) promotes
the main module's go directive to the highest dependency floor — hand-editing
`go 1.27` back in is silently reverted on the next tidy (observed live: edit

- `toolchain go1.27.2` line → tidy folded it right back to `go 1.27.2`).
  BuildFlow's go-version-auto-configure flags any patch component in the go
  directive ("must be major.minor only") — with go-appkit pinned at v0.9.0 that
  warning CANNOT be fixed repo-side; treat it as documented accepted noise, not
  a finding to chase. Also from the same run: govulncheck's GO-2026-6809
  (crypto/x509) is fixed in go1.27.3 — the repo's environments (nix go_1_27,
  setup-go latest-patch) already ship it; only stale global go binaries report
  it, so a stdlib vuln "finding" can be purely a toolchain-version artifact.

## A concurrent session's dirty tree fakes red gates (paid for, 2026-10-23)

This repo's auto-commit daemon plus concurrent AI sessions mean the tree can
be mid-refactor while a gate runs: a BuildFlow run tripped 4 step failures
(internal/web detail-join tests, nix-build "a dependency derivation failed",
vulnix crash) that ALL vanished once the concurrent session's web-layer
commit landed — the failing nix drv was literally `<rev>-dirty` at the
half-refactored commit. Before triaging gate reds: check `git log --since`
for interleaved commits and re-run the failing step against the settled tree
first. Corollary: a "stale FOD hash" preflight warning can be an mtime
heuristic misfire (go DIRECTIVE change, no dep change) — a green `nix build`
is the verdict, not the warning.

## `reader.Exists` answers false for rows `reader.Get` finds (paid for, 2026-10-25)

The collapse of duplicate source rows needed "does the CDR sibling exist?" —
and the obvious call, `metaengine.TypedReader.Exists`, is a trap for this
collection: it prefers the engine's `SetBackend`, and sqliteengine's set
table is a DIFFERENT table from the map-backed rows `metaengine.Query`
projections write. Verified live: `Get("freeswitch-cdr:uuid")` returned the
row while `Exists("freeswitch-cdr:uuid")` returned false, with no error. The
symptom is silent under-collapse (every AI call still rendered twice).

Use `calls.LookupView` (a `Get`) for presence. It also accepts the legacy
marker-prefixed key, so pre-flip rows answer correctly. Regression:
`internal/web/representative_test.go` seeds a CDR + transcript pair and
asserts ONE list row.

## versions.json lies; git tags are the truth (verified, 2026-10-25)

The go-cqrs-lite checkout's `versions.json` said systemscenario v4.2.0 while
the actual latest tag was v4.2.1 (and other modules drift the same way). Any
"what version shipped X?" question must be answered with
`git -C <checkout> tag --contains <commit>` and `git merge-base --is-ancestor`,
never with versions.json. Filing upstream about it is on the watchlist.

## The layout-stamp drift WARN never covered call_views (correction, 2026-10-25)

The 2026-10-24 audit's F2 claimed metaengine v4.20.0's layout-stamp drift
WARN closed "a live silent-drift exposure" on call_views. Wrong mechanism:
the stamp only records collections registered WITH a layout plan (declared
filter/sort fields, or the `columnarLayout` opt-in). `call_views` registers
neither — `BuildLayoutPlan` derives columns from filter+sort fields only —
so registration skips the plan entirely and there is nothing to stamp or
drift. The WARN mechanism itself is real (proven on a scratch collection:
same field names, changed column TYPES across two boots of one engine →
`metaengine: layout stamp drift ... recorded=v1:… declared=v1:…`). T21
verdict: the `WithLayoutStampsDisabled` opt-out is NOT needed — nothing
stamps. Consequence for boot logs: a quiet stamp section is normal here.

## systemscenario Then* windows are cumulative since the FIRST act (paid for, 2026-10-25)

`markActStart` fires once per scenario: every Then*/ThenNoEvents measures
the journal delta since the FIRST When act, not since the most recent one.
A second `When(cmd).ThenNoEvents()` inside one scenario sees the first
act's events and "fails". Structure instead: one act per scenario, seed
history with `Given(sc.Event(...))`, and use `systemscenario.Adopt` for a
second scenario over the same booted system (fresh baseline). Also:
`ThenEventsSatisfy` is act-scoped the same way — journal-shape assertions
belong at t-level via `System().EventStore()` (the documented escape
hatch). And a race fixture that appends out-of-band INSIDE a decide
(`AppendBatch` mid-handler) rolls the whole dispatch transaction back
silently — the OCC loser is better proven at the store's `Save` gate or
with true concurrent dispatches.

## A local core.hooksPath override silently disarmed the repo's own gates (paid for, 2026-10-25)

The primary clone carried a LOCAL `core.hooksPath=.git/hooks`, overriding
the machine global `.githooks` — so `.githooks/pre-commit` (budget, JS
no-undef, docs counts, doc links) never fired while the auto-commit daemon
kept committing, and scripts/README.md still claimed "SET since
2026-10-18". Paid for by the 2026-10-25 plan's errata 5: three seeded
refusal proofs in a scratch clone (a 220-line AGENTS, an undefined JS
symbol, a 999-report count claim) each refused with the gate's own
message. Two lessons: a hook chain is real only in clones where
`git config --get core.hooksPath` resolves to `.githooks` — verify per
clone — and a guard proof must assert the refusal MESSAGE, not just a
non-zero exit (a `git commit … | tail` pipeline hides the exit code and
fakes a pass; the first proof run here did exactly that).

## `nix flake check` never builds the devShell — verify devshell consumers after touching flake pins (paid for, 2026-10-26)

The 2026-10-25 dependency sweep overwrote TWO of cqrs-lint's fixed-output
pins in flake.nix (source hash → placeholder, vendorHash ← this repo's
pbx-lens hash) but re-derived only `vendorHash.nix`. Every consumer nix
build died at cqrs-lint's go-modules FOD, yet this repo's `nix flake
check` stayed GREEN — flake check evaluates and builds packages/checks
but never builds `devShells.*`, so the broken devshell input surfaced
only when a devshell consumer ran (`nix develop -c …`). Three hashes
now exist and must never be cross-pasted: `vendorHash.nix` (pbx-lens's
module set), flake.nix's cqrs-lint `vendorHash` (cqrs-lint's OWN module
set at its pinned rev — it does not track this repo's go.mod), and
cqrs-lint's source `hash` (immutable rev → deterministic). The recipe
that would have caught it in one command: after any flake pin touch,
run `nix develop -c echo ok` BESIDE `nix flake check` (the failed FOD's
mismatch output names the correct hash — pin it and the same build
verifies it).

## A committed rapid `.fail` file is a regression PIN — the policy (decided, 2026-10-26)

`testdata/rapid/<TestName>/*.fail` files are rapid's automatic replay
seeds: the owning test replays every one of them on every run. The four
committed seeds (the CDR lone-`;` trim bug's failfile among them — found
live by a fresh seed, fixed, and its committed failfile has replayed
green ever since) are therefore permanent regression pins, and that is
the POLICY, not an accident of the auto-commit daemon:

1. **Keep them in git — never gitignore.** A `.gitignore` rule would
   fight the daemon (which sweeps future seeds either way) and silently
   discard the exact input that broke the property.
2. **Never hand-edit one.** The bytes ARE the failing input; an edited
   pin is a lie about what once failed.
3. **Committing a seed means the diagnosis is done.** An unexplained
   `.fail` file is a live property failure — diagnose first (the replay
   must pass), then let it land as the pin. A committed seed that
   replays RED means the old bug is back; that is the pin working.
4. **Curation is deliberate deletion, never a bulk sweep.** When a
   pin's coverage is subsumed by an explicit table case (or it outlives
   its value), delete that one file with a CHANGELOG/gotcha note.

No PII concern: failfiles hold synthetic rapid-generated bytes, never
source data.

## metaengine `WithSort(col, true)` is DESCENDING — "true" is not "ascending" (paid for, 2026-10-26)

`WithSort("started_at", true)` sorts NEWEST first (`desc bool` is the
second parameter), and `OldViews`' old "oldest first" comment lied about
it for weeks without anyone noticing — it filters a full bounded scan, so
order never mattered there. The lie bit when `retention.OldestView` copied
the sort for a `WithLimit(1)` "oldest row" read: it silently returned the
NEWEST row, and the ops SLO figures came out 60 s off (one seed minute),
caught only because the new test pinned exact numbers. Lesson: a limit-1
extreme-value read must name its direction and TEST the extreme — and a
comment that describes order without a test pinning it is a future
60-second-off production figure.
