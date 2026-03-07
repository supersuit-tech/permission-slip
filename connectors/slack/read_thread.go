package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// readThreadAction implements connectors.Action for slack.read_thread.
// It fetches replies in a thread via POST /conversations.replies.
type readThreadAction struct {
	conn *SlackConnector
}

// readThreadParams is the user-facing parameter schema.
type readThreadParams struct {
	Channel  string `json:"channel"`
	ThreadTS string `json:"thread_ts"`
	// Limit is the max number of replies to return (1-1000, default 50).
	Limit int `json:"limit,omitempty"`
	// Cursor is a pagination cursor from a previous response.
	Cursor string `json:"cursor,omitempty"`
}

func (p *readThreadParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.ThreadTS == "" {
		return &connectors.ValidationError{Message: "missing required parameter: thread_ts"}
	}
	if err := validateLimit(p.Limit); err != nil {
		return err
	}
	return nil
}

// readThreadRequest is the Slack API request body for conversations.replies.
type readThreadRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Limit   int    `json:"limit,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

// Execute fetches replies in a Slack thread.
func (a *readThreadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readThreadParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := readThreadRequest{
		Channel: params.Channel,
		TS:      params.ThreadTS,
		Limit:   params.Limit,
		Cursor:  params.Cursor,
	}
	if body.Limit == 0 {
		body.Limit = 50
	}

	var resp messagesResponse
	if err := a.conn.doPost(ctx, "conversations.replies", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(toMessagesResult(&resp))
}
