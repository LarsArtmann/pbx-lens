package cdr

import (
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

const fixture13 = `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","f0e1d2c3-1111-4aaa-9aaa-000000000001","b0a9-2222",""
"","1000","+17287289311","default","2026-10-09 20:20:00","2026-10-09 20:20:02","2026-10-09 20:20:28","28","26","NORMAL_CLEARING","aaaa2222-3333-4bbb-8bbb-000000000002","","artmann"
`

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
	if obs.DurationSecs != 9 || obs.Direction != calls.DirectionOutbound {
		t.Fatalf("mapped fields wrong: %+v", obs)
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
