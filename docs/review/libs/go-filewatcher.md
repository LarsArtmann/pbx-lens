# go-filewatcher — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-filewatcher` v2.4.3 (checkout last commit ≈2026-11-02; single module) · **Status in pbx-lens:** **NOT USED** (not in the graph; `fsnotify` v1.10.1 is already indirect via koanf/go-appkit, go.mod:57)
**Verdict:** **REJECT** — the one watchable file already has a better, deterministic reload mechanism.
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- Recursive fsnotify wrapper: 17+ composable filters, 18 middleware, global/per-path debounce, .gitignore-aware walk, ENOSPC resilience/self-heal, NFS/FUSE polling fallback, content hashing, Prometheus/OTel integration.

## Why pbx-lens rejects it

1. **The watch target is the htpasswd file, and SIGHUP already owns it.** `internal/web/auth.go:142-151` (`watchSIGHUP` → `reload`) is operator-driven, deterministic, and test-pinned; a broken reload **keeps the running table** (design note `auth.go:49-51`) — a safety property that must survive any reload mechanism.
2. **File-watching adds failure modes for zero operator value:** inotify limit edges, a debounce race against atomic-rename credential rotations (the exact pattern an operator's `htpasswd`/deploy tooling uses), and NFS edge cases — all to replace a one-line `systemctl kill -s HUP`.
3. **fsnotify rides the graph already** (koanf config watching inside go-appkit) — but that is go-appkit's business, not pbx-lens'.
4. Rotation frequency here is human-scale (rare); hot-reload latency requirements are seconds, which SIGHUP meets.

## When this verdict flips

Only if the auth surface grows beyond one operator-managed file (e.g. per-source API keys with automated rotation landing as files). Then revisit — but go-credentials-store (see [that report](go-credentials-store.md)) would likely be the better answer first.

## How to verify

```console
rg -n "SIGHUP|reload" internal/web/auth.go | head -6
rg -n "go-filewatcher" go.mod go.sum   # absent
rg -n "fsnotify" go.mod
```
