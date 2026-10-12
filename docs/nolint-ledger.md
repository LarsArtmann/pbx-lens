<!-- docs/nolint-ledger.md — every nolint in the Go sources with its why, so
     the rationale survives refactors and lint-config changes. Extracted
     2026-10-12 (templ-UI Pareto plan T26); pattern follows
     docs/cqrs-lint-policy.md. Rule: a nolint without a row here and without
     an inline why is a bug — fix one or both. -->

# Nolint ledger

Every `//nolint:` in `internal/` (non-test, non-generated), with the reason
it survives. Inline comments are the primary home; this ledger is the
survivable copy. When removing the suppressed condition, delete both.

| Location                       | Linter           | Why it survives                                                                                                                                                                         |
| ------------------------------ | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/web/queries.go:122`  | gochecknoglobals | `defaultListPageSize` is test-shrinkable on purpose — keyset seam tests shrink the page to exercise multi-page walks without config plumbing.                                           |
| `internal/web/queries.go:305`  | nilerr           | A malformed kind/id in `CallDetail` is a plain miss (404), not a server error — the error is folded into `found=false` deliberately.                                                    |
| `internal/web/export.go:47`    | gosec            | The bytes are JSON-encoded ndjson served as an attachment; never interpolated into HTML, so the injection class gosec flags on dynamic writes does not apply.                           |
| `internal/web/export.go:57`    | gosec            | Same as above for the terminal `EXPORT_INCOMPLETE` marker row/object.                                                                                                                   |
| `internal/web/server.go:436`   | gosec            | The search-jump redirect is same-origin by construction: fixed `/calls/` prefix, path-escaped identity, query-escaped search text. Property pinned by `TestSearchJumpTargetSameOrigin`. |
| `internal/web/dashboard.go:34` | gochecknoglobals | `dashboardLocation` is injectable so tests can pin the zone (save/restore) without a config knob on the query.                                                                          |
| `internal/web/dashboard.go:92` | gochecknoglobals | `durationBuckets` is a declarative histogram table, read-only — a struct-of-data table, not mutable state.                                                                              |
