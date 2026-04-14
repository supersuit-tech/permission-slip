package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// renameChannelAction implements connectors.Action for slack.rename_channel.
// It renames a channel via POST /conversations.rename.
type renameChannelAction struct {
	conn *SlackConnector
}

type renameChannelParams struct {
	Channel string `json:"channel"`
	Name    string `json:"name"`
}

func (p *renameChannelParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

type renameChannelRequest struct {
	Channel string `json:"channel"`
	Name    string `json:"name"`
}

type renameChannelResponse struct {
	slackResponse
	Channel *channelInfo `json:"channel,omitempty"`
}

// Execute renames a Slack channel.
func (a *renameChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params renameChannelParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := renameChannelRequest{Channel: params.Channel, Name: params.Name}
	var resp renameChannelResponse
	if err := a.conn.doPost(ctx, "conversations.rename", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	out := map[string]string{"channel": params.Channel, "name": params.Name}
	if resp.Channel != nil {
		out["id"] = resp.Channel.ID
		out["new_name"] = resp.Channel.Name
	}
	return connectors.JSONResult(out)
}
