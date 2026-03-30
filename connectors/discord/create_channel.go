package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/supersuit-tech/permission-slip/connectors"
)

var channelNameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

// createChannelAction implements connectors.Action for discord.create_channel.
// Discord API: POST /guilds/{guild.id}/channels
// See: https://discord.com/developers/docs/resources/guild#create-guild-channel
type createChannelAction struct {
	conn *DiscordConnector
}

type createChannelParams struct {
	GuildID  string `json:"guild_id"`
	Name     string `json:"name"`
	Type     int    `json:"type,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Topic    string `json:"topic,omitempty"`
}

func (p *createChannelParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if len(p.Name) < 2 || len(p.Name) > 100 {
		return &connectors.ValidationError{Message: "name must be between 2 and 100 characters"}
	}
	if !channelNameRe.MatchString(p.Name) {
		return &connectors.ValidationError{Message: "name must be lowercase alphanumeric with hyphens or underscores only (no spaces)"}
	}
	if p.ParentID != "" {
		if err := validateSnowflake(p.ParentID, "parent_id"); err != nil {
			return err
		}
	}
	if len(p.Topic) > 1024 {
		return &connectors.ValidationError{Message: "topic must be 1024 characters or fewer"}
	}
	return nil
}

type createChannelRequest struct {
	Name     string `json:"name"`
	Type     int    `json:"type"`
	ParentID string `json:"parent_id,omitempty"`
	Topic    string `json:"topic,omitempty"`
}

type createChannelResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

func (a *createChannelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createChannelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := createChannelRequest{
		Name:     params.Name,
		Type:     params.Type,
		ParentID: params.ParentID,
		Topic:    params.Topic,
	}

	var resp createChannelResponse
	if err := a.conn.doRequest(ctx, "POST", "/guilds/"+params.GuildID+"/channels", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"id":   resp.ID,
		"name": resp.Name,
		"type": resp.Type,
	})
}
