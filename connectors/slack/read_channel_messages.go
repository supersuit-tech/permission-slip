package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	// Oldest is a Unix timestamp; only messages after this time are returned.
	Oldest string `json:"oldest,omitempty"`
	// Latest is a Unix timestamp; only messages before this time are returned.
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

type readChannelMessagesResponse struct {
	slackResponse
	Messages []slackMessage                 `json:"messages,omitempty"`
	HasMore  bool                           `json:"has_more,omitempty"`
	Meta     *paginationMeta   `json:"response_metadata,omitempty"`
}

type slackMessage struct {
	Type      string `json:"type"`
	User      string `json:"user,omitempty"`
	BotID     string `json:"bot_id,omitempty"`
	Text      string `json:"text"`
	TS        string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	ReplyCount int   `json:"reply_count,omitempty"`
}

// readChannelMessagesResult is the action output.
type readChannelMessagesResult struct {
	Messages   []messageSummary `json:"messages"`
	HasMore    bool             `json:"has_more"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type messageSummary struct {
	User       string `json:"user,omitempty"`
	BotID      string `json:"bot_id,omitempty"`
	Text       string `json:"text"`
	TS         string `json:"ts"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
}

// Execute fetches recent messages from a Slack channel.
func (a *readChannelMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readChannelMessagesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := readChannelMessagesRequest{
		Channel: params.Channel,
		Limit:   params.Limit,
		Oldest:  params.Oldest,
		Latest:  params.Latest,
		Cursor:  params.Cursor,
	}
	if body.Limit == 0 {
		body.Limit = 20
	}

	var resp readChannelMessagesResponse
	if err := a.conn.doPost(ctx, "conversations.history", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	result := readChannelMessagesResult{
		Messages: make([]messageSummary, 0, len(resp.Messages)),
		HasMore:  resp.HasMore,
	}
	for _, msg := range resp.Messages {
		result.Messages = append(result.Messages, messageSummary{
			User:       msg.User,
			BotID:      msg.BotID,
			Text:       msg.Text,
			TS:         msg.TS,
			ThreadTS:   msg.ThreadTS,
			ReplyCount: msg.ReplyCount,
		})
	}
	if resp.Meta != nil {
		result.NextCursor = resp.Meta.NextCursor
	}

	return connectors.JSONResult(result)
}
