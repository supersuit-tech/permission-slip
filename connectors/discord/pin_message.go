package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// executePinUnpin is the shared implementation for pin and unpin actions.
func executePinUnpin(ctx context.Context, conn *DiscordConnector, req connectors.ActionRequest, method, status string) (*connectors.ActionResult, error) {
	var params pinMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/channels/%s/pins/%s", params.ChannelID, params.MessageID)

	if err := conn.doRequest(ctx, method, path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":     status,
		"message_id": params.MessageID,
		"channel_id": params.ChannelID,
	})
}

func (a *pinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePinUnpin(ctx, a.conn, req, http.MethodPut, "pinned")
}

func (a *unpinMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePinUnpin(ctx, a.conn, req, http.MethodDelete, "unpinned")
}
