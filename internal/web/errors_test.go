package web

import (
	"errors"
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
func TestWriteIndexErrorFamilyContract(t *testing.T) {
	s := &server{}
	rec := func(jsonBody bool) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.writeIndexError(w, errIndexBoom, http.StatusInternalServerError, "reading the call index failed", jsonBody)

		return w
	}

	text := rec(false)
	if text.Code != http.StatusInternalServerError {
		t.Errorf("unclassified text status = %d, want fallback 500", text.Code)
	}

	if body := text.Body.String(); strings.Contains(body, "secrets") || strings.Contains(body, "midnight") {
		t.Errorf("unclassified text body leaks the raw error: %q", body)
	}

	if !strings.Contains(text.Body.String(), "reading the call index failed") {
		t.Errorf("unclassified text body lost the crafted message: %q", text.Body.String())
	}

	js := rec(true)
	if js.Code != http.StatusInternalServerError {
		t.Errorf("unclassified JSON status = %d, want fallback 500", js.Code)
	}

	if body := js.Body.String(); strings.Contains(body, "secrets") || strings.Contains(body, "midnight") {
		t.Errorf("unclassified JSON body leaks the raw error: %q", body)
	}

	if !strings.Contains(js.Header().Get("Content-Type"), "application/json") {
		t.Errorf("JSON body content type = %q", js.Header().Get("Content-Type"))
	}

	fam := httptest.NewRecorder()
	s.writeIndexError(fam, errorfamily.NewTransient("lens.test.scan", "internal detail"), http.StatusInternalServerError, "reading the call index failed", true)

	if fam.Code != http.StatusServiceUnavailable {
		t.Errorf("transient JSON status = %d, want family 503", fam.Code)
	}

	body := fam.Body.String()
	for _, want := range []string{`"family":"transient"`, `"code":"lens.test.scan"`} {
		if !strings.Contains(body, want) {
			t.Errorf("transient JSON body missing %s: %s", want, body)
		}
	}

	if strings.Contains(body, "internal detail") {
		t.Errorf("transient JSON body ships the raw message: %s", body)
	}

	infra := httptest.NewRecorder()
	s.writeIndexError(infra, errorfamily.NewInfrastructure("lens.test.unwired", "x"), http.StatusInternalServerError, "unwired", false)

	if infra.Code != http.StatusServiceUnavailable {
		t.Errorf("infrastructure text status = %d, want family 503", infra.Code)
	}
}
