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

	index, err := Open(filepath.Join(t.TempDir(), "search.db"))
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
