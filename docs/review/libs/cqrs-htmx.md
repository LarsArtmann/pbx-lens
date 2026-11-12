# cqrs-htmx — utilization report (pbx-lens)

**Date:** 2026-11-04 (audit) · **Refreshed 2026-11-12: FLIP-TRIGGER EXECUTED — the 2026-11-11 operator order adopted the identity stack; verdict below is post-adoption.**
**Upstream/pins (go.mod:7-9,92-93):** direct `usermgmt/v4 v4.20.0`, `loginpage/v4 v4.18.0`, `usermgmt/webauthn/v4 v4.13.0`; indirect riders `v4 (root) v4.18.1`, `identity-model/v4 v4.17.0` (MVS via usermgmt). Latest upstream at refresh: root v4.18.1, usermgmt v4.20.0, loginpage v4.18.0, setup v4.17.0 — **pins current**.
**Status in pbx-lens:** **ADOPTED (leaf modules only, exactly as the 2026-11-04 guardrails demanded)** — identity/login only; root stack, adminui, dashboardui, datastar, offline-sync, OpenAPI, health-bridge, security middleware all NOT imported (verified: `rg "cqrs-htmx" internal/ cmd/` hits only usermgmt/loginpage/webauthn paths).
**Verdict:** **ADOPTED for identity; the rest stays REJECT/N-A** with unchanged reasons. Score: **78/100 (high)** for the adopted surface (see breakdown below).
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers (unchanged census)

- **Root/v4:** `app.Command`/`app.Query` endpoint composition, templ duck-typing, pagination, embedded htmx + SSE extension, error-family→status mapping.
- **Modules:** usermgmt (password-less passkeys/WebAuthn + password factor, sessions, lockout, max-users cap), loginpage (templ login UI), adminui, dashboardui, setup (one-call wiring), datastar, offline sync, OpenAPI builder, health bridge, security middleware.

## What pbx-lens adopted (2026-11-12, `internal/web/userauth.go`, 376 lines)

- **One identity DB, four stores, one handle:** `usermgmt.NewSQLEventStore` + `NewSQLSessionStore` + SQL read models + `storage.NewSQLiteCheckpointStore` (go-cqrs-lite/storage/v4; `SQLiteCheckpointSchema()` hand-applied at `userauth.go:238` because usermgmt only auto-migrates its own tables), then `usermgmt.OptimizeSQLiteDB` (`:192`) and `SetMaxOpenConns(1)` (`:226` — one connection serializes the identity journal).
- **WebAuthn provider:** `webauthn.New(webauthn.Config{RPID, RPDisplayName, RPOrigins})` (`:108`); `deriveAuthDefaults` (`serve.go:458`) maps an IP/localhost listen address to RPID `localhost` because go-webauthn refuses an IP RPID.
- **Single-operator bootstrap:** `MaxUsers: 1` (`:135`) — the first completed signup becomes the operator and registration closes; later `/auth/register` is refused without a session (test-pinned).
- **Lockout:** `usermgmt.NewAccountLockout()` (`:134`, 5 fails/15 min) — per account, distinct from the serve auth-failure throttle (per client address).
- **Request path:** `NewSessionMiddleware(service, cookieName)` → CSRF (httputil/nosurf) → the gate (`buildGate`, `:278`); public paths `/login`, `/auth/*`, `/app.css`, `/robots.txt` (`publicAuthPath`); browsers (HTML Accept or `Hx-Request`) 303→/login, others 401.
- **Routes:** loginpage `panel.Mount(mux, "GET /login")` (`:323`, auto CSRF meta via `httputil.CSRFTokenHTMLMeta`); `NewAuthHandler(service, HandlerConfig{RegistrationRateLimit: 10/min, WebAuthnRateLimit: 30/min}).RegisterRoutes(mux)` (`:325-334`).
- **Serve wiring:** opt-in via `-auth-db` (`serve.go:135-147` — without it serve runs unauthenticated, the localhost dev posture the probes rely on; the sibling NixOS module passes the state path so production is gated); flags `-auth-rpid`/`-auth-origin` (`serve.go:67-78`); `identity.Middleware()` heads the ExtraMiddlewares chain (`:159-163`); `identity.Ready()` (projection statuses live/stopped, `:356`) ANDs into `serviceConfig.ReadyCheck` (`serve.go:175-181`); `identity.Register(svc.Mux)` (`:188-190`); `Close()` = `errors.Join(service.Close(), db.Close())` (`:372`) — never closes the shared DB from the service side.
- **Not-adopted and why (unchanged):** root stack (Casbin + embedded assets + a second composition story next to `system.New` — ADR-0123 forbids), adminui/dashboardui (no operator demand), datastar/offline-sync (R9 htmx-polling doctrine), health bridge (bridges to go-health, rejected; the readiness IDEA was adopted in-house instead — see below), OpenAPI builder (no external API).
- **The 2026-11-04 "worth adopting" row resolved:** the projection-readiness idea landed BOTH ways — the index context got `cli.ProjectionWorkersReady` (failure-vote predicate, CHANGELOG 2026-11-05) and the identity context now reuses `usermgmt`'s own projection-status `Ready()` — no health-bridge module needed.

## Why not wholesale (historical, kept for the refresh rule)

1. Single-operator posture — RESOLVED 2026-11-12: the operator now has a real account, session, and passkey slot; the multi-user machinery (roles, admin UI, OAuth2/TOTP factors) remains unadopted.
2. SSE/datastar excluded by R9 — unchanged.
3. Read-only UI, no command surface — unchanged; the adopted usermgmt routes are the ONE sanctioned write surface (identity), outside the calls domain.

## Adoption guardrails (post-adoption status)

Leaf modules only — HELD: only usermgmt/loginpage/webauthn import paths appear; root stays indirect via MVS with zero direct imports. No second composition root: `OpenIdentity` composes usermgmt stores; `system.New` remains the only event/command composition (ADR-0123).

## Risks

- Low. The identity DB is now the only non-re-derivable state next to the index DB — it rides the nightly backup (sibling module `auth.db` staging) and is NOT purged by retention.
- WebAuthn over LAN http stays dormant (secure-context requirement); password factor active. TLS posture is an open owner decision (TODO_LIST Blocked, 2026-11-12).
- `identityMaxOpenConns = 1` is deliberate (journal serialization) — do not "fix" it without measuring lock contention.

## Recommendations

1. Logout affordance in the shell nav (TODO_LIST Unblocked; `/auth/logout` exists, no UI link).
2. Watch usermgmt releases for a session-revocation/admin surface before ever flipping MaxUsers above 1.
3. The setup module (one-call wiring) remains rejected — `OpenIdentity` is the right-sized in-house composition; revisit only if a second consumer needs the identical stack.

## How to verify

```console
rg -n "cqrs-htmx" go.mod                                  # 5 module lines (3 direct, 2 indirect)
rg -n "usermgmt\.|loginpage|webauthn\.New" internal/web/userauth.go | head -12
rg -n "auth-db|deriveAuthDefaults|identity\.Ready" cmd/pbx-lens/serve.go | head -8
git -C /home/lars/projects/cqrs-htmx tag --sort=-v:refname | head -4
```
