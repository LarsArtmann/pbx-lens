package cdr

import (
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

const fixture13 = `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","f0e1d2c3-1111-4aaa-9aaa-000000000001","b0a9-2222",""
"","1000","+17287289311","default","2026-10-09 20:20:00","2026-10-09 20:20:02","2026-10-09 20:20:28","28","26","NORMAL_CLEARING","aaaa2222-3333-4bbb-8bbb-000000000002","","artmann"
`

// fixture13MOS is the default template with mod_cdr_csv's extra
// rtp_audio_in_mos column appended.
const fixture13MOS = `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","f0e1d2c3-1111-4aaa-9aaa-000000000001","b0a9-2222","","4.4"` + "\n"

// fixture13MOSGarbage has an appended extra column that is not a MOS value.
const fixture13MOSGarbage = `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","f0e1d2c3-1111-4aaa-9aaa-000000000001","b0a9-2222","","not-a-mos"` + "\n"

const fixture18 = `"","+14084757593","2000","default","Lars","SOFTA","SOFTB","bridge","park","2026-10-09 21:00:00","2026-10-09 21:00:01","2026-10-09 21:00:10","10","9","NORMAL_CLEARING","2","cccc4444-5555-4ccc-8ccc-000000000003","";` + "\n"

func TestParseDefaultTemplate(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture13))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 2 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	in := res.Observations[0]
	if in.NativeID != "f0e1d2c3-1111-4aaa-9aaa-000000000001" || in.Direction != calls.DirectionInbound {
		t.Fatalf("inbound row wrong: %+v", in)
	}

	if in.FromNumber != "17287289311" || in.ToNumber != "2000" || in.HangupCause != "NORMAL_CLEARING" {
		t.Fatalf("participants wrong: %+v", in)
	}

	if in.StartedAt.UTC().Format(stampLayout) != "2026-10-09 19:51:25" {
		t.Fatalf("stamp wrong: %v", in.StartedAt)
	}

	if in.DurationSecs != 25 {
		t.Fatalf("billsec wrong: %d", in.DurationSecs)
	}

	out := res.Observations[1]
	if out.Direction != calls.DirectionOutbound {
		t.Fatalf("default context must be outbound: %+v", out)
	}

	if out.Kind != calls.SourceFreeSwitchCDR {
		t.Fatalf("kind wrong: %+v", out)
	}
}

func TestParseFallbackTemplateWithQuirks(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture18))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 1 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	obs := res.Observations[0]
	if obs.NativeID != "cccc4444-5555-4ccc-8ccc-000000000003" {
		t.Fatalf("uuid column wrong: %+v", obs)
	}

	// An inbound PSTN call transferred into "default": destination 2000 is an
	// internal extension and the caller is an external number, so the transfer
	// heuristic reads it as inbound even though the context column no longer
	// says "public". This fixture IS the live shape the heuristic exists for.
	if obs.DurationSecs != 9 || obs.Direction != calls.DirectionInbound {
		t.Fatalf("mapped fields wrong: %+v", obs)
	}
}

// fixture13BareQuote is the shape vanilla mod_cdr_csv actually writes
// on every call: the compiled-in default template's tail carries
// `,"", ""` — a comma-space before the final empty fields, i.e. a bare
// quote in a non-quoted field. Strict CSV rejects the line outright
// (the telephony-lens VM suite caught this live, 2026-10-12).
const fixture13BareQuote = `"","0000000000","2000","default","2026-10-12 04:56:29","","2026-10-12 04:56:29","0","0","NORMAL_CLEARING","88b26a7e-e923-439b-b92e-c48418cb4d03","", ""
"","0000000000","2000","default","2026-10-12 04:56:29","","2026-10-12 04:56:29","0","0","NORMAL_CLEARING","639677df-0787-48db-81f2-27adf6587eff","", ""
`

func TestParseVanillaBareQuoteTail(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture13BareQuote))
	if err != nil {
		t.Fatalf("vanilla Master.csv tail must parse: %v", err)
	}

	if len(res.Observations) != 2 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	for _, in := range res.Observations {
		if in.ToNumber != "2000" || in.HangupCause != "NORMAL_CLEARING" {
			t.Fatalf("row wrong: %+v", in)
		}

		if in.FromNumber != "0000000000" {
			t.Fatalf("bare-quote residue leaked into participants: %+v", in)
		}
	}
}

func TestParseSkipsMalformedRows(t *testing.T) {
	input := "not,enough,fields\n" + fixture13 + `"2026-13-99 99:99:99",x`

	res, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 2 {
		t.Fatalf("want 2 good rows, got %d", len(res.Observations))
	}

	if res.Skipped != 2 {
		t.Fatalf("want 2 skipped, got %d", res.Skipped)
	}
}

func TestParseStreamIDsDeterministic(t *testing.T) {
	a, err := Parse(strings.NewReader(fixture13))
	if err != nil {
		t.Fatal(err)
	}

	b, err := Parse(strings.NewReader(fixture13))
	if err != nil {
		t.Fatal(err)
	}

	for i := range a.Observations {
		sa, err := a.Observations[i].StreamID()
		if err != nil {
			t.Fatal(err)
		}

		sb, err := b.Observations[i].StreamID()
		if err != nil {
			t.Fatal(err)
		}

		if sa != sb {
			t.Fatalf("re-parse changed stream id: %s vs %s", sa, sb)
		}
	}
}

const fixtureAIDest = `"John Doe","+17287289311","9100","public","2026-10-09 21:30:00","2026-10-09 21:30:02","2026-10-09 21:30:20","18","16","NORMAL_CLEARING","ddddeeee-ffff-4aaa-9aaa-000000000009","b0a9-3333",""`

const fixtureHumanDest = `"John Doe","+17287289311","17287289311","public","2026-10-09 21:31:00","2026-10-09 21:31:02","2026-10-09 21:31:20","18","16","NORMAL_CLEARING","ddddeeee-ffff-4aaa-9aaa-000000000010","b0a9-4444",""`

func TestParseAIBadgeByDestination(t *testing.T) {
	t.Parallel()

	defaults, err := Parse(strings.NewReader(fixtureAIDest))
	if err != nil {
		t.Fatal(err)
	}

	if len(defaults.Observations) != 1 || !defaults.Observations[0].IsAI {
		t.Fatalf("destination 9100 should badge AI by default: %+v", defaults.Observations)
	}

	human, err := Parse(strings.NewReader(fixtureHumanDest))
	if err != nil {
		t.Fatal(err)
	}

	if len(human.Observations) != 1 || human.Observations[0].IsAI {
		t.Fatalf("non-AI destination must not badge: %+v", human.Observations)
	}

	custom, err := ParseWithOptions(strings.NewReader(fixtureHumanDest), ParseOptions{
		AIDestinations: ParseAIDestinations("17287289311, +1999"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(custom.Observations) != 1 || !custom.Observations[0].IsAI {
		t.Fatalf("custom -ai-dest should badge the listed destination: %+v", custom.Observations)
	}
}

func TestParseExtractsAppendedMOS(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture13MOS))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 1 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	if mos := res.Observations[0].MOS; mos != 4.4 {
		t.Fatalf("MOS = %v, want 4.4", mos)
	}
}

func TestParseNonMOSExtraColumnReadsAsUnknown(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture13MOSGarbage))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 1 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	if mos := res.Observations[0].MOS; mos != 0 {
		t.Fatalf("a non-MOS extra column must read as unknown (0), got %v", mos)
	}
}

// fixture18MOS is the fallback template with mod_cdr_csv's extra
// rtp_audio_in_mos column appended (the semicolon quirk preserved).
const fixture18MOS = `"","+14084757593","2000","default","Lars","SOFTA","SOFTB","bridge","park","2026-10-09 21:00:00","2026-10-09 21:00:01","2026-10-09 21:00:10","10","9","NORMAL_CLEARING","2","cccc4444-5555-4ccc-8ccc-000000000003","","3.9";` + "\n"

func TestParseFallbackTemplateWithAppendedMOS(t *testing.T) {
	res, err := Parse(strings.NewReader(fixture18MOS))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 1 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	obs := res.Observations[0]
	if obs.NativeID != "cccc4444-5555-4ccc-8ccc-000000000003" {
		t.Fatalf("native id wrong: %+v", obs)
	}

	if mos := obs.MOS; mos != 3.9 {
		t.Fatalf("18-column template must parse the appended MOS, got %v", mos)
	}
}

// fixtureInboundTransferredDID is the live 2026-10-23 shape: an external
// caller reaches the AI DID, the dialplan transfers the a-leg into "default",
// and the CDR's context column therefore says nothing about the orientation.
const fixtureInboundTransferredDID = `"","4084757593","9100","default","2026-10-23 09:14:02","2026-10-23 09:14:03","2026-10-23 09:15:11","69","68","NORMAL_CLEARING","11112222-3333-4444-8555-666677778888","",""
`

func TestParseDirectionFromTransferredInboundDID(t *testing.T) {
	t.Parallel()

	res, err := Parse(strings.NewReader(fixtureInboundTransferredDID))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 1 {
		t.Fatalf("got %d observations, want 1", len(res.Observations))
	}

	obs := res.Observations[0]
	if obs.Direction != calls.DirectionInbound {
		t.Fatalf("external caller to an internal extension is inbound, got %q", obs.Direction)
	}

	if !obs.IsAI {
		t.Error("destination 9100 must still badge AI")
	}
}

// fixtureDeclaredDirection is the 13-column default template with the
// sibling's two extra columns: rtp_audio_in_mos and call_direction.
const fixtureDeclaredDirection = `"","4084757593","9100","default","2026-10-23 09:14:02","2026-10-23 09:14:03","2026-10-23 09:15:11","69","68","NORMAL_CLEARING","11112222-3333-4444-8555-666677778888","","","4.1","outbound"
` + `"","1000","+4930123456","default","2026-10-23 10:00:00","2026-10-23 10:00:02","2026-10-23 10:01:00","60","58","NORMAL_CLEARING","99992222-3333-4444-8555-666677778889","","","outbound","4.2"
`

func TestParseDeclaredDirectionColumnWins(t *testing.T) {
	t.Parallel()

	res, err := Parse(strings.NewReader(fixtureDeclaredDirection))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 2 || res.Skipped != 0 {
		t.Fatalf("got %d observations, %d skipped", len(res.Observations), res.Skipped)
	}

	// Row one looks inbound by the heuristic (external caller → internal
	// extension), but the dialplan declares outbound; the declaration wins.
	if got := res.Observations[0].Direction; got != calls.DirectionOutbound {
		t.Errorf("declared direction ignored: got %q", got)
	}

	if mos := res.Observations[0].MOS; mos != 4.1 {
		t.Errorf("MOS = %v, want 4.1 alongside a declared direction", mos)
	}

	// Row two puts the extras in the opposite order: both are still read by
	// value, so the parser does not depend on the template's column order.
	if got := res.Observations[1].Direction; got != calls.DirectionOutbound {
		t.Errorf("reordered extras: direction = %q, want outbound", got)
	}

	if mos := res.Observations[1].MOS; mos != 4.2 {
		t.Errorf("reordered extras: MOS = %v, want 4.2", mos)
	}
}

func TestParseRejectsOverlongExtraColumns(t *testing.T) {
	t.Parallel()

	line := `"","1000","+4930123456","default","2026-10-23 10:00:00","","","60","58","NORMAL_CLEARING","aaaabbbb-3333-4444-8555-666677778890","","","4.2","outbound","extra"`

	res, err := Parse(strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 0 || res.Skipped != 1 {
		t.Fatalf("a row wider than the known extras is a mismatch: %d observations, %d skipped",
			len(res.Observations), res.Skipped)
	}
}

func TestParseSkipsUnparseableStartStamp(t *testing.T) {
	t.Parallel()

	line := `"","1000","+4930123456","default","not-a-stamp","","","60","58","NORMAL_CLEARING","aaaabbbb-3333-4444-8555-666677778890","","","4.2","outbound"`

	res, err := Parse(strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Observations) != 0 || res.Skipped != 1 {
		t.Fatalf("a malformed start_stamp must skip the row, not crash: %d observations, %d skipped",
			len(res.Observations), res.Skipped)
	}
}

func TestParseDirectionHeuristicEdges(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		context  string
		caller   string
		dest     string
		expected calls.Direction
	}{
		{"public context stays inbound", "public", "4084757593", "9100", calls.DirectionInbound},
		{"internal outbound call", "default", "1000", "+4930123456", calls.DirectionOutbound},
		{"internal to internal stays outbound", "default", "1000", "2000", calls.DirectionOutbound},
		{"external to external stays outbound", "default", "+14084757593", "+4930123456", calls.DirectionOutbound},
		{"empty partners stay outbound", "default", "", "", calls.DirectionOutbound},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			byColumn := map[string]string{
				"context":            testCase.context,
				"caller_id_number":   testCase.caller,
				"destination_number": testCase.dest,
			}

			if got := direction(byColumn, ""); got != testCase.expected {
				t.Errorf("direction = %q, want %q", got, testCase.expected)
			}
		})
	}
}
