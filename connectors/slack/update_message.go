package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateMessageAction implements connectors.Action for slack.update_message.
// It edits an existing message via POST /chat.update.
type updateMessageAction struct {
	conn *SlackConnector
}

// updateMessageParams is the user-facing parameter schema.
type updateMessageParams struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Message string `json:"message"`
}

func (p *updateMessageParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if err := validateMessageTS(p.TS); err != nil {
		return err
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

// updateMessageRequest is the Slack API request body for chat.update.
type updateMessageRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Text    string `json:"text"`
}

type updateMessageResponse struct {
	slackResponse
	TS      string `json:"ts,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// Execute updates an existing Slack message and returns the updated metadata.
func (a *updateMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := updateMessageRequest{
		Channel: params.Channel,
		TS:      params.TS,
		Text:    params.Message,
	}

	creds := credentialsForChat(req.Credentials)
	var resp updateMessageResponse
	if err := a.conn.doPost(ctx, "chat.update", creds, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(map[string]string{
		"ts":      resp.TS,
		"channel": resp.Channel,
	})
}
