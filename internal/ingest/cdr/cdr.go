package cdr

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest"
)

// fieldTemplates returns the two Master.csv column layouts FreeSWITCH
// writes (13- and 18-column). They are values derived from the dialplan
// template, not mutable state, so they are built per Parse call.
func fieldTemplates() ([]string, []string) {
	thirteen := []string{
		"caller_id_name",
		"caller_id_number",
		"destination_number",
		"context",
		"start_stamp",
		"answer_stamp",
		"end_stamp",
		"duration",
		"billsec",
		"hangup_cause",
		"uuid",
		"bleg_uuid",
		"accountcode",
	}

	eighteen := []string{
		"accountcode",
		"caller_id_number",
		"destination_number",
		"context",
		"caller_id",
		"channel_name",
		"bridge_channel",
		"last_app",
		"last_arg",
		"start_stamp",
		"answer_stamp",
		"end_stamp",
		"duration",
		"billsec",
		"hangup_cause",
		"amaflags",
		"uuid",
		"userfield",
	}

	return thirteen, eighteen
}

const (
	stampLayout = "2006-01-02 15:04:05"
)

// DefaultAIDestination is the deployment's agent extension (nix-international-
// telephony options.nix: agent.extension default "9100"); CDR rows whose
// destination_number hits an AI destination badge IsAI.
const DefaultAIDestination = "9100"

// ParseOptions carries the dialplan knowledge the CDR text cannot express.
type ParseOptions struct {
	// AIDestinations badges a call AI when destination_number matches
	// (comma-free values); nil means {DefaultAIDestination} only.
	AIDestinations map[string]bool
}

func (o ParseOptions) isAI(destination string) bool {
	if len(o.AIDestinations) == 0 {
		return destination == DefaultAIDestination
	}

	return o.AIDestinations[destination]
}

// ParseAIDestinations turns a comma-separated -ai-dest flag value into the
// lookup set, trimming whitespace and dropping empties.
func ParseAIDestinations(flag string) map[string]bool {
	out := map[string]bool{}

	for part := range strings.SplitSeq(flag, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out[trimmed] = true
		}
	}

	return out
}

func Parse(r io.Reader) (ingest.Result, error) {
	return ParseWithOptions(r, ParseOptions{})
}

func ParseWithOptions(r io.Reader, opts ParseOptions) (ingest.Result, error) {
	res := ingest.Result{}
	fields13, fields18 := fieldTemplates()
	sc := ingest.NewScanner(r)

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r\n")
		if line == "" {
			continue
		}

		line = strings.TrimSuffix(line, ";")

		cr := csv.NewReader(strings.NewReader(line))
		cr.FieldsPerRecord = -1
		// Vanilla mod_cdr_csv writes a bare quote in the compiled-in
		// default template's tail (", \"\" — a comma-space before the
		// final empty fields) on every call; Go's strict CSV rejects it
		// ("bare \" in non-quoted-field", proven live by the
		// nix-international-telephony telephony-lens VM suite
		// 2026-10-12). LazyQuotes accepts the stray quote; normalize
		// below already strips the residue from the field values.
		cr.LazyQuotes = true

		row, err := cr.Read()
		if err != nil {
			return ingest.Result{}, errorfamily.NewCorruption(
				"cdr.parse.read",
				fmt.Sprintf("reading Master.csv: %v", err),
			)
		}

		if len(row) == 0 {
			continue
		}

		fields := normalize(row)

		var byColumn map[string]string

		var mosRaw string

		switch len(fields) {
		case len(fields13):
			byColumn = zip(fields13, fields)
		case len(fields18):
			byColumn = zip(fields18, fields)
		case len(fields13) + 1:
			// mod_cdr_csv appends extra configured fields AFTER the template;
			// this deployment's only extra field is rtp_audio_in_mos. The
			// parse below rejects anything outside the MOS scale, so a
			// different extra field degrades to "no MOS" rather than a bad
			// value.
			byColumn = zip(fields13, fields[:len(fields13)])
			mosRaw = fields[len(fields13)]
		case len(fields18) + 1:
			byColumn = zip(fields18, fields[:len(fields18)])
			mosRaw = fields[len(fields18)]
		default:
			res.Skipped++

			continue
		}

		obs, err := observation(byColumn, opts, mosRaw)
		if err != nil {
			res.Skipped++

			continue
		}

		res.Observations = append(res.Observations, obs)
	}

	if err := sc.Err(); err != nil {
		return ingest.Result{}, errorfamily.NewCorruption("cdr.parse.scan", fmt.Sprintf("scanning Master.csv: %v", err))
	}

	return res, nil
}

func normalize(row []string) []string {
	out := make([]string, 0, len(row))
	for _, f := range row {
		out = append(out, strings.TrimSpace(strings.Trim(strings.TrimSuffix(strings.TrimSpace(f), ";"), `"`)))
	}

	return out
}

func zip(names, values []string) map[string]string {
	m := make(map[string]string, len(names))
	for i, name := range names {
		m[name] = values[i]
	}

	return m
}

// cdrMOSMax is FreeSWITCH's Mean Opinion Score ceiling (the 1..5 scale).
const cdrMOSMax = 5.0

// parseMOS accepts a reported rtp_audio_in_mos value; anything empty,
// unparseable, or outside the MOS scale reads as "not reported" (0).
func parseMOS(raw string) float64 {
	if raw == "" {
		return 0
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || v > cdrMOSMax {
		return 0
	}

	return v
}

func observation(byColumn map[string]string, opts ParseOptions, mosRaw string) (calls.Observation, error) {
	started, err := time.Parse(stampLayout, byColumn["start_stamp"])
	if err != nil {
		return calls.Observation{}, fmt.Errorf("start_stamp %q: %w", byColumn["start_stamp"], err)
	}

	billsec := max(atoi(byColumn["billsec"]), 0)

	obs := calls.Observation{
		NativeID:     calls.NativeID(byColumn["uuid"]),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    direction(byColumn["context"]),
		StartedAt:    started.UTC(),
		DurationSecs: billsec,
		FromNumber:   byColumn["caller_id_number"],
		ToNumber:     byColumn["destination_number"],
		HangupCause:  byColumn["hangup_cause"],
		IsAI:         opts.isAI(byColumn["destination_number"]),
		MOS:          parseMOS(mosRaw),
	}

	return obs, obs.Validate()
}

// direction derives the call leg orientation from the FreeSWITCH dialplan:
// calls that ended in the "public" context came in from outside.
func direction(dialplanContext string) calls.Direction {
	if dialplanContext == "public" {
		return calls.DirectionInbound
	}

	return calls.DirectionOutbound
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}

	return n
}
