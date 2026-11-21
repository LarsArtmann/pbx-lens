package query

import (
	"testing"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestMergeCallKeepsPrimaryAuthority pins the read-time join's merge rule:
// the requested row wins every fact; a sibling only fills true gaps.
func TestMergeCallKeepsPrimaryAuthority(t *testing.T) {
	primaryObs := aiAgentTranscript()
	primaryObs.Transcript = []calls.TranscriptTurn{{Role: calls.TurnRoleUser, Text: "primary words"}}
	primaryObs.HangupCause = "NORMAL_CLEARING"

	primary := calls.ViewFrom(primaryObs, calls.Enrichment{
		CostMicros:     1234,
		CostCurrency:   "USD",
		InsightSummary: "collector",
	})

	siblingObs := aiAgentCDR()
	siblingObs.Transcript = []calls.TranscriptTurn{{Role: calls.TurnRoleUser, Text: "sibling words"}}
	siblingObs.ToNumber = "9999"

	sibling := calls.ViewFrom(siblingObs, calls.Enrichment{})

	merged, changed := mergeCall(primary, sibling)
	if changed {
		t.Error("merge reported a change although the primary carried every fact")
	}

	if merged.NativeID != primary.NativeID || merged.Kind != primary.Kind {
		t.Errorf("identity changed: %+v", merged)
	}

	if len(merged.Transcript) != 1 || merged.Transcript[0].Text != "primary words" {
		t.Errorf("primary transcript was overwritten: %+v", merged.Transcript)
	}

	if merged.ToNumber != primary.ToNumber {
		t.Errorf("primary ToNumber overwritten: %q", merged.ToNumber)
	}

	if merged.CostMicros != 1234 || merged.CostCurrency != "USD" || merged.InsightSummary != "collector" {
		t.Errorf("enrichment lost across merge: %+v", merged)
	}
}

// TestSiblingKinds pins the join map: only the FreeSWITCH uuid is shared
// between sources, so no other pair may be joined on its native id.

func TestSiblingKinds(t *testing.T) {
	tests := []struct {
		kind calls.SourceKind
		want []calls.SourceKind
	}{
		{kind: calls.SourceFreeSwitchCDR, want: []calls.SourceKind{calls.SourceAgentTranscript}},
		{kind: calls.SourceAgentTranscript, want: []calls.SourceKind{calls.SourceFreeSwitchCDR}},
		{kind: calls.SourceTelnyxConversation, want: nil},
		{kind: calls.SourceWebhookLog, want: nil},
	}

	for _, test := range tests {
		got := siblingKinds(test.kind)

		if len(got) != len(test.want) {
			t.Errorf("siblingKinds(%q) = %v, want %v", test.kind, got, test.want)

			continue
		}

		for i := range got {
			if got[i] != test.want[i] {
				t.Errorf("siblingKinds(%q)[%d] = %q, want %q", test.kind, i, got[i], test.want[i])
			}
		}
	}
}
