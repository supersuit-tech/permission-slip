package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// banUserAction implements connectors.Action for discord.ban_user.
// Discord API: PUT /guilds/{guild.id}/bans/{user.id}
// See: https://discord.com/developers/docs/resources/guild#create-guild-ban
type banUserAction struct {
	conn *DiscordConnector
}

type banUserParams struct {
	GuildID              string `json:"guild_id"`
	UserID               string `json:"user_id"`
	DeleteMessageSeconds int    `json:"delete_message_seconds,omitempty"`
}

func (p *banUserParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if err := validateSnowflake(p.UserID, "user_id"); err != nil {
		return err
	}
	if p.DeleteMessageSeconds < 0 || p.DeleteMessageSeconds > 604800 {
		return &connectors.ValidationError{Message: "delete_message_seconds must be between 0 and 604800"}
	}
	return nil
}

type banUserRequest struct {
	DeleteMessageSeconds int `json:"delete_message_seconds,omitempty"`
}

func (a *banUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params banUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/guilds/%s/bans/%s", params.GuildID, params.UserID)
	body := banUserRequest{DeleteMessageSeconds: params.DeleteMessageSeconds}

	// PUT /guilds/{guild.id}/bans/{user.id} returns 204 No Content.
	if err := a.conn.doRequest(ctx, "PUT", path, req.Credentials, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":  "banned",
		"user_id": params.UserID,
	})
}
