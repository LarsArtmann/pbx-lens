package webhook

import (
	"slices"
	"strings"
	"testing"
)

const fixtureJSONL = `
{"received_at":"2026-10-09T19:51:35Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.initiated","occurred_at":"2026-10-09T19:51:35.123Z","payload":{"call_control_id":"ccid-1","call_session_id":"cs-1","direction":"incoming","from":"+17287289311","to":"+15551230001"}}}}
{"received_at":"2026-10-09T19:51:37Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.answered","occurred_at":"2026-10-09T19:51:37.456Z","payload":{"call_control_id":"ccid-1","direction":"incoming","from":"+17287289311","to":"+15551230001"}}}}
{"received_at":"2026-10-09T19:52:03Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.hangup","occurred_at":"2026-10-09T19:52:03.789Z","payload":{"call_control_id":"ccid-1","direction":"incoming","from":"+17287289311","to":"+15551230001","hangup_cause":"normal_clearing"}}}}
{"received_at":"2026-10-09T20:00:00Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.initiated","occurred_at":"2026-10-09T20:00:00Z","payload":{"call_control_id":"ccid-2","direction":"outgoing","from":"+15551230999","to":"+491701234567"}}}}
{"received_at":"2026-10-09T20:00:05Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.hangup","occurred_at":"2026-10-09T20:00:05Z","payload":{"call_control_id":"ccid-2","direction":"outgoing","from":"+15551230999","to":"+491701234567","hangup_cause":"busy"}}}}
{"received_at":"2026-10-09T20:01:00Z","remote":"192.0.2.10","body":{"data":{"event_type":"recording.saved","occurred_at":"2026-10-09T20:01:00Z","payload":{"call_control_id":"ccid-2"}}}}
not json at all
`

func mustParse(t *testing.T, input string) Result {
	t.Helper()

	res, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return res
}

func findObservation(t *testing.T, res Result, nativeID string) (bool, int) {
	t.Helper()

	for i, obs := range res.Observations {
		if string(obs.NativeID) == nativeID {
			return true, i
		}
	}

	return false, 0
}

func TestParseAggregatesPerCallControlID(t *testing.T) {
	t.Parallel()

	res := mustParse(t, fixtureJSONL)

	if len(res.Observations) != 2 {
		t.Fatalf("got %d observations, want 2: %+v", len(res.Observations), res.Observations)
	}

	ok, i := findObservation(t, res, "ccid-1")
	if !ok {
		t.Fatal("ccid-1 missing")
	}

	obs := res.Observations[i]
	if obs.Direction != "inbound" {
		t.Errorf("ccid-1 direction = %q, want inbound", obs.Direction)
	}

	if obs.DurationSecs != 28 {
		t.Errorf("ccid-1 duration = %d, want 28 (hangup 19:52:03 - initiated 19:51:35)", obs.DurationSecs)
	}

	if obs.FromNumber != "+17287289311" || obs.ToNumber != "+15551230001" {
		t.Errorf("ccid-1 parties = %q -> %q", obs.FromNumber, obs.ToNumber)
	}

	if obs.HangupCause != "normal_clearing" {
		t.Errorf("ccid-1 hangup cause = %q", obs.HangupCause)
	}

	if obs.Kind != "webhook-log" {
		t.Errorf("ccid-1 kind = %q", obs.Kind)
	}

	ok, j := findObservation(t, res, "ccid-2")
	if !ok {
		t.Fatal("ccid-2 missing")
	}

	if res.Observations[j].Direction != "outbound" {
		t.Errorf("ccid-2 direction = %q, want outbound", res.Observations[j].Direction)
	}

	if res.Observations[j].DurationSecs != 5 {
		t.Errorf("ccid-2 duration = %d, want 5", res.Observations[j].DurationSecs)
	}
}

func TestParseSkipsGarbageAndNonCallEvents(t *testing.T) {
	t.Parallel()

	res := mustParse(t, fixtureJSONL)

	// one non-call event (recording.saved) + one undecodable line
	if res.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", res.Skipped)
	}
}

func TestParseOrderIndependent(t *testing.T) {
	t.Parallel()

	lines := strings.Split(strings.TrimSpace(fixtureJSONL), "\n")

	forward := mustParse(t, strings.Join(lines, "\n"))

	reversed := make([]string, 0, len(lines))
	for _, line := range slices.Backward(lines) {
		reversed = append(reversed, line)
	}

	backward := mustParse(t, strings.Join(reversed, "\n"))

	fwdHash := map[string]string{}
	for _, obs := range forward.Observations {
		fwdHash[string(obs.NativeID)] = obs.ContentHash()
	}

	for _, obs := range backward.Observations {
		if fwdHash[string(obs.NativeID)] != obs.ContentHash() {
			t.Errorf("hash for %s differs between orders: %s vs %s",
				obs.NativeID, fwdHash[string(obs.NativeID)], obs.ContentHash())
		}
	}
}

func TestParseMissingPartiesRejected(t *testing.T) {
	t.Parallel()

	input := `{"received_at":"2026-10-09T21:00:00Z","remote":"192.0.2.10","body":{"data":{"event_type":"call.initiated","occurred_at":"2026-10-09T21:00:00Z","payload":{"call_control_id":"ccid-3","direction":"outgoing"}}}}`

	res := mustParse(t, input)

	if len(res.Observations) != 0 || res.Skipped != 1 {
		t.Fatalf("partyless call = %+v skipped=%d, want rejected", res.Observations, res.Skipped)
	}
}
