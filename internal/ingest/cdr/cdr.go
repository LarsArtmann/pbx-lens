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
		line = strings.TrimSuffix(line, ";")

		if line == "" {
			continue
		}

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

		byColumn, extras, ok := mapRow(fields, fields13, fields18)
		if !ok {
			res.Skipped++

			continue
		}

		mos, declaredDirection := classifyExtras(extras)

		obs, err := observation(byColumn, opts, mos, declaredDirection)
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

// mapRow classifies one normalized CSV row against the two built-in
// templates and returns its column map plus any extra fields, reporting
// false for a row that is a parse mismatch rather than a guess (the caller
// counts it as skipped).
//
// mod_cdr_csv appends the template's extra configured fields AFTER the
// built-in ones. The content-typed classifier (classifyExtras) reads each
// extra column by its VALUE (a MOS score, a direction word, or neither), so
// the deployment can add rtp_audio_in_mos and call_direction in any order
// without the parser hard-coding a position. A row longer than the built-in
// template plus maxExtraColumns stays a skipped row rather than a guessed
// parse (the same guard the exact-length cases had).
func mapRow(fields, fields13, fields18 []string) (map[string]string, []string, bool) {
	switch len(fields) {
	case len(fields13):
		return zip(fields13, fields), nil, true
	case len(fields18):
		return zip(fields18, fields), nil, true
	}

	switch {
	case len(fields) > len(fields13) && len(fields) <= len(fields13)+maxExtraColumns:
		return zip(fields13, fields[:len(fields13)]), fields[len(fields13):], true
	case len(fields) > len(fields18) && len(fields) <= len(fields18)+maxExtraColumns:
		return zip(fields18, fields[:len(fields18)]), fields[len(fields18):], true
	default:
		return nil, nil, false
	}
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

// maxExtraColumns bounds the trailing extra columns this parser accepts
// (rtp_audio_in_mos, call_direction). A longer row is a parse mismatch, not
// a wider deployment.
const maxExtraColumns = 2

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

// classifyExtras reads the template's trailing extra columns by VALUE: a
// score in the MOS scale is the quality report, an "inbound"/"outbound" word
// is the dialplan's own direction verdict, and anything else is ignored (the
// historical "garbage extra column degrades to no data" behaviour). Reading
// by value instead of by position lets the sibling template add
// call_direction without re-ordering the MOS column or breaking old exports.
func classifyExtras(extras []string) (float64, calls.Direction) {
	mos := 0.0
	direction := calls.Direction("")

	for _, raw := range extras {
		if parsed, ok := parseDeclaredDirection(raw); ok {
			direction = parsed

			continue
		}

		if score := parseMOS(raw); score > 0 {
			mos = score
		}
	}

	return mos, direction
}

// parseDeclaredDirection accepts the sibling template's call_direction
// column, and nothing else: an unknown word must never be read as a verdict.
func parseDeclaredDirection(raw string) (calls.Direction, bool) {
	declared := calls.Direction(strings.ToLower(strings.TrimSpace(raw)))
	if !declared.Valid() {
		return "", false
	}

	return declared, true
}

// parseStartStamp parses the row's start_stamp column against the CDR stamp
// layout; the rejection names the offending value so a malformed row is
// diagnosable from the skip count alone.
func parseStartStamp(raw string) (time.Time, error) {
	started, err := time.Parse(stampLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("start_stamp %q: %w", raw, err)
	}

	return started, nil
}

func observation(
	byColumn map[string]string,
	opts ParseOptions,
	mos float64,
	declared calls.Direction,
) (calls.Observation, error) {
	started, err := parseStartStamp(byColumn["start_stamp"])
	if err != nil {
		return calls.Observation{}, err
	}

	billsec := max(atoi(byColumn["billsec"]), 0)

	obs := calls.Observation{
		NativeID:     calls.NativeID(byColumn["uuid"]),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    direction(byColumn, declared),
		StartedAt:    started.UTC(),
		DurationSecs: billsec,
		FromNumber:   byColumn["caller_id_number"],
		ToNumber:     byColumn["destination_number"],
		HangupCause:  byColumn["hangup_cause"],
		IsAI:         opts.isAI(byColumn["destination_number"]),
		MOS:          mos,
	}

	return obs, obs.Validate()
}

// Direction derivation, in precedence order:
//
//  1. A template-declared call_direction column — the dialplan's own verdict,
//     authoritative when present.
//  2. The "public" context — an a-leg that never left the inbound dialplan.
//  3. The transfer heuristic: an external caller ringing an internal
//     extension arrived from outside. This is what the context column cannot
//     say: the sibling dialplan transfers every inbound DID into "default",
//     so an inbound AI call's a-leg ends in the default context and used to
//     index as OUTBOUND (live evidence 2026-10-23: 4084757593 → 9100).
//  4. Otherwise outbound.
func direction(byColumn map[string]string, declared calls.Direction) calls.Direction {
	if declared.Valid() {
		return declared
	}

	if byColumn["context"] == "public" {
		return calls.DirectionInbound
	}

	if isInternalExtension(byColumn["destination_number"]) && isExternalNumber(byColumn["caller_id_number"]) {
		return calls.DirectionInbound
	}

	return calls.DirectionOutbound
}

// internalExtensionMaxDigits is the longest number an extension plan uses
// (the deployment's agent extension is 9100; four-digit plans are the norm).
const internalExtensionMaxDigits = 6

// externalNumberMinDigits is the shortest PSTN number: a national subscriber
// number is never shorter, and internal extensions never reach it.
const externalNumberMinDigits = 7

// isInternalExtension reports a bare dialplan extension: digits only, no
// international prefix, no more digits than an extension plan allocates.
func isInternalExtension(number string) bool {
	trimmed := strings.TrimSpace(number)
	if trimmed == "" || strings.HasPrefix(trimmed, "+") || len(trimmed) > internalExtensionMaxDigits {
		return false
	}

	return isAllDigits(trimmed)
}

// isExternalNumber reports a PSTN number, bare or E.164.
func isExternalNumber(number string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSpace(number), "+")

	return len(trimmed) >= externalNumberMinDigits && isAllDigits(trimmed)
}

func isAllDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return value != ""
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}

	return n
}
