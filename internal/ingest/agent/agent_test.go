package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

const fixtureTranscript = `{"type":"start","ts":1760041885,"uuid":"f0e1d2c3-1111-4aaa-9aaa-000000000001","models":["gpt-rt"]}` + "\n" +
	`{"type":"turn","role":"agent","text":" Hi, this is Woolly calling. "}` + "\n" +
	`{"type":"turn","role":"caller","text":"Did you take my two eggs?"}` + "\n" +
	`{"type":"turn","role":"caller","text":""}` + "\n" +
	`{"type":"turn","role":"system","text":"invisible"}` + "\n" +
	`{"type":"end","reason":"caller_hungup"}` + "\n"

func TestParseNormalizesRolesAndFilters(t *testing.T) {
	t.Parallel()

	transcript, err := Parse(strings.NewReader(fixtureTranscript), "fallback-uuid")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if transcript.UUID != "f0e1d2c3-1111-4aaa-9aaa-000000000001" {
		t.Errorf("uuid = %q, want start-line uuid", transcript.UUID)
	}

	if !transcript.StartedAt.Equal(time.Unix(1760041885, 0).UTC()) {
		t.Errorf("started at = %v", transcript.StartedAt)
	}

	if len(transcript.Turns) != 2 {
		t.Fatalf("turns = %d, want 2 (empty + system filtered): %+v", len(transcript.Turns), transcript.Turns)
	}

	if transcript.Turns[0].Role != "assistant" || transcript.Turns[0].Text != "Hi, this is Woolly calling." {
		t.Errorf("turn 0 = %+v", transcript.Turns[0])
	}

	if transcript.Turns[1].Role != "user" {
		t.Errorf("turn 1 role = %q, want user", transcript.Turns[1].Role)
	}

	if transcript.Turns[1].At.IsZero() == false {
		t.Errorf("per-turn At should stay zero (writer stamps none)")
	}

	if transcript.EndReason != "caller_hungup" {
		t.Errorf("end reason = %q", transcript.EndReason)
	}
}

func TestObservationWithoutCDRJoin(t *testing.T) {
	t.Parallel()

	transcript, err := Parse(strings.NewReader(fixtureTranscript), "u1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	obs, err := Observation(transcript, nil)
	if err != nil {
		t.Fatalf("Observation without join should validate (agent transcripts may lack numbers): %v", err)
	}

	if !obs.IsAI {
		t.Error("agent transcripts must badge IsAI")
	}

	if obs.Kind != calls.SourceAgentTranscript {
		t.Errorf("kind = %q", obs.Kind)
	}

	if obs.TranscriptExcerpt != "Did you take my two eggs?" {
		t.Errorf("excerpt = %q, want first user turn", obs.TranscriptExcerpt)
	}

	if obs.Direction != calls.DirectionOutbound {
		t.Errorf("direction = %q, want default outbound", obs.Direction)
	}

	if obs.HangupCause != "caller_hungup" {
		t.Errorf("hangup cause = %q, want end reason as fallback", obs.HangupCause)
	}

	if obs.StartedAt.Unix() != 1760041885 {
		t.Errorf("started at anchored by start ts: %v", obs.StartedAt)
	}
}

func TestObservationWithCDRJoin(t *testing.T) {
	t.Parallel()

	transcript, err := Parse(strings.NewReader(fixtureTranscript), "u1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	joined := &calls.CallView{
		Direction:    calls.DirectionInbound,
		StartedAt:    time.Unix(1760041880, 0).UTC(),
		DurationSecs: 42,
		FromNumber:   "+17287289311",
		ToNumber:     "+15551230001",
		HangupCause:  "normal_clearing",
	}

	obs, err := Observation(transcript, joined)
	if err != nil {
		t.Fatalf("Observation: %v", err)
	}

	if obs.Direction != calls.DirectionInbound || obs.DurationSecs != 42 ||
		obs.FromNumber != "+17287289311" || obs.ToNumber != "+15551230001" {
		t.Errorf("joined fields wrong: %+v", obs)
	}

	if obs.HangupCause != "normal_clearing" {
		t.Errorf("cdr cause should win over end reason: %q", obs.HangupCause)
	}
}

func TestParseDirSkipsEmptyAndUsesStem(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "stem-uuid-1.jsonl"), []byte(fixtureTranscript), 0o600); err != nil {
		t.Fatal(err)
	}

	// turns exist but neither start uuid nor usable content
	if err := os.WriteFile(filepath.Join(dir, "empty.jsonl"), []byte("\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// stem is the fallback uuid when the start line carries none
	noStart := `{"type":"turn","role":"caller","text":"hello"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "stem-only.jsonl"), []byte(noStart), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a transcript"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	if len(res.Transcripts) != 2 || res.Skipped != 1 {
		t.Fatalf("transcripts=%d skipped=%d, want 2/1", len(res.Transcripts), res.Skipped)
	}

	byUUID := map[string]Transcript{}
	for _, tr := range res.Transcripts {
		byUUID[tr.UUID] = tr
	}

	if _, ok := byUUID["f0e1d2c3-1111-4aaa-9aaa-000000000001"]; !ok {
		t.Error("start-line uuid missing")
	}

	if _, ok := byUUID["stem-only"]; !ok {
		t.Error("stem fallback uuid missing")
	}
}
