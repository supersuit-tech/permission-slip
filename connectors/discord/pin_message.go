package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// pinMessageAction implements connectors.Action for discord.pin_message.
type pinMessageAction struct {
	conn *DiscordConnector
}

// unpinMessageAction implements connectors.Action for discord.unpin_message.
type unpinMessageAction struct {
	conn *DiscordConnector
}

type pinMessageParams struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

func (p *pinMessageParams) validate() error {
	if err := validateSnowflake(p.ChannelID, "channel_id"); err != nil {
		return err
	}
	if err := validateSnowflake(p.MessageID, "message_id"); err != nil {
		return err
	}
	return nil
}

func (a *pinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params pinMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/channels/%s/pins/%s", params.ChannelID, params.MessageID)

	// PUT /channels/{channel.id}/pins/{message.id} returns 204 No Content.
	if err := a.conn.doRequest(ctx, "PUT", path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":     "pinned",
		"message_id": params.MessageID,
		"channel_id": params.ChannelID,
	})
}

func (a *unpinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params pinMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/channels/%s/pins/%s", params.ChannelID, params.MessageID)

	// DELETE /channels/{channel.id}/pins/{message.id} returns 204 No Content.
	if err := a.conn.doRequest(ctx, "DELETE", path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":     "unpinned",
		"message_id": params.MessageID,
		"channel_id": params.ChannelID,
	})
}
