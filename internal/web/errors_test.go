package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errIndexBoom = errors.New("secrets: disk 3 hums at midnight")

// TestWriteIndexErrorFamilyContract pins the classified-error-response
// contract: family-classified errors take the family's HTTP status and a
// JSON body naming family/code, unclassified errors keep the fallback
// status — and in BOTH shapes the raw error text never reaches the body
// (it can embed storage internals); only the caller's crafted line ships.
// Classified errors additionally log through the family emitter (structured
// family/code fields, WARN for retryable); unclassified keep the plain log.
func TestWriteIndexErrorFamilyContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var logged [][]any

	s := &server{log: func(msg string, args ...any) { logged = append(logged, append([]any{msg}, args...)) }}

	rec := func(err error, jsonBody bool) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.writeIndexError(w, req, err, http.StatusInternalServerError, "reading the call index failed", jsonBody)

		return w
	}

	text := rec(errIndexBoom, false)
	if text.Code != http.StatusInternalServerError {
		t.Errorf("unclassified text status = %d, want fallback 500", text.Code)
	}

	if body := text.Body.String(); strings.Contains(body, "secrets") || strings.Contains(body, "midnight") {
		t.Errorf("unclassified text body leaks the raw error: %q", body)
	}

	if !strings.Contains(text.Body.String(), "reading the call index failed") {
		t.Errorf("unclassified text body lost the crafted message: %q", text.Body.String())
	}

	js := rec(errIndexBoom, true)
	if js.Code != http.StatusInternalServerError {
		t.Errorf("unclassified JSON status = %d, want fallback 500", js.Code)
	}

	if body := js.Body.String(); strings.Contains(body, "secrets") || strings.Contains(body, "midnight") {
		t.Errorf("unclassified JSON body leaks the raw error: %q", body)
	}

	if !strings.Contains(js.Header().Get("Content-Type"), "application/json") {
		t.Errorf("JSON body content type = %q", js.Header().Get("Content-Type"))
	}

	if len(logged) != 2 {
		t.Errorf("unclassified errors logged %d records, want 2 (one per response)", len(logged))
	}

	logged = nil

	fam := errorfamily.NewTransient("lens.test.scan", "internal detail").WithContext("call_id", "conv-x")
	famRec := rec(fam, true)

	if famRec.Code != http.StatusServiceUnavailable {
		t.Errorf("transient JSON status = %d, want family 503", famRec.Code)
	}

	body := famRec.Body.String()
	for _, want := range []string{`"family":"transient"`, `"code":"lens.test.scan"`} {
		if !strings.Contains(body, want) {
			t.Errorf("transient JSON body missing %s: %s", want, body)
		}
	}

	if strings.Contains(body, "internal detail") {
		t.Errorf("transient JSON body ships the raw message: %s", body)
	}

	if len(logged) != 1 {
		t.Fatalf("classified error logged %d records, want 1 (the family emitter)", len(logged))
	}

	// The family emitter's record: message = err text, args carry
	// family/code/retryable/exit_code/context.*.
	fields := fmt.Sprint(logged[0]...)

	for _, want := range []string{"lens.test.scan", "transient", "call_id", "conv-x"} {
		if !strings.Contains(fields, want) {
			t.Errorf("classified log record missing %s: %v", want, logged[0])
		}
	}

	infra := rec(errorfamily.NewInfrastructure("lens.test.unwired", "x"), false)

	if infra.Code != http.StatusServiceUnavailable {
		t.Errorf("infrastructure text status = %d, want family 503", infra.Code)
	}
}

// TestExportErrorMarkerCarriesContext pins the marker shape (plan T08):
// the terminal JSONL object carries the family code AND any WithContext
// keys (call_id …), so a truncated download says where it stopped.
func TestExportErrorMarkerCarriesContext(t *testing.T) {
	err := errorfamily.NewTransient("lens.test.stream", "read failed").WithContext("call_id", "conv-42")

	marker := exportErrorMarker(err)

	if marker.ExportIncomplete != exportIncompleteCode {
		t.Errorf("marker code = %q, want %q", marker.ExportIncomplete, exportIncompleteCode)
	}

	if marker.Code != "lens.test.stream" {
		t.Errorf("marker Code = %q, want the family code", marker.Code)
	}

	if marker.Context["call_id"] != "conv-42" {
		t.Errorf("marker Context = %v, want call_id conv-42", marker.Context)
	}

	plain := exportErrorMarker(errors.New("plain"))

	if plain.Context != nil {
		t.Errorf("unclassified marker Context = %v, want nil", plain.Context)
	}
}
