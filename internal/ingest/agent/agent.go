// Package agent turns the voice agent's per-call transcript JSONL files
// (`/var/lib/telephony/recordings/transcripts/{call-uuid}.jsonl`, written by
// nix-international-telephony's voice-agent) into Observations.
//
// Line shapes (verified against the writer source, modules/telephony/voice-agent.py):
//
//	{"type":"start","ts":<unix>,"uuid":"...","models":[...]}
//	{"type":"turn","role":"caller"|"agent","text":"..."}   (no per-turn ts)
//	{"type":"end","reason":"..."}
//
// The filename IS the FreeSWITCH call uuid — the join key back to the
// freeswitch-cdr Observation. Roles normalize to the domain's user/assistant
// vocabulary. Per-turn At stays zero (the writer stamps no per-turn time);
// the whole-second doctrine is preserved because zero is already exact.
package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest"
)

// DefaultDirection is used when no CDR view joins: this deployment's agent
// places outbound calls (same caller-declared doctrine as Telnyx
// conversations; revisit when inbound AI interception lands).
const DefaultDirection = calls.DirectionOutbound

// line is one JSONL record of a transcript file; the variants share the
// discriminator field.
type line struct {
	Type   string  `json:"type"`
	TS     float64 `json:"ts"`
	UUID   string  `json:"uuid"`
	Models []any   `json:"models"`
	Role   string  `json:"role"`
	Text   string  `json:"text"`
	Reason string  `json:"reason"`
}

// Transcript is one parsed {uuid}.jsonl file.
type Transcript struct {
	// UUID is the FreeSWITCH call uuid (join key to freeswitch-cdr).
	UUID string
	// StartedAt is the start line's unix ts; zero when the file has none.
	StartedAt time.Time
	// Turns are the normalized user/assistant utterances in file order.
	Turns []calls.TranscriptTurn
	// EndReason is the end line's reason, when present.
	EndReason string
}

// Result is one pass over a transcripts directory.
type Result struct {
	Transcripts []Transcript
	// Skipped counts files that produced no usable transcript.
	Skipped int
}

// Parse reads one transcript file.
func Parse(r io.Reader, fallbackUUID string) (Transcript, error) {
	out := Transcript{UUID: fallbackUUID}

	err := ingest.EachLine(r, func(raw string) {
		var ln line
		if err := json.Unmarshal([]byte(raw), &ln); err != nil {
			return
		}

		switch ln.Type {
		case "start":
			if ln.UUID != "" {
				out.UUID = ln.UUID
			}

			if ln.TS > 0 {
				out.StartedAt = time.Unix(int64(ln.TS), 0).UTC()
			}
		case "turn":
			if turn, ok := normalizeTurn(ln.Role, ln.Text); ok {
				out.Turns = append(out.Turns, turn)
			}
		case "end":
			out.EndReason = ln.Reason
		}
	})
	if err != nil {
		return Transcript{}, err
	}

	return out, nil
}

func normalizeTurn(role, text string) (calls.TranscriptTurn, bool) {
	mapped, ok := map[string]string{"caller": "user", "agent": "assistant"}[role]
	if !ok {
		return calls.TranscriptTurn{}, false
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return calls.TranscriptTurn{}, false
	}

	return calls.TranscriptTurn{Role: mapped, Text: trimmed}, true
}

// ParseDir reads every *.jsonl transcript file in dir (non-recursive).
func ParseDir(dir string) (Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, err
	}

	res := Result{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		stem := strings.TrimSuffix(entry.Name(), ".jsonl")

		file, err := os.Open(path)
		if err != nil {
			return Result{}, fmt.Errorf("open transcript %s: %w", path, err)
		}

		transcript, err := Parse(file, stem)
		closeErr := file.Close()

		if err != nil {
			return Result{}, fmt.Errorf("read transcript %s: %w", path, err)
		}

		if closeErr != nil {
			return Result{}, fmt.Errorf("close transcript %s: %w", path, closeErr)
		}

		if transcript.UUID == "" || len(transcript.Turns) == 0 {
			res.Skipped++

			continue
		}

		res.Transcripts = append(res.Transcripts, transcript)
	}

	return res, nil
}

// MapTranscript maps a transcript to its agent-transcript Observation. A
// joined freeswitch-cdr CallView fills the parties, direction, duration, and
// the started-at; without a join the transcript stands alone (numbers may
// stay empty — validation relaxes for agent transcripts) and the start
// line's ts anchors the call in time.
func MapTranscript(transcript Transcript, cdrView *calls.CallView) (calls.Observation, error) {
	obs := calls.Observation{
		NativeID:          calls.NativeID(transcript.UUID),
		Kind:              calls.SourceAgentTranscript,
		Direction:         DefaultDirection,
		IsAI:              true,
		Transcript:        transcript.Turns,
		TranscriptExcerpt: excerpt(transcript.Turns),
	}

	if cdrView != nil {
		obs.Direction = cdrView.Direction
		obs.StartedAt = cdrView.StartedAt
		obs.DurationSecs = cdrView.DurationSecs
		obs.FromNumber = cdrView.FromNumber
		obs.ToNumber = cdrView.ToNumber

		if cdrView.HangupCause != "" {
			obs.HangupCause = cdrView.HangupCause
		}
	}

	if obs.StartedAt.IsZero() {
		obs.StartedAt = transcript.StartedAt
	}

	if obs.HangupCause == "" && transcript.EndReason != "" {
		obs.HangupCause = transcript.EndReason
	}

	return obs, obs.Validate()
}

// excerpt is the first user turn (the call opening), falling back to the
// first assistant turn (the greeting) — same semantics as the Telnyx mapper,
// including the domain's excerpt ceiling: an uncapped opening line would
// flood the list page's single table cell.
func excerpt(turns []calls.TranscriptTurn) string {
	opening := ""
	for _, turn := range turns {
		if turn.Role == "user" {
			opening = turn.Text

			break
		}
	}

	if opening == "" && len(turns) > 0 {
		opening = turns[0].Text
	}

	runes := []rune(opening)
	if len(runes) > calls.DefaultExcerptMax {
		return string(runes[:calls.DefaultExcerptMax])
	}

	return opening
}
