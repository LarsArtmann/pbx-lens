# go-credentials-store — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-credentials-store` v0.3.1 (checkout last commit 2026-11-04; single module; built on go-cqrs-lite system+metaengine) · **Status in pbx-lens:** **NOT USED** (not in the graph)
**Verdict:** **REJECT** — same pedigree, wrong problem: it manages many rotating secrets at rest; pbx-lens has exactly one auth surface and stores no secret material.
**Fleet report:** [index.html](index.html)

## What the library offers

- Event-sourced, encrypted-at-rest credential store (XChaCha20-Poly1305, KeyID rotation) for API keys / Playwright storageState / passwords; lifecycle audit (created/rotated/probed/used/revealed); opt-in admin UI (SSE live updates, audited one-time reveal, `/status`, `/metrics`, API-key auth beyond loopback); SDK-first (`credentials.Build` + `Resolve`).

## Why pbx-lens rejects it

1. **One secret surface, human-managed:** auth is a bcrypt htpasswd file with SIGHUP rotation and keep-on-failure semantics (`internal/web/auth.go:46-61,142-176`) — operator-owned, deterministic, test-pinned. KeyID rotation and reveal-audit would be dead weight around a file that `htpasswd` manages.
2. **The index never stores secret material:** read-only over sources (AGENTS.md north star); Telnyx API access is operator-configured, not stored in the DB. There is nothing to encrypt-at-rest.
3. **Sibling overlap, not pbx-lens concern:** the store's own ecosystem (probing/rotation tooling) belongs to the credential-*producing* services; pbx-lens consumes nothing credential-shaped at runtime beyond the htpasswd path.

## When this verdict flips

If the lens ever ingests authenticated sources whose credentials must rotate automatically and be audited (e.g. multi-tenant Telnyx API keys), the credential store — being cqrs-lite-native — is the right first choice. That is a product expansion, not a current need.

## How to verify

```console
rg -n "go-credentials-store" go.mod go.sum   # absent
rg -n "bcrypt|htpasswd" internal/web/auth.go | head -5
rg -n "SIGHUP" internal/web/auth.go | head -2
```
