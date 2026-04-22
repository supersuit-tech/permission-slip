package slack

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// readChannelMessagesAction implements connectors.Action for slack.read_channel_messages.
// It fetches message history from a channel via POST /conversations.history.
type readChannelMessagesAction struct {
	conn *SlackConnector
}

// readChannelMessagesParams is the user-facing parameter schema.
type readChannelMessagesParams struct {
	Channel string `json:"channel"`
	// Limit is the max number of messages to return (1-1000, default 20).
	Limit int `json:"limit,omitempty"`
	// Oldest accepts an RFC 3339 datetime or a Unix timestamp string.
	Oldest string `json:"oldest,omitempty"`
	// Latest accepts an RFC 3339 datetime or a Unix timestamp string.
	Latest string `json:"latest,omitempty"`
	// Cursor is a pagination cursor from a previous response.
	Cursor string `json:"cursor,omitempty"`
}

func (p *readChannelMessagesParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if err := validateLimit(p.Limit); err != nil {
		return err
	}
	return nil
}

// readChannelMessagesRequest is the Slack API request body for conversations.history.
type readChannelMessagesRequest struct {
	Channel string `json:"channel"`
	Limit   int    `json:"limit,omitempty"`
	Oldest  string `json:"oldest,omitempty"`
	Latest  string `json:"latest,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

// Execute fetches recent messages from a Slack channel.
func (a *readChannelMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readChannelMessagesParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	oldest, err := toSlackTimestamp(params.Oldest)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid oldest: %s", err)}
	}
	latest, err := toSlackTimestamp(params.Latest)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid latest: %s", err)}
	}

	body := readChannelMessagesRequest{
		Channel: params.Channel,
		Limit:   params.Limit,
		Oldest:  oldest,
		Latest:  latest,
		Cursor:  params.Cursor,
	}
	if body.Limit == 0 {
		body.Limit = 20
	}

	var resp messagesResponse
	if err := a.conn.doPost(ctx, "conversations.history", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, resp.asError()
	}

	return connectors.JSONResult(toMessagesResult(&resp))
}

// toSlackTimestamp converts a timestamp value to the format Slack expects
// (Unix seconds as a string, e.g. "1711234567.000000"). It accepts:
//   - empty string → returned as-is
//   - a numeric Unix timestamp string (e.g. "1711234567" or "1711234567.123456")
//   - an RFC 3339 datetime string (e.g. "2026-03-20T09:00:00-04:00")
func toSlackTimestamp(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	// If it's already a numeric timestamp, pass through.
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value, nil
	}
	// Try parsing as RFC 3339 (with optional fractional seconds).
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", fmt.Errorf("expected a date/time or Unix timestamp, got %q", value)
	}
	return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000), nil
}
