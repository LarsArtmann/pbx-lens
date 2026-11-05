# go-appkit — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pins:** `github.com/larsartmann/go-appkit v0.9.0` + `go-appkit/systemd v0.1.0` (go.mod:7-8) · **Status:** USED
**Adoption score:** **73/100 (high)** — 11 of 15 applicable capabilities (unchanged from the prior report; pin is still the latest release).
**Prior report:** [2026-10-25_go-appkit-templ-deep-dive.html](../../research/2026-10-25_go-appkit-templ-deep-dive.html) (73/100 = go-appkit; the same report's 86/100 belongs to a-h/templ — score-card ambiguity resolved there). This report re-verifies the census at today's HEAD and adds one forward finding; nothing supersedes the prior scores.

## Verdict

pbx-lens leans on appkit exactly as designed: `NewService` → `svc.Mux` → `svc.Run` owns the HTTP shell, signals, and graceful drain; the `ExtraMiddlewares` chain carries the auth→SecurityHeaders→Server-Timing→Compression→ETag doctrine; health endpoints + `/version` + opt-in `/metrics` come from the shell; `systemd.Install` handles READY/STOPPING/WATCHDOG; and the testkit adoption (the prior report's F1 fix) is intact at all 3 sites, keeping stack tests fast. The four unwired seams are wired-out by documented decisions, not neglect. One delta worth planning: upstream `[Unreleased]` makes Server-Timing a default outermost middleware — which will make pbx-lens's own `web.ServerTimingMiddleware()` wiring redundant.

## Usage census (re-verified)

- **Service shell:** `cmd/pbx-lens/serve.go:150,182,212` (`NewService`, `svc.Mux`, `svc.Run`); `ServiceConfig{Addr, Version → /version, Metrics → /metrics (AllowUnauthenticated opt-in), WriteTimeout: NoTimeout}` (`serve.go:257-272`; NoTimeout:263 — the export/recording streaming contract, docs/http-stack.md:10-15).
- **Middleware chain:** auth → SecurityHeaders → Server-Timing → Compression → ETag via `ExtraMiddlewares` (`serve.go:284,300,328,329` + `internal/web/caching.go:92`).
- **systemd:** `systemd.Install` (`serve.go:12,267`; no-op without `NOTIFY_SOCKET`).
- **testkit:** `testkit.NewService`/`testkit.Serve` ×3 (`cmd/pbx-lens/serve_test.go:59,187,298`; regression note 53-55) — the prior report's 5s-drain → 0.81s fix still holds.
- **Health/ready:** appkit `/health,/live,/ready` routes (docs/http-stack.md; `internal/web` does not import appkit — `server.go:570` is a version comment only).

## Capability disposition (unchanged where the prior report ruled)

| Capability                                                                               | Verdict                                                                                                                     |
| ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Shell, default stack, health, `/metrics` opt-in, `/version`, NoTimeout, testkit, systemd | Fully leveraged (census above)                                                                                              |
| `ReadyCheck` / `OuterMiddlewares` / `StartHooks` / `DrainHooks` / `ShutdownHooks`        | Unwired by design — deps open pre-bind; teardown rides `defer` (prior report F2 "not scheduled")                            |
| `InitLogger`, error re-exports                                                           | Deliberate non-adoptions (documented)                                                                                       |
| errorpages / security / realtime / cqrs / docs / otel / flightrecorder sibling modules   | REJECTED 2026-10-12 with recorded dispositions — still stand; realtime/SSE exclusion re-affirmed by the R9 polling decision |

## Forward finding (D4)

Upstream `[Unreleased]` (checkout CHANGELOG 3-67, master `a14337bc`): **Server-Timing becomes the default outermost middleware**, plus testkit `HTTPGetURL`/`WaitForRunning`/`FreeAddr`. When v0.10.0 ships:

1. `serve.go:328` (`web.ServerTimingMiddleware()` in ExtraMiddlewares) likely becomes deletable — verify semantics (header stamped at commit; SSE-safe) before deleting.
2. `serve_test.go` can upgrade to `WaitForRunning`/`FreeAddr` for free.

## Version currency

v0.9.0 (2026-10-12) **is the latest tag**; 23 days of unreleased master behind it. systemd v0.1.0 (2026-10-07) likewise current. Nothing to bump; the Unreleased items are fold-in watch material for the v0.10.0 window.

## Risks

- Low. The go 1.27.2 floor still matches; no deprecations in the used surface.

## Recommendations

1. Watch for the v0.10.0 release; on bump, de-duplicate the Server-Timing wiring (pairs with the [server_timing report](httputil-server-timing.md) finding) and adopt the new testkit helpers.

## How to verify

```console
rg -n "appkit\.(NewService|testkit)|svc\.(Mux|Run)" cmd/pbx-lens/serve.go cmd/pbx-lens/serve_test.go
git -C /home/lars/projects/go-appkit tag --sort=-v:refname | head -3
sed -n '3,20p' /home/lars/projects/go-appkit/CHANGELOG.md   # Unreleased: default Server-Timing
```
