package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendMessageAction implements connectors.Action for discord.send_message.
// Discord API: POST /channels/{channel.id}/messages
// See: https://discord.com/developers/docs/resources/message#create-message
type sendMessageAction struct {
	conn *DiscordConnector
}

type sendMessageParams struct {
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
}

func (p *sendMessageParams) validate() error {
	if err := validateSnowflake(p.ChannelID, "channel_id"); err != nil {
		return err
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if len(p.Content) > 2000 {
		return &connectors.ValidationError{Message: "content must be 2000 characters or fewer"}
	}
	return nil
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type sendMessageResponse struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
}

func (a *sendMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := sendMessageRequest{Content: params.Content}
	var resp sendMessageResponse
	if err := a.conn.doRequest(ctx, "POST", "/channels/"+params.ChannelID+"/messages", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":         resp.ID,
		"channel_id": resp.ChannelID,
	})
}
