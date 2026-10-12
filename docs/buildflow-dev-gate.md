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
  23:59) is FILED as BuildFlow#45 (2026-10-12) — until it lands, treat that
  message under load as an environment casualty and rerun the step.
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
  carries `go 1.27.1` because go-appkit v0.8.0's module floor demands it —
  `go mod tidy` reverts any major.minor downgrade. The checker's own
  severity policy treats dep-forced patch floors as correct. If a stray
  `go-mod-*` step ever rewrites the directive to `go 1.27`, restore it
  (`git restore go.mod`); a 1.27 directive breaks `go list`.
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
- **Gate closed green in FULL mode (2026-10-12 00:45)**: first full-mode
  devShell pass — `buildflow --fix --build-mode full --max-time 15m` =
  74/74 steps, 0 failed (test-race ✔, test-coverage ✔ incl. -race -cover,
  nix-build ✔, license-check ✔, go-generate/govalid ✔ on devShell go
  1.27.2). Full mode needs `--max-time ≥ 15m`: at the previous session's
  5 m cap, nix-build was killed by the budget, not by OOM. Remaining
  findings are the documented advisories above.
