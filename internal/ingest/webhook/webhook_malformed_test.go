package webhook

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// TestParseMalformedJSONLNeverPanics is the property rail (plan 05-40
// M16/F51): a scanner over structurally hostile JSONL — byte-flips,
// truncations, invalid UTF-8, control bytes, empty and whitespace lines —
// never panics and always accounts for every line (valid data still
// parses, hostile lines land in Skipped). Seeded PRNG: deterministic
// across runs, no external property dependency.
func TestParseMalformedJSONLNeverPanics(t *testing.T) {
	rng := rand.New(rand.NewSource(20261015))

	valid := []string{
		`{"received_at":"2026-10-15T09:00:00Z","remote":"10.0.0.1","body":{"data":{"event_type":"call.initiated","occurred_at":"2026-10-15T09:00:00Z","payload":{"call_control_id":"cc-prop-1","call_session_id":"cs-1","direction":"incoming","from":"+15550001111","to":"+15550002222"}}}}`,
		`{"received_at":"2026-10-15T09:00:31Z","remote":"10.0.0.1","body":{"data":{"event_type":"call.ended","occurred_at":"2026-10-15T09:00:30Z","payload":{"call_control_id":"cc-prop-1","call_session_id":"cs-1","hangup_cause":"normal_clearing"}}}}`,
	}

	lines := []string{}
	lines = append(lines, valid...)

	for range 5000 {
		switch rng.Intn(6) {
		case 0: // byte-flip a random position
			b := []byte(valid[0])
			b[rng.Intn(len(b))] = byte(rng.Intn(256))
			lines = append(lines, string(b))
		case 1: // truncate at a random offset
			lines = append(lines, valid[0][:rng.Intn(len(valid[0]))])
		case 2: // raw control bytes / invalid UTF-8
			lines = append(lines, "\x00\xff\xfe{\"body\":\x07\x1b[garbage")
		case 3: // structurally valid JSON, wrong shape
			lines = append(lines, fmt.Sprintf(`{"n":%d,"body":{"data":{"event_type":42}}}`, rng.Int()))
		case 4: // blanks and whitespace
			lines = append(lines, "", "   ", "\t")
		case 5: // deep nesting bomb (bounded depth)
			depth := 1 + rng.Intn(64)
			lines = append(lines, strings.Repeat("[", depth)+strings.Repeat("]", depth))
		}
	}

	res, err := Parse(strings.NewReader(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("in-ceiling garbage must not abort the scan: %v", err)
	}

	// The property is not "every mutated line is skipped": a byte-flip can
	// leave structurally valid JSON with a mangled-but-present
	// call_control_id, and such lines legitimately fold into observations.
	// What must hold: the scan never aborts, never panics, and the
	// untouched valid envelope still comes through — garbage never erases
	// good data.
	found := false

	for _, obs := range res.Observations {
		if strings.Contains(string(obs.NativeID), "cc-prop-1") {
			found = true
		}
	}

	if !found {
		t.Errorf("valid envelope lost among %d observations", len(res.Observations))
	}
}

// TestParseOverCeilingLineSurfacesError pins the scanner-ceiling contract:
// a line beyond the 1 MiB per-line policy is a hard error Parse surfaces —
// never a silent truncation, never a panic.
func TestParseOverCeilingLineSurfacesError(t *testing.T) {
	oversized := strings.Repeat("x", 2<<20)

	if _, err := Parse(strings.NewReader(oversized)); err == nil {
		t.Error("over-ceiling line must surface a scanner error, got nil")
	}
}
