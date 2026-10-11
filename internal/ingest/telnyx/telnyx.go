package telnyx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/wire"
)

const (
	DefaultBase     = "https://api.telnyx.com/v2"
	defaultPageSize = 100
	maxPages        = 500
	maxErrorBytes   = 300
	maxBodyBytes    = 8 << 20
	// detailRecordsPageSize is /detail_records' page[size] maximum per the
	// endpoint schema (other list endpoints allow 100).
	detailRecordsPageSize = 50
)

type Client struct {
	base     string
	apiKey   string
	http     *http.Client
	PageSize int
}

func NewClient(apiKey, baseURL string) *Client {
	base := baseURL
	if base == "" {
		base = DefaultBase
	}

	return &Client{base: base, apiKey: apiKey, http: http.DefaultClient, PageSize: defaultPageSize}
}

type ListOptions struct {
	Since time.Time
	Limit int
}

type Conversation struct {
	ID            string         `json:"id"`
	CreatedAt     wire.FlexTime  `json:"created_at"`
	LastMessageAt wire.FlexTime  `json:"last_message_at"`
	Name          string         `json:"name"`
	Metadata      map[string]any `json:"metadata"`
}

func (c Conversation) MetadataString(key string) string {
	s, _ := c.Metadata[key].(string)

	return s
}

type Message struct {
	Role      string        `json:"role"`
	Text      string        `json:"text"`
	CreatedAt wire.FlexTime `json:"created_at"`
	SentAt    wire.FlexTime `json:"sent_at"`
}

type DownloadURLs struct {
	MP3 string `json:"mp3"`
	WAV string `json:"wav"`
}

type Recording struct {
	ID                 string        `json:"id"`
	CallControlID      string        `json:"call_control_id"`
	Channels           string        `json:"channels"`
	CreatedAt          wire.FlexTime `json:"created_at"`
	DownloadURLs       DownloadURLs  `json:"download_urls"`
	DurationMillis     int           `json:"duration_millis"`
	RecordingStartedAt wire.FlexTime `json:"recording_started_at"`
	RecordingEndedAt   wire.FlexTime `json:"recording_ended_at"`
	From               string        `json:"from"`
	To                 string        `json:"to"`
	Status             string        `json:"status"`
}

func (c *Client) ListConversations(ctx context.Context, opts ListOptions) ([]Conversation, error) {
	q := url.Values{}
	if !opts.Since.IsZero() {
		q.Set("last_message_at", "gte."+opts.Since.UTC().Format(time.RFC3339))
	}

	return fetchAll[Conversation](ctx, c, "/ai/conversations", q, opts.Limit, 0)
}

func (c *Client) Messages(ctx context.Context, conversationID string) ([]Message, error) {
	if conversationID == "" {
		return nil, errorfamily.NewRejection(
			"telnyx.empty_conversation_id",
			"conversation id is required",
		)
	}

	return fetchAll[Message](
		ctx,
		c,
		"/ai/conversations/"+url.PathEscape(conversationID)+"/messages",
		url.Values{},
		0,
		0,
	)
}

func (c *Client) Recording(ctx context.Context, recordingID string) (Recording, error) {
	if recordingID == "" {
		return Recording{}, errorfamily.NewRejection(
			"telnyx.empty_recording_id",
			"recording id is required",
		)
	}

	var out struct {
		Data Recording `json:"data"`
	}

	if err := c.get(ctx, "/recordings/"+url.PathEscape(recordingID), nil, &out); err != nil {
		return Recording{}, err
	}

	return out.Data, nil
}

func (c *Client) ListRecordings(ctx context.Context, limit int) ([]Recording, error) {
	return fetchAll[Recording](ctx, c, "/recordings", url.Values{}, limit, 0)
}

// DetailRecord is one /detail_records row. The public OpenAPI spec publishes
// no call-control record schema (its DetailRecord union lists only
// messaging/conference/amd/verify/sim/media), so the decode is deliberately
// tolerant: both id spellings (id, uuid), both call-control id spellings
// (call_control_id, telnyx_call_control_id), and cost as either a decimal
// string ("0.002") or a JSON number. Live shape unverified (no API key at
// implementation time) — re-verify against the live API like the rest of the
// Telnyx cluster, quarterly per AGENTS.
type DetailRecord struct {
	ID            string
	CallControlID string
	Cost          string
	Currency      string
	StartedAt     wire.FlexTime
}

func (d *DetailRecord) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID            *string          `json:"id"`
		UUID          *string          `json:"uuid"`
		CallControlID *string          `json:"call_control_id"`
		TelnyxCCID    *string          `json:"telnyx_call_control_id"`
		Cost          *json.RawMessage `json:"cost"`
		Currency      *string          `json:"currency"`
		StartedAt     wire.FlexTime    `json:"started_at"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.ID = firstString(raw.ID, raw.UUID)
	d.CallControlID = firstString(raw.CallControlID, raw.TelnyxCCID)
	d.Cost = costLiteral(raw.Cost)
	d.Currency = firstString(raw.Currency)
	d.StartedAt = raw.StartedAt

	return nil
}

func firstString(candidates ...*string) string {
	for _, c := range candidates {
		if c != nil && *c != "" {
			return *c
		}
	}

	return ""
}

// costLiteral accepts the documented decimal-string cost and tolerates a
// bare JSON number, normalizing both to the string form MicrosFromCost
// parses.
func costLiteral(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}

	var s string
	if err := json.Unmarshal(*raw, &s); err == nil {
		return s
	}

	var f float64
	if err := json.Unmarshal(*raw, &f); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}

	return ""
}

// ConversationInsight is one /ai/conversations/{id}/conversations-insights
// row (schema verified against the Telnyx OpenAPI spec 2026-10-11). Status
// is pending|in_progress|completed|failed; only completed rows carry final
// results.
type ConversationInsight struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	CreatedAt wire.FlexTime   `json:"created_at"`
	Results   []InsightResult `json:"conversation_insights"`
}

type InsightResult struct {
	InsightID string `json:"insight_id"`
	Result    string `json:"result"`
}

// DetailRecords lists call-control detail records since the given lower
// bound (zero = unfiltered). filter[record_type] is required by the
// endpoint; started_at is one of the three call-control filter fields the
// docs publish. Page size caps at 50 per the endpoint's own schema.
func (c *Client) DetailRecords(ctx context.Context, since time.Time) ([]DetailRecord, error) {
	q := url.Values{}
	q.Set("filter[record_type]", "call-control")

	if !since.IsZero() {
		q.Set("filter[started_at][gte]", since.UTC().Format(time.RFC3339))
	}

	return fetchAll[DetailRecord](ctx, c, "/detail_records", q, 0, detailRecordsPageSize)
}

// ConversationInsights lists the insight rows generated for one conversation
// (path verified against the OpenAPI spec: conversations-insights, not
// insights).
func (c *Client) ConversationInsights(
	ctx context.Context,
	conversationID string,
) ([]ConversationInsight, error) {
	if conversationID == "" {
		return nil, errorfamily.NewRejection(
			"telnyx.empty_conversation_id",
			"conversation id is required",
		)
	}

	path := "/ai/conversations/" + url.PathEscape(conversationID) + "/conversations-insights"

	return fetchAll[ConversationInsight](ctx, c, path, url.Values{}, 0, 0)
}

// DetailRecordByCallControl finds the detail record correlated to one call
// control id, mirroring the recordings correlation pattern: correlation is
// client-side because the endpoint's field-level filters beyond the
// documented time stamps are unverified.
func DetailRecordByCallControl(recs []DetailRecord, callControlID string) (DetailRecord, bool) {
	for _, r := range recs {
		if r.CallControlID == callControlID {
			return r, true
		}
	}

	return DetailRecord{}, false
}

// fetchAll pages a list endpoint until exhausted, capped, or the page
// shortens. pageSize 0 uses the client default (the endpoint's own maximum
// is the caller's responsibility to pass).
func fetchAll[T any](
	ctx context.Context,
	client *Client,
	path string,
	base url.Values,
	limit, pageSize int,
) ([]T, error) {
	if pageSize <= 0 {
		pageSize = client.PageSize
	}

	var all []T

	for page := 1; page <= maxPages; page++ {
		q := url.Values{}
		maps.Copy(q, base)

		q.Set("page[size]", strconv.Itoa(pageSize))
		q.Set("page[number]", strconv.Itoa(page))

		var pg struct {
			Data []T `json:"data"`
		}

		if err := client.get(ctx, path, q, &pg); err != nil {
			return nil, err
		}

		all = append(all, pg.Data...)

		if len(pg.Data) < pageSize {
			break
		}

		if limit > 0 && len(all) >= limit {
			all = all[:limit]

			break
		}
	}

	return all, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errorfamily.NewRejection(
			"telnyx.request_build",
			fmt.Sprintf("GET %s: %v", path, err),
		)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return errorfamily.NewTransient(
			"telnyx.request_failed",
			fmt.Sprintf("GET %s: %v", path, err),
		)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return errorfamily.NewCorruption("telnyx.read_body", fmt.Sprintf("GET %s: %v", path, err))
	}

	if resp.StatusCode != http.StatusOK {
		family := errorfamily.Rejection
		if resp.StatusCode >= http.StatusInternalServerError {
			family = errorfamily.Transient
		}

		return errorfamily.New(family,
			fmt.Sprintf("telnyx.http.%d", resp.StatusCode),
			fmt.Sprintf("GET %s: %s", path, truncate(string(body), maxErrorBytes)),
		)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return errorfamily.NewCorruption("telnyx.decode", fmt.Sprintf("GET %s: %v", path, err))
	}

	return nil
}

func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) > maxRunes {
		return string(r[:maxRunes])
	}

	return s
}
