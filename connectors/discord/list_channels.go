package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChannelsAction implements connectors.Action for discord.list_channels.
type listChannelsAction struct {
	conn *DiscordConnector
}

type listChannelsParams struct {
	GuildID string `json:"guild_id"`
}

type discordChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id,omitempty"`
	Topic    string `json:"topic,omitempty"`
}

type listChannelSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id,omitempty"`
	Topic    string `json:"topic,omitempty"`
}

func (a *listChannelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChannelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validateSnowflake(params.GuildID, "guild_id"); err != nil {
		return nil, err
	}

	var channels []discordChannel
	if err := a.conn.doRequest(ctx, "GET", "/guilds/"+params.GuildID+"/channels", req.Credentials, nil, &channels); err != nil {
		return nil, err
	}

	result := make([]listChannelSummary, 0, len(channels))
	for _, ch := range channels {
		result = append(result, listChannelSummary{
			ID:       ch.ID,
			Name:     ch.Name,
			Type:     ch.Type,
			Position: ch.Position,
			ParentID: ch.ParentID,
			Topic:    ch.Topic,
		})
	}

	return connectors.JSONResult(map[string]any{
		"channels": result,
	})
}
