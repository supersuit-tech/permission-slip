package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteMessageAction implements connectors.Action for slack.delete_message.
// It deletes a message from a channel via POST /chat.delete.
type deleteMessageAction struct {
	conn *SlackConnector
}

// deleteMessageParams is the user-facing parameter schema.
type deleteMessageParams struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

func (p *deleteMessageParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if err := validateMessageTS(p.TS); err != nil {
		return err
	}
	return nil
}

// deleteMessageRequest is the Slack API request body for chat.delete.
type deleteMessageRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

type deleteMessageResponse struct {
	slackResponse
	TS      string `json:"ts,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// Execute deletes a message from a Slack channel.
func (a *deleteMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := deleteMessageRequest{
		Channel: params.Channel,
		TS:      params.TS,
	}

	var resp deleteMessageResponse
	if err := a.conn.doPost(ctx, "chat.delete", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, resp.asError()
	}

	return connectors.JSONResult(map[string]string{
		"ts":      resp.TS,
		"channel": resp.Channel,
	})
}
