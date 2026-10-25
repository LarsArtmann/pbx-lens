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
  statix now actually says so. The rejected alternative (restyle
  `deploy/nixos/module-draft.nix` to avoid W20 entirely) stays rejected —
  the sealed design record is not restyled for a linter (01-45 §f24).
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

- **Gate closed green in FULL mode from the clean checkout — the T2
  receipt (2026-10-18 06:17, commit b4a9970, tree clean and daemon-quiet
  throughout)**: `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow
  --build-mode full --max-time 15m` = **57 success, 0 failed, 0 skipped
  (+5 via config)**, exit 0, wall 1m0.8s, run `20261018-041735-4709f40d`,
  launched at load **14.41/35.00** — strictly inside the load policy. A
  first run 6 minutes earlier (06:11, run `20261018-041204-bc0e3b6d`, wall
  2m18.7s cold nix) also closed 57/0/0 with 1-min load 17.24 — 2.24 over
  the 1-min threshold at launch after the 79-minute window wait; zero
  starvation casualties either run and exit 0 (not 75), so both stand,
  the strict one is the banked receipt. test-race ✔ (27.7s),
  test-coverage ✔ (26.0s), nix-build ✔, nix-flake-check ✔,
  license-check ✔, and go-generate/govalid ran green IN-GATE — the
  BuildFlow#56 tool-floor fix holds (binary ef32fa0 reports go1.27.2).
  The literal count is 57, not the 74 of the 2026-10-12 receipt: the
  pipeline compiles 118 tools → 85 DAG nodes and the executed set drifts
  with the fleet — bank the literal numbers per run. Findings: 292 across
  14 tools; the receipt is CI-equivalent on lint because golangci-lint
  reports ZERO findings (the advisory-gap bullet above is the doctrine).
  The receipt surfaced two real nits, fixed immediately: dprint-format on
  FEATURES.md (table padding) and prettier-format on the GENERATED
  styles.css — excluded in `.buildflow.yml` next to `*.min.js` (scoped
  `buildflow -s prettier-format` re-run clean). SQLITE_BUSY
  resume-checkpoint warnings were state-DB contention noise; no step was
  affected. The timing-regression verdicts at run end are load variance
  against the baseline, not step failures.
- **Gate tools trail module toolchain floors (paid for, 2026-10-15; CLOSED
  2026-10-18)**: the go-appkit v0.9.0 bump moved go.mod's floor to 1.27.2,
  and the full gate then failed TWO steps — `go` and `govalid` — with
  "requires go >= 1.27.2 (running go 1.27.0)": buildflow's bundled tool
  binaries are compiled in its own nix closure, which still carried go
  1.27.0. The repo's own environments were already fine (devShell go_1_27
  = 1.27.2, shell go auto-switches), so `go build/test`, `golangci-lint`,
  and `nix build` all passed individually — the documented per-casualty
  fallback that made the v0.2.0-era receipt a workaround instead of a
  literal pass. Fix landed upstream: BuildFlow#56 CLOSED 2026-10-17, the
  fleet binary rebuilt (ef32fa0, `buildflow version` reports go1.27.2),
  and the 2026-10-18 T2 receipt above ran go-generate/govalid green
  IN-GATE — the workaround is retired. Lesson kept: after ANY go.mod
  floor bump, expect the gate's tool steps to trail until the fleet
  binary rebuilds; if a stray `go-mod-*` step ever rewrites the go.mod
  directive to `go 1.27`, restore it (`git restore go.mod`) — a 1.27
  directive breaks `go list`.

- **golangci findings are ADVISORY in the gate, FATAL in CI (paid for,
  2026-10-18, the ca31221 §e4 audit)**: the 00:37:27Z dev-gate green and
  the CI lint red landed on the SAME commit, and run
  `20261018-003727-d08c3565` explains it — its step table shows
  `golangci-lint success 1.3s` while its findings section carries the
  exact CI trio (`gocyclo ×1, ineffassign ×1, modernize ×1`). Buildflow
  reports lint findings as non-fatal advisories; CI's golangci action
  exits non-zero on any finding. NOT a stale pass and NOT version skew:
  no-cache dry-runs of dev AND full both plan the identical
  `golangci-lint run --allow-parallel-runners ./...` (v2.14.0 in the
  devShell = the CI pin; repo config read in all three lanes). Receipt
  consequence: "N/N steps success" is NOT lint parity — a receipt is
  CI-equivalent only when the run's findings section shows ZERO
  golangci-lint entries (`buildflow explain <run-id> | grep
  'golangci-lint:'` empty). The pre-push `golangci-lint run ./...`
  discipline stays the belt to this braces.

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
  2 user-intent skips and no dead-config warnings. Re-verified 2026-10-23
  (docs-health pass; the `29 tools` receipt had aged): `buildflow doctor`
  = 65 passed / 3 warnings / 30 failed, and every failure is a fleet tool
  that lives in the devShell or targets a language with no files here
  (dprint/shellcheck/prettier/pytest/ruff/tsc/vulnix/lychee/... — the
  dispositions above plus "devShell-only, not on PATH"). The warnings are
  the BuildFlow binary predating its HEAD, go.mod freshness, go-line
  churn, the FOD-hash advisory, and AGENTS size. Counts drift with the
  fleet's tool list, never with this repo's policy: judge a doctor run by
  whether every mark fits a disposition, not by the number.

### The gate's "N tools unavailable (health check failed)" list, named (2026-10-23)

`buildflow --dry-run --verbose` names every entry; the count is 10 and each
one has a standing disposition. Nothing here is repo work.

| Tool                                           | Why it reports unavailable                                                                                                                           | Disposition                                                                                                                                                                                                                                                                                       |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `jest-test`, `vitest-test`, `js-coverage` (c8) | no JS test runner: this repo has no `package.json` project                                                                                           | accepted — the probe scripts are self-contained instruments, not a test suite; their contracts are proven by `--self-test` lanes and the browser probes                                                                                                                                           |
| `knip`, `madge`                                | no JS dependency graph to analyse                                                                                                                    | accepted — no JS module graph exists                                                                                                                                                                                                                                                              |
| `publint`, `svelte-check`, `vue-tsc`           | no publishable JS package, no Svelte/Vue sources                                                                                                     | accepted — never applicable                                                                                                                                                                                                                                                                       |
| `interrogate`                                  | no Python sources                                                                                                                                    | accepted — never applicable                                                                                                                                                                                                                                                                       |
| `oxlint-auto-configure`                        | the repo now carries `scripts`-owning `.oxlintrc.json`, which differs from the fleet-generated config (advisory: "drift detected", nothing modified) | accepted and INTENTIONAL: the file enables `no-undef` with browser+node globals (the 2026-10-23 `bin is not defined` class); regenerating from the fleet default would replace an explicit rule set with an implicit one. Reconcile only if the fleet starts shipping a config with `no-undef` on |

The five "skipped via config" entries are the documented on-demand tools
(`markdown-lint`, `gitleaks`, `codespell`) plus `lychee`/`nix-build-verify`
from `.buildflow.yml`. Re-run `buildflow --dry-run --verbose` after any
fleet tool-list bump and add the new name here instead of re-diagnosing it.

### Detect-only findings: the 2026-10-25 triage (fix / document / accept)

Three steps report findings as ADVISORIES (they never fail the gate); the
counts drift with code size, so each has a standing verdict instead of a
number to chase. Re-read this section rather than re-diagnosing them.

| Step              | Count | Verdict                                                | Why                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ----------------- | ----- | ------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `art-dupl`        | 101   | **accept — the CI gate is authoritative**              | BuildFlow runs the tool with its default lens (no `--min-lines`), so the report is dominated by 1-line signature/type matches between unrelated functions. The repo's CI gate (`art-dupl check -t 4 --min-lines 2` vs a ZERO-group baseline) is green: 0 clone groups, verified 2026-10-25. Mechanics: docs/art-dupl-baseline.md                                                                                                                                                                                                                                      |
| `branching-flow`  | 116   | **accept — one false-positive class + taste calls**    | 21× "Pointer dereference of 'db' may panic if nil" (and the same shape for `maxAge`, `dryRun`, `keyFile`, …): every flag pointer in `cmd/pbx-lens` comes from `flag.*`, which never returns nil, and a nil would panic at startup in a CLI, not in a request. The rest are mixin suggestions (two throttler structs, two ops rows) and "brand these two string params" notes on values that ARE branded types (`calls.SourceKind`, `calls.NativeID`) — the analyzer sees the underlying `string`. No fix without a lint-suppression mechanism this repo does not have |
| `file-size-check` | 16    | **accept — the 350-line ceiling is a fleet heuristic** | The shape of this repo is many small files plus a few cohesive ones: `internal/web/server.go` (654) is the route table + handlers, `cmd/pbx-lens/serve_test.go` (860) is one end-to-end stack suite, `internal/web/queries.go` (608) is the read-model query layer. Splitting them would scatter one concern across files to satisfy a line count. Revisit when a file's concern genuinely forks (the art-dupl/branching signals are the coupling evidence that matters, not line count)                                                                              |

The `cqrs-lint` half of the same triage pass lives in
docs/cqrs-lint-policy.md — every finding there is a DECISION, and the
2026-10-25 pass added F021/F026/F031 plus the time-typed-field entries.

## The gate entry point is the wrapper (2026-10-25)

Run the gate through `scripts/poll-gate.sh`, never bare:

```console
nix develop -c bash scripts/poll-gate.sh -- env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode full
```

The wrapper polls the 15-minute load average below the policy ceiling,
then `exec`s the gate IN THE SAME PROCESS — so the START LOAD a receipt
records is the load the loop itself verified. A `while`-loop-plus-`&&`
leaves a poll→launch gap (the load can cross the ceiling between the
check and the exec — two sessions paid for exactly that before the
wrapper existed). `--once` exits 75 when not ready; everything after
`--` reaches the gate verbatim; `--self-test` proves the over-policy,
ready, and unreadable-load paths. AGENTS.md's Commands block points
here.

## The first calm-window FULL receipt (2026-10-25 11:34, run 3)

START LOAD **15m=6.06** (policy < 40), `BUILDFLOW_NO_RESULT_CACHE=1
buildflow --build-mode full` → **60 success / 0 failed / 0 skipped**,
exit 0, wall 2m19s, frozen tree (`095e62c`). The morning's three starved
attempts at 15-min ≈ 100 are superseded. Two same-day runs preceded the
receipt and teach the receipt rules:

- Run #1 (11:26) exited 69 on four REAL erraudit findings — the
  scenario-test rejection sentinels were declared `*errorfamily.Error`
  instead of `error`. That is the gate working, not failing.
- Run #2 (11:31) closed 60/0/0 but on a tree the session's own P2 edits
  moved mid-run — void as a receipt under docs/concurrency.md's
  no-receipt-from-a-moving-tree rule.
- Run #3 ran on the frozen post-fix tree and stands.

A green run on a moving tree is not a receipt; a red run with real
findings is the gate doing its job.
