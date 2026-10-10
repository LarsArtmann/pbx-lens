package telnyx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := NewClient("test-key", srv.URL)
	c.PageSize = 2

	return c, srv
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

func TestListConversationsPaginatesAndFilters(t *testing.T) {
	var gotAuth, gotFilter bool
	var gotPages []string

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer header")
		}

		gotAuth = true
		gotFilter = gotFilter || strings.Contains(r.URL.RawQuery, "last_message_at.gte.")
		gotPages = append(gotPages, r.URL.Query().Get("page[number]"))

		page, err := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if err != nil {
			t.Fatalf("page[number]: %v", err)
		}

		data := []Conversation{}
		if page == 1 {
			data = []Conversation{
				{ID: "conv-1", CreatedAt: time.Unix(1000, 0).UTC(), Metadata: map[string]string{"assistant_id": "assistant-x"}},
				{ID: "conv-2", CreatedAt: time.Unix(2000, 0).UTC()},
			}
		} else if page == 2 {
			data = []Conversation{
				{ID: "conv-3", CreatedAt: time.Unix(3000, 0).UTC()},
			}
		}

		writeJSON(t, w, map[string]any{"data": data})
	})

	since := time.Unix(900, 0).UTC()
	convs, err := c.ListConversations(context.Background(), ListOptions{Since: since})
	if err != nil {
		t.Fatal(err)
	}

	if !gotAuth || !gotFilter {
		t.Fatalf("auth=%t filter=%t", gotAuth, gotFilter)
	}

	if strings.Join(gotPages, ",") != "1,2" {
		t.Fatalf("pages walked: %v", gotPages)
	}

	if len(convs) != 3 || convs[0].ID != "conv-1" || convs[2].ID != "conv-3" {
		t.Fatalf("stitched wrong: %+v", convs)
	}
}

func TestListConversationsLimitCapsFetch(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		writeJSON(t, w, map[string]any{"data": []Conversation{
			{ID: "conv-" + strconv.Itoa(page)},
			{ID: "conv-" + strconv.Itoa(page) + "b"},
		}})
	})

	convs, err := c.ListConversations(context.Background(), ListOptions{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}

	if len(convs) != 3 {
		t.Fatalf("want limit 3, got %d", len(convs))
	}
}

func TestMessagesDecodesNullTextAsEmpty(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/ai/conversations/conv-9/messages") {
			t.Fatalf("path: %s", r.URL.Path)
		}

		writeJSON(t, w, map[string]any{"data": []map[string]any{
			{"role": "assistant", "text": "You WANT eggs?!", "created_at": "2026-10-09T19:50:12Z", "sent_at": "2026-10-09T19:50:13Z"},
			{"role": "tool", "text": nil, "created_at": "2026-10-09T19:50:14Z"},
		}})
	})

	msgs, err := c.Messages(context.Background(), "conv-9")
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}

	if msgs[0].Text != "You WANT eggs?!" || msgs[0].Role != "assistant" {
		t.Fatalf("assistant message wrong: %+v", msgs[0])
	}

	if msgs[1].Role != "tool" || msgs[1].Text != "" {
		t.Fatalf("tool message null-text must decode to empty: %+v", msgs[1])
	}

	if msgs[0].SentAt.UTC().Format(time.RFC3339) != "2026-10-09T19:50:13Z" {
		t.Fatalf("sent_at wrong: %v", msgs[0].SentAt)
	}
}

func TestRecordingFetchShape(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/recordings/rec-1") {
			t.Fatalf("path: %s", r.URL.Path)
		}

		writeJSON(t, w, map[string]any{"data": map[string]any{
			"id":                "rec-1",
			"call_control_id":   "v3:abc:def",
			"channels":          "dual",
			"duration_millis":   26000,
			"recording_urls":    nil,
			"download_urls":     map[string]string{"mp3": "https://cdn.example/mp3", "wav": "https://cdn.example/wav"},
			"recording_started_at": "2026-10-09T19:50:10Z",
			"recording_ended_at":   "2026-10-09T19:50:36Z",
			"from":              "+17287289311",
			"to":                "+14084757593",
			"status":            "completed",
		}})
	})

	rec, err := c.Recording(context.Background(), "rec-1")
	if err != nil {
		t.Fatal(err)
	}

	if rec.ID != "rec-1" || rec.DurationMillis != 26000 || rec.CallControlID != "v3:abc:def" {
		t.Fatalf("recording fields wrong: %+v", rec)
	}

	if rec.DownloadURLs.MP3 != "https://cdn.example/mp3" {
		t.Fatalf("download_urls.mp3 wrong: %+v", rec.DownloadURLs)
	}

	if !rec.RecordingEndedAt.After(rec.RecordingStartedAt) {
		t.Fatalf("recording timestamps wrong: %v -> %v", rec.RecordingStartedAt, rec.RecordingEndedAt)
	}
}

func TestHTTPErrorFamilies(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   errorfamily.Family
	}{
		{http.StatusUnauthorized, errorfamily.Rejection},
		{http.StatusNotFound, errorfamily.Rejection},
		{http.StatusInternalServerError, errorfamily.Transient},
		{http.StatusBadGateway, errorfamily.Transient},
	} {
		c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"errors":[{"code":"10015","title":"nope"}]}`, tc.status)
		})

		_, err := c.Messages(context.Background(), "conv-x")
		if err == nil {
			t.Fatalf("status %d: want error", tc.status)
		}

		var ef *errorfamily.Error
		if !errors.As(err, &ef) {
			t.Fatalf("status %d: not an errorfamily error: %v", tc.status, err)
		}

		if ef.Family() != tc.want {
			t.Fatalf("status %d: family %v, want %v", tc.status, ef.Family(), tc.want)
		}

		if !strings.Contains(ef.Code(), strconv.Itoa(tc.status)) {
			t.Fatalf("status %d: code %q lacks status", tc.status, ef.Code())
		}
	}
}

func TestListRecordingsLimit(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if page > 1 {
			writeJSON(t, w, map[string]any{"data": []map[string]string{}})

			return
		}

		writeJSON(t, w, map[string]any{"data": []map[string]string{
			{"id": "rec-a", "call_control_id": "v3:one"},
			{"id": "rec-b", "call_control_id": "v3:two"},
		}})
	})

	recs, err := c.ListRecordings(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(recs) != 2 {
		t.Fatalf("want 2 recordings, got %d", len(recs))
	}
}

const eggsConversationJSON = `{
  "id": "ba15158e-aaaa-4bbb-8ccc-00000000egg1",
  "created_at": "2026-10-09T19:50:03Z",
  "last_message_at": "2026-10-09T19:50:29Z",
  "name": "Woolly calls Lars",
  "metadata": {
    "assistant_id": "assistant-ce5d8cb7-5edb-4752-af98-77a0563d2296",
    "call_control_id": "v3:eggcall:leg1",
    "telnyx_agent_target": "+17287289311",
    "telnyx_end_user_target": "+14084757593",
    "telnyx_conversation_channel": "phone_call"
  }
}`

func eggsMessages() []Message {
	return []Message{
		{Role: "user", Text: "  Woolly, did you take my eggs?  ", CreatedAt: time.Unix(10, 0).UTC()},
		{Role: "tool", Text: ""},
		{Role: "assistant", Text: "[angry] I did NOT take your eggs! [angry]", CreatedAt: time.Unix(20, 0).UTC()},
	}
}

func eggsRecording() Recording {
	return Recording{ID: "c8daa4ef-020c-49c5-8a35-888b41e33900", CallControlID: "v3:eggcall:leg1", DurationMillis: 26000}
}

func TestMapConversationEggsFixture(t *testing.T) {
	var conv Conversation
	if err := json.Unmarshal([]byte(eggsConversationJSON), &conv); err != nil {
		t.Fatal(err)
	}

	rec := eggsRecording()

	obs, err := MapConversation(conv, eggsMessages(), &rec, DefaultMapOptions())
	if err != nil {
		t.Fatal(err)
	}

	if obs.NativeID != calls.NativeID("ba15158e-aaaa-4bbb-8ccc-00000000egg1") {
		t.Fatalf("native id wrong: %+v", obs)
	}

	if obs.Kind != calls.SourceTelnyxConversation {
		t.Fatalf("kind wrong: %+v", obs)
	}

	if !obs.IsAI {
		t.Fatalf("assistant_id must set the AI badge: %+v", obs)
	}

	if obs.FromNumber != "+17287289311" || obs.ToNumber != "+14084757593" {
		t.Fatalf("participants wrong: %+v", obs)
	}

	if obs.DurationSecs != 26 {
		t.Fatalf("duration must come from the recording: %d", obs.DurationSecs)
	}

	if obs.RecordingRef != "c8daa4ef-020c-49c5-8a35-888b41e33900" {
		t.Fatalf("recording ref wrong: %q", obs.RecordingRef)
	}

	if obs.TranscriptExcerpt != "Woolly, did you take my eggs?" {
		t.Fatalf("excerpt must be the first user line, trimmed: %q", obs.TranscriptExcerpt)
	}

	if obs.StartedAt.UTC().Format(time.RFC3339) != "2026-10-09T19:50:03Z" {
		t.Fatalf("started_at wrong: %v", obs.StartedAt)
	}

	if obs.Direction != calls.DirectionOutbound {
		t.Fatalf("default direction wrong: %+v", obs)
	}

	if err := obs.Validate(); err != nil {
		t.Fatalf("observation must validate: %v", err)
	}
}

func TestMapConversationFallbacks(t *testing.T) {
	conv := Conversation{
		ID:            "conv-2",
		CreatedAt:     time.Unix(0, 0).UTC(),
		LastMessageAt: time.Unix(95, 0).UTC(),
	}

	obs, err := MapConversation(conv, nil, nil, MapOptions{Direction: calls.DirectionInbound})
	if err != nil {
		t.Fatal(err)
	}

	if obs.DurationSecs != 95 {
		t.Fatalf("duration must fall back to last_message_at - created_at: %d", obs.DurationSecs)
	}

	if obs.Direction != calls.DirectionInbound {
		t.Fatalf("caller-provided direction must win: %+v", obs)
	}

	if obs.IsAI || obs.TranscriptExcerpt != "" || obs.RecordingRef != "" {
		t.Fatalf("empty metadata must stay empty: %+v", obs)
	}
}

func TestMapConversationTruncatesExcerptByRunes(t *testing.T) {
	conv := Conversation{ID: "conv-3", CreatedAt: time.Unix(0, 0).UTC(), Metadata: map[string]string{"assistant_id": "a1"}}
	msgs := []Message{{Role: "assistant", Text: "äöü" + strings.Repeat("x", 30)}}

	obs, err := MapConversation(conv, msgs, nil, MapOptions{Direction: calls.DirectionOutbound, ExcerptMax: 5})
	if err != nil {
		t.Fatal(err)
	}

	if obs.TranscriptExcerpt != "äöüxx" {
		t.Fatalf("rune truncation wrong: %q", obs.TranscriptExcerpt)
	}
}

func TestMapConversationRejectsParticipantlessConv(t *testing.T) {
	conv := Conversation{ID: "conv-4", CreatedAt: time.Unix(0, 0).UTC()}

	_, err := MapConversation(conv, nil, nil, MapOptions{Direction: calls.DirectionOutbound})
	if err == nil {
		t.Fatal("both participants missing must reject")
	}
}

func TestRecordingByCallControl(t *testing.T) {
	recs := []Recording{
		{ID: "rec-a", CallControlID: "v3:one"},
		{ID: "rec-b", CallControlID: "v3:two"},
	}

	got, ok := RecordingByCallControl(recs, "v3:two")
	if !ok || got.ID != "rec-b" {
		t.Fatalf("match wrong: %t %+v", ok, got)
	}

	if _, ok := RecordingByCallControl(recs, "v3:missing"); ok {
		t.Fatal("missing call control id must not match")
	}
}
