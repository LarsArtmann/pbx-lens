package cli

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
)

// benchCDR is the fixture row: a realistic 13-column FreeSWITCH row (the
// live shape including the MOS column); the native id is substituted per
// row so every dispatch lands a fresh stream.
const benchCDR = `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","conv-bench-%d","b0a9-2222","","4.4"` + "\n"

func benchObservations(b *testing.B, n int) []calls.Observation {
	b.Helper()

	var buf strings.Builder

	for i := range n {
		buf.WriteString(rowFor(i))
	}

	res, err := cdr.Parse(strings.NewReader(buf.String()))
	if err != nil {
		b.Fatal(err)
	}

	if len(res.Observations) != n {
		b.Fatalf("parsed %d observations, want %d", len(res.Observations), n)
	}

	return res.Observations
}

func rowFor(i int) string {
	return strings.ReplaceAll(benchCDR, "%d", strconv.Itoa(i))
}

// BenchmarkIngestCDRParse isolates the parse half: raw CDR text to
// observations, no engine involved. ns/row is the ingest floor.
func BenchmarkIngestCDRParse(b *testing.B) {
	text := rowFor(1)

	for range b.N {
		if _, err := cdr.Parse(strings.NewReader(text)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIngestCDRDispatch measures the full ingest hop: dispatch one
// fresh-stream observation through the memory engine and fold the
// projection — the poller's per-new-row ceiling (parse excluded; see
// BenchmarkIngestCDRParse).
func BenchmarkIngestCDRDispatch(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sys, err := app.New(ctx, "")
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = sys.Close() })

	proto := benchObservations(b, 1)[0]

	b.ResetTimer()

	for i := range b.N {
		obs := proto
		obs.NativeID = calls.NativeID("conv-bench-" + strconv.Itoa(i))

		cmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			b.Fatal(err)
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			b.Fatal(err)
		}
	}
}
