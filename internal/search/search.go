// Package search maintains a disposable FTS5 shadow index over the call
// read model and answers content queries. The index lives in the same
// sqlite file as the read model, is rebuilt wholesale after every ingest
// (personal-PBX volumes; the DB is disposable), and is the ONLY piece of
// pbx-lens that writes SQL of its own — the read model itself stays
// metaengine-owned.
package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	_ "modernc.org/sqlite"
)

// highlightStart/Stop are control-char snippet markers: they can never
// collide with user content, and they are stripped before rendering.
const (
	highlightStart = "\x01"
	highlightStop  = "\x02"

	// DefaultLimit caps one search result page.
	DefaultLimit = 50

	// MaxLimit is the hard ceiling on one query's page, however large the
	// caller asks — the web layer forwards a user-controlled ?limit, and an
	// unbounded page would let one request scan the whole index.
	MaxLimit = 200

	// searchBodyFixedFields counts the non-transcript parts of the indexed
	// body: from, to, excerpt.
	searchBodyFixedFields = 3

	// minSearchDigits is the shortest digit run that triggers the substring
	// number search (R6): shorter runs would flood a small index with
	// area-code noise, while 4+ digits pin a subscriber.
	minSearchDigits = 4

	maxQueryRunes = 256

	tableName = "lens_search"

	// digitsTableName is the substring-number shadow table (R6): FTS5
	// tokenizes "+1728…" as ONE token, so a partial number silently matched
	// nothing. Digits live in a plain column and match via LIKE — the index
	// rebuilds wholesale (disposable-DB doctrine), so a scan-priced lookup
	// at personal-PBX volume is free.
	digitsTableName = "lens_digits"
)

// Hit is one matching call with a highlight-flagged snippet. The identity
// fields carry the domain types so a Kind/NativeID swap cannot compile —
// they are the same strings the detail route addresses streams by.
type Hit struct {
	Kind     calls.SourceKind
	NativeID calls.NativeID
	Started  time.Time
	Parts    []Part
}

// Part is one snippet segment; Highlight marks matched terms.
type Part struct {
	Text      string
	Highlight bool
}

// Index is the FTS5 shadow index over the call_views read model.
type Index struct {
	db *sql.DB
}

// Open connects to the index sqlite file (same DSN as the read model). The
// connection carries a busy timeout so FTS rebuilds coexist with the read
// model's own connection instead of failing fast with SQLITE_BUSY.
func Open(ctx context.Context, dsn string) (*Index, error) {
	db, err := sql.Open("sqlite", withBusyTimeout(dsn))
	if err != nil {
		return nil, errorfamily.NewRejection("lens.search.open", fmt.Sprintf("opening %s: %v", dsn, err))
	}

	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, errorfamily.NewRejection("lens.search.ping", fmt.Sprintf("pinging %s: %v", dsn, err))
	}

	return &Index{db: db}, nil
}

// withBusyTimeout appends the busy_timeout pragma to a modernc sqlite DSN
// (query params are stripped from non-file: DSNs by the driver before
// opening, so plain paths are fine).
func withBusyTimeout(dsn string) string {
	if strings.Contains(dsn, "busy_timeout") {
		return dsn
	}

	const pragma = "_pragma=busy_timeout(5000)"

	if strings.Contains(dsn, "?") {
		return dsn + "&" + pragma
	}

	return dsn + "?" + pragma
}

// Close releases the connection.
func (i *Index) Close() error {
	if err := i.db.Close(); err != nil {
		return errorfamily.NewTransient("lens.search.close", err.Error())
	}

	return nil
}

// Rebuild recreates the FTS table from the given views — a full recompute,
// so the shadow index can never drift from the read model.
func (i *Index) Rebuild(ctx context.Context, views []calls.CallView) error {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return errorfamily.NewTransient("lens.search.begin", err.Error())
	}

	defer func() { _ = tx.Rollback() }()

	if err := resetSearchTables(ctx, tx); err != nil {
		return err
	}

	insert, err := tx.PrepareContext(ctx,
		"INSERT INTO "+tableName+"(body, kind, native_id, started_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return errorfamily.NewTransient("lens.search.prepare", err.Error())
	}

	defer func() { _ = insert.Close() }()

	insertDigit, err := tx.PrepareContext(ctx,
		"INSERT INTO "+digitsTableName+"(kind, native_id, started_at, number, digits) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return errorfamily.NewTransient("lens.search.prepare_digits", err.Error())
	}

	defer func() { _ = insertDigit.Close() }()

	for _, view := range views {
		started := view.StartedAt.UTC().Format(time.RFC3339Nano)

		if _, err := insert.ExecContext(ctx, searchBody(view),
			string(view.Kind), string(view.NativeID), started); err != nil {
			return errorfamily.NewTransient("lens.search.insert",
				fmt.Sprintf("indexing %s/%s: %v", view.Kind, view.NativeID, err))
		}

		for _, number := range []string{view.FromNumber, view.ToNumber} {
			if number == "" {
				continue
			}

			if _, err := insertDigit.ExecContext(ctx,
				string(view.Kind), string(view.NativeID), started, number, digitsOnly(number)); err != nil {
				return errorfamily.NewTransient("lens.search.insert_digits",
					fmt.Sprintf("indexing %s/%s: %v", view.Kind, view.NativeID, err))
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errorfamily.NewTransient("lens.search.commit", err.Error())
	}

	return nil
}

// resetSearchTables drops and recreates the FTS and digits tables inside
// tx.
func resetSearchTables(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+tableName); err != nil {
		return errorfamily.NewTransient("lens.search.drop", err.Error())
	}

	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+digitsTableName); err != nil {
		return errorfamily.NewTransient("lens.search.drop_digits", err.Error())
	}

	create := "CREATE VIRTUAL TABLE " + tableName +
		" USING fts5(body, kind UNINDEXED, native_id UNINDEXED, started_at UNINDEXED)"
	if _, err := tx.ExecContext(ctx, create); err != nil {
		return errorfamily.NewTransient("lens.search.create", err.Error())
	}

	createDigits := "CREATE TABLE " + digitsTableName +
		" (kind TEXT NOT NULL, native_id TEXT NOT NULL, started_at TEXT NOT NULL, number TEXT NOT NULL, digits TEXT NOT NULL)"
	if _, err := tx.ExecContext(ctx, createDigits); err != nil {
		return errorfamily.NewTransient("lens.search.create_digits", err.Error())
	}

	return nil
}

// searchBody is the indexed text: parties, excerpt, and every transcript
// turn — anything a person would grep for.
func searchBody(view calls.CallView) string {
	parts := make([]string, 0, len(view.Transcript)+searchBodyFixedFields)
	parts = append(parts, view.FromNumber, view.ToNumber, view.TranscriptExcerpt)

	for _, turn := range view.Transcript {
		parts = append(parts, turn.Text)
	}

	return strings.Join(parts, "\n")
}

// Query runs a MATCH search, best rank first, and merges in substring
// number hits (R6): a 4+ digit run matches any indexed party number that
// contains it, so partial numbers finally find their calls.
func (i *Index) Query(ctx context.Context, query string, limit int) ([]Hit, error) {
	match, err := MatchExpression(query)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = DefaultLimit
	}

	limit = min(limit, MaxLimit)

	rows, err := i.db.QueryContext(ctx,
		"SELECT kind, native_id, started_at, snippet("+tableName+", 0, ?, ?, '…', 12) "+
			"FROM "+tableName+" WHERE "+tableName+" MATCH ? ORDER BY rank LIMIT ?",
		highlightStart, highlightStop, match, limit)
	if err != nil {
		return nil, errorfamily.NewRejection("lens.search.query", err.Error())
	}

	defer func() { _ = rows.Close() }()

	hits := make([]Hit, 0, limit)

	for rows.Next() {
		var (
			hit              Hit
			started, snippet string
		)

		if err := rows.Scan(&hit.Kind, &hit.NativeID, &started, &snippet); err != nil {
			return nil, errorfamily.NewCorruption("lens.search.scan", err.Error())
		}

		startedAt, err := time.Parse(time.RFC3339Nano, started)
		if err != nil {
			return nil, errorfamily.NewCorruption("lens.search.parse_time",
				fmt.Sprintf("hit %s/%s: %v", hit.Kind, hit.NativeID, err))
		}

		hit.Started = startedAt
		hit.Parts = parseSnippet(snippet)
		hits = append(hits, hit)
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.NewTransient("lens.search.rows", err.Error())
	}

	if digits := digitsOnly(query); len(digits) >= minSearchDigits {
		extra, err := i.digitHits(ctx, digits)
		if err != nil {
			return nil, err
		}

		hits = mergeHits(hits, extra, limit)
	}

	return hits, nil
}

// digitHits finds every indexed party number whose digit projection
// contains the run. digitsOnly already stripped everything but 0-9, so the
// LIKE pattern carries no wildcard metacharacters.
func (i *Index) digitHits(ctx context.Context, digits string) ([]Hit, error) {
	rows, err := i.db.QueryContext(ctx,
		"SELECT kind, native_id, started_at, number FROM "+digitsTableName+" WHERE digits LIKE ?",
		"%"+digits+"%")
	if err != nil {
		return nil, errorfamily.NewRejection("lens.search.digits", err.Error())
	}

	defer func() { _ = rows.Close() }()

	hits := make([]Hit, 0)

	for rows.Next() {
		var (
			hit             Hit
			started, number string
		)

		if err := rows.Scan(&hit.Kind, &hit.NativeID, &started, &number); err != nil {
			return nil, errorfamily.NewCorruption("lens.search.scan_digits", err.Error())
		}

		startedAt, err := time.Parse(time.RFC3339Nano, started)
		if err != nil {
			return nil, errorfamily.NewCorruption("lens.search.parse_time",
				fmt.Sprintf("digit hit %s/%s: %v", hit.Kind, hit.NativeID, err))
		}

		hit.Started = startedAt
		// The whole matching number is the snippet: the operator searched
		// for digits, so the honest highlight is the number itself.
		hit.Parts = []Part{{Text: number, Highlight: true}}
		hits = append(hits, hit)
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.NewTransient("lens.search.rows_digits", err.Error())
	}

	return hits, nil
}

// mergeHits appends unique extras after the ranked primary hits, capped.
func mergeHits(primary, extra []Hit, limit int) []Hit {
	seen := make(map[string]struct{}, len(primary)+len(extra))

	for _, hit := range primary {
		seen[hitKey(hit)] = struct{}{}
	}

	out := primary

	for _, hit := range extra {
		if len(out) >= limit {
			break
		}

		key := hitKey(hit)
		if _, dup := seen[key]; dup {
			continue
		}

		seen[key] = struct{}{}

		out = append(out, hit)
	}

	return out
}

func hitKey(hit Hit) string {
	return string(hit.Kind) + "\x00" + string(hit.NativeID)
}

// digitsOnly keeps the 0-9 projection of a string: the normalized form a
// partial-number search runs against.
func digitsOnly(s string) string {
	var builder strings.Builder

	for _, r := range s {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// MatchExpression turns free text into a safe FTS5 MATCH: every term is
// double-quoted (internal quotes doubled) and terms AND together. No raw
// user syntax ever reaches sqlite, so nothing can error or inject.
func MatchExpression(query string) (string, error) {
	if len([]rune(query)) > maxQueryRunes {
		return "", errorfamily.NewRejection("lens.search.query_too_long",
			fmt.Sprintf("query longer than %d characters", maxQueryRunes))
	}

	var terms []string

	for field := range strings.FieldsSeq(query) {
		if strings.ContainsAny(field, "\x01\x02") {
			return "", errorfamily.NewRejection("lens.search.control_chars",
				"queries must not contain control characters")
		}

		quoted := `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
		terms = append(terms, quoted)
	}

	if len(terms) == 0 {
		return "", errorfamily.NewRejection("lens.search.empty_query", "enter at least one search term")
	}

	return strings.Join(terms, " "), nil
}

// parseSnippet splits the marker-flagged snippet into render-safe parts.
func parseSnippet(snippet string) []Part {
	var parts []Part

	remaining := snippet

	for {
		start := strings.Index(remaining, highlightStart)
		if start < 0 {
			if remaining != "" {
				parts = append(parts, Part{Text: remaining})
			}

			return parts
		}

		if before := remaining[:start]; before != "" {
			parts = append(parts, Part{Text: before})
		}

		rest := remaining[start+len(highlightStart):]
		before, after, ok := strings.Cut(rest, highlightStop)

		if !ok {
			if rest != "" {
				parts = append(parts, Part{Text: rest, Highlight: true})
			}

			return parts
		}

		parts = append(parts, Part{Text: before, Highlight: true})
		remaining = after
	}
}

// ErrNoSearchIndex is returned by callers that need to explain a missing
// sqlite index (memory deployments have no FTS).
var ErrNoSearchIndex = errors.New("search requires a sqlite index (memory deployments have none)")
