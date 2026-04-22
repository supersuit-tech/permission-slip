package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// markReadAction implements slack.mark_read (conversations.mark).
type markReadAction struct {
	conn *SlackConnector
}

type markReadParams struct {
	ChannelID string `json:"channel_id"`
	TS        string `json:"ts"`
}

type markReadRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

func (p *markReadParams) validate() error {
	if err := validateChannelID(p.ChannelID); err != nil {
		return err
	}
	return validateMessageTS(p.TS)
}

func (a *markReadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params markReadParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	if err := a.conn.verifyChannelAccess(ctx, req.Credentials, params.ChannelID, req.UserEmail); err != nil {
		return nil, err
	}

	body := markReadRequest{
		Channel: params.ChannelID,
		TS:      params.TS,
	}
	var resp slackResponse
	if err := a.conn.doPost(ctx, "conversations.mark", req.Credentials, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}

	return connectors.JSONResult(map[string]any{"ok": true})
}
