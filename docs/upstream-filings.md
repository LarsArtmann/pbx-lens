# Upstream filings (D6, 2026-10-11 — all source-verified, voice-checked, review boxes left UNCHECKED for Lars)

Extracted from AGENTS.md 2026-10-11 (go-structure-linter size limit).
Update these notes when upstream lands.

**State check 2026-10-12 (read-only, gh api): all five filings still OPEN
upstream — no upstream movement to fold in yet.**

- **go-cqrs-lite#62**: sqliteengine binds filter values raw (`filter_clause.go:44`,
  `planned_filter.go:77`) — time.Time miscompares vs RFC3339 JSON (Le/Lt leak the day,
  Ge/Gt drop it, Eq never). In-repo fix stays: bind `rfc3339Nano` strings.
- **go-cqrs-lite#63**: memory engine resolves filter/sort/cursor columns by Go FIELD name
  (`typed_reader_scan.go:271` → `reflect.go:307`) vs sqlite's JSON keys.
- **go-cqrs-lite#64**: cqrs-lint C009 `len(name) > 4` flags the bare `must` idiom;
  E005 doesn't recognize `system.RegisterCommand` (ADR-0123 root API).
- **go-nix-helpers#7**: go-standard's templ formatter inherits treefmt-nix's `pkgs.go`
  wrap (`go-standard.nix:1297`) instead of the module's resolved goPkg (line 665).
- **numtide/treefmt-nix#545**: already filed upstream 2026-10-02 — corroborating comment
  posted with our independent repro (nixpkgs go 1.26.8 vs floor 1.27.1).
- **BuildFlow#45** (filed 2026-10-12): go-mod-update misreports a
  load-starved compile gate (`tool.timeout` kill) as "dependency bump
  introduced compile errors" and silently reverts go.mod/go.sum;
  source-verified chain + draft in `docs/drafts/`.
- C009 verification note (2026-10-12): the `len(name) > 4` guard is
  confirmed at source (`cmd/cqrs-lint/pkg/rules/correctness/c009.go:95`,
  same shape for `New` at line 99) and IS covered by go-cqrs-lite#64 — no
  separate filing.
- R12 pass (2026-10-12, owner: annotate-and-keep): all six filed drafts in
  `docs/drafts/` now carry a FILED banner naming their issue; they stay as
  historical write-ups. `go-atomic-write-commit-primitive.md` remains the
  one LIVE draft (unfiled; owner review box unchecked by design).
- Drafts live in `docs/drafts/` (never /tmp).
