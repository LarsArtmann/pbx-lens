# cqrs-htmx — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/cqrs-htmx` root v4.15.0, usermgmt v4.18.0, setup v4.16.0 (commit pace ≈30/day; single root go.mod) · **Status in pbx-lens:** **NOT USED** (not in the graph)
**Verdict:** **ADOPT-PARTIAL (aspirational)** — one concrete, adoptable idea today (the health bridge's projection-readiness check for `/ready`); the rest is either N/A by doctrine or held until the UI gains an action. Pull leaf modules only, never the stack.
**Fleet report:** [index.html](index.html)

## What the library offers

- **Root/v4:** `app.Command`/`app.Query` endpoint composition (decode → Casbin authz → validate → dispatch → HX headers, partial-vs-full rendering, error-family→status mapping, typed variants), templ duck-typing, pagination, embedded htmx v2.0.10 + SSE extension with ETag caching.
- **Modules:** usermgmt (passkeys/TOTP/OAuth2, sessions, lockout), adminui, dashboardui (event-store observability), loginpage, setup (one-call wiring), datastar, offline sync, OpenAPI builder, **health bridge** (projection readiness → go-health), security middleware.

## Why not wholesale

1. **No user management to buy:** single-operator htpasswd (`internal/web/auth.go`); usermgmt/adminui/loginpage target multi-user auth pbx-lens deliberately lacks.
2. **SSE/datastar/offline-sync excluded by R9** (htmx polling doctrine); embedded-htmx assets already owned by `templ-components/htmx v1.26.0` (go.mod:27).
3. **The UI has no command surface:** every route is a query or proxy (AGENTS.md serve/UI law) — the `app.Command` endpoint glue has nothing to compose. The only POST-ish surface today is recording rate-limiting (`internal/web/recording.go:61-62`).

## What IS worth adopting

- **The health bridge's ProjectionReadinessCheck idea:** `/ready` currently reflects only the appkit process; projection worker readiness is invisible to it even though every poll pass drains projections (`internal/cli/poller.go:198`, `DrainProjections` via `internal/cli/system.go:41-74`). A small in-house readiness check that reports `projectionhost.Status()` workers as ready/not-ready is the right-sized adaptation (503 until workers live) — *the idea*, not necessarily the module (the module bridges to go-health, which pbx-lens rejects).
- **The command/query HandlerOption glue + error-family→HTTP mapping** as the reference pattern *if* the UI ever gains an action (e.g. a "trigger refresh now" button). Cross-ref: go-error-family's unused `HTTPHandler` covers the mapping half ([go-error-family.md](go-error-family.md)).

## Adoption guardrails

Leaf modules only (`cqrs-htmx/health/v4` shape), never the root stack — the root pulls Casbin + embedded assets pbx-lens cannot use. Any adoption must not introduce a second composition story next to `system.New`.

## How to verify

```console
rg -n "cqrs-htmx" go.mod go.sum                       # absent
rg -n "Status\(\)" internal/cli/system.go | head -3   # projectionhost status polling
rg -n "POST|PostForm|MethodPost" internal/web --type go | head -5
```
