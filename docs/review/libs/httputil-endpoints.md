# httputil/endpoints — adoption-fit report (pbx-lens)

**Date:** 2026-11-11 · **Refreshed 2026-11-12** (tag re-verified current: nothing stale, stamp-completeness only — plan P09.3) · **Upstream:** `github.com/larsartmann/httputil/endpoints` v0.1.0 (first release 2026-11-06, tagged from the endpoints sub-module; == proxy latest; stdlib-only, zero external deps; owner ruling: pre-1.0 iteration tier — 0.x semantics govern) · **Status in pbx-lens:** **NOT USED** (not in the graph — `go.sum` carries zero entries)
**Verdict:** **REJECT** — a good answer to a question pbx-lens does not have: its route surface is small, static, and documented; a self-describing runtime listing would be new public surface with no consumer.
**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

`http.ServeMux` cannot enumerate its own patterns, so this module makes the registry the source of truth: `Registry.Handle`/`HandleFunc` mounts the route on the mux AND records it in one declaration; `endpoints.Register(mux, routes)` mounts `GET /endpoints`, a self-describing listing that content-negotiates — browsers get an HTML table, API clients JSON, on one URL with `Vary: Accept`. Stdlib-only (embeddable where the root httputil dependency set is too heavy), composes with httputil middleware without requiring it. Root repo posture: the shared q-value corpus pins both copies' parser behavior (architecture-review F1 closed 2026-11-06).

## Why pbx-lens declines it

1. **The inventory need is already served.** Every route is declared in one place (`web.Register`), the middleware chain + route contract live in `docs/http-stack.md`, and `/api/ops` already publishes the machine-readable operational state the ops page renders. There is no "what does this service serve?" question a listing would answer that the docs do not.
2. **New public surface, single-operator posture.** `/endpoints` would enumerate route internals on a server whose one gate is opt-in (`-auth-db` unset ⇒ unauthenticated dev posture). The Server-Timing `When` decision (docs/http-stack.md) set the precedent: route internals don't leak on ungated runs. A listing endpoint would need the same gate — an escape hatch built for a need that has not arisen.
3. **Static surface, zero drift risk:** routes change rarely (the last UI change added identity routes, not route plumbing). The failure mode endpoints exists to prevent — docs and mux disagreeing — is structurally hard here because the docs cite the one `Register` file.
4. **Pre-1.0 API:** v0.1.0 by owner ruling keeps iteration freedom; adopting a 0.x module for a nice-to-have listing buys churn risk without a consumer.

## Flip trigger (falsifiable)

Adopt when ANY of: (a) the route count or a second mount point (admin surface, API version) makes the one-file answer stale — concretely, when `docs/http-stack.md` and `web.Register` disagree for the first time, or a route is added outside `Register`; (b) a machine consumer needs route discovery (a probe suite or deploy module that derives expectations from the live server); (c) pbx-lens wants the listing for the ops page (it would compose with the identity gate cleanly).

## Version currency

endpoints/v0.1.0 == proxy latest (2026-11-11); first tag, active development (the root repo's `[Unreleased]` carries the shared q-value corpus CI sync work). Nothing to pin.

## How to verify

```console
rg -n "httputil/endpoints" go.mod go.sum internal/ cmd/   # zero hits
rg -n "func Register" internal/web/server.go | head -3    # the one wiring surface
ls /home/lars/projects/httputil/endpoints/                # the module exists upstream
go list -m -versions github.com/larsartmann/httputil/endpoints   # v0.1.0
```
