package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// archiveChannelAction implements connectors.Action for slack.archive_channel.
// It archives a channel via POST /conversations.archive.
type archiveChannelAction struct {
	conn *SlackConnector
}

type archiveChannelParams struct {
	Channel string `json:"channel"`
}

func (p *archiveChannelParams) validate() error {
	return validateChannelID(p.Channel)
}

type archiveChannelRequest struct {
	Channel string `json:"channel"`
}

type archiveChannelResponse struct {
	slackResponse
}

// Execute archives a Slack channel.
func (a *archiveChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params archiveChannelParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := archiveChannelRequest{Channel: params.Channel}
	var resp archiveChannelResponse
	if err := a.conn.doPost(ctx, "conversations.archive", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	return connectors.JSONResult(map[string]string{"channel": params.Channel})
}
