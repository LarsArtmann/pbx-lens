package ingest_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/ingest/agent"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/webhook"
	"pgregory.net/rapid"
)

// TestParsersSurviveArbitraryInput is the fuzz-seed property (plan P19.3):
// ARBITRARY bytes through any parser must never panic, never error (parsers
// skip rows they cannot use — one bad row never fails the whole index), and
// every emitted observation must be recordable, i.e. pass the domain's own
// Validate. A parser that emits an observation NewRecordCommand would reject
// is a contract break this property catches at the parse boundary, before
// any dispatcher sees it.
func TestParsersSurviveArbitraryInput(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		input := rapid.SliceOf(rapid.Byte()).Draw(rt, "input")

		cdrResult, err := cdr.Parse(bytes.NewReader(input))
		if err != nil {
			rt.Fatalf("cdr.Parse errored on arbitrary input: %v", err)
		}

		if cdrResult.Skipped < 0 {
			rt.Fatalf("cdr negative skip count: %d", cdrResult.Skipped)
		}

		for i, obs := range cdrResult.Observations {
			if err := obs.Validate(); err != nil {
				rt.Fatalf("cdr emitted an unrecordable observation at %d: %v", i, err)
			}
		}

		webhookResult, err := webhook.Parse(bytes.NewReader(input))
		if err != nil {
			rt.Fatalf("webhook.Parse errored on arbitrary input: %v", err)
		}

		if webhookResult.Skipped < 0 {
			rt.Fatalf("webhook negative skip count: %d", webhookResult.Skipped)
		}

		for i, obs := range webhookResult.Observations {
			if err := obs.Validate(); err != nil {
				rt.Fatalf("webhook emitted an unrecordable observation at %d: %v", i, err)
			}
		}

		if _, err := agent.Parse(bytes.NewReader(input), "property-fallback"); err != nil {
			rt.Fatalf("agent.Parse errored on arbitrary input: %v", err)
		}
	})
}
