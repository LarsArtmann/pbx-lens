package cli

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// fakeTelnyx stubs the four endpoints one poll pass touches: conversations,
// messages, recordings, detail records, and per-conversation insights. Hit
// counters assert what the enrichment pass skips.
type fakeTelnyx struct {
	detailHits     int
	insightsHits   int
	insightsStatus string
	insightsResult string
}

func (f *fakeTelnyx) server(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/ai/conversations":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{{
				"id":              "conv-1",
				"created_at":      "2026-10-09T19:50:03Z",
				"last_message_at": "2026-10-09T19:50:29Z",
				"metadata": map[string]any{
					"assistant_id":           "assistant-1",
					"call_control_id":        "v3:eggcall:leg1",
					"telnyx_agent_target":    "+17287289311",
					"telnyx_end_user_target": "+14084757593",
				},
			}}})
		case "/ai/conversations/conv-1/messages":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{
				{
					"role":       "user",
					"text":       "Woolly, did you take my eggs?",
					"created_at": "2026-10-09T19:50:10Z",
				},
				{
					"role":       "assistant",
					"text":       "I did NOT take your eggs!",
					"created_at": "2026-10-09T19:50:20Z",
				},
			}})
		case "/recordings":
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{}})
		case "/detail_records":
			f.detailHits++
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{{
				"id":              "dr-1",
				"call_control_id": "v3:eggcall:leg1",
				"cost":            "0.002",
				"currency":        "USD",
				"started_at":      "2026-10-09T19:50:03Z",
			}}})
		case "/ai/conversations/conv-1/conversations-insights":
			f.insightsHits++
			_ = json.MarshalWrite(w, map[string]any{"data": []map[string]any{{
				"id":         "ins-1",
				"status":     f.insightsStatus,
				"created_at": "2026-10-09T19:51:35",
				"conversation_insights": []map[string]string{
					{"insight_id": "tpl", "result": f.insightsResult},
				},
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestEnrichTelnyxCallsFillsCostAndSummary(t *testing.T) {
	ctx := context.Background()

	fake := &fakeTelnyx{
		insightsStatus: "completed",
		insightsResult: "Caller wanted to know about the eggs.",
	}
	srv := fake.server(t)
	t.Cleanup(srv.Close)

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	index, err := search.Open(ctx, filepath.Join(t.TempDir(), "enrich.db"))
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	client := telnyx.NewClient("test-key", srv.URL)
	logs := &pollTestLog{}

	PollOnce(
		ctx,
		sys,
		index,
		PollConfig{Telnyx: client, SinceAgo: 48 * time.Hour, Direction: calls.DirectionOutbound},
		logs.printf,
	)

	view := telnyxView(t, ctx, sys, "conv-1")
	if view.CostMicros != 2000 || view.CostCurrency != "USD" {
		t.Fatalf("cost enrichment wrong: %+v", view)
	}

	if view.InsightSummary != "Caller wanted to know about the eggs." {
		t.Fatalf("insight enrichment wrong: %+v", view)
	}

	hits, err := index.Query(ctx, "eggs", 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if len(hits) != 1 || hits[0].NativeID != "conv-1" {
		t.Fatalf("FTS must carry the enriched call, got %+v", hits)
	}

	joined := strings.Join(logs.lines, "\n")
	if !strings.Contains(
		joined,
		"poll telnyx enrich: 1 considered, 1 enriched, 0 unchanged, 0 missing, 0 rejected",
	) {
		t.Fatalf("first pass enrichment log wrong:\n%s", joined)
	}

	// The second pass replays to zero everywhere: the observation is
	// idempotent AND the enrichment is unchanged, so no event is appended.
	logs.lines = nil
	PollOnce(
		ctx,
		sys,
		index,
		PollConfig{Telnyx: client, SinceAgo: 48 * time.Hour, Direction: calls.DirectionOutbound},
		logs.printf,
	)

	joined = strings.Join(logs.lines, "\n")
	if !strings.Contains(
		joined,
		"poll telnyx enrich: 1 considered, 0 enriched, 1 unchanged, 0 missing, 0 rejected",
	) {
		t.Fatalf("second pass must replay to zero:\n%s", joined)
	}
}

func TestEnrichTelnyxCallsSkipsFetchWhenAlreadyEnriched(t *testing.T) {
	ctx := context.Background()

	fake := &fakeTelnyx{insightsStatus: "completed", insightsResult: "final summary"}
	srv := fake.server(t)
	t.Cleanup(srv.Close)

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	client := telnyx.NewClient("test-key", srv.URL)
	logs := &pollTestLog{}

	PollOnce(
		ctx,
		sys,
		nil,
		PollConfig{Telnyx: client, SinceAgo: 48 * time.Hour, Direction: calls.DirectionOutbound},
		logs.printf,
	)

	if fake.detailHits != 1 || fake.insightsHits != 1 {
		t.Fatalf(
			"first pass must fetch both sources: detail=%d insights=%d",
			fake.detailHits,
			fake.insightsHits,
		)
	}

	logs.lines = nil
	PollOnce(
		ctx,
		sys,
		nil,
		PollConfig{Telnyx: client, SinceAgo: 48 * time.Hour, Direction: calls.DirectionOutbound},
		logs.printf,
	)

	if fake.detailHits != 1 || fake.insightsHits != 1 {
		t.Fatalf(
			"already-enriched fields must skip their fetch: detail=%d insights=%d",
			fake.detailHits, fake.insightsHits,
		)
	}
}

func TestEnrichTelnyxCallsCountsMissingAndRejects(t *testing.T) {
	ctx := context.Background()

	fake := &fakeTelnyx{insightsStatus: "pending", insightsResult: ""}
	srv := fake.server(t)
	t.Cleanup(srv.Close)

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	client := telnyx.NewClient("test-key", srv.URL)

	// A seed whose observation never landed counts Missing, not Rejected:
	// the next pass retries once the observation ingest catches up. The key
	// derives the same way the fetch pass builds real seeds — never
	// hand-assembled with the library's marker prefix.
	ghost := calls.Observation{NativeID: "ghost", Kind: calls.SourceTelnyxConversation}

	ghostKey, err := ghost.StreamID()
	if err != nil {
		t.Fatalf("ghost stream id: %v", err)
	}

	missing := EnrichmentSeed{
		ConversationID: "ghost",
		StreamKey:      ghostKey.String(),
	}

	stats, err := EnrichTelnyxCalls(ctx, sys, client, []EnrichmentSeed{missing}, time.Now().UTC())
	if err != nil {
		t.Fatalf("EnrichTelnyxCalls: %v", err)
	}

	if stats.Missing != 1 || stats.Enriched != 0 {
		t.Fatalf("missing seed wrong: %+v", stats)
	}
}

// telnyxView reads one conversation's projected view through the same key
// derivation the enrich pass uses.
func telnyxView(
	t *testing.T,
	ctx context.Context,
	sys *system.System,
	conversationID string,
) calls.CallView {
	t.Helper()

	sid, err := calls.Observation{
		NativeID: calls.NativeID(conversationID),
		Kind:     calls.SourceTelnyxConversation,
	}.StreamID()
	if err != nil {
		t.Fatalf("stream id: %v", err)
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	view, found, err := reader.Get(ctx, sid.String())
	if err != nil || !found {
		t.Fatalf("view %s: found=%t err=%v", sid, found, err)
	}

	return view
}
