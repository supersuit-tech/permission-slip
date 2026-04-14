package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// removeFromChannelAction implements connectors.Action for slack.remove_from_channel.
// It removes a user from a channel via POST /conversations.kick.
type removeFromChannelAction struct {
	conn *SlackConnector
}

type removeFromChannelParams struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
}

func (p *removeFromChannelParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	return validateUserID(p.User)
}

type removeFromChannelRequest struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
}

type removeFromChannelResponse struct {
	slackResponse
}

// Execute removes a user from a Slack channel.
func (a *removeFromChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params removeFromChannelParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := removeFromChannelRequest{Channel: params.Channel, User: params.User}
	var resp removeFromChannelResponse
	if err := a.conn.doPost(ctx, "conversations.kick", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	return connectors.JSONResult(map[string]string{
		"channel": params.Channel,
		"user":    params.User,
	})
}
