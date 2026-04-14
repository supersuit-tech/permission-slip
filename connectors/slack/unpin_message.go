package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// unpinMessageAction implements connectors.Action for slack.unpin_message.
// It unpins a message via POST /pins.remove.
type unpinMessageAction struct {
	conn *SlackConnector
}

type unpinMessageParams struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

func (p *unpinMessageParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	return validateMessageTS(p.TS)
}

type unpinMessageRequest struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
}

type unpinMessageResponse struct {
	slackResponse
}

// Execute unpins a message in a Slack channel.
func (a *unpinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params unpinMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := unpinMessageRequest{Channel: params.Channel, Timestamp: params.TS}
	var resp unpinMessageResponse
	if err := a.conn.doPost(ctx, "pins.remove", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	return connectors.JSONResult(map[string]string{"channel": params.Channel, "ts": params.TS})
}
