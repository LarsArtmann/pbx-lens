# go-daemon — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-daemon` v0.4.0 (checkout last commit 2026-11-04; single module) · **Status in pbx-lens:** **NOT USED** (not in the graph)
**Verdict:** **REJECT** — its sd_notify slice is already owned by go-appkit/systemd; its unix-socket premise contradicts the deployment shape.
**Fleet report:** [index.html](index.html)

## What the library offers

- Unix-socket `SocketServer`: stale-socket cleanup, 0600 perms, lifecycle hooks, graceful drain.
- Standalone sd_notify (READY/STOPPING/WATCHDOG), JSON/CBOR content negotiation, WPT-conformant SSE parse helper, flight recorder, unix-socket HTTP clients. (Deliberately no pidfiles/detach — systemd owns supervision.)

## Why pbx-lens rejects it

1. **The sd_notify slice is taken.** `go-appkit/systemd v0.1.0` + `systemd.Install` (go.mod:8; `cmd/pbx-lens/serve.go:12,267`) provide READY/STOPPING/WATCHDOG with a no-op fallback outside the manager; signal handling + drain come from appkit `svc.Run` (serve.go:212). Adopting go-daemon would put two sd_notify implementations in one binary.
2. **Wrong service shape:** pbx-lens is a TCP HTTP service behind a systemd unit (`deploy/systemd/pbx-lens.service`, htpasswd + LAN posture), not a `/run` socket daemon with per-connection negotiation.
3. **The extras are unused surface:** content negotiation (UI is htmx/HTML + JSON endpoints), SSE parse helper (R9 rejected SSE), flight recorder (go-appkit's flightrecorder module was likewise declined in the 2026-10-12 audit).

## When this verdict flips

Only if pbx-lens sprouts a second, local control interface (e.g. a root-only `/run/pbx-lens.sock` admin socket for a local CLI). Today `pbx-lens serve` + systemd is the whole story.

## How to verify

```console
rg -n "systemd.Install|NOTIFY_SOCKET" cmd/pbx-lens/serve.go | head -4
rg -n "go-daemon" go.mod go.sum   # absent
cat deploy/systemd/pbx-lens.service | head -15
```
