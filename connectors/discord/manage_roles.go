package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// manageRolesAction implements connectors.Action for discord.manage_roles.
// Discord API: PUT/DELETE /guilds/{guild.id}/members/{user.id}/roles/{role.id}
// See: https://discord.com/developers/docs/resources/guild#add-guild-member-role
type manageRolesAction struct {
	conn *DiscordConnector
}

type manageRolesParams struct {
	GuildID string `json:"guild_id"`
	UserID  string `json:"user_id"`
	RoleID  string `json:"role_id"`
	Action  string `json:"action"` // "assign" or "remove"
}

func (p *manageRolesParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if err := validateSnowflake(p.UserID, "user_id"); err != nil {
		return err
	}
	if err := validateSnowflake(p.RoleID, "role_id"); err != nil {
		return err
	}
	if p.Action != "assign" && p.Action != "remove" {
		return &connectors.ValidationError{Message: "action must be \"assign\" or \"remove\""}
	}
	return nil
}

func (a *manageRolesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params manageRolesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/guilds/%s/members/%s/roles/%s", params.GuildID, params.UserID, params.RoleID)

	method := http.MethodPut
	if params.Action == "remove" {
		method = http.MethodDelete
	}

	// These endpoints return 204 No Content on success.
	if err := a.conn.doRequest(ctx, method, path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":  "success",
		"action":  params.Action,
		"user_id": params.UserID,
		"role_id": params.RoleID,
	})
}
