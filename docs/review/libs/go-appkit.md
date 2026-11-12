# go-appkit — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pins:** `github.com/larsartmann/go-appkit v0.9.0` + `go-appkit/systemd v0.1.0` (go.mod:7-8) · **Status:** USED

> **Refreshed 2026-11-11:** lib latest root tag is now **v0.12.0** (2026-11-07) — three minors ahead of the pin. v0.10.0 (2026-11-06) RELEASED the D4 flip-trigger (Server-Timing default outermost); v0.11.0 raised the go floor to 1.27.3 (pbx-lens matched, `539a136`); v0.12.0 recomposed `Service` on `httputil.Server`. Facts below keep the 2026-11-04 snapshot basis; the v0.10-v0.12 deltas live in the forward-finding and version-currency sections.
> **Adoption score:** **73/100 (high)** — 11 of 15 applicable capabilities (unchanged from the prior report; pin is still v0.9.0, the adoption didn't move).
> **Prior report:** [2026-10-25_go-appkit-templ-deep-dive.html](../../research/2026-10-25_go-appkit-templ-deep-dive.html) (73/100 = go-appkit; the same report's 86/100 belongs to a-h/templ — score-card ambiguity resolved there). This report re-verifies the census at today's HEAD and adds one forward finding; nothing supersedes the prior scores.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

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

## Forward finding (D4) — flip-trigger RELEASED upstream 2026-11-06, pending the pin bump

v0.10.0 shipped exactly what this report flagged: **`servertiming.ServerTimingMiddleware()` sits OUTERMOST in the default chain** (Server-Timing → Recovery → RequestID → Logging → [Timeout] → SecurityHeaders), plus testkit `HTTPGetURL`/`WaitForRunning`/`FreeAddr`, plus `appkit.RegisterClassifications()` (fleet-default error-family classifications — call once at boot). v0.11.0 (2026-11-07) raised the go floor to 1.27.3 and classified run-hook failures via `errorfamily.WrapOnce`. v0.12.0 (2026-11-07) delegated the server loop to a fresh `httputil.Server` per `Start` — a full shutdown no longer poisons a restart. When pbx-lens bumps v0.9.0 → v0.12.0:

1. `serve.go:328` (`web.ServerTimingMiddleware()` in ExtraMiddlewares) becomes deletable — verify semantics first (header stamped at commit; SSE/NoTimeout-safe per the v0.10.0 entry).
2. `serve_test.go` can upgrade to `WaitForRunning`/`FreeAddr` for free.
3. Decide on `RegisterClassifications()` at boot (pairs with the [go-error-family report](go-error-family.md) classification census).
4. Config contract change: configs the delegated server rejects (e.g. `ReadHeaderTimeout > ReadTimeout > 0`) now fail `Start` synchronously — none in pbx-lens' `ServiceConfig`, verified by its green build against v0.12-era upstream.
5. New direct dep of appkit at v0.12: `httputil` (+ server_timing) — already direct in pbx-lens, zero new graph surface.

## Version currency

**Pin behind upstream by three minors (2026-11-11):** pinned v0.9.0 (2026-10-12); upstream v0.10.0 (2026-11-06), v0.11.0 (2026-11-07), v0.12.0 (2026-11-07). systemd v0.1.0 (2026-10-07) remains current. The lag is a deliberate hold (the bump carries the Server-Timing de-duplication + config-contract work above), not neglect — but schedule it: every minor held back accumulates fold-in work, and the v0.10 default middleware removes pbx-lens code rather than adding it.

## Risks

- Low. The go 1.27.3 floor (upstream v0.11.0) matches pbx-lens' floor since `539a136`. No deprecations in the used surface.
- The v0.12 `httputil.Server` delegation changes restart semantics — irrelevant to pbx-lens (single `Run`, no restart path), relevant if the deploy module ever grows one.

## Recommendations

1. Bump v0.9.0 → v0.12.0 on the next deps train: de-duplicate the Server-Timing wiring (pairs with the [server_timing report](httputil-server-timing.md) finding), adopt the new testkit helpers, and decide on `RegisterClassifications()`.

## How to verify

```console
rg -n "appkit\.(NewService|testkit)|svc\.(Mux|Run)" cmd/pbx-lens/serve.go cmd/pbx-lens/serve_test.go
git -C /home/lars/projects/go-appkit tag --sort=-v:refname | head -3   # expect v0.12.0
sed -n '/\[0.10.0\]/,/\[0.9.0\]/p' /home/lars/projects/go-appkit/CHANGELOG.md | head -30   # the released flip-trigger
```
