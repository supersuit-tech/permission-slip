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
	// ThreadTS posts in a thread (chat.postMessage thread_ts). Optional.
	ThreadTS string `json:"thread_ts,omitempty"`
	// InResponseToTS anchors approval context only; not sent to Slack (issue #981).
	InResponseToTS string `json:"in_response_to_ts,omitempty"`
}

func (p *sendMessageParams) validate() error {
	if p.Channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.ThreadTS != "" {
		if err := validateMessageTS(p.ThreadTS); err != nil {
			return err
		}
	}
	if p.InResponseToTS != "" {
		if err := validateMessageTS(p.InResponseToTS); err != nil {
			return err
		}
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

	var resp sendMessageResponse
	var postErr error
	if params.ThreadTS != "" {
		postErr = a.conn.doPost(ctx, "chat.postMessage", req.Credentials, struct {
			Channel  string `json:"channel"`
			Text     string `json:"text"`
			ThreadTS string `json:"thread_ts,omitempty"`
		}{
			Channel:  params.Channel,
			Text:     params.Message,
			ThreadTS: params.ThreadTS,
		}, &resp)
	} else {
		postErr = a.conn.doPost(ctx, "chat.postMessage", req.Credentials, sendMessageRequest{
			Channel: params.Channel,
			Text:    params.Message,
		}, &resp)
	}
	if postErr != nil {
		return nil, postErr
	}

	if !resp.OK {
		return nil, resp.asError()
	}

	return connectors.JSONResult(map[string]string{
		"ts":      resp.TS,
		"channel": resp.Channel,
	})
}
