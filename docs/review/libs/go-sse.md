# go-sse — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-sse` v0.7.0 (+ `ssetest` v0.5.0, `sseparse` v0.2.1; checkout last commit 2026-11-04) · **Status:** **INDIRECT only** via `go-cqrs-lite/metaengine/v4 v4.20.0` (go.mod:94, `go mod why` verified) · **Status in UI:** NOT USED
**Refreshed 2026-11-12:** pin still v0.7.0 indirect (go.mod:119) == latest tag, verified via `git fetch --tags` + `git tag --sort=-v:refname` in `/home/lars/projects/go-sse` (only two chore commits after v0.7.0). `rg "text/event-stream" internal/ --type go` still returns zero surfaces → verdict **REJECT/INDIRECT** stands (R9 htmx-polling doctrine unchanged); no score impact.

**Verdict:** **REJECT** — a well-built library whose premise (server push) this project deliberately declined.
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- Allocation-minimized SSE wire types + serializer.
- `Stream` (headers/heartbeat/Last-Event-ID replay), generic `Broadcaster[T]`, filtered subscriptions, `EventStore` + replay.
- Companion modules: WPT-conformant parser (`sseparse`), test harness (`ssetest`).

## Why pbx-lens rejects it

1. **R9 decided this.** The UI is htmx polling by recorded decision: `/frag/dashboard` 30 s, `/frag/staleness` 60 s, runtime self-hosted (`AGENTS.md` serve/UI section). Freshness is a JSON poll endpoint (`internal/web/freshness.go:24`); ops is polled JSON (`internal/web/ops.go`). No push consumer exists anywhere in the product.
2. **Push would fight the deployment posture:** the 2026-10-12 HTTP-stack audit explicitly rejected the appkit realtime/SSE-hub family, and the htpasswd/loopback single-operator surface has no broadcast audience. SSE also complicates the unbounded-stream doctrine (docs/http-stack.md) rather than simplifying it.
3. **The indirect presence is correct:** metaengine offers SSE subscribers for live-tail projection modes pbx-lens never enables; pbx-lens pays nothing for it (blank dependency of its pin).

## When this verdict flips

Only if a genuinely push-shaped feature appears (e.g. live multi-client dashboards). Then go-sse is the obvious in-house choice — its Broadcaster/EventStore/replay match the freshness-model needs — but that is a product change, not a library gap.

## How to verify

```console
go mod why github.com/larsartmann/go-sse   # → metaengine/v4 → go-sse
rg -n "text/event-stream" internal/ --type go   # zero SSE surfaces
rg -n "PollLoop|poll" internal/web/freshness.go | head -3
```
