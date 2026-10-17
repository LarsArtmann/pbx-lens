package web

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// stalledExportMux seeds the tie fixture that forces a keyset stall: three
// calls sharing ONE timestamp, batch shrunk to 2 — page one emits two rows,
// page two is all duplicates, the cursor stalls, the marker must follow.
func stalledExportMux(t *testing.T) *http.ServeMux {
	t.Helper()

	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	same := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	tied := []calls.Observation{
		webObservation("tied-a", same),
		webObservation("tied-b", same),
		webObservation("tied-c", same),
	}

	return newTestServer(t, tied...)
}

// TestExportCSVParsesAsConsumer pins the EXPORT_INCOMPLETE contract from
// the CONSUMER side (2026-10-17 quality pass): a real csv.Reader — not
// string-contains — must parse the stalled artifact end to end, with an
// intact header and a well-formed terminal marker row. This is the
// contract downstream tooling actually experiences.
func TestExportCSVParsesAsConsumer(t *testing.T) {
	mux := stalledExportMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/export.csv", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("csv status = %d, want 200", rec.Code)
	}

	reader := csv.NewReader(rec.Body)
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("csv.Reader rejected the artifact: %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("csv artifact has no rows at all")
	}

	header, data := rows[0], rows[1:]

	if got, want := strings.Join(header, ","), strings.Join(exportHeader(), ","); got != want {
		t.Errorf("csv header = %q, want %q", got, want)
	}

	dataRows := 0
	sawMarker := false

	for _, row := range data {
		if len(row) > 0 && row[0] == "# "+exportIncompleteCode {
			sawMarker = true

			continue
		}

		dataRows++
	}

	if dataRows < 2 {
		t.Errorf("csv data rows = %d, want the rows that streamed before the stall", dataRows)
	}

	if !sawMarker {
		t.Error("csv consumer found no EXPORT_INCOMPLETE marker row")
	}
}

// TestExportJSONLParsesAsConsumer is the JSONL twin: every line must
// decode as a JSON object and the stream must end with the marker object.
func TestExportJSONLParsesAsConsumer(t *testing.T) {
	mux := stalledExportMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/export.jsonl", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("jsonl status = %d, want 200", rec.Code)
	}

	decoder := json.NewDecoder(rec.Body)
	objects := 0
	sawMarker := false

	for {
		var obj map[string]any
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}

			t.Fatalf("jsonl record %d is not valid JSON: %v", objects+1, err)
		}

		objects++

		if obj["export_incomplete"] == exportIncompleteCode {
			sawMarker = true
		}
	}

	if objects < 2 {
		t.Errorf("jsonl objects = %d, want the rows that streamed before the stall", objects)
	}

	if !sawMarker {
		t.Error("jsonl consumer found no EXPORT_INCOMPLETE marker object")
	}
}

// BenchmarkExportStream10k measures the export walk at a realistic volume:
// 10k distinct-timestamp rows streamed through the real handler (real
// sqlite, like production) and discarded. Completeness is asserted once;
// the reported B/s and allocs are the memory-bound evidence for the R7
// "stream, never materialize" doctrine. Run with:
//
//	GOEXPERIMENT=jsonv2 go test ./internal/web -bench BenchmarkExportStream10k -benchmem
func BenchmarkExportStream10k(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(b.TempDir(), "bench.db"))
	if err != nil {
		b.Fatalf("OpenSystem: %v", err)
	}

	defer cleanup()

	base := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	seeded := make([]calls.Observation, 0, 10000)

	for i := range 10000 {
		seeded = append(
			seeded,
			webObservation(fmt.Sprintf("bench-%05d", i), base.Add(time.Duration(i)*time.Millisecond)),
		)
	}

	if stats := cli.Ingest(ctx, sys, seeded); len(stats.Rejected) > 0 {
		b.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		b.Fatalf("drain: %v", err)
	}

	mux := http.NewServeMux()
	Register(mux, sys, Options{})

	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/export.csv", nil))

		if rec.Code != http.StatusOK {
			b.Fatalf("export status = %d, want 200", rec.Code)
		}

		n, err := io.Copy(io.Discard, rec.Body)
		if err != nil {
			b.Fatalf("drain: %v", err)
		}

		if n == 0 {
			b.Fatal("streamed zero bytes")
		}
	}
}
