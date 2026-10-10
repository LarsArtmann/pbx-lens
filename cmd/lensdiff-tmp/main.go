package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

func main() {
	ctx := context.Background()
	key := os.Getenv("TELNYX_API_KEY")

	if len(key) < 10 {
		raw, err := os.ReadFile(os.Getenv("HOME") + "/.pbx-prod-secrets/telnyx_api_key")
		if err != nil {
			panic(err)
		}

		key = strings.TrimSpace(string(raw))
	}

	c := telnyx.NewClient(key, "")
	convs, err := c.ListConversations(ctx, telnyx.ListOptions{Limit: 1})
	if err != nil {
		panic(err)
	}

	conv := convs[0]
	msgs, err := c.Messages(ctx, conv.ID)
	if err != nil {
		panic(err)
	}

	recs, err := c.ListRecordings(ctx, 300)
	if err != nil {
		panic(err)
	}

	var rec *telnyx.Recording
	if r, ok := telnyx.RecordingByCallControl(recs, conv.MetadataString("call_control_id")); ok {
		rec = &r
	}

	obs, err := telnyx.MapConversation(conv, msgs, rec, telnyx.MapOptions{Direction: "outbound"})
	if err != nil {
		panic(err)
	}

	fresh, _ := json.MarshalIndent(obs, "", "  ")
	fmt.Println("FRESH:", string(fresh))

	db, err := sql.Open("sqlite", os.Args[1])
	if err != nil {
		panic(err)
	}

	var stored string
	if err := db.QueryRow(`select value from meta_map where collection='call_views' and key=?`,
		"StreamMarker:telnyx-conversation:"+conv.ID).Scan(&stored); err != nil {
		panic(err)
	}

	fmt.Println("STORED:", stored)
}
