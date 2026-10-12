> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

`testkit.Serve` cleans up through the full production `Shutdown`, so every test pays the 5s default `DrainDelay` — and the `Serve(tb, *Service)` signature takes an already-built service, so a test cannot avoid it. Ask: a config-taking constructor that defaults `DrainDelay` to `NoDrainDelay`, or at least a doc line naming the cost and the escape hatch.

## Problem

`testkit/testkit.go` at v0.8.0: `Serve` registers cleanup that enters `stop` (`testkit.go:179`), which calls the full drain:

```go
if err := ts.service.Shutdown(ctx); err != nil { // testkit.go:181
```

`Service.Shutdown` waits out `DrainDelay`, defaulting to 5s (`config.go:18`: `defaultDrainDelay = 5 * time.Second`, applied in `NewService`). But `Serve`'s signature is:

```go
func Serve(tb testing.TB, svc *appkit.Service) *TestServer // testkit.go:80
```

— the service is already built, so the config is already applied. A test only escapes the 5s if it happens to know, before constructing the service, that it must set `DrainDelay: appkit.NoDrainDelay` itself. Nothing in the `Serve` doc comment mentions it.

## Measured cost

One testkit-based e2e test in a real consumer (pbx-lens, `cmd/pbx-lens/serve_test.go`):

```text
ok  github.com/larsartmann/pbx-lens/cmd/pbx-lens  5.035s
```

— 5.0s of the 5.0s package is the drain wait. Suites with N `Serve` tests pay 5N seconds.

## Suggested fix

Either shape works, no strong preference:

- [ ] `ServeConfig(tb testing.TB, cfg appkit.ServiceConfig) (*TestServer, *appkit.Service)` — applies test-safe defaults (`DrainDelay: NoDrainDelay` unless the caller set it) and builds the service, keeping `Serve` for drain-behavior tests that pair with `DrainWindowProbe`
- [ ] Minimally: extend `Serve`'s doc comment with the 5s-per-test cost and the `NoDrainDelay` escape hatch

Checked open and closed issues for "testkit"/"drain" — no existing report.

💘 Generated with Crush
