> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

sqliteengine binds filter values raw — a `time.Time` filter silently miscompares against the RFC3339 stored in the view JSON.

## Problem

`metaengine/sqliteengine/filter_clause.go:44` (master `f2c892b`) appends the filter value to the driver args with no type coercion:

```go
fmt.Fprintf(b, ` AND json_extract(value, '%s') %s ?`, path, string(f.Op))
*args = append(*args, f.Value)
```

The planned-table path has the same raw bind (`metaengine/planned_filter.go:77`). The doc comment above `appendStandardFilter` documents numeric coercion ("SQLite json_extract returns native types") but there is no time branch.

Views serialize as JSON, so a `time.Time` column is stored as RFC3339 text (`2026-10-11T00:00:00Z`). A consumer filtering with a raw `time.Time` value gets modernc.org/sqlite's default Go rendering bound instead — `2026-10-11 00:00:00 +0000 UTC` (space separator). Lexicographic TEXT comparison then corrupts same-day bounds, observed on sqliteengine v4.5.2 in a real consumer (pbx-lens):

| Op        | Same-day behavior with raw time.Time bound                                       |
| --------- | -------------------------------------------------------------------------------- |
| `Le`/`Lt` | leak the day (space sorts before `T`, so the upper bound includes the whole day) |
| `Ge`/`Gt` | drop the day (lower bound excludes every same-day row)                           |
| `Eq`      | never matches                                                                    |

## Workaround that proves the diagnosis

Binding `t.UTC().Format(time.RFC3339Nano)` — the exact stored wire format — makes eq/le/ge/lt all compare correctly for the consumer. That is only necessary because the engine forwards the raw value.

## Suggested fix

Coerce at the two bind sites, e.g. `case time.Time: v = t.UTC().Format(time.RFC3339Nano)` (matching the stored format), or honor `driver.Valuer` so consumers can carry the choice.

Related but distinct: #50 (nanosecond drift through the journal storage path). This one is the query/filter path.

💘 Generated with Crush
