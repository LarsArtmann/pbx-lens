# httputil — utilization audit (pbx-lens)

**Date:** 2026-11-04 (refreshed **2026-11-11**) · **Pin:** `github.com/larsartmann/httputil v1.8.0` (go.mod:24; == proxy latest; upstream `[Unreleased]` is CI/process work + the new `endpoints` sub-module — zero root API change, see [httputil-endpoints.md](httputil-endpoints.md)) · **Status:** USED
**Refreshed 2026-11-12:** pin still v1.8.0 (go.mod:29) == latest root tag, verified via `git fetch --tags` + `git tag --sort=-v:refname` in `/home/lars/projects/httputil` (sub-module tags share the repo: latest `endpoints/v0.1.0`, `server_timing/v1.0.3`). The 36 post-v1.8.0 commits are chore/docs + the endpoints/v0.1.0 cut + fixes in areas pbx-lens does not use (CORS `Vary: Origin`); the used surface is untouched since the tag — security.go/nonce.go have no commits after v1.8.0, and compression.go is net-unchanged (a sentinel-typing experiment was reverted). `NonceConfig` still has no Report-Only mode at HEAD (`git grep -il ReportOnly` → no hits), so recommendation 1 below is not yet actionable. Verdict/score unchanged (**80/100**).
**Adoption score:** **80/100 (high)** — 8 of 10 applicable symbols used (5 entry points incl. config structs; 4 of 4 capability areas served). Prior audit: [2026-10-12_http-stack-adoption-audit.html](../../research/2026-10-12_http-stack-adoption-audit.html) — this report supersedes its version-currency rows (pin moved v1.4.1 → v1.8.0) and resolves its "watchlist: no needed feature in the delta" disposition (the delta did deliver).
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## Verdict

The four middleware this read-only UI needs — SecurityHeaders, Nonce/CSP, Compression, plus the appkit-owned chain — are adopted and doctrine-documented (`docs/http-stack.md`). Since the 2026-10-12 audit the pin moved four minor versions and quietly delivered two things pbx-lens now depends on: the v1.5.0 compression Content-Length contract (the cache doctrine's foundation, pinned by `TestCacheSemanticsHardening`) and the v1.6.0 Range-aware compression guard (protects `/recording` byte-range seeking). The unused ~120 exports are correctly refused with per-feature doctrine, not neglect. **2026-11-11: the CSRF area left the N/A column** — the cqrs-htmx identity adoption wires `httputil.CSRFMiddleware(httputil.CSRFConfig{})` (`internal/web/userauth.go:170`) around the login/auth ceremonies, the UI's first unsafe-method surface. The read-only app routes stay GET-only; CSRF guards exactly the library-owned POST surface.

## Capability areas vs use

Ground truth: module cache `~/go/pkg/mod/github.com/larsartmann/httputil@v1.8.0` (~130 exported identifiers across security, nonce, compression, rate-limit, CSRF, server, observability, language, misc).

| Area                                                                                                                                                               | Used by pbx-lens                                                                                                                                                   | Verdict                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------- |
| Security headers                                                                                                                                                   | `SecurityHeaders(+SecurityHeadersConfig)` — `cmd/pbx-lens/serve.go:284`                                                                                            | Fully leveraged                                                                     |
| CSP nonce                                                                                                                                                          | `Nonce(+NonceConfig)` — `serve.go:300`; `NonceFromRequest` — `internal/web/server.go` consumers (`internal/web/search.go:20`), `internal/web/nonce_csp_test.go:49` | Fully leveraged                                                                     |
| Compression                                                                                                                                                        | `Compression(DefaultCompressionConfig())` — `serve.go:329`                                                                                                         | Fully leveraged (v1.5.0 semantics load-bearing)                                     |
| Chain                                                                                                                                                              | `httputil.Middleware` type only — composition owned by go-appkit `ExtraMiddlewares`                                                                                | Partially used (correct: appkit owns the chain, docs/http-stack.md)                 |
| CSP Report-Only                                                                                                                                                    | hand-rolled at `serve.go:315-325`                                                                                                                                  | Justified hand-roll — `NonceConfig` (nonce.go:59-70) has no Report-Only mode at pin |
| CSRF (post-2026-11-11)                                                                                                                                             | `CSRFMiddleware(CSRFConfig{})` — `internal/web/userauth.go:170` (identity ceremonies only)                                                                         | Partially used — new adoption, default config; the read-only routes still don't need it             |
| Rate limit / Language / Decompression / MaxBodySize / Server / Health / Metrics / Timeout / CORS / ClientIP / ResponseRecorder / MiddlewareStack / httpspec         | ❌                                                                                                                                                                 | Not applicable — reasons below                                                                      |

**Rate-limit Hold — defined, not vague (2026-11-05, plan M27.4).** "Revisit on
abuse" was an unfalsifiable trigger. It now reopens on any ONE of these
observations, and on nothing else:

1. **One address dominates traffic.** A single client address accounts for
   > 50% of requests in a 10-minute window and is not the loopback/operator
   > address — the posture this Hold rests on (loopback plus htpasswd) has
   > stopped describing reality.
2. **An unauthenticated storm.** >20 rejected requests per minute from one
   address that is not the operator's. The per-address auth-failure buckets
   of the deleted htpasswd gate counted this until 2026-11-11; the identity
   stack now carries the instrument — usermgmt's `AccountLockout`
   (5 failures / 15 min, `userauth.go:134,321`) plus its own session
   throttling. No new pbx-lens instrument needed.
3. **A second user exists.** The deploy gains a non-operator consumer (a
   webphone user, a second PC), which is a posture change rather than abuse —
   but it voids the single-operator assumption the whole Hold cites.

Cheapest first instrument if (1) or (2) looks close: a per-address 4xx count
in the `/api/ops` payload (today the ops counters are global). Adding
`httputil.RateLimit` before that evidence exists would be the exact
"capability without a consumer" the fleet audit scores as misuse.

N/A reasons (one line each): rate limiting — doctrine Hold from the 2026-10-12 audit ("revisit on abuse"; loopback + single-operator posture, now enforced by usermgmt lockout); CSRF was N/A until the identity adoption made the library's CSRFMiddleware the guard of the auth ceremonies (2026-11-11); Language — single-locale tool; Decompression/MaxBodySize — no request bodies; Server/NewServerListener/Health/Metrics/Timeout — appkit owns lifecycle and `WriteTimeout: NoTimeout` is the streaming contract (docs/http-stack.md:10-15); CORS/ClientIP — no cross-origin or proxy posture; ResponseRecorder/Chain/MiddlewareStack — appkit owns composition; httpspec — the library's own test harness; `DetectCapabilities` — explicitly frozen-but-unused upstream; `code.go` error classifiers — go-error-family is used directly instead (see [go-error-family.md](go-error-family.md)).

## Usage census

`cmd/pbx-lens/serve.go`: SecurityHeaders 284, Nonce 300, `NonceFromRequest` 321 (via internal/web), Compression 329. `internal/web/userauth.go:170`: `CSRFMiddleware(CSRFConfig{})` (2026-11-11). Tests: `internal/web/nonce_csp_test.go:49`, cache-semantics suites pin the v1.5.0 contract. The middlewares are ordered auth → SecurityHeaders → Server-Timing → Compression → ETag per docs/http-stack.md.

## Version currency (delta since the 2026-10-12 audit pin v1.4.1)

| Version | Notable change                                                                        | Relevance                                                                                                                      |
| ------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| v1.8.0  | process/CI only (apidiff gate, policy-vet CI)                                         | None — zero API change                                                                                                         |
| v1.7.0  | `NewServerListener`, Shutdown-before-Start no-op                                      | None (appkit owns lifecycle)                                                                                                   |
| v1.6.0  | **Range-aware compression guard**; Language; trusted-proxy keying; `Recovery.OnPanic` | **High** — protects `/recording` Range seeking through the compression middleware (`internal/web/recording.go:156-158`)        |
| v1.5.0  | **Compression Content-Length delete/preserve semantics**                              | **Load-bearing** — the ETag/cache doctrine depends on it; pinned by `TestCacheSemanticsHardening` (docs/http-stack.md:102-108) |

Pin v1.8.0 == proxy latest (re-verified 2026-11-11). Upstream `[Unreleased]` carries CI/policy work and the **`endpoints` sub-module's first release (endpoints/v0.1.0, 2026-11-06)** — a stdlib-only route registry with a self-describing `/endpoints` listing; pbx-lens doesn't need it (every route is declared in one mux in `web.Register`, [httputil-endpoints.md](httputil-endpoints.md) carries the full assessment). **Current.**

## Risks

- CSP Report-Only stays a hand-roll until upstream adds a Report-Only mode to `NonceConfig` — revisit on the next httputil minor.
- Rate-limiting doctrine Hold is a _decision_, not an omission — re-litigate only on abuse evidence.

## Recommendations

1. None urgent. On the next httputil bump, check whether `NonceConfig` gained Report-Only (deletes `serve.go:315-325`).

## How to verify

```console
rg -n "httputil\." cmd/pbx-lens/serve.go internal/web -g '!*_templ.go'
go doc github.com/larsartmann/httputil NonceConfig | head -20
git -C /home/lars/projects/httputil tag --sort=-v:refname | head -5
```
