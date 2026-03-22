package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// scheduleMessageAction implements connectors.Action for slack.schedule_message.
// It schedules a message for future delivery via POST /chat.scheduleMessage.
type scheduleMessageAction struct {
	conn *SlackConnector
}

// scheduleMessageParams is the user-facing parameter schema.
type scheduleMessageParams struct {
	Channel string       `json:"channel"`
	Message string       `json:"message"`
	PostAt  flexDateTime `json:"post_at"`
}

// flexDateTime accepts both a string (RFC 3339, datetime-local) and a legacy
// integer (Unix timestamp) from JSON, converting either to a string. This
// ensures backward compatibility with existing stored approvals that have
// integer post_at values.
type flexDateTime string

func (f *flexDateTime) UnmarshalJSON(data []byte) error {
	// Handle null — leave the field empty for validate() to catch.
	if string(data) == "null" {
		*f = ""
		return nil
	}
	// Try string first (new format: RFC 3339, datetime-local, or numeric string)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexDateTime(s)
		return nil
	}
	// Try integer (legacy stored approvals with Unix timestamp)
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexDateTime(time.Unix(n, 0).UTC().Format(time.RFC3339))
		return nil
	}
	// Try float (JSON numbers can be decoded as floats)
	var fl float64
	if err := json.Unmarshal(data, &fl); err == nil {
		*f = flexDateTime(time.Unix(int64(fl), 0).UTC().Format(time.RFC3339))
		return nil
	}
	return &connectors.ValidationError{
		Message: fmt.Sprintf("post_at must be a datetime string or Unix timestamp, got %s", string(data)),
	}
}

func (p *scheduleMessageParams) validate() error {
	if p.Channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.PostAt == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_at"}
	}
	return nil
}

// postAtUnix parses the post_at field and returns the Unix timestamp.
// Accepts RFC 3339 (with optional fractional seconds) and datetime-local
// formats (with or without seconds, no timezone — treated as UTC).
// Returns a ValidationError if the value is unparseable or in the past.
func (p *scheduleMessageParams) postAtUnix() (int64, error) {
	postAt := string(p.PostAt)

	// If the value is purely numeric, it's already a Unix timestamp (legacy path).
	if n, err := strconv.ParseInt(postAt, 10, 64); err == nil {
		if n <= time.Now().Unix() {
			return 0, &connectors.ValidationError{
				Message: fmt.Sprintf("post_at must be in the future (got %s)", postAt),
			}
		}
		return n, nil
	}

	// Try RFC 3339 / RFC 3339 Nano (e.g. "2026-03-20T09:00:00Z" or "…000Z").
	t, err := time.Parse(time.RFC3339Nano, postAt)
	if err != nil {
		// Fall back to datetime-local formats (no TZ) emitted by HTML
		// <input type="datetime-local">, treating them as UTC.
		for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
			if t2, err2 := time.Parse(layout, postAt); err2 == nil {
				t = t2.UTC()
				err = nil
				break
			}
		}
		if err != nil {
			return 0, &connectors.ValidationError{
				Message: fmt.Sprintf("post_at must be a valid RFC 3339 datetime (e.g. 2026-03-20T09:00:00Z), got %q", postAt),
			}
		}
	}
	unix := t.Unix()
	if unix <= time.Now().Unix() {
		return 0, &connectors.ValidationError{
			Message: fmt.Sprintf("post_at must be in the future (got %s)", postAt),
		}
	}
	return unix, nil
}

// scheduleMessageRequest is the Slack API request body for chat.scheduleMessage.
type scheduleMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
	PostAt  int64  `json:"post_at"`
}

type scheduleMessageResponse struct {
	slackResponse
	ScheduledMessageID string `json:"scheduled_message_id,omitempty"`
	PostAt             int64  `json:"post_at,omitempty"`
	Channel            string `json:"channel,omitempty"`
}

// Execute schedules a message for future delivery and returns the scheduled message metadata.
func (a *scheduleMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params scheduleMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	postAtUnix, err := params.postAtUnix()
	if err != nil {
		return nil, err
	}

	body := scheduleMessageRequest{
		Channel: params.Channel,
		Text:    params.Message,
		PostAt:  postAtUnix,
	}

	creds := credentialsForChat(req.Credentials)
	var resp scheduleMessageResponse
	if err := a.conn.doPost(ctx, "chat.scheduleMessage", creds, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(map[string]any{
		"scheduled_message_id": resp.ScheduledMessageID,
		"post_at":              resp.PostAt,
		"channel":              resp.Channel,
	})
}
