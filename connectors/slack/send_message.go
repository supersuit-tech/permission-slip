package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendMessageAction implements connectors.Action for slack.send_message.
// It sends a message to a Slack channel via POST /chat.postMessage.
type sendMessageAction struct {
	conn *SlackConnector
}

// sendMessageParams is the user-facing parameter schema.
// Uses "message" (user-friendly) rather than Slack's "text" field name.
type sendMessageParams struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

func (p *sendMessageParams) validate() error {
	if p.Channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

// sendMessageRequest is the Slack API request body for chat.postMessage.
type sendMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type sendMessageResponse struct {
	slackResponse
	TS      string `json:"ts,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// Execute sends a message to a Slack channel and returns the message metadata.
func (a *sendMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := sendMessageRequest{
		Channel: params.Channel,
		Text:    params.Message,
	}

	var resp sendMessageResponse
	if err := a.conn.doPost(ctx, "chat.postMessage", req.Credentials, body, &resp); err != nil {
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
