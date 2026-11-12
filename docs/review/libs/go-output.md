# go-output — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-output` root v0.38.7 (+ per-module tags; 19 sub-modules) · **Status in pbx-lens:** **NOT IMPORTED — PATTERN-ONLY adoption**
**Verdict:** **PATTERN-ONLY** — pbx-lens copies its _engineering patterns_ (flake structure, CI lane) but has no Go-level use for a rendering framework.
> **Refreshed 2026-11-12:** re-verified: still not imported — `rg 'go-output' go.mod go.sum internal cmd` is empty; latest tag re-verified **v0.38.7** (tags fetched). PATTERN-ONLY verdict stands.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- One data model → 16 formats: markdown/CSV/JSON serialization, DOT/Mermaid/D2 graphs, HTML; CQRS build-freeze-render discipline; NOM-style live progress; streaming renderers; branded IDs.

## The pattern adoption (verified, with one precision fix)

- **Flake:** `flake.nix:12-14` consumes `go-nix-helpers`' `flakeModules.go-standard` — the _go-standard_ flakeModule. AGENTS.md phrases this as "go-standard flakeModule from go-nix-helpers (go-output's pattern)": the pattern provenance is go-output, the consuming module is go-nix-helpers'. Both true; keep the attribution precise.
- **CI:** the GitHub Actions lane (`.github/workflows/ci.yml`, build+test+lint+coverage+tidy) follows go-output's single-module pattern per AGENTS.md.
- **What is NOT adopted:** any Go import. `rg 'go-output' go.mod go.sum` is empty; `internal/cli/catalog.go` renders the eventcatalog tree via go-cqrs-lite `catalog`, not go-output.

## Why not a dependency

1. **The UI already has a rendering stack:** templ-components pages + JSON endpoints. go-output's renderer framework would be a competing abstraction over the same tables.
2. **The export surface is streaming, not rendered:** keyset-paginated CSV/JSONL (`StreamExportCalls`, 5k-row pages, memory-bounded — AGENTS.md serve/UI) — deliberately row-streamed; a build-freeze-render model is the opposite shape.
3. **19 sub-modules** = graph weight (hermetic nix vendorHash cost) for surface pbx-lens would not call.

## When this verdict flips

If pbx-lens grows a _static report_ surface (e.g. a generated weekly call digest in markdown/graph form), go-output's build-freeze-render is exactly that tool. Until then: pattern-only, correctly.

## How to verify

```console
rg -n "go-output" go.mod go.sum internal cmd --type go   # pattern-only: zero Go hits
sed -n '10,16p' flake.nix                                 # go-nix-helpers go-standard
rg -n "StreamExportCalls" internal/web/export.go | head -3
```
