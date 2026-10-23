package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// joinUUID is the FreeSWITCH call uuid shared by every source that observed
// the call. The live instance of this shape (2026-10-23, 12:09): a 30 s
// inbound call to the voice agent's extension 9100 whose CDR row carried the
// facts, whose transcript row carried the words, and whose recording lived at
// {uuid}_ai_agent.wav — none reachable from the CDR detail page.
const joinUUID = "9d69060b-1003-401a-8a0a-f083a81c3b9c"

func joinStarted() time.Time {
	return time.Date(2026, 10, 23, 10, 9, 51, 0, time.UTC)
}

func aiAgentCDR() calls.Observation {
	return calls.Observation{
		NativeID:     joinUUID,
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    calls.DirectionOutbound,
		StartedAt:    joinStarted(),
		DurationSecs: 30,
		FromNumber:   "4084757593",
		ToNumber:     "9100",
		HangupCause:  "NORMAL_CLEARING",
		IsAI:         true,
	}
}

func aiAgentTranscript() calls.Observation {
	return calls.Observation{
		NativeID:     joinUUID,
		Kind:         calls.SourceAgentTranscript,
		Direction:    calls.DirectionOutbound,
		StartedAt:    joinStarted(),
		DurationSecs: 30,
		FromNumber:   "4084757593",
		ToNumber:     "9100",
		IsAI:         true,
		Transcript: []calls.TranscriptTurn{
			{Role: calls.TurnRoleAssistant, Text: "Guten Tag, hier ist Woolly."},
			{Role: calls.TurnRoleUser, Text: "Ich wollte wegen der Rechnung fragen."},
		},
	}
}

// TestDetailCDRJoinsAgentTranscript pins the cross-source read: a call the
// agent transcribed is ONE call, so the CDR page shows the transcript the
// agent-transcript row recorded by the same uuid. Before the join the page
// said "No transcript" for the very call whose transcript sat one stream
// away.
func TestDetailCDRJoinsAgentTranscript(t *testing.T) {
	mux := newTestServer(t, aiAgentCDR(), aiAgentTranscript())

	status, body := getBody(t, mux, "/calls/freeswitch-cdr/"+joinUUID)
	if status != 200 {
		t.Fatalf("detail status = %d, want 200", status)
	}

	for _, want := range []string{"Guten Tag, hier ist Woolly.", "Ich wollte wegen der Rechnung fragen.", "agent-transcript"} {
		if !strings.Contains(body, want) {
			t.Errorf("CDR detail missing %q — the joined transcript must render", want)
		}
	}

	if strings.Contains(body, "No transcript") {
		t.Error("CDR detail still renders the no-transcript empty state despite a joined transcript")
	}
}

// TestDetailTranscriptJoinsCDRFacts is the mirror direction: a transcript
// row opened directly must show the call's parties and duration even when the
// ingest join could not fill them (the transcript file existed before the CDR
// did).
func TestDetailTranscriptJoinsCDRFacts(t *testing.T) {
	transcript := aiAgentTranscript()
	transcript.FromNumber = ""
	transcript.ToNumber = ""
	transcript.DurationSecs = 0

	mux := newTestServer(t, aiAgentCDR(), transcript)

	status, body := getBody(t, mux, "/calls/agent-transcript/"+joinUUID)
	if status != 200 {
		t.Fatalf("detail status = %d, want 200", status)
	}

	for _, want := range []string{"4084757593", "9100", "freeswitch-cdr"} {
		if !strings.Contains(body, want) {
			t.Errorf("transcript detail missing %q — the joined CDR facts must render", want)
		}
	}
}

// TestDetailCDRShowsLocalAIAgentRecording pins the recording half: the
// dialplan names agent recordings {uuid}_ai_agent.wav, which matched none of
// the pre-join candidate stems, so the page claimed no recording and the
// proxy 404'd. The player must render and the route must serve the bytes.
func TestDetailCDRShowsLocalAIAgentRecording(t *testing.T) {
	dir := t.TempDir()

	wav := []byte("RIFF-ai-agent-recording")
	if err := os.WriteFile(filepath.Join(dir, joinUUID+"_ai_agent.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		aiAgentCDR(), aiAgentTranscript(),
	)

	status, body := getBody(t, mux, "/calls/freeswitch-cdr/"+joinUUID)
	if status != 200 {
		t.Fatalf("detail status = %d, want 200", status)
	}

	if !strings.Contains(body, "<audio") {
		t.Error("detail page renders no player although the local recording exists")
	}

	if strings.Contains(body, "No recording reference indexed") {
		t.Error("detail page still claims no recording")
	}

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+joinUUID+"/recording")
	if status != 200 {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, audio)
	}

	if audio != string(wav) {
		t.Errorf("recording body = %q, want the local file bytes", audio)
	}
}

// TestDetailCDRShowsLocalPSTNRecording covers the other dialplan stem this
// deployment writes: outbound PSTN legs record {uuid}_pstn.wav, while the CDR
// destination is the dialled number.
func TestDetailCDRShowsLocalPSTNRecording(t *testing.T) {
	dir := t.TempDir()

	wav := []byte("RIFF-pstn-recording")
	if err := os.WriteFile(filepath.Join(dir, joinUUID+"_pstn.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cdr := aiAgentCDR()
	cdr.ToNumber = "+4915112345678"
	cdr.IsAI = false

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		cdr,
	)

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+joinUUID+"/recording")
	if status != 200 {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, audio)
	}

	if audio != string(wav) {
		t.Errorf("recording body = %q, want the local file bytes", audio)
	}
}

// TestDetailCDRWithoutRecordingStaysHonest pins the negative: no file on
// disk means no player and an honest empty state, not a broken <audio>.
func TestDetailCDRWithoutRecordingStaysHonest(t *testing.T) {
	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: t.TempDir()}},
		aiAgentCDR(), aiAgentTranscript(),
	)

	_, body := getBody(t, mux, "/calls/freeswitch-cdr/"+joinUUID)

	if strings.Contains(body, "<audio") {
		t.Error("detail page renders a player without a recording file")
	}

	if !strings.Contains(body, "No recording file") {
		t.Error("detail page must explain that no recording file exists for the call")
	}
}
