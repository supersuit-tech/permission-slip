package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendChannelMessageAction implements connectors.Action for microsoft.send_channel_message.
// It posts a message to a Teams channel via POST /teams/{team-id}/channels/{channel-id}/messages.
type sendChannelMessageAction struct {
	conn *MicrosoftConnector
}

// sendChannelMessageParams is the user-facing parameter schema.
type sendChannelMessageParams struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

func (p *sendChannelMessageParams) validate() error {
	if p.TeamID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: team_id"}
	}
	if p.ChannelID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel_id"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

// graphChannelMessageRequest is the Microsoft Graph API request body for posting a channel message.
type graphChannelMessageRequest struct {
	Body graphChannelMessageBody `json:"body"`
}

type graphChannelMessageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// graphChannelMessageResponse is the Microsoft Graph API response for a posted message.
type graphChannelMessageResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdDateTime"`
}

// Execute posts a message to a Teams channel.
func (a *sendChannelMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendChannelMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	graphReq := graphChannelMessageRequest{
		Body: graphChannelMessageBody{
			ContentType: detectContentType(params.Message),
			Content:     params.Message,
		},
	}

	path := fmt.Sprintf("/teams/%s/channels/%s/messages", params.TeamID, params.ChannelID)

	var resp graphChannelMessageResponse
	if err := a.conn.doRequest(ctx, http.MethodPost, path, req.Credentials, graphReq, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":     "sent",
		"message_id": resp.ID,
		"created_at": resp.CreatedAt,
	})
}
