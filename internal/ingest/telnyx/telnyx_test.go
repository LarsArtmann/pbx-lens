package telnyx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/wire"
)

func ft(unix int64) wire.FlexTime {
	return wire.FlexTime{Time: time.Unix(unix, 0).UTC()}
}

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
	var (
		gotAuth, gotFilter bool
		gotPages           []string
	)

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer header")
		}

		gotAuth = true
		gotFilter = gotFilter || strings.Contains(r.URL.RawQuery, "last_message_at=gte.")
		gotPages = append(gotPages, r.URL.Query().Get("page[number]"))

		page, err := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if err != nil {
			t.Fatalf("page[number]: %v", err)
		}

		data := []Conversation{}

		switch page {
		case 1:
			data = []Conversation{
				{ID: "conv-1", CreatedAt: ft(1000), Metadata: map[string]any{"assistant_id": "assistant-x"}},
				{ID: "conv-2", CreatedAt: ft(2000)},
			}
		case 2:
			data = []Conversation{
				{ID: "conv-3", CreatedAt: ft(3000)},
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

		if r.URL.Query().Get("page[number]") != "1" {
			writeJSON(t, w, map[string]any{"data": []map[string]any{}})

			return
		}

		writeJSON(t, w, map[string]any{"data": []map[string]any{
			{
				"role":       "assistant",
				"text":       "You WANT eggs?!",
				"created_at": "2026-10-09T19:50:12Z",
				"sent_at":    "2026-10-09T19:50:13Z",
			},
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
			"id":              "rec-1",
			"call_control_id": "v3:abc:def",
			"channels":        "dual",
			"duration_millis": 26000,
			"recording_urls":  nil,
			"download_urls": map[string]string{
				"mp3": "https://cdn.example/mp3",
				"wav": "https://cdn.example/wav",
			},
			"recording_started_at": "2026-10-09T19:50:10Z",
			"recording_ended_at":   "2026-10-09T19:50:36Z",
			"from":                 "+17287289311",
			"to":                   "+14084757593",
			"status":               "completed",
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

	if !rec.RecordingEndedAt.After(rec.RecordingStartedAt.Time) {
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

		ef, ok := errors.AsType[*errorfamily.Error](err)
		if !ok {
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
    "telnyx_conversation_channel": "phone_call",
    "called_tools": []
  }
}`

func eggsMessages() []Message {
	return []Message{
		{Role: "user", Text: "  Woolly, did you take my eggs?  ", CreatedAt: ft(10)},
		{Role: "tool", Text: ""},
		{Role: "assistant", Text: "[angry] I did NOT take your eggs! [angry]", CreatedAt: ft(20)},
	}
}

func eggsRecording() Recording {
	return Recording{
		ID:             "c8daa4ef-020c-49c5-8a35-888b41e33900",
		CallControlID:  "v3:eggcall:leg1",
		DurationMillis: 26000,
	}
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
		CreatedAt:     ft(0),
		LastMessageAt: ft(95),
		Metadata:      map[string]any{"telnyx_agent_target": "+17287289311"},
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
	conv := Conversation{
		ID:        "conv-3",
		CreatedAt: ft(0),
		Metadata:  map[string]any{"assistant_id": "a1", "telnyx_end_user_target": "+14084757593"},
	}
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
	conv := Conversation{ID: "conv-4", CreatedAt: ft(0)}

	_, err := MapConversation(conv, nil, nil, MapOptions{Direction: calls.DirectionOutbound})
	if err == nil {
		t.Fatal("both participants missing must reject")
	}
}

func TestFlexTimeAcceptsOffsetlessTimestamps(t *testing.T) {
	var c Conversation
	if err := json.Unmarshal(
		[]byte(`{"id":"c1","created_at":"2026-10-09T19:51:35","last_message_at":"2026-10-09T19:52:00.123456Z"}`),
		&c,
	); err != nil {
		t.Fatal(err)
	}

	if c.CreatedAt.UTC().Format(time.RFC3339) != "2026-10-09T19:51:35Z" {
		t.Fatalf("offset-less timestamp wrong: %v", c.CreatedAt.Time)
	}

	if c.LastMessageAt.UTC().Format(time.RFC3339Nano) != "2026-10-09T19:52:00.123456Z" {
		t.Fatalf("fractional timestamp wrong: %v", c.LastMessageAt.Time)
	}
}

func TestMapConversationTimestampPrecisionStable(t *testing.T) {
	nanos := `{"id":"conv-p","created_at":"2026-10-09T19:50:27.401017904Z","last_message_at":"2026-10-09T19:50:59.888888888Z","metadata":{"assistant_id":"a1","telnyx_agent_target":"+17287289311"}}`
	micros := `{"id":"conv-p","created_at":"2026-10-09T19:50:27.401018Z","last_message_at":"2026-10-09T19:50:59.888889Z","metadata":{"assistant_id":"a1","telnyx_agent_target":"+17287289311"}}`

	var a, b Conversation
	if err := json.Unmarshal([]byte(nanos), &a); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal([]byte(micros), &b); err != nil {
		t.Fatal(err)
	}

	obsA, err := MapConversation(a, nil, nil, MapOptions{Direction: calls.DirectionOutbound})
	if err != nil {
		t.Fatal(err)
	}

	obsB, err := MapConversation(b, nil, nil, MapOptions{Direction: calls.DirectionOutbound})
	if err != nil {
		t.Fatal(err)
	}

	if obsA.ContentHash() != obsB.ContentHash() {
		t.Fatalf("timestamp precision leaked into the content hash: %s vs %s", obsA.ContentHash(), obsB.ContentHash())
	}

	if obsA.StartedAt.Format(time.RFC3339) != "2026-10-09T19:50:27Z" {
		t.Fatalf("started_at must truncate to seconds: %v", obsA.StartedAt)
	}

	if obsA.DurationSecs != 32 {
		t.Fatalf("fallback duration must use truncated timestamps: %d", obsA.DurationSecs)
	}
}

func TestExcerptOrderIndependent(t *testing.T) {
	conv := Conversation{
		ID:            "conv-ord",
		CreatedAt:     ft(0),
		LastMessageAt: ft(30),
		Metadata:      map[string]any{"assistant_id": "a1", "telnyx_agent_target": "+17287289311"},
	}

	// The API serves newest-first; both orders must map to ONE observation
	// (same ContentHash) or re-poll would "correct" every conversation.
	oldestFirst := []Message{
		{Role: "assistant", Text: "Hi! This is Woolly.", CreatedAt: ft(5)},
		{Role: "user", Text: "Did you take my eggs?", CreatedAt: ft(10)},
		{Role: "assistant", Text: "Never!", CreatedAt: ft(20)},
	}
	newestFirst := slices.Clone(oldestFirst)
	slices.Reverse(newestFirst)

	asc, err := MapConversation(conv, oldestFirst, nil, DefaultMapOptions())
	if err != nil {
		t.Fatal(err)
	}

	desc, err := MapConversation(conv, newestFirst, nil, DefaultMapOptions())
	if err != nil {
		t.Fatal(err)
	}

	if asc.ContentHash() != desc.ContentHash() {
		t.Fatalf("message order leaked into the content hash:\n asc %s\ndesc %s", asc.ContentHash(), desc.ContentHash())
	}

	// The excerpt is the call opening: the first user line, not the last
	// utterance the newest-first API happens to serve first.
	if asc.TranscriptExcerpt != "Did you take my eggs?" {
		t.Fatalf("excerpt must be the first user line: %q", asc.TranscriptExcerpt)
	}
}

func TestExcerptFallsBackToAssistantGreeting(t *testing.T) {
	conv := Conversation{
		ID:        "conv-greet",
		CreatedAt: ft(0),
		Metadata:  map[string]any{"assistant_id": "a1", "telnyx_agent_target": "+17287289311"},
	}
	msgs := []Message{{Role: "assistant", Text: "Hi Lars! This is Woolly.", CreatedAt: ft(5)}}

	obs, err := MapConversation(conv, msgs, nil, DefaultMapOptions())
	if err != nil {
		t.Fatal(err)
	}

	if obs.TranscriptExcerpt != "Hi Lars! This is Woolly." {
		t.Fatalf("no-user-turn call must excerpt the greeting: %q", obs.TranscriptExcerpt)
	}
}

func TestTranscriptChronologicalAndFiltered(t *testing.T) {
	conv := Conversation{
		ID:        "conv-tr",
		CreatedAt: ft(0),
		Metadata:  map[string]any{"telnyx_agent_target": "+17287289311"},
	}
	msgs := []Message{
		{Role: "tool", Text: "internal", CreatedAt: ft(1)},
		{Role: "assistant", Text: "second", CreatedAt: ft(20)},
		{Role: "user", Text: "first", CreatedAt: ft(10)},
		{Role: "user", Text: "  ", CreatedAt: ft(15)},
	}

	obs, err := MapConversation(conv, msgs, nil, DefaultMapOptions())
	if err != nil {
		t.Fatal(err)
	}

	if len(obs.Transcript) != 2 {
		t.Fatalf("tool turns and blank turns must drop: %+v", obs.Transcript)
	}

	if obs.Transcript[0].Role != "user" || obs.Transcript[0].Text != "first" {
		t.Fatalf("turns must be chronological: %+v", obs.Transcript)
	}

	if obs.Transcript[1].Role != "assistant" || obs.Transcript[1].Text != "second" {
		t.Fatalf("turn order wrong: %+v", obs.Transcript)
	}

	if !obs.Transcript[0].At.Equal(time.Unix(10, 0).UTC()) {
		t.Fatalf("turn timestamps must truncate to seconds: %v", obs.Transcript[0].At)
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

func TestDetailRecordsSendsRequiredFilter(t *testing.T) {
	var gotQuery url.Values

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/detail_records" {
			t.Fatalf("path: %s", r.URL.Path)
		}

		gotQuery = r.URL.Query()
		writeJSON(t, w, map[string]any{"data": []map[string]any{}})
	})

	since := time.Unix(1700000000, 0).UTC()

	recs, err := c.DetailRecords(context.Background(), since)
	if err != nil {
		t.Fatal(err)
	}

	if recs != nil {
		t.Fatalf("empty page must decode to nothing: %+v", recs)
	}

	if got := gotQuery.Get("filter[record_type]"); got != "call-control" {
		t.Fatalf("record_type filter = %q, want call-control", got)
	}

	if got := gotQuery.Get("filter[started_at][gte]"); got != since.Format(time.RFC3339) {
		t.Fatalf("started_at gte = %q, want %q", got, since.Format(time.RFC3339))
	}

	if got := gotQuery.Get("page[size]"); got != "50" {
		t.Fatalf("detail records page size = %q, want the endpoint max 50", got)
	}
}

func TestDetailRecordDecodesTolerantly(t *testing.T) {
	// The public spec publishes no call-control record shape, so the decode
	// must accept both id spellings, both call-control id spellings, and
	// cost as string or number (unverified live — fixtures document the
	// tolerance contract).
	tests := []struct {
		name string
		body string
		want DetailRecord
	}{
		{
			name: "documented shape: id + call_control_id + string cost",
			body: `{"id":"dr-1","call_control_id":"v3:a:1","cost":"0.002","currency":"USD","started_at":"2026-10-09T19:50:03Z"}`,
			want: DetailRecord{ID: "dr-1", CallControlID: "v3:a:1", Cost: "0.002", Currency: "USD", StartedAt: wire.FlexTime{Time: time.Date(2026, 10, 9, 19, 50, 3, 0, time.UTC)}},
		},
		{
			name: "webrtc-doc shape: uuid + telnyx_call_control_id",
			body: `{"uuid":"dr-2","telnyx_call_control_id":"v3:b:2","cost":"0.002","currency":"USD","started_at":"2026-10-09T19:50:03Z"}`,
			want: DetailRecord{ID: "dr-2", CallControlID: "v3:b:2", Cost: "0.002", Currency: "USD", StartedAt: wire.FlexTime{Time: time.Date(2026, 10, 9, 19, 50, 3, 0, time.UTC)}},
		},
		{
			name: "numeric cost",
			body: `{"id":"dr-3","call_control_id":"v3:c:3","cost":0.0031,"currency":"USD"}`,
			want: DetailRecord{ID: "dr-3", CallControlID: "v3:c:3", Cost: "0.0031", Currency: "USD"},
		},
		{
			name: "missing everything optional",
			body: `{"id":"dr-4"}`,
			want: DetailRecord{ID: "dr-4"},
		},
	}

	for _, test := range tests {
		var got DetailRecord
		if err := json.Unmarshal([]byte(test.body), &got); err != nil {
			t.Fatalf("%s: decode: %v", test.name, err)
		}

		if got.ID != test.want.ID || got.CallControlID != test.want.CallControlID ||
			got.Cost != test.want.Cost || got.Currency != test.want.Currency ||
			!got.StartedAt.Equal(test.want.StartedAt.Time) {
			t.Errorf("%s: got %+v, want %+v", test.name, got, test.want)
		}
	}
}

func TestDetailRecordByCallControl(t *testing.T) {
	recs := []DetailRecord{
		{ID: "dr-a", CallControlID: "v3:one"},
		{ID: "dr-b", CallControlID: "v3:two"},
	}

	got, ok := DetailRecordByCallControl(recs, "v3:one")
	if !ok || got.ID != "dr-a" {
		t.Fatalf("match wrong: %t %+v", ok, got)
	}

	if _, ok := DetailRecordByCallControl(recs, "v3:missing"); ok {
		t.Fatal("missing call control id must not match")
	}
}

func TestConversationInsightsPathAndShape(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ai/conversations/conv-7/conversations-insights" {
			t.Fatalf("path: %s", r.URL.Path)
		}

		writeJSON(t, w, map[string]any{"data": []map[string]any{
			{
				"id":         "ins-1",
				"status":     "completed",
				"created_at": "2026-10-09T19:51:35",
				"conversation_insights": []map[string]string{
					{"insight_id": "tpl-summarize", "result": "Caller wanted eggs."},
				},
			},
		}})
	})

	insights, err := c.ConversationInsights(context.Background(), "conv-7")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 || insights[0].Status != "completed" {
		t.Fatalf("insight row wrong: %+v", insights)
	}

	if got := InsightSummary(insights); got != "Caller wanted eggs." {
		t.Fatalf("summary through the wire decode wrong: %q", got)
	}
}

func TestConversationInsightsRejectsEmptyID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("must not hit the wire with an empty conversation id")
	})

	if _, err := c.ConversationInsights(context.Background(), ""); err == nil {
		t.Fatal("empty conversation id must reject")
	}
}
