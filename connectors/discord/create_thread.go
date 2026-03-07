package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createThreadAction implements connectors.Action for discord.create_thread.
type createThreadAction struct {
	conn *DiscordConnector
}

type createThreadParams struct {
	ChannelID           string `json:"channel_id"`
	Name                string `json:"name"`
	MessageID           string `json:"message_id,omitempty"`
	AutoArchiveDuration int    `json:"auto_archive_duration,omitempty"`
}

func (p *createThreadParams) validate() error {
	if err := validateSnowflake(p.ChannelID, "channel_id"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if len(p.Name) > 100 {
		return &connectors.ValidationError{Message: "name must be 100 characters or fewer"}
	}
	if p.MessageID != "" {
		if err := validateSnowflake(p.MessageID, "message_id"); err != nil {
			return err
		}
	}
	validDurations := map[int]bool{0: true, 60: true, 1440: true, 4320: true, 10080: true}
	if !validDurations[p.AutoArchiveDuration] {
		return &connectors.ValidationError{Message: "auto_archive_duration must be one of 0 (use default), 60, 1440, 4320, or 10080"}
	}
	return nil
}

type createThreadRequest struct {
	Name                string `json:"name"`
	AutoArchiveDuration int    `json:"auto_archive_duration,omitempty"`
	Type                int    `json:"type,omitempty"` // 11 = public thread, 12 = private thread
}

type createThreadResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a *createThreadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createThreadParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	archiveDuration := params.AutoArchiveDuration
	if archiveDuration == 0 {
		archiveDuration = 1440 // 24 hours
	}

	var path string
	var body any

	if params.MessageID != "" {
		// Start thread from an existing message.
		path = fmt.Sprintf("/channels/%s/messages/%s/threads", params.ChannelID, params.MessageID)
		body = createThreadRequest{
			Name:                params.Name,
			AutoArchiveDuration: archiveDuration,
		}
	} else {
		// Start thread without a message (forum-style).
		path = fmt.Sprintf("/channels/%s/threads", params.ChannelID)
		body = createThreadRequest{
			Name:                params.Name,
			AutoArchiveDuration: archiveDuration,
			Type:                11, // public thread
		}
	}

	var resp createThreadResponse
	if err := a.conn.doRequest(ctx, "POST", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.ID,
		"name": resp.Name,
	})
}
