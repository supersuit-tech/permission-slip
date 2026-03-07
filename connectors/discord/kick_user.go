package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// kickUserAction implements connectors.Action for discord.kick_user.
// Discord API: DELETE /guilds/{guild.id}/members/{user.id}
// See: https://discord.com/developers/docs/resources/guild#remove-guild-member
type kickUserAction struct {
	conn *DiscordConnector
}

type kickUserParams struct {
	GuildID string `json:"guild_id"`
	UserID  string `json:"user_id"`
}

func (p *kickUserParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if err := validateSnowflake(p.UserID, "user_id"); err != nil {
		return err
	}
	return nil
}

func (a *kickUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params kickUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/guilds/%s/members/%s", params.GuildID, params.UserID)

	// DELETE /guilds/{guild.id}/members/{user.id} returns 204 No Content.
	if err := a.conn.doRequest(ctx, "DELETE", path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":  "kicked",
		"user_id": params.UserID,
	})
}
