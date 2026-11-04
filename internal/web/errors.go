package web

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// writeIndexError maps one handler error onto the response through the
// error family. Family-classified errors set the family's HTTP status (a
// Transient scan failure surfaces as 503 — the service genuinely cannot
// answer — not a lying 500); unclassified errors keep fallbackStatus.
// Either way the BODY carries only the caller's crafted safe text:
// err.Error() never reaches a client (it can embed storage internals),
// the raw error goes to the server log instead. jsonBody selects the
// /api/* shape ({"error", "family", "code"}) over the http.Error text
// shape; family/code are empty strings for unclassified errors.
func (s *server) writeIndexError(w http.ResponseWriter, r *http.Request, err error, fallbackStatus int, safeMsg string, jsonBody bool) {
	status := fallbackStatus
	family := ""
	code := errorfamily.Code(err)

	if classified, ok := errors.AsType[errorfamily.Classified](err); ok {
		status = classified.ErrorFamily().HTTPStatus()
		family = classified.ErrorFamily().String()

		// Classified errors log through the family emitter: family, code,
		// retryable, exit_code, and context.* ride the record, retryables
		// land at WARN, the rest at ERROR.
		errorfamily.LogErrorContext(r.Context(), err, s.slogger())
	} else {
		s.logf("%s failed: %v (status %d)", safeMsg, err, status)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if jsonBody {
		w.WriteHeader(status)

		body := struct {
			Error  string `json:"error"`
			Family string `json:"family,omitempty"`
			Code   string `json:"code,omitempty"`
		}{Error: safeMsg, Family: family, Code: code}

		if marshalErr := json.MarshalWrite(w, body); marshalErr != nil {
			s.logf("writing %s error body: %v", safeMsg, marshalErr)
		}

		return
	}

	http.Error(w, safeMsg, status)
}
