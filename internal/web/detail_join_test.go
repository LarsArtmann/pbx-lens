package web

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/query"
	"github.com/larsartmann/pbx-lens/internal/query/querytest"
)

// aiAgentCDR and aiAgentTranscript delegate to the shared querytest fixtures
// — one forensic join pair serves the query, web, and serve suites.
func aiAgentCDR() calls.Observation        { return querytest.AgentCDR() }
func aiAgentTranscript() calls.Observation { return querytest.AgentTranscript() }

// TestDetailCDRJoinsAgentTranscript pins the cross-source read: a call the
// agent transcribed is ONE call, so the CDR page shows the transcript the
// agent-transcript row recorded by the same uuid. Before the join the page
// said "No transcript" for the very call whose transcript sat one stream
// away.
func TestDetailCDRJoinsAgentTranscript(t *testing.T) {
	mux := newTestServer(t, aiAgentCDR(), aiAgentTranscript())

	status, body := getBody(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID)
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

	status, body := getBody(t, mux, "/calls/agent-transcript/"+querytest.JoinUUID)
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
	if err := os.WriteFile(filepath.Join(dir, querytest.JoinUUID+"_ai_agent.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		aiAgentCDR(), aiAgentTranscript(),
	)

	status, body := getBody(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID)
	if status != 200 {
		t.Fatalf("detail status = %d, want 200", status)
	}

	if !strings.Contains(body, "<audio") {
		t.Error("detail page renders no player although the local recording exists")
	}

	if strings.Contains(body, "No recording reference indexed") {
		t.Error("detail page still claims no recording")
	}

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID+"/recording")
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
	if err := os.WriteFile(filepath.Join(dir, querytest.JoinUUID+"_pstn.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cdr := aiAgentCDR()
	cdr.ToNumber = "+4915112345678"
	cdr.IsAI = false

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		cdr,
	)

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID+"/recording")
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

	_, body := getBody(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID)

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
// TestRecordingStemSweepFindsUnknownDialplanName pins the general fix: a
// destination the dialplan invents later (here a made-up suffix) must still
// resolve through the {uuid}_* sweep, not silently orphan the audio.
func TestRecordingStemSweepFindsUnknownDialplanName(t *testing.T) {
	dir := t.TempDir()

	wav := []byte("RIFF-unknown-stem-recording")
	if err := os.WriteFile(filepath.Join(dir, querytest.JoinUUID+"_conference_room_9.wav"), wav, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	mux := newTestServerWith(t,
		Options{Recordings: RecordingOptions{RecordingsDir: dir}},
		aiAgentCDR(),
	)

	status, audio, _ := getWithHeaders(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID+"/recording")
	if status != 200 {
		t.Fatalf("recording status = %d, want 200 (body: %s)", status, audio)
	}

	if audio != string(wav) {
		t.Errorf("recording body = %q, want the local file bytes", audio)
	}

	_, body := getBody(t, mux, "/calls/freeswitch-cdr/"+querytest.JoinUUID)
	if !strings.Contains(body, querytest.JoinUUID+"_conference_room_9.wav") {
		t.Error("detail page does not name the swept recording file")
	}
}

// TestDetailCounterpartHistoryCountsEveryCall pins the "who called this
// number before?" join: every representative call involving the detail
// call's outside party counts once (a CDR+transcript pair is ONE call), the
// newest one names the "latest", and the item drills into ?number=.
func TestDetailCounterpartHistoryCountsEveryCall(t *testing.T) {
	t.Parallel()

	base := querytest.JoinStarted()
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

	page, found, err := query.CallDetailPage(t.Context(), sys, string(calls.SourceTelnyxConversation), "hist-newest")
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
