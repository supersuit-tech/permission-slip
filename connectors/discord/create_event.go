package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createEventAction implements connectors.Action for discord.create_event.
// Discord API: POST /guilds/{guild.id}/scheduled-events
// See: https://discord.com/developers/docs/resources/guild-scheduled-event#create-guild-scheduled-event
type createEventAction struct {
	conn *DiscordConnector
}

type createEventParams struct {
	GuildID            string          `json:"guild_id"`
	Name               string          `json:"name"`
	Description        string          `json:"description,omitempty"`
	ScheduledStartTime string          `json:"scheduled_start_time"`
	ScheduledEndTime   string          `json:"scheduled_end_time,omitempty"`
	PrivacyLevel       int             `json:"privacy_level,omitempty"`
	EntityType         int             `json:"entity_type"`
	ChannelID          string          `json:"channel_id,omitempty"`
	EntityMetadata     json.RawMessage `json:"entity_metadata,omitempty"`
}

func (p *createEventParams) validate() error {
	if err := validateSnowflake(p.GuildID, "guild_id"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.ScheduledStartTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: scheduled_start_time"}
	}
	if p.EntityType < 1 || p.EntityType > 3 {
		return &connectors.ValidationError{Message: "entity_type must be 1 (stage), 2 (voice), or 3 (external)"}
	}
	if p.ChannelID != "" {
		if err := validateSnowflake(p.ChannelID, "channel_id"); err != nil {
			return err
		}
	}
	return nil
}

type createEventRequest struct {
	Name               string          `json:"name"`
	Description        string          `json:"description,omitempty"`
	ScheduledStartTime string          `json:"scheduled_start_time"`
	ScheduledEndTime   string          `json:"scheduled_end_time,omitempty"`
	PrivacyLevel       int             `json:"privacy_level"`
	EntityType         int             `json:"entity_type"`
	ChannelID          string          `json:"channel_id,omitempty"`
	EntityMetadata     json.RawMessage `json:"entity_metadata,omitempty"`
}

type createEventResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a *createEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	privacyLevel := params.PrivacyLevel
	if privacyLevel == 0 {
		privacyLevel = 2 // guild only
	}

	body := createEventRequest{
		Name:               params.Name,
		Description:        params.Description,
		ScheduledStartTime: params.ScheduledStartTime,
		ScheduledEndTime:   params.ScheduledEndTime,
		PrivacyLevel:       privacyLevel,
		EntityType:         params.EntityType,
		ChannelID:          params.ChannelID,
		EntityMetadata:     params.EntityMetadata,
	}

	var resp createEventResponse
	if err := a.conn.doRequest(ctx, "POST", "/guilds/"+params.GuildID+"/scheduled-events", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.ID,
		"name": resp.Name,
	})
}
