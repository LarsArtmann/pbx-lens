package search

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func searchViews() []calls.CallView {
	base := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

	return []calls.CallView{
		{
			NativeID: "conv-a", Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: "+17287289311", ToNumber: "+15555550123",
			Transcript: []calls.TranscriptTurn{
				{Role: "user", Text: "hi woolly, can you check the invoice?"},
				{Role: "assistant", Text: "sure, pulling it up now"},
			},
		},
		{
			NativeID: "conv-b", Kind: calls.SourceTelnyxConversation,
			StartedAt: base.Add(time.Hour), FromNumber: "+17287289311", ToNumber: "+15555550999",
			Transcript: []calls.TranscriptTurn{
				{Role: "user", Text: "woolly double mention"},
			},
		},
		{
			NativeID: "cdr-c", Kind: calls.SourceFreeSwitchCDR,
			StartedAt: base.Add(2 * time.Hour), FromNumber: "+17287289311", ToNumber: "+15555550777",
		},
	}
}

func openTestIndex(t *testing.T) *Index {
	t.Helper()

	index, err := Open(context.Background(), filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	return index
}

func TestSearchMatchAndHighlight(t *testing.T) {
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := index.Query(context.Background(), "woolly", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2 (conv-a, conv-b)", len(hits))
	}

	// conv-b mentions the term twice: bm25 ranks it first.
	if hits[0].NativeID != "conv-b" {
		t.Errorf("top hit = %s, want conv-b (double mention ranks first)", hits[0].NativeID)
	}

	highlighted := false

	for _, hit := range hits {
		for _, part := range hit.Parts {
			if part.Highlight && strings.Contains(part.Text, "woolly") {
				highlighted = true
			}
		}
	}

	if !highlighted {
		t.Error("no highlighted snippet part contains the matched term")
	}
}

func TestSearchCoversPartiesAndNoMatch(t *testing.T) {
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	byNumber, err := index.Query(context.Background(), "+15555550777", 0)
	if err != nil {
		t.Fatalf("Query by number: %v", err)
	}

	if len(byNumber) != 1 || byNumber[0].NativeID != "cdr-c" {
		t.Fatalf("number search = %+v, want exactly cdr-c", byNumber)
	}

	none, err := index.Query(context.Background(), "nonexistent-term", 0)
	if err != nil {
		t.Fatalf("Query no-match: %v", err)
	}

	if len(none) != 0 {
		t.Fatalf("no-match returned %d hits", len(none))
	}
}

func TestSearchRebuildIsFullRecompute(t *testing.T) {
	index := openTestIndex(t)
	ctx := context.Background()

	if err := index.Rebuild(ctx, searchViews()); err != nil {
		t.Fatalf("first Rebuild: %v", err)
	}

	if err := index.Rebuild(ctx, searchViews()[:1]); err != nil {
		t.Fatalf("second Rebuild: %v", err)
	}

	hits, err := index.Query(ctx, "woolly", 0)
	if err != nil {
		t.Fatalf("Query after re-rebuild: %v", err)
	}

	if len(hits) != 1 || hits[0].NativeID != "conv-a" {
		t.Fatalf("rebuild left stale rows: %+v", hits)
	}
}

func TestMatchExpression(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    string
		wantErr error
	}{
		{name: "plain terms", query: `woolly invoice`, want: `"woolly" "invoice"`},
		{name: "quotes are neutralized", query: `say "woolly" OR 1=1`, want: `"say" """woolly""" "OR" "1=1"`},
		{name: "empty", query: "   ", wantErr: errorfamily.NewRejection("lens.search.empty_query", "")},
		{
			name:    "control chars",
			query:   "bad\x01marker",
			wantErr: errorfamily.NewRejection("lens.search.control_chars", ""),
		},
	}

	for _, test := range tests {
		got, err := MatchExpression(test.query)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("%s: expected error, got %q", test.name, got)
			}

			continue
		}

		if err != nil {
			t.Errorf("%s: %v", test.name, err)

			continue
		}

		if got != test.want {
			t.Errorf("%s: MatchExpression = %q, want %q", test.name, got, test.want)
		}
	}

	if _, err := MatchExpression(strings.Repeat("x", maxQueryRunes+1)); err == nil {
		t.Error("overlong query should reject")
	}
}

func TestPartialNumberSubstringMatches(t *testing.T) {
	// R6: FTS5 tokenizes "+17287289311" as one token, so partial numbers
	// matched nothing before. A 4+ digit run must now find every call whose
	// party digits contain it — prefix, suffix, and mid-number alike.
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	for _, digits := range []string{"7289", "17287289311", "8931"} {
		hits, err := index.Query(context.Background(), digits, 0)
		if err != nil {
			t.Fatalf("Query(%q): %v", digits, err)
		}

		if len(hits) != 3 {
			t.Errorf("partial %q hit %d calls, want all 3", digits, len(hits))
		}
	}

	// To-side numbers surface too.
	hits, err := index.Query(context.Background(), "550999", 0)
	if err != nil {
		t.Fatalf("Query(to): %v", err)
	}

	if len(hits) != 1 || hits[0].NativeID != "conv-b" {
		t.Errorf("to-side partial hit %v, want conv-b", hits)
	}
}

func TestDigitHitSnippetHighlightsTheNumber(t *testing.T) {
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := index.Query(context.Background(), "50123", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(hits) != 1 {
		t.Fatalf("hits = %d, want 1", len(hits))
	}

	if len(hits[0].Parts) != 1 || !hits[0].Parts[0].Highlight {
		t.Fatalf("digit hit parts = %+v, want one highlighted part", hits[0].Parts)
	}

	if got := hits[0].Parts[0].Text; got != "+15555550123" {
		t.Errorf("highlighted snippet = %q, want the matching raw number", got)
	}
}

func TestShortDigitRunsStayFTSOnly(t *testing.T) {
	// The min-length guard (R6): a 3-digit run would flood the index with
	// area-code noise, so it must not open the substring path.
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := index.Query(context.Background(), "311", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(hits) != 0 {
		t.Errorf("3-digit run must not substring-match, got %d hits", len(hits))
	}
}

func TestAbsentDigitsMatchNothing(t *testing.T) {
	// False-positive guard: digits no indexed number carries return no hits
	// instead of wildcard noise.
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := index.Query(context.Background(), "999999", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(hits) != 0 {
		t.Errorf("absent digits matched %d calls", len(hits))
	}
}

func TestMultiWordQuerySemantics(t *testing.T) {
	// The multi-word quality pass (2026-10-17, untouched since the
	// digits-shadow work): terms AND together regardless of order, an
	// absent second term empties the result, case folds, and a words +
	// number mix hits exactly once through the FTS leg + the digit shadow
	// (merge dedupes).
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	ctx := context.Background()

	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"both terms AND", "woolly invoice", []string{"conv-a"}},
		{"order is free", "invoice woolly", []string{"conv-a"}},
		{"absent term empties", "woolly zanzibar", nil},
		{"case folds", "WOOLLY INVOICE", []string{"conv-a"}},
		{"words + number mix", "woolly +15555550123", []string{"conv-a"}},
	}

	for _, tc := range cases {
		hits, err := index.Query(ctx, tc.query, 0)
		if err != nil {
			t.Fatalf("%s: Query(%q): %v", tc.name, tc.query, err)
		}

		got := make([]string, 0, len(hits))
		for _, hit := range hits {
			got = append(got, string(hit.NativeID))
		}

		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s: Query(%q) = %v, want %v", tc.name, tc.query, got, tc.want)
		}
	}
}

func TestRebuildRefreshesDigitTable(t *testing.T) {
	// The digits shadow table rides the wholesale rebuild: after reindexing
	// without a view, its numbers must stop matching (no drift).
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), searchViews()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	if err := index.Rebuild(context.Background(), nil); err != nil {
		t.Fatalf("Rebuild(empty): %v", err)
	}

	hits, err := index.Query(context.Background(), "7289", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(hits) != 0 {
		t.Errorf("digits matched after an empty rebuild: %d hits", len(hits))
	}
}
