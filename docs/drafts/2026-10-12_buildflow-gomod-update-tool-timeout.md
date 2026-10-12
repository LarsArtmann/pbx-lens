> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

When the box is under heavy load, the `go-mod-update` compile gate gets killed by the step deadline with zero compile output — and that kill is classified as a compile failure: go.mod/go.sum are silently reverted and the run reports "Dependency bump introduced compile errors". The tree was fine. The deadline error is already typed (`tool.timeout`, Transient) but the gate's error handling never asks for it.

## Problem

BuildFlow at `51d31b1a4`. The chain, source-verified:

**1. The command runner types deadline kills** — `tools/providers/cmdrun_streaming.go:84`:

```go
// cmdrun_streaming.go:84 (streamCommandOutput)
if errors.Is(ctx.Err(), context.DeadlineExceeded) {
	return output, errorfamily.Wrap(waitErr, errorfamily.Transient, "tool.timeout",
		hangPhaseDescription(toolName, spawnedAt, emitter.firstOutputAt()))
}
```

**2. The compile gate classifies two special cases while the error is raw, then wraps everything else generically** — `tools/gomod/update.go:289-302`:

```go
// update.go:289 (compileGate)
if floor, ok := ClassifyToolchainFloorErr(err); ok {
	return NewToolchainFloorError(floor, "go build ./... (compile gate)")
}

if vi, ok := ClassifyVendoringInconsistencyErr(err); ok {
	return NewVendoringInconsistencyError(vi, "go build ./... (compile gate)")
}

return errorfamily.WrapInfrastructure(err, "gomod.update",
	"go build ./... failed after dependency upgrade: \n"+output)
```

**3. The caller only escapes the revert path for those two classes** — `tools/gomod/update.go:199-251`. A `tool.timeout` matches neither, so it falls through to:

```go
// update.go:238
revertErr := snapshot.revert()
...
r.Logger.Warnf("Dependency bump introduced compile errors; reverted go.mod/go.sum to previous state. "+
	"Use --dep-update-mode patch for safer updates. Error: %v", buildErr)
```

## Observed instance

pbx-lens, 2026-10-11 23:59, load average 40-85 from unrelated jobs: `go-mod-update` died with the starvation kill ("killed after Ns: no output at all"), the step reported the bump as the cause, and go.mod/go.sum were auto-reverted. `go build ./...` on the reverted-but-identical tree was clean immediately after — the only thing that had failed was the machine. The same session lost four other steps to the same starvations, each a different step per run (documented in pbx-lens `docs/buildflow-dev-gate.md`).

The revert is the worst part: it is silent state destruction of a legitimate bump. Nothing was wrong with the dependency change; the environment starved the gate.

## Fix / Proposal

Classify the deadline kill in the same classify-while-raw pass as the floor and vendoring cases — a `tool.timeout` Transient from the gate means the bump was never evaluated:

1. In `compileGate`, check for the `tool.timeout` error (family Transient, key `tool.timeout`) before the generic wrap and return it typed, e.g. a `NewToolTimeoutError` sibling of `NewToolchainFloorError`.
2. In `updateSingleModule`'s `buildErr` branch, return it next to the `IsToolchainFloorError` case: **no revert** (nothing was evaluated, and reverting discards a legitimate bump) and no "bump introduced compile errors" framing — the honest message is "the compile gate was killed by its deadline under load; rerun the step".

This is the same contract the file already states for the floor case: "without the misleading 'dependency bump broke compilation' framing (the bump was fine, the environment is not) and without reverting".

💘 Generated with Crush
