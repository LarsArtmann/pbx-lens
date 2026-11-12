# go-datastar — adoption-fit report (pbx-lens)

**Date:** 2026-11-11 · **Upstream:** `github.com/larsartmann/go-datastar` v0.7.0 (checkout last commit 2026-11-10; == proxy latest; single module + `broadcast` submodule; built on go-sse; embedded DataStar JS client v1.0.4) · **Status in pbx-lens:** **NOT USED** (not in the graph — `go.sum` carries zero entries)
**Verdict:** **REJECT** — the same recorded decision as [go-sse.md](go-sse.md), one layer up: its entire premise (server push over SSE) is what R9 deliberately declined.
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- The DataStar protocol vocabulary as Go values: every patch (`NewElementsPatch`, signals, scripts) is a first-class `Patch` implementing `Event() sse.Event` — constructible without a live connection, routable through a `Broadcaster`, storable for Last-Event-ID replay, filterable per subscriber.
- `broadcast` submodule (`NewBroadcaster` + replay capacity + multi-instance `WithStore`), `ScriptHandler()` (ETag + Cache-Control, embedded zero-dep client), `datastartest` (SSE parsing, typed decoding, assertions), every error classified with the fleet's error-family vocabulary, templ rendering support.
- vs the official `starfederation/datastar-go` SDK: patches-as-values, broadcast/replay/filtering, classified errors — the architectural bet that pays off only when a server PUSHES.

## Why pbx-lens declines it

1. **R9 is the architecture:** the UI refreshes by htmx polling (`/frag/dashboard` 30 s, `/frag/staleness` 60 s — AGENTS.md serve/UI law). There is no live-push surface to compose patches for. Adopting go-datastar would add the whole SSE stack (go-sse, broadcaster, replay) to serve a need the doctrine says does not exist.
2. **The dashboard is read-only analytics** over a disposable, re-derivable index; 30 s staleness is the honest contract, not a defect. The ops page already surfaces poll freshness (`ageText` renders "unknown"/"never" honestly).
3. **Not even transitively present:** unlike go-sse (indirect via metaengine) or go-idempotency (indirect via loginpage), go-datastar appears nowhere in pbx-lens' module graph — `templ-components/datastar` is likewise unimported (the root pin imports display/forms/feedback/layout/navigation/htmx/icons/utils only).

## Pairing to watch (the one live connection)

`templ-components` `[Unreleased]` (the likely v1.28.0) ships **typed Datastar reactive builders** (`Show`, `Text`, `Bind`, `Signals`, `Computed`, chainable `On` with Prevent/Debounce/Throttle) and the **CSP nonce mode** (`layout.DatastarNonceAttrs` — nonce'd inline scripts instead of `'unsafe-eval'`, browser-proven, ~5.7 µs/eval). If R9 is EVER revisited, the fleet-blessed stack is go-datastar + go-sse + templ-components' builders — this report's flip-trigger fires first, not the library's readiness.

## Flip trigger (falsifiable)

Adopt when a feature genuinely needs server push faster than the 30 s poll — the concrete candidate: **live call-event streaming into the dashboard** (a `call.record` landing should move the volume chart without a poll cycle). Before that: no second trigger exists; polling covers every current surface.

## Version currency

v0.7.0 == proxy latest (2026-11-11); upstream pace healthy (last commit 2026-10-10/11-10 window, released v0.7.0). Nothing to pin; nothing to read for the current architecture.

## How to verify

```console
rg -n "go-datastar" go.mod go.sum                        # zero hits
rg -rn "datastar" internal/ cmd/ --type go               # zero imports
rg -n "text/event-stream" internal/ --type go            # zero SSE surfaces (the R9 proof)
rg -n "R9" AGENTS.md | head -3                           # the recorded doctrine
```
