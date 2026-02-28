package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createChannelAction implements connectors.Action for slack.create_channel.
// It creates a new channel via POST /conversations.create.
type createChannelAction struct {
	conn *SlackConnector
}

// createChannelParams maps 1:1 to the Slack API request body for
// conversations.create, so a single type serves both roles.
type createChannelParams struct {
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
}

func (p *createChannelParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

type createChannelResponse struct {
	slackResponse
	Channel *channelInfo `json:"channel,omitempty"`
}

type channelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Execute creates a Slack channel and returns the channel metadata.
func (a *createChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createChannelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp createChannelResponse
	if err := a.conn.doPost(ctx, "conversations.create", req.Credentials, params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	if resp.Channel == nil {
		return nil, &connectors.ExternalError{
			StatusCode: 200,
			Message:    "Slack API returned ok=true but no channel data",
		}
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.Channel.ID,
		"name": resp.Channel.Name,
	})
}
