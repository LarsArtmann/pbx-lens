//go:build live

// Package telnyx's live-tagged correction-wave measurement.
//
// Run it BEFORE deploying a hash-affecting change (a ContentHash schema bump,
// a turn-time correction, a mapper change). It answers the one question a
// green unit suite cannot: how many real streams will the deploy rewrite?
//
//	TELNYX_API_KEY_FILE=~/.pbx-prod-secrets/telnyx_api_key \
//	  go test -tags live ./internal/ingest/telnyx/ -run TestCorrectionWave -v -timeout 10m
//
// It only READS: conversations and their messages. Recordings are not fetched
// (their refs do not change the turn-time question) and no index is opened.
package telnyx

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// waveBaselineUnderTest is the turn-time rule this measurement compares the
// CURRENT mapper against: v1 ordered and stamped turns by created_at, which
// is Telnyx's PERSISTENCE time, not the speech time (see Message.SpokeAt).
func waveBaselineTurns(msgs []Message) []string {
	type turn struct {
		at   time.Time
		text string
	}

	kept := make([]turn, 0, len(msgs))

	for _, msg := range msgs {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}

		if strings.TrimSpace(msg.Text) == "" {
			continue
		}

		kept = append(kept, turn{at: msg.CreatedAt.Time, text: msg.Text})
	}

	// Stable chronological order: the API serves newest-first, so the
	// baseline sorts exactly like turnsByTime did.
	for i := 1; i < len(kept); i++ {
		for j := i; j > 0 && kept[j].at.Before(kept[j-1].at); j-- {
			kept[j], kept[j-1] = kept[j-1], kept[j]
		}
	}

	out := make([]string, 0, len(kept))
	for _, t := range kept {
		out = append(out, t.at.UTC().Truncate(time.Second).Format(time.RFC3339)+"|"+t.text)
	}

	return out
}

func TestCorrectionWave(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("TELNYX_API_KEY"))
	if key == "" {
		if file := strings.TrimSpace(os.Getenv("TELNYX_API_KEY_FILE")); file != "" {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read key file %s: %v", file, err)
			}

			key = strings.TrimSpace(string(raw))
		}
	}

	if key == "" {
		t.Skip("no TELNYX_API_KEY or TELNYX_API_KEY_FILE: the live measurement cannot run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	client := NewClient(key, "")

	convs, err := client.ListConversations(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}

	var (
		hashShifted  int
		timeShifted  int
		orderShifted int
		silenceOnly  int
		markerTotal  int
		turnTotal    int
		fetchFailure int
	)

	for _, conv := range convs {
		msgs, msgErr := client.ListMessages(ctx, conv.ID)
		if msgErr != nil {
			fetchFailure++

			t.Logf("%s: messages unavailable: %v", conv.ID, msgErr)

			continue
		}

		current := turnsByTimeOnly(msgs)
		baseline := waveBaselineTurns(msgs)

		if strings.Join(current, "\n") != strings.Join(baseline, "\n") {
			hashShifted++

			// Split the two causes: the transcript hash is order-sensitive,
			// so a stable set with a different sequence moves it too.
			if !sameStringSet(current, baseline) {
				timeShifted++
			} else {
				orderShifted++
			}
		}

		userTurns := 0

		for _, msg := range msgs {
			switch msg.Role {
			case "user":
				userTurns++
				turnTotal++
			case "assistant":
				turnTotal++
			default:
				markerTotal++
			}
		}

		if userTurns == 0 {
			silenceOnly++
		}
	}

	t.Logf("correction wave over %d conversations:", len(convs))
	t.Logf("  transcript hash moves: %d", hashShifted)
	t.Logf("    of which turn TIMES move (created_at -> sent_at): %d", timeShifted)
	t.Logf("    of which only the ORDER moves (identical stamps): %d", orderShifted)
	t.Logf("  zero-user-turn conversations (marker backlog): %d", silenceOnly)
	t.Logf("  non-spoken source events available to persist: %d", markerTotal)
	t.Logf("  spoken turns total: %d", turnTotal)
	t.Logf("  message-fetch failures: %d", fetchFailure)
	t.Logf("NOTE: every conversation rewrites once regardless (ObservationSchemaVersion bump);")
	t.Logf("      the local-file sources (CDR/webhook/transcripts) heal on the next poller pass for free.")
}

// sameStringSet reports whether two turn projections carry the same stamps and
// texts regardless of order — the split between "the times were wrong" and
// "only the sequence changed".
func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	sorted := make([]string, len(left))
	copy(sorted, left)

	compare := make([]string, len(right))
	copy(compare, right)

	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	for i := 1; i < len(compare); i++ {
		for j := i; j > 0 && compare[j] < compare[j-1]; j-- {
			compare[j], compare[j-1] = compare[j-1], compare[j]
		}
	}

	for i := range sorted {
		if sorted[i] != compare[i] {
			return false
		}
	}

	return true
}

// turnsByTimeOnly projects the CURRENT mapper's turn ordering onto the plain
// strings the baseline uses, so the comparison stays about TIMES, not types.
func turnsByTimeOnly(msgs []Message) []string {
	turns, _ := turnsByTime(msgs)

	out := make([]string, 0, len(turns))
	for _, turn := range turns {
		out = append(out, turn.At.UTC().Truncate(time.Second).Format(time.RFC3339)+"|"+turn.Text)
	}

	return out
}
