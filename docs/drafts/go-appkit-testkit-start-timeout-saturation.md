> [!NOTE]
> **LIVE DRAFT — do NOT file yet.**
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation (the 2026-10-18 flake triage, pareto plan T6), not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# testkit: the 5s `startTimeout` flakes under machine saturation — same class `leakSettleTimeout` already outgrew

## The constant and its loop (verified at v0.9.0)

`testkit/testkit.go` pins:

```go
// startTimeout bounds how long Serve waits for the listener to come up.
startTimeout = 5 * time.Second
```

and `Serve` polls for start with a tight sleep:

```go
deadline := time.Now().Add(startTimeout)
for !svc.Running() {
	if time.Now().After(deadline) {
		tb.Fatal("testkit.Serve: service did not start within 5s")
	}
	time.Sleep(time.Millisecond)
}
```

## Observed failure

One CI failure in `pbx-lens` (`TestCacheSemanticsHardening`, `go test -race` with
all packages in parallel): the test's reported duration was **5.06 s** — the
signature of this exact deadline — with the service start delayed past 5 s by
scheduler starvation (11 race-instrumented test binaries saturating the runner).
Every run before and after, local and CI, is green; the one-off shape matches the
saturation class this file's own `leakSettleTimeout` comment documents and fixed
(2s → 5s → 10s after two proven false-leak failures, 2026-09-29 and 2026-10-12).

`startTimeout` is the same medicine applied to a sibling constant that never got
the dose: `leakSettleTimeout` grew 5× with recorded proof; `startTimeout` still
holds the value the leak loop outgrew at its FIRST proven flake.

## Why the 1ms sleep makes it worse

Under saturation, `time.Sleep(time.Millisecond)` routinely over-sleeps to
scheduler-quantum delays, so the loop checks `svc.Running()` far less often than
nominal, while the goroutine that flips it waits behind the same queue — the
deadline is consumed by descheduling, not by the service being genuinely broken.

## The ask

Either (either is fine; both is better):

1. **Raise the bound** — 15s mirrors the `leakSettleTimeout` headroom doctrine
   (early-exit green path untouched: a healthy start flips in milliseconds).
2. **Expose the knob** — a package-level var (e.g. `StartTimeout`) or a
   `ServiceConfig` field, so saturation-heavy suites can tune without an
   upstream release, matching how consumers already scale `leakSettleTimeout`
   via the documented constants.

## Local reproduction note

`go test -race -count=50 -run TestCacheSemanticsHardening` on a saturated
machine (load 140-200) at the pin: [loop result pending — fill before filing;
green-under-saturation does not refute the CI one-off, it bounds the frequency].

## Consumer context

`pbx-lens` uses `testkit.Serve` in `cmd/pbx-lens/serve_test.go` (4 call sites)
through go-appkit v0.9.0; the flake triage lives in the repo's
docs/status/archived/2026-10-17_01-45 report (§ postscript).
