package cdr

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
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

	scannerInitialBuf = 64 * 1024
	scannerMaxLine    = 1024 * 1024
)

type Result struct {
	Observations []calls.Observation
	Skipped      int
}

func Parse(r io.Reader) (Result, error) {
	res := Result{}
	fields13, fields18 := fieldTemplates()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxLine)

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r\n")
		if line == "" {
			continue
		}

		line = strings.TrimSuffix(line, ";")

		cr := csv.NewReader(strings.NewReader(line))
		cr.FieldsPerRecord = -1

		row, err := cr.Read()
		if err != nil {
			return Result{}, errorfamily.NewCorruption("cdr.parse.read", fmt.Sprintf("reading Master.csv: %v", err))
		}

		if len(row) == 0 {
			continue
		}

		fields := normalize(row)

		var shape map[string]string

		switch len(fields) {
		case len(fields13):
			shape = zip(fields13, fields)
		case len(fields18):
			shape = zip(fields18, fields)
		default:
			res.Skipped++

			continue
		}

		obs, err := observation(shape)
		if err != nil {
			res.Skipped++

			continue
		}

		res.Observations = append(res.Observations, obs)
	}

	if err := sc.Err(); err != nil {
		return Result{}, errorfamily.NewCorruption("cdr.parse.scan", fmt.Sprintf("scanning Master.csv: %v", err))
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

func observation(shape map[string]string) (calls.Observation, error) {
	started, err := time.Parse(stampLayout, shape["start_stamp"])
	if err != nil {
		return calls.Observation{}, fmt.Errorf("start_stamp %q: %w", shape["start_stamp"], err)
	}

	billsec := max(atoi(shape["billsec"]), 0)

	obs := calls.Observation{
		NativeID:     calls.NativeID(shape["uuid"]),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    direction(shape["context"]),
		StartedAt:    started.UTC(),
		DurationSecs: billsec,
		FromNumber:   shape["caller_id_number"],
		ToNumber:     shape["destination_number"],
		HangupCause:  shape["hangup_cause"],
	}

	return obs, obs.Validate()
}

func direction(context string) calls.Direction {
	if context == "public" {
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
