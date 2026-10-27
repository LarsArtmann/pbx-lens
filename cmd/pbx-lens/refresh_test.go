package main

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// refreshStub serves the three endpoints one refresh pass touches. It is
// deliberately tiny: the verb's contract is the CLASSIFICATION (new vs stale
// vs unchanged), and one conversation with one user turn is enough to tell
// "wrote nothing" from "wrote the row".
func refreshStub(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/ai/conversations":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{{
				"id":              "conv-refresh-1",
				"created_at":      "2026-10-23T03:23:29Z",
				"last_message_at": "2026-10-23T03:24:04Z",
				"metadata": map[string]any{
					"assistant_id":           "assistant-1",
					"telnyx_agent_target":    "+17287289311",
					"telnyx_end_user_target": "+14084757593",
				},
			}}})
		case "/ai/conversations/conv-refresh-1/messages":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{
				{
					"role":       "user",
					"text":       "Ja, hallo?",
					"created_at": "2026-10-23T03:24:17Z",
					"sent_at":    "2026-10-23T03:23:31Z",
				},
				{
					"role":       "assistant",
					"text":       "Guten Tag, hier ist Mara.",
					"created_at": "2026-10-23T03:24:17Z",
					"sent_at":    "2026-10-23T03:23:30Z",
				},
				{
					"role":       "system",
					"text":       "[long silence]",
					"created_at": "2026-10-23T03:24:18Z",
					"sent_at":    "2026-10-23T03:24:03Z",
				},
			}})
		case "/recordings":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestRefreshClassifiesThenBackfills drives the verb end to end: the first
// pass must classify the unseen stream and write NOTHING, the -apply pass must
// record it, and the third pass must report the row as unchanged (which is
// also the proof the second pass actually landed).
func TestRefreshClassifiesThenBackfills(t *testing.T) {
	srv := refreshStub(t)
	defer srv.Close()

	dir := t.TempDir()
	db := filepath.Join(dir, "refresh.db")
	keyFile := filepath.Join(dir, "key")

	if err := os.WriteFile(keyFile, []byte("stub-key\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	baseArgs := []string{"-db", db, "-key-file", keyFile, "-api-base", srv.URL, "-ai-did", "+17287289311"}

	var classify bytes.Buffer

	if err := runRefresh(baseArgs, &classify); err != nil {
		t.Fatalf("runRefresh(classify): %T %v", err, err)
	}

	for _, want := range []string{"1 new", "would write 1", "classify-only"} {
		if !strings.Contains(classify.String(), want) {
			t.Errorf("classify output %q must contain %q", classify.String(), want)
		}
	}

	var apply bytes.Buffer

	if err := runRefresh(append(append([]string{}, baseArgs...), "-apply"), &apply); err != nil {
		t.Fatalf("runRefresh(apply): %v", err)
	}

	if !strings.Contains(apply.String(), "1 recorded") {
		t.Errorf("apply output %q must report the recorded stream", apply.String())
	}

	var replay bytes.Buffer

	if err := runRefresh(baseArgs, &replay); err != nil {
		t.Fatalf("runRefresh(replay): %v", err)
	}

	for _, want := range []string{"1 unchanged", "would write 0"} {
		if !strings.Contains(replay.String(), want) {
			t.Errorf("replay output %q must contain %q (stored row must now match the source)", replay.String(), want)
		}
	}
}

// TestRefreshRequiresDB pins the one flag the verb cannot default.
func TestRefreshRequiresDB(t *testing.T) {
	var out bytes.Buffer

	err := runRefresh(nil, &out)
	if err == nil || !strings.Contains(err.Error(), "-db is required") {
		t.Fatalf("runRefresh(no db) = %v, want the -db rejection", err)
	}
}
