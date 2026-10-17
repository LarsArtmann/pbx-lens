# BuildFlow dev-gate residual findings policy (pbx-lens)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit).
Status at extraction: 2026-10-11 full pass — gate PASSES with warnings.

- **Run the gate INSIDE the devShell (PAID FOR, 2026-10-11)**: invoke
  `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev`.
  Run from a bare shell, buildflow resolves a go older than the 1.27.1 floor
  and `go-generate`/`workspace-build-verify` fail with "requires go >= 1.27.1
  (running go 1.27.0)". The devShell ships go 1.27.2 with GOTOOLCHAIN=local,
  which satisfies the floor. A bare-shell failure is an environment artifact,
  not a code defect — BUT rerun in the devShell before dismissing it: the
  2026-10-12 bare-shell `golangci-lint` failure was the skew AND masked
  3 real findings (cyclop/forcetypeassert/wsl_v5 in internal/web/recording.go,
  fixed same day); the devShell rerun is what surfaced them.
- **Gate under heavy machine load (PAID FOR, 2026-10-11)**: at load average
  40-85 (unrelated llama-server/clickhouse jobs saturating the box), five
  gate runs each lost DIFFERENT steps to "killed after Ns: no output at all"
  (go-fix, test-race, nix-build, go-mod-tidy, golangci:detect) — starvation,
  not code. Before believing a gate failure, verify the casualties
  individually: `gofmt -l`, `go test -race ./...`, `nix build`,
  `go mod tidy -diff`, `buildflow -s golangci-lint`. The go-mod-update
  flavor of this (a starved compile gate reported as "dependency bump
  introduced compile errors" + silent go.mod/go.sum revert, 2026-10-11
  23:59) was FILED as BuildFlow#45 and is FIXED (2026-10-13, BuildFlow
  v0.8.0): starved steps now get one automatic transient retry, an
  all-starvation run exits 75 (EX_TEMPFAIL, `pipeline.environment_starvation`,
  with the 15-minute load average in the message) instead of 69, and the
  summary names "N lost to starvation". After bumping the fleet binary:
  exit 75 = rerun when the machine is quieter (or `--max-concurrency N`),
  never a code regression; exit 69 keeps meaning code/findings.
  `vulnix` exits 1
  persistently (buildflow-internal CVE scan, network/advisory-class; not in
  the devShell — CI's govulncheck lane is the trusted one).
- **Gate test steps run under `-test.shuffle` (2026-10-12)**: two full-mode
  runs under load ~80-160 lost DIFFERENT test steps (test-race, then
  test-coverage) while the direct `go test -race -cover ./...` was green.
  The step log prints the seed — reproduce deterministically with
  `GOEXPERIMENT=jsonv2 go test -race -shuffle=<seed> ./internal/web/`
  before diagnosing. The timing-sensitive recording-proxy tests (1 ms
  upstream-timeout budgets) are the expected flake candidates under load;
  the seed replay passed in both cases.
- **`go mod tidy -diff` before CI (PAID FOR, 2026-10-11)**: dependency bumps
  can leave removable go.sum entries that build fine but fail a CI
  tidy-diff; tidy, then re-pin via `nix run .#update-vendor-hash` (the FOD
  hash survived this trim unchanged — it covers the module tree, not go.sum).
- **go-version-auto-configure (dep-forced, advisory by design)**: go.mod
  carries `go 1.27.2` because go-appkit v0.9.0's module floor demands it
  (its own go.mod declares 1.27.2) — `go mod tidy` reverts any
  major.minor downgrade. The checker's own severity policy treats
  dep-forced patch floors as correct; the 2026-10-15 run still flagged it
  as a warning from the stale fleet binary (built 518739c, 101 commits
  behind) — re-verify after a binary rebuild before treating it as a
  regression. If a stray `go-mod-*` step ever rewrites the directive to
  `go 1.27`, restore it (`git restore go.mod`); a 1.27 directive breaks
  `go list`.
- **go-auto-upgrade encoding/json v1→v2 suggestions**: deliberate — the
  boundary types (metaengine DTOs) are non-stdlib-owned; the v2 swap waits
  for T4 per the stack decisions. samber/lo suggestions ignored (stdlib
  - go-cqrs-lite + go-error-family only).
- **go-structure-linter assets/ suggestion**: `internal/web/static/` is
  deliberate (go:embed already scopes it; moving buys nothing).
- **art-dupl (20) / branching-flow (51) / nix-checker (hash bookkeeping
  infos)**: advisory reports; revisit only if a dedicated dedup/nix pass
  is scheduled (see TODO_LIST).
- **treefmt templ formatter go (PAID FOR, 2026-10-11)**: treefmt-nix's
  templ program wraps `templ fmt` with nixpkgs' DEFAULT `pkgs.go`
  (1.26.8 at this lock) — older than this repo's floor, so the sandboxed
  pre-commit check tried a forbidden toolchain download and failed. Fixed
  in flake.nix by force-overriding `settings.formatter.templ.command`
  with a writeShellApplication carrying OUR `go`. Upstream fix belongs in
  go-nix-helpers (templ formatter should resolve goPkg when enableTempl).
- **golangci exclusions added (2026-10-11)**: contextcheck in internal/web
  (templ-generated closures don't thread r.Context; framework false
  positive), noctx in tests (httptest.NewRequest stdlib pattern), and the
  godoclint "more than one godoc" templ-header false positive (fleet
  precedent: samber-do-auditlog). `.golangci.yml` `run.go` must track
  go.mod's floor — govalid reads it. Removed `tagliatelle.case.use-case`
  (invalid for the installed golangci-lint schema; `golangci-lint-config-verify`
  rejected it and the runner terminated with "configuration contains invalid
  elements").
- **license-check RESOLVED (2026-10-11 23:23)**: the note previously
  recorded here (sandbox `go` 1.27.0 vs go-appkit's 1.27.1 floor, buildflow
  binary 200ab99) is gone because its own fix condition landed: the binary
  was rebuilt (now at BuildFlow HEAD) and `buildflow -s license-check` is
  green (13.6s, 1/1). The same rerun closed the 22:26 full pass's only
  failure: `buildflow -s nix-build` green (23.6s) after load-98+ starvation.
- **dprint-format: markdown tables must be dprint-padded (PAID FOR,
  2026-10-11 23:24)**: a TODO_LIST table whose cells were padded to a
  narrower width than its longest row failed `dprint-format` (exit 20) and
  BLOCKED 7 downstream steps (the gate DAG stops at the first failure —
  nix-build never ran). Fix: `nix develop -c dprint fmt <file>` (dprint
  re-pads every row to the widest cell; content byte-identical, only
  whitespace). Hand-maintained table padding is the failure class — run
  dprint fmt after any manual table edit, don't eyeball it.
  (2026-10-12: dprint joined the devShell packages, so the documented
  command works as written; before that it needed
  `nix shell nixpkgs#dprint --command dprint fmt`.)
- **go-generate transient "updates to go.mod needed" (2026-10-11 23:27)**:
  one full-gate run failed `go-generate` with `go: updates to go.mod
  needed; to update it: go mod tidy` while `go build` and `go mod tidy`
  were simultaneously clean; a standalone step rerun and the next full
  gate both passed with zero changes. GOMODCACHE is the SHARED
  `/mnt/buildcache/go-mod` mount — under concurrent sessions the first
  suspect is a cache race, not the module. Rerun the step before
  diagnosing; only treat it as real if `go mod tidy` shows a diff.
- **Gate closed green (2026-10-11 23:30)**: full
  `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode
  dev` = 73 success, 0 failed, 0 skipped; nix-build ✔ (737 ms, FOD cached
  via the vendor-hash verification build). All remaining findings are the
  documented advisories above.
- **NEVER edit the tree while a gate run is in flight (PAID FOR,
  2026-10-12 run #4)**: a mid-run docs edit re-dirtied the source, nix
  re-evaluated the `-dirty` derivations mid-run and started a SECOND
  go-modules + full build next to the still-running test steps; nix-build
  was killed 8s in ("OOM or timeout"). The DAG runs heavy steps in
  PARALLEL — every extra dirty rebuild competes with them. Rule: write
  docs BEFORE launching the gate or after it exits; the tree is frozen
  for the duration.
- **test-race exposed a REAL test teardown race (PAID FOR, 2026-10-12
  run #4)**: `TestSqliteEndToEndAndReplay` fired `Start(hostCtx)` in a
  bare goroutine and never waited for it — `defer sys2.Close()` runs
  BEFORE `t.Cleanup(stop)`, so `Host.Stop()` (spawned by `System.Close`)
  wrote host state while `Start` was still reading it; the race detector
  caught it only under package-wide shuffle + parallel load (not
  reproducible with `-run` alone, 3 attempts). Fix: `startProjections`
  returns `stopAndWait` (cancel + `<-stopped`), called explicitly before
  EVERY `System.Close`, with `t.Cleanup` as the fatal-path net.
  Rule: any test that starts a background host must own a waitable stop,
  and the gate's shuffle seed in the step log is the reproduction tool
  for order-dependent flakes.
- **Gate closed green in FULL mode (2026-10-12 00:45)**: first full-mode
  devShell pass — `buildflow --fix --build-mode full --max-time 15m` =
  74/74 steps, 0 failed (test-race ✔, test-coverage ✔ incl. -race -cover,
  nix-build ✔, license-check ✔, go-generate/govalid ✔ on devShell go
  1.27.2). Full mode needs `--max-time ≥ 15m`: at the previous session's
  5 m cap, nix-build was killed by the budget, not by OOM. Remaining
  findings are the documented advisories above.
- **agents-md-size line counter over-counts trailing-newline files (source
  verified, 2026-10-15)**: BuildFlow counts AGENTS.md with
  `strings.Count(data, "\n") + 1` (discovery/doctor/checks/agents_md.go:51,
  also feeds go-structure-linter's error finding) — for a POSIX file ending
  in `\n` that is +1 phantom line, so a 220-line file reports 221 and the
  gate errors. Repo-side convention until fixed upstream: keep AGENTS.md at
  ≤219 `wc -l` lines (the tool then sees ≤220). Upstream fix: drop the `+1`
  when the data ends with `\n` (or scan lines). The 2026-10-15 trim moved
  the buildflow-dev-gate bullet to one line for this.
- **Gate closed green, post-lint-zero (2026-10-12 11:36, run #5)**: full
  `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode
  dev` = 75 success, 0 failed, 0 skipped (+3 via config) in 55.3s —
  launched at load 6.5/8.6/13.4 (1/5/15-min), tree committed and
  UNTOUCHED for the duration; test-race ✔ (after the startProjections
  stop-and-wait fix above), test-coverage ✔ (web 72.8%), nix-build ✔
  (40.3s), nix-flake-check ✔ ("all checks passed": pre-commit, treefmt,
  vendor-hash incl.). Findings remaining are exactly the documented
  advisories. This green closed R1 and unlocked R4 (v0.1.0).
- **Gate load policy (evidence, 2026-10-12)**: full gate ONLY when 1-min
  load < 15 AND 15-min < 40; otherwise scoped-verify the suspect steps
  individually and defer the full run. Paid for: runs #1/#2 starved at
  1-min ~47-54 (DIFFERENT steps each time — go-mod-update+lychee, then
  test-coverage+1); run #4 died in nix-build after a mid-run tree edit
  doubled the dirty derivations beside running tests. At/under the
  thresholds, three consecutive runs (23:30, 00:45, 11:36) closed green.
- **oxfmt/oxlint un-minify vendored `*.min.js` (PAID FOR, 2026-10-12)**:
  BuildFlow's JS formatter provider `oxfmt` triggers on `**/*.js`
  (`webFilePatterns`) with NO `*.min.js` exclude — the `*.min.js` default
  lives in the JS _profile_ `DefaultExcludes`, which the provider never
  consumes; oxlint's `jsFilePatterns` has the same gap. Two gate runs
  pretty-printed the vendored `internal/web/static/htmx.min.js`
  (1 line → 3,415, daemon committed one copy at 939f194) and run #3's
  oxlint reported 1 finding against it. Fix (repo-owned):
  `.buildflow.yml` now carries a global `exclude: ["*.min.js"]` —
  `MatchExcludePattern` globs basenames, so the nested path matches —
  plus the pinned bytes restored from history
  (`git show bd51d29:internal/web/static/htmx.min.js`). Verified: scoped
  `buildflow -s oxfmt` / `-s oxlint` report `filesScanned: 0` and the file
  survives. Rule: vendored browser assets are exclude-pinned, never
  reformatted in place.
- **templ fmt import-sort couples the pre-commit check to hand edits
  (PAID FOR, 2026-10-12)**: the flake's sandboxed pre-commit check runs
  treefmt → templ fmt in fail-on-change mode, and templ fmt SORTS Go
  imports inside `.templ` files. `dashboard.templ` carried
  `forms` before `feedback` after hand edits, so `checks.pre-commit`
  failed "unexpected changes detected" and took down nix-build +
  nix-flake-check with it (fast, deterministic — not load starvation).
  Fix: `nix develop -c templ fmt ./internal/web/`, then
  `nix build .#checks.x86_64-linux.pre-commit` standalone (exit 0).
  Rule: run templ fmt after ANY hand edit to a `.templ` import block,
  before spending a gate run.
- **statix suppression: file-level `ignore`, after a five-day green-but-dead
  lane (2026-10-12 → 2026-10-17)**: the repeated-attribute-set-keys hint
  (W20 `repeated_keys`) fires 15× on `deploy/nixos/module-draft.nix` — the
  sealed design record superseded by the landed NixOS module in
  nix-international-telephony. The repeats are intentional structure, not
  debt, so that ONE file is ignored via root `statix.toml`
  (`ignore = ["deploy/nixos/module-draft.nix"]`) and the rule stays live
  everywhere else. The original `disabled = [20]` (integer) was a TOML
  parse error, and statix exits 0 on a config parse error without linting
  anything — this gate's statix step read green while the check had been a
  no-op since the day the config landed (the only clue: the "config error"
  line on `statix check .`'s stderr). Full statix contract: the statix
  entry in gotchas.md. If a real repeated-keys smell shows up elsewhere,
  statix now actually says so.
- **The auto-commit daemon races the gate's repair steps (paid for,
  2026-10-12)**: go-mod-normalize rewrote go.mod mid-run, the daemon
  committed the rewrite seconds later, and go-generate then failed with
  "go: updates to go.mod needed; go mod tidy" against a tree that was
  tidy again by the time anyone looked. Same class: a gate run that
  starts on a dirty tree builds `-dirty` derivations, so every daemon
  sweep re-dirties and forces a rebuild (the repeated nix-build
  OOM/timeout kills during a concurrent session's bursts are this,
  amplified). Triage rule: after a gate failure, check `git status` and
  `git log -1` FIRST — if a daemon commit landed mid-run, re-run the
  failed step on the settled tree before diagnosing deeper.

- **Gate tools trail module toolchain floors (paid for, 2026-10-15)**: the
  go-appkit v0.9.0 bump moved go.mod's floor to 1.27.2, and the full gate
  then failed TWO steps — `go` and `govalid` — with "requires go >= 1.27.2
  (running go 1.27.0)": buildflow's bundled tool binaries are compiled in
  its own nix closure, which still carried go 1.27.0. The repo's own
  environments were already fine (devShell go_1_27 = 1.27.2, shell go
  auto-switches), so `go build/test`, `golangci-lint`, and `nix build`
  (exit 0, `pbx-lens 0.2.0` printed from the packaged binary) all passed
  individually — the documented per-casualty fallback. `buildflow upgrade`
  refuses (dirty-checkout build has no install channel). Fix belongs in
  the BuildFlow repo (rebuild from a clean checkout so the closure picks
  up current nixpkgs go); filed as BuildFlow#56. Until it lands: after
  ANY go.mod floor bump, expect the buildflow gate's go/govalid steps to
  fail first and verify them individually per this file's doctrine.

- **Tooling triage bundle (2026-10-17, TODO row closed)**: the recurring
  "N tools unavailable (health check failed)" warning is dispositioned.
  `buildflow doctor` marks 32 fleet-wide tools missing, but the gate only
  ever runs REGISTERED providers with matching files — for this repo that
  set is small, and `buildflow --dry-run` is the arbiter (it warns on
  dead `skip_steps` entries, which proved eslint/markdownlint/prettier are
  NOT registered here: their doctor ✗ marks are cosmetic, never gate
  steps; markdownlint was additionally trial-run via
  `nix run nixpkgs#markdownlint-cli2` and its MD013/MD060 defaults fight
  the house format's 650+-char evidence tables — a deliberate non-adoption
  even if a provider registers). Applied:
  (1) `shellcheck` joined the devShell packages (trial run: ZERO findings
  on scripts/*.sh — pure value);
  (2) `.buildflow.yml` skips `lychee` (needs the fleet GITHUB_TOKEN policy
  first — GitHub links 401/429 without it; the decision line: exclude
  until a fleet-wide token story exists, revisit upstream) and
  `nix-build-verify` (12/12 historical failures on this machine — the
  equivalent check runs as the MANUAL `nix build` step of every receipt
  procedure, and statix/deadnix/nix-flake-check still cover the nix lane
  in-gate);
  (3) accepted as-is: `nix-hash-fix`'s 82% failure record (environmental —
  load/OOM/tmp-disk, and it remains the sanctioned repair mechanism), the
  cargo/python/js-ts/docker/proto doctor marks (no matching files in this
  repo), `codespell` (on-demand-only by design), `gci` (golangci-lint runs
  it internally when configured), `vulnix` (owned by the R27.x advisory
  row; CI's govulncheck lane is the trusted CVE gate), and the stale
  binary-freshness advisory (P1-T12, owner-gated rebuild).
  Verification: `buildflow --dry-run` shows 37 ok / 1 warn / 0 fail with
  2 user-intent skips and no dead-config warnings.
