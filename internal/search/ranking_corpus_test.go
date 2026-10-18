package search

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// rankingCorpus builds the 100+ view synthetic corpus the ranking spot
// checks run against (T21/F82): 100 filler conversations (unique words,
// number tails restricted to digits 0-4 so no filler accidentally carries a
// probe run) plus six instrumented targets with controlled term and digit
// structure.
func rankingCorpus() []calls.CallView {
	base := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)

	views := make([]calls.CallView, 0, 106)

	for i := range 100 {
		views = append(views, calls.CallView{
			NativeID:   calls.NativeID(fmt.Sprintf("filler-%03d", i)),
			Kind:       calls.SourceTelnyxConversation,
			StartedAt:  base.Add(time.Duration(i) * time.Minute),
			FromNumber: fmt.Sprintf("+155501%d%03d", i%5, i),
			ToNumber:   fmt.Sprintf("+155502%d%03d", (i+1)%5, i),
			Transcript: []calls.TranscriptTurn{
				{Role: "user", Text: fmt.Sprintf("routine call %03d about paperwork", i)},
			},
		})
	}

	thermo := func(id, number string, mentions int) calls.CallView {
		turns := make([]calls.TranscriptTurn, 0, mentions)

		for range mentions {
			turns = append(turns, calls.TranscriptTurn{Role: "user", Text: "the thermostat reading looks wrong"})
		}

		return calls.CallView{
			NativeID: calls.NativeID(id), Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: number, ToNumber: "+15550000001",
			Transcript: turns,
		}
	}

	views = append(views,
		thermo("rank-hot", "+15558765001", 3),
		thermo("rank-cold", "+15558765002", 1),
		calls.CallView{
			NativeID: "rank-both", Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: "+15558765003", ToNumber: "+15550000001",
			Transcript: []calls.TranscriptTurn{
				{Role: "user", Text: "the gizmo and the thermostat both flaked"},
			},
		},
		calls.CallView{
			NativeID: "digits-inside", Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: "+493012340001", ToNumber: "+15550000001",
			Transcript: []calls.TranscriptTurn{{Role: "user", Text: "call about the meter"}},
		},
		calls.CallView{
			NativeID: "digits-overlap-a", Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: "+15558887111", ToNumber: "+15550000001",
			Transcript: []calls.TranscriptTurn{{Role: "user", Text: "overlap corpus call a"}},
		},
		calls.CallView{
			NativeID: "digits-overlap-b", Kind: calls.SourceTelnyxConversation,
			StartedAt: base, FromNumber: "+15558887222", ToNumber: "+15550000001",
			Transcript: []calls.TranscriptTurn{{Role: "user", Text: "overlap corpus call b"}},
		},
	)

	return views
}

func corpusQuery(t *testing.T, index *Index, query string) []string {
	t.Helper()

	hits, err := index.Query(context.Background(), query, 0)
	if err != nil {
		t.Fatalf("Query(%q): %v", query, err)
	}

	got := make([]string, 0, len(hits))
	for _, hit := range hits {
		got = append(got, string(hit.NativeID))
	}

	return got
}

// TestRankingOnSyntheticCorpus pins the bm25 ORDERING expectations on the
// 100+ view corpus (T21/F83-F85): term frequency ranks the heavier doc
// first, a doc matching both terms of a multi-word query outranks a
// single-term doc, and the digit shadow finds runs inside longer numbers
// and across overlapping prefixes with the FTS+digit merge deduping.
func TestRankingOnSyntheticCorpus(t *testing.T) {
	index := openTestIndex(t)

	if err := index.Rebuild(context.Background(), rankingCorpus()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	t.Run("term frequency ranks the heavier document first", func(t *testing.T) {
		got := corpusQuery(t, index, "thermostat")

		// rank-both legitimately matches too (its turn mentions the word);
		// the pinned fact is the ORDER of the frequency pair.
		if len(got) != 3 {
			t.Fatalf("thermostat hits = %v, want the three matching docs", got)
		}

		if got[0] != "rank-hot" || got[1] != "rank-cold" {
			t.Errorf("thermostat order = %v, want rank-hot before rank-cold (bm25 term frequency)", got)
		}
	})

	t.Run("both-terms document outranks the single-term heavyweight", func(t *testing.T) {
		got := corpusQuery(t, index, "gizmo thermostat")

		if len(got) == 0 {
			t.Fatal("gizmo thermostat returned nothing")
		}

		if got[0] != "rank-both" {
			t.Errorf("gizmo thermostat first hit = %s, want rank-both (both terms beat one term's frequency)", got[0])
		}
	})

	t.Run("four-digit run inside a longer number", func(t *testing.T) {
		got := corpusQuery(t, index, "1234")

		if strings.Join(got, ",") != "digits-inside" {
			t.Errorf("1234 hits = %v, want exactly [digits-inside]", got)
		}
	})

	t.Run("shared prefix run hits every overlapping number", func(t *testing.T) {
		got := corpusQuery(t, index, "8887")

		if len(got) != 2 {
			t.Fatalf("8887 hits = %v, want both overlap docs", got)
		}
	})

	t.Run("discriminating run isolates one overlap member", func(t *testing.T) {
		for _, tc := range []struct{ run, want string }{
			{"8711", "digits-overlap-a"},
			{"8722", "digits-overlap-b"},
		} {
			got := corpusQuery(t, index, tc.run)

			if strings.Join(got, ",") != tc.want {
				t.Errorf("%s hits = %v, want exactly [%s]", tc.run, got, tc.want)
			}
		}
	})

	t.Run("full number query dedupes the FTS and digit legs", func(t *testing.T) {
		got := corpusQuery(t, index, "15558765001")

		if strings.Join(got, ",") != "rank-hot" {
			t.Errorf("full-number query hits = %v, want exactly [rank-hot] once", got)
		}
	})
}
