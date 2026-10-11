// Package ingest carries the scaffolding shared by the per-source parsers
// (cdr, telnyx, webhook, agent): the bounded line scanner and the row-level
// parse result.
package ingest

import (
	"bufio"
	"io"
	"strings"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

const (
	scannerInitialBuf = 64 * 1024
	scannerMaxLine    = 1024 * 1024
)

// Result is one pass over a source: the Observations emitted plus Skipped,
// the count of rows consumed but not emitted (undecodable, malformed, or
// carrying no usable call). Parsers skip such rows instead of failing the
// whole index on one bad one.
type Result struct {
	Observations []calls.Observation
	Skipped      int
}

// NewScanner returns a line scanner for r with the ingest buffer policy:
// 64 KiB initial buffer growing to a 1 MiB per-line ceiling — no realistic
// CDR row, webhook envelope, or transcript line is refused.
func NewScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxLine)

	return sc
}

// EachLine invokes fn for every non-blank line of r, trimmed of surrounding
// whitespace.
func EachLine(r io.Reader, fn func(line string)) error {
	sc := NewScanner(r)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			fn(line)
		}
	}

	return sc.Err()
}
