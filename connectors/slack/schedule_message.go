package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// scheduleMessageAction implements connectors.Action for slack.schedule_message.
// It schedules a message for future delivery via POST /chat.scheduleMessage.
type scheduleMessageAction struct {
	conn *SlackConnector
}

// scheduleMessageParams is the user-facing parameter schema.
type scheduleMessageParams struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
	PostAt  int64  `json:"post_at"`
}

func (p *scheduleMessageParams) validate() error {
	if p.Channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.PostAt <= 0 {
		return &connectors.ValidationError{Message: "missing required parameter: post_at (must be a future Unix timestamp)"}
	}
	return nil
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
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := scheduleMessageRequest{
		Channel: params.Channel,
		Text:    params.Message,
		PostAt:  params.PostAt,
	}

	var resp scheduleMessageResponse
	if err := a.conn.doPost(ctx, "chat.scheduleMessage", req.Credentials, body, &resp); err != nil {
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
