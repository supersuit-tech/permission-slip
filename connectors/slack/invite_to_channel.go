package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// inviteToChannelAction implements connectors.Action for slack.invite_to_channel.
// It invites user(s) to a channel via POST /conversations.invite.
type inviteToChannelAction struct {
	conn *SlackConnector
}

// inviteToChannelParams is the user-facing parameter schema.
type inviteToChannelParams struct {
	Channel string `json:"channel"`
	Users   string `json:"users"`
}

func (p *inviteToChannelParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.Users == "" {
		return &connectors.ValidationError{Message: "missing required parameter: users"}
	}
	return nil
}

// inviteToChannelRequest is the Slack API request body for conversations.invite.
type inviteToChannelRequest struct {
	Channel string `json:"channel"`
	Users   string `json:"users"`
}

type inviteToChannelResponse struct {
	slackResponse
	Channel *inviteChannelInfo `json:"channel,omitempty"`
}

type inviteChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Execute invites user(s) to a Slack channel and returns the channel metadata.
func (a *inviteToChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params inviteToChannelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Normalize: trim spaces around comma-separated user IDs.
	users := strings.ReplaceAll(params.Users, " ", "")

	body := inviteToChannelRequest{
		Channel: params.Channel,
		Users:   users,
	}

	var resp inviteToChannelResponse
	if err := a.conn.doPost(ctx, "conversations.invite", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	result := map[string]string{
		"channel": params.Channel,
	}
	if resp.Channel != nil {
		result["channel_name"] = resp.Channel.Name
	}

	return connectors.JSONResult(result)
}
