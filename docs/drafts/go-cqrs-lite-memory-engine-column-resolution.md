> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

The memory engine resolves declarative filter columns by Go FIELD name while SQL engines resolve them by JSON key — the same `WithFilter("started_at", ...)` is correct on sqlite and silently matches nothing/everything on memory.

## Problem

The closure-fallback filter path extracts the row value with `itemFieldByName` (`metaengine/typed_reader_scan.go:271`):

```go
if evalFilterOp(spec.Op, itemFieldByName(item, spec.Column), spec.Value) {
```

For a struct row that ends in `extractValueByName` (`metaengine/reflect.go:307-327`), which matches **Go field names** case-insensitively and never consults the json tag. `EqualFold("StartedAt", "started_at")` is false (underscore), so the extraction returns nil. For `map[string]any` rows (SQL engines) the same helper matches JSON keys, where `started_at` is exactly the key.

Consequences on the memory engine for a JSON-tag-named filter column:

| Op        | Behavior (value compared against nil) |
| --------- | ------------------------------------- |
| `Le`/`Lt` | match EVERYTHING                      |
| `Ge`/`Gt` | match NOTHING                         |
| `Eq`      | never matches                         |

Sort (`typed_reader_scan.go:246`) and keyset cursor comparisons (`typed_reader_scan.go:294-295`, `313-314`) go through the same extractor, so ordering and pagination carry the same split brain.

## Consumer impact

pbx-lens ships filter columns named after JSON tags (they must match what sqlite's `json_extract` sees). Its web tests therefore cannot use the memory engine: filter semantics differ from production. Nothing warns; the memory engine just answers a different query.

## Suggested fix

One of: (a) resolve columns through the JSON tag first with field-name fallback, (b) accept the field name on both engine classes and normalize SQL-side, or (c) at minimum, error when a declarative filter column resolves to nil on every row (today it degrades into a wrong answer).

Observed on metaengine v4.17.0 / master `f2c892b`.

💘 Generated with Crush
