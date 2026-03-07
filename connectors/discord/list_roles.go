package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listRolesAction implements connectors.Action for discord.list_roles.
// Discord API: GET /guilds/{guild.id}/roles
// See: https://discord.com/developers/docs/resources/guild#get-guild-roles
type listRolesAction struct {
	conn *DiscordConnector
}

type listRolesParams struct {
	GuildID string `json:"guild_id"`
}

type discordRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

type roleSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Position    int    `json:"position"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

func (a *listRolesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listRolesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validateSnowflake(params.GuildID, "guild_id"); err != nil {
		return nil, err
	}

	var roles []discordRole
	if err := a.conn.doRequest(ctx, "GET", "/guilds/"+params.GuildID+"/roles", req.Credentials, nil, &roles); err != nil {
		return nil, err
	}

	result := make([]roleSummary, 0, len(roles))
	for _, r := range roles {
		result = append(result, roleSummary{
			ID:          r.ID,
			Name:        r.Name,
			Color:       r.Color,
			Position:    r.Position,
			Managed:     r.Managed,
			Mentionable: r.Mentionable,
		})
	}

	return connectors.JSONResult(map[string]any{
		"roles": result,
	})
}
