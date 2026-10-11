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
	DefaultBase   = "https://api.telnyx.com/v2"
	maxPages      = 500
	maxErrorBytes = 300
	maxBodyBytes  = 8 << 20
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

	return &Client{base: base, apiKey: apiKey, http: http.DefaultClient, PageSize: 100}
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

	return fetchAll[Conversation](ctx, c, "/ai/conversations", q, opts.Limit)
}

func (c *Client) Messages(ctx context.Context, conversationID string) ([]Message, error) {
	if conversationID == "" {
		return nil, errorfamily.NewRejection("telnyx.empty_conversation_id", "conversation id is required")
	}

	return fetchAll[Message](ctx, c, "/ai/conversations/"+url.PathEscape(conversationID)+"/messages", url.Values{}, 0)
}

func (c *Client) Recording(ctx context.Context, recordingID string) (Recording, error) {
	if recordingID == "" {
		return Recording{}, errorfamily.NewRejection("telnyx.empty_recording_id", "recording id is required")
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
	return fetchAll[Recording](ctx, c, "/recordings", url.Values{}, limit)
}

func fetchAll[T any](ctx context.Context, c *Client, path string, base url.Values, limit int) ([]T, error) {
	var all []T

	for page := 1; page <= maxPages; page++ {
		q := url.Values{}
		maps.Copy(q, base)

		q.Set("page[size]", strconv.Itoa(c.PageSize))
		q.Set("page[number]", strconv.Itoa(page))

		var pg struct {
			Data []T `json:"data"`
		}

		if err := c.get(ctx, path, q, &pg); err != nil {
			return nil, err
		}

		all = append(all, pg.Data...)

		if len(pg.Data) < c.PageSize {
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
		return errorfamily.NewRejection("telnyx.request_build", fmt.Sprintf("GET %s: %v", path, err))
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return errorfamily.NewTransient("telnyx.request_failed", fmt.Sprintf("GET %s: %v", path, err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return errorfamily.NewCorruption("telnyx.read_body", fmt.Sprintf("GET %s: %v", path, err))
	}

	if resp.StatusCode != http.StatusOK {
		family := errorfamily.Rejection
		if resp.StatusCode >= 500 {
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

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}

	return s
}
