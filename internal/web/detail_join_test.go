package web

import (
	"net/url"
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

// TestMergeCallKeepsPrimaryAuthority pins the precedence rule: the requested
// row's own observations win, and merging never rewrites the row's identity
// or the enrichment another pass attached.
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

// TestRecordingStemSweepFindsUnknownDialplanName pins the general fix: a
// destination the dialplan invents later (here a made-up suffix) must still
// resolve through the {uuid}_* sweep, not silently orphan the audio.
func TestRecordingStemSweepFindsUnknownDialplanName(t *testing.T) {
	dir := t.TempDir()

	wav := []byte("RIFF-unknown-stem-recording")
	if err := os.WriteFile(filepath.Join(dir, joinUUID+"_conference_room_9.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		aiAgentCDR(),
	)

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+joinUUID+"/recording")
	if status != 200 {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, audio)
	}

	if audio != string(wav) {
		t.Errorf("recording body = %q, want the local file bytes", audio)
	}

	_, body := getBody(t, mux, "/calls/freeswitch-cdr/"+joinUUID)
	if !strings.Contains(body, joinUUID+"_conference_room_9.wav") {
		t.Error("detail page does not name the swept recording file")
	}
}

// TestDetailCounterpartHistoryCountsEveryCall pins the "who called this
// number before?" join: every representative call involving the detail
// call's outside party counts once (a CDR+transcript pair is ONE call), the
// newest one names the "latest", and the item drills into ?number=.
func TestDetailCounterpartHistoryCountsEveryCall(t *testing.T) {
	t.Parallel()

	base := joinStarted()
	theirs := "+15557770111"

	// Three calls with the same counterpart, the third the newest; the
	// CDR+transcript pair shares one uuid so it must collapse to one call.
	first := webObservation("hist-first", base)
	first.FromNumber = "9100"
	first.ToNumber = theirs

	pairCDR := aiAgentCDR()
	pairCDR.FromNumber = "9100"
	pairCDR.ToNumber = theirs
	pairCDR.StartedAt = base.Add(time.Hour)

	newest := webObservation("hist-newest", base.Add(2*time.Hour))
	newest.FromNumber = "9100"
	newest.ToNumber = theirs

	// A call to a DIFFERENT counterpart must not count.
	other := webObservation("hist-other", base.Add(3*time.Hour))
	other.FromNumber = "9100"
	other.ToNumber = "+15559999999"

	sys := newTestSystem(t, first, pairCDR, aiAgentTranscript(), newest, other)

	page, found, err := CallDetailPage(t.Context(), sys, string(calls.SourceTelnyxConversation), "hist-newest")
	if err != nil || !found {
		t.Fatalf("CallDetailPage: found=%v err=%v", found, err)
	}

	if page.CounterpartHistory == nil {
		t.Fatal("CounterpartHistory = nil for a call with a counterpart")
	}

	if page.CounterpartHistory.Number != theirs {
		t.Errorf("counterpart = %q, want %q", page.CounterpartHistory.Number, theirs)
	}

	if page.CounterpartHistory.Calls != 3 {
		t.Errorf("counterpart calls = %d, want 3 (the pair collapses to one)", page.CounterpartHistory.Calls)
	}

	if !page.CounterpartHistory.LastSeen.Equal(newest.StartedAt) {
		t.Errorf("last seen = %v, want the newest call %v", page.CounterpartHistory.LastSeen, newest.StartedAt)
	}

	_, body := getBody(t, muxForSystem(sys), "/calls/telnyx-conversation/hist-newest")
	if !strings.Contains(body, "3 calls with "+theirs) {
		t.Error("detail page missing the counterpart history line")
	}

	if !strings.Contains(body, "/?number="+url.QueryEscape(theirs)) {
		t.Error("counterpart history does not drill into ?number=")
	}
}
