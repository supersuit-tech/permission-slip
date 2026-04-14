package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// pinMessageAction implements connectors.Action for slack.pin_message.
// It pins a message via POST /pins.add.
type pinMessageAction struct {
	conn *SlackConnector
}

type pinMessageParams struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

func (p *pinMessageParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	return validateMessageTS(p.TS)
}

type pinMessageRequest struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
}

type pinMessageResponse struct {
	slackResponse
}

// Execute pins a message in a Slack channel.
func (a *pinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params pinMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := pinMessageRequest{Channel: params.Channel, Timestamp: params.TS}
	var resp pinMessageResponse
	if err := a.conn.doPost(ctx, "pins.add", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	return connectors.JSONResult(map[string]string{"channel": params.Channel, "ts": params.TS})
}
