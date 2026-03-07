package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listMembersAction implements connectors.Action for discord.list_members.
type listMembersAction struct {
	conn *DiscordConnector
}

type listMembersParams struct {
	GuildID string `json:"guild_id"`
	Limit   int    `json:"limit,omitempty"`
	After   string `json:"after,omitempty"`
}

func (p *listMembersParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 1000) {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 1000, got %d", p.Limit)}
	}
	return nil
}

type discordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name,omitempty"`
}

type discordMember struct {
	User   *discordUser `json:"user,omitempty"`
	Nick   string       `json:"nick,omitempty"`
	Roles  []string     `json:"roles"`
	Joined string       `json:"joined_at"`
}

type memberSummary struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Nick     string   `json:"nick,omitempty"`
	Roles    []string `json:"roles"`
	JoinedAt string   `json:"joined_at"`
}

func (a *listMembersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listMembersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 100
	}

	path := fmt.Sprintf("/guilds/%s/members?limit=%d", params.GuildID, limit)
	if params.After != "" {
		path += "&after=" + params.After
	}

	var members []discordMember
	if err := a.conn.doRequest(ctx, "GET", path, req.Credentials, nil, &members); err != nil {
		return nil, err
	}

	result := make([]memberSummary, 0, len(members))
	for _, m := range members {
		s := memberSummary{
			Nick:     m.Nick,
			Roles:    m.Roles,
			JoinedAt: m.Joined,
		}
		if m.User != nil {
			s.UserID = m.User.ID
			s.Username = m.User.Username
		}
		result = append(result, s)
	}

	return connectors.JSONResult(map[string]any{
		"members": result,
	})
}
