# httputil — utilization audit (pbx-lens)

**Date:** 2026-11-04 · **Pin:** `github.com/larsartmann/httputil v1.8.0` (go.mod:24) · **Status:** USED
**Adoption score:** **80/100 (high)** — 8 of 10 applicable symbols used (5 entry points incl. config structs; 4 of 4 capability areas served). Prior audit: [2026-10-12_http-stack-adoption-audit.html](../../research/2026-10-12_http-stack-adoption-audit.html) — this report supersedes its version-currency rows (pin moved v1.4.1 → v1.8.0) and resolves its "watchlist: no needed feature in the delta" disposition (the delta did deliver).
**Fleet report:** [index.html](index.html)

## Verdict

The four middleware this read-only UI needs — SecurityHeaders, Nonce/CSP, Compression, plus the appkit-owned chain — are adopted and doctrine-documented (`docs/http-stack.md`). Since the 2026-10-12 audit the pin moved four minor versions and quietly delivered two things pbx-lens now depends on: the v1.5.0 compression Content-Length contract (the cache doctrine's foundation, pinned by `TestCacheSemanticsHardening`) and the v1.6.0 Range-aware compression guard (protects `/recording` byte-range seeking). The unused ~120 exports are correctly refused with per-feature doctrine, not neglect.

## Capability areas vs use

Ground truth: module cache `~/go/pkg/mod/github.com/larsartmann/httputil@v1.8.0` (~130 exported identifiers across security, nonce, compression, rate-limit, CSRF, server, observability, language, misc).

| Area                                                                                                                                                               | Used by pbx-lens                                                                                                                                                   | Verdict                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------- |
| Security headers                                                                                                                                                   | `SecurityHeaders(+SecurityHeadersConfig)` — `cmd/pbx-lens/serve.go:284`                                                                                            | Fully leveraged                                                                     |
| CSP nonce                                                                                                                                                          | `Nonce(+NonceConfig)` — `serve.go:300`; `NonceFromRequest` — `internal/web/server.go` consumers (`internal/web/search.go:20`), `internal/web/nonce_csp_test.go:49` | Fully leveraged                                                                     |
| Compression                                                                                                                                                        | `Compression(DefaultCompressionConfig())` — `serve.go:329`                                                                                                         | Fully leveraged (v1.5.0 semantics load-bearing)                                     |
| Chain                                                                                                                                                              | `httputil.Middleware` type only — composition owned by go-appkit `ExtraMiddlewares`                                                                                | Partially used (correct: appkit owns the chain, docs/http-stack.md)                 |
| CSP Report-Only                                                                                                                                                    | hand-rolled at `serve.go:315-325`                                                                                                                                  | Justified hand-roll — `NonceConfig` (nonce.go:59-70) has no Report-Only mode at pin |
| Rate limit / CSRF / Language / Decompression / MaxBodySize / Server / Health / Metrics / Timeout / CORS / ClientIP / ResponseRecorder / MiddlewareStack / httpspec | ❌                                                                                                                                                                 | Not applicable — reasons below                                                      |

N/A reasons (one line each): rate limiting — doctrine Hold from the 2026-10-12 audit ("revisit on abuse"; loopback + htpasswd posture); CSRF (~20 symbols) — no unsafe methods, the UI is read-only GET; Language — single-locale tool; Decompression/MaxBodySize — no request bodies; Server/NewServerListener/Health/Metrics/Timeout — appkit v0.9.0 owns lifecycle and `WriteTimeout: NoTimeout` is the streaming contract (docs/http-stack.md:10-15); CORS/ClientIP — no cross-origin or proxy posture; ResponseRecorder/Chain/MiddlewareStack — appkit owns composition; httpspec — the library's own test harness; `DetectCapabilities` — explicitly frozen-but-unused upstream; `code.go` error classifiers — go-error-family is used directly instead (see [go-error-family.md](go-error-family.md)).

## Usage census

`cmd/pbx-lens/serve.go`: SecurityHeaders 284, Nonce 300, `NonceFromRequest` 321 (via internal/web), Compression 329. Tests: `internal/web/nonce_csp_test.go:49`, cache-semantics suites pin the v1.5.0 contract. The middlewares are ordered auth → SecurityHeaders → Server-Timing → Compression → ETag per docs/http-stack.md.

## Version currency (delta since the 2026-10-12 audit pin v1.4.1)

| Version | Notable change                                                                        | Relevance                                                                                                                      |
| ------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| v1.8.0  | process/CI only (apidiff gate, policy-vet CI)                                         | None — zero API change                                                                                                         |
| v1.7.0  | `NewServerListener`, Shutdown-before-Start no-op                                      | None (appkit owns lifecycle)                                                                                                   |
| v1.6.0  | **Range-aware compression guard**; Language; trusted-proxy keying; `Recovery.OnPanic` | **High** — protects `/recording` Range seeking through the compression middleware (`internal/web/recording.go:156-158`)        |
| v1.5.0  | **Compression Content-Length delete/preserve semantics**                              | **Load-bearing** — the ETag/cache doctrine depends on it; pinned by `TestCacheSemanticsHardening` (docs/http-stack.md:102-108) |

Pin v1.8.0 == the checkout's pending release cut (its `[Unreleased]` freezes exactly the v1.8.0 content). **Current.**

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
