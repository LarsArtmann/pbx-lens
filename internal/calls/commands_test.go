package calls

import (
	"strings"
	"testing"
)

func TestCommandConstructorsValidateAndAddress(t *testing.T) {
	obs := validObservation()

	record, err := NewRecordCommand(obs)
	if err != nil {
		t.Fatalf("NewRecordCommand(valid): %v", err)
	}

	if record.Type() != CommandRecord {
		t.Errorf("record command type = %s, want %s", record.Type(), CommandRecord)
	}

	if _, err := NewRecordCommand(Observation{}); err == nil {
		t.Error("NewRecordCommand must reject an empty observation")
	}

	enrich, err := NewEnrichCommand(obs, Enrichment{})
	if err != nil {
		t.Fatalf("NewEnrichCommand(valid): %v", err)
	}

	if enrich.Type() != CommandEnrich {
		t.Errorf("enrich command type = %s, want %s", enrich.Type(), CommandEnrich)
	}

	if _, err := NewEnrichCommand(Observation{}, Enrichment{}); err == nil {
		t.Error("NewEnrichCommand must reject an empty observation")
	}

	retire, err := NewRetireCommand(SourceFreeSwitchCDR, "cdr-1", "retention:age")
	if err != nil {
		t.Fatalf("NewRetireCommand(valid): %v", err)
	}

	if retire.Type() != CommandRetire || retire.Reason != "retention:age" || retire.NativeID != "cdr-1" {
		t.Errorf("retire command = %+v, want the addressed identity + reason", retire)
	}

	if _, err := NewRetireCommand(SourceFreeSwitchCDR, "cdr-1", ""); err == nil {
		t.Error("NewRetireCommand must reject an empty audit reason")
	}

	if _, err := NewRetireCommand(SourceKind("bogus"), "cdr-1", "retention:age"); err == nil {
		t.Error("NewRetireCommand must reject an unknown source kind")
	}
}

func TestExcerptSelectsAndTruncates(t *testing.T) {
	cases := []struct {
		name     string
		turns    []TranscriptTurn
		maxRunes int
		want     string
	}{
		{
			"first user turn wins",
			[]TranscriptTurn{
				{Role: TurnRoleAssistant, Text: "hello"},
				{Role: TurnRoleUser, Text: "the thermostat is broken"},
			},
			200,
			"the thermostat is broken",
		},
		{
			"no user turn falls back to the first turn",
			[]TranscriptTurn{{Role: TurnRoleAssistant, Text: "hello there"}},
			200,
			"hello there",
		},
		{
			"truncates at the rune ceiling",
			[]TranscriptTurn{{Role: TurnRoleUser, Text: strings.Repeat("x", 300)}},
			200,
			strings.Repeat("x", 200),
		},
		{
			"no turns is empty",
			nil,
			200,
			"",
		},
	}

	for _, tc := range cases {
		if got := Excerpt(tc.turns, tc.maxRunes); got != tc.want {
			t.Errorf("%s: Excerpt = %q, want %q", tc.name, got, tc.want)
		}
	}
}
