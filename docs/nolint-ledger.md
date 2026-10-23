<!-- docs/nolint-ledger.md — every nolint in the Go sources with its why, so
     the rationale survives refactors and lint-config changes. Extracted
     2026-10-12 (templ-UI Pareto plan T26); pattern follows
     docs/cqrs-lint-policy.md. Rule: a nolint without a row here and without
     an inline why is a bug — fix one or both. ENFORCED by
     scripts/check-nolint-ledger.sh (rows matched by file basename; line
     numbers here are human context) as a named CI step (plan T5.4). -->

# Nolint ledger

Every `//nolint:` in `internal/` and `cmd/` (non-test, non-generated), with the reason
it survives. Inline comments are the primary home; this ledger is the
survivable copy. When removing the suppressed condition, delete both.

| Location                        | Linter           | Why it survives                                                                                                                                                                         |
| ------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/web/queries.go:122`   | gochecknoglobals | `defaultListPageSize` is test-shrinkable on purpose — keyset seam tests shrink the page to exercise multi-page walks without config plumbing.                                           |
| `internal/web/queries.go:305`   | nilerr           | A malformed kind/id in `CallDetail` is a plain miss (404), not a server error — the error is folded into `found=false` deliberately.                                                    |
| `internal/web/export.go:47`     | gosec            | The bytes are JSON-encoded ndjson served as an attachment; never interpolated into HTML, so the injection class gosec flags on dynamic writes does not apply.                           |
| `internal/web/export.go:57`     | gosec            | Same as above for the terminal `EXPORT_INCOMPLETE` marker row/object.                                                                                                                   |
| `internal/web/server.go:445`    | gosec            | The search-jump redirect is same-origin by construction: fixed `/calls/` prefix, path-escaped identity, query-escaped search text. Property pinned by `TestSearchJumpTargetSameOrigin`. |
| `internal/web/dashboard.go:34`  | gochecknoglobals | `dashboardLocation` is injectable so tests can pin the zone (save/restore) without a config knob on the query.                                                                          |
| `internal/web/dashboard.go:92`  | gochecknoglobals | `durationBuckets` is a declarative histogram table, read-only — a struct-of-data table, not mutable state.                                                                              |
| `internal/web/recording.go:248` | gosec            | The served path comes from directory entries or basename-locked stems, never the caller's string: the resolver only returns files that already exist inside the recordings directory.   |
| `cmd/pbx-lens/catalog.go:23`    | erraudit         | CLI output stream: a failing write of the "wrote eventcatalog tree" status line has no recovery action and must not mask the command's real result.                                     |
