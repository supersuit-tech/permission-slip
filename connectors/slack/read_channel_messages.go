package slack

import (
	"context"

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

// Execute fetches recent messages from a Slack channel.
func (a *readChannelMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readChannelMessagesParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	// Verify the Permission Slip user has access to this channel.
	if err := a.conn.verifyChannelAccess(ctx, req.Credentials, params.Channel, req.UserEmail); err != nil {
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

	var resp messagesResponse
	if err := a.conn.doPost(ctx, "conversations.history", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(toMessagesResult(&resp))
}
