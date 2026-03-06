package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChannelMessagesAction implements connectors.Action for microsoft.list_channel_messages.
// It lists recent messages from a Teams channel via GET /teams/{team-id}/channels/{channel-id}/messages.
type listChannelMessagesAction struct {
	conn *MicrosoftConnector
}

// listChannelMessagesParams is the user-facing parameter schema.
type listChannelMessagesParams struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	Top       int    `json:"top"`
}

func (p *listChannelMessagesParams) validate() error {
	if err := validateGraphID("team_id", p.TeamID); err != nil {
		return err
	}
	if err := validateGraphID("channel_id", p.ChannelID); err != nil {
		return err
	}
	return nil
}

func (p *listChannelMessagesParams) defaults() {
	if p.Top <= 0 {
		p.Top = 20
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

// graphChannelMessagesResponse is the Microsoft Graph API response for listing channel messages.
type graphChannelMessagesResponse struct {
	Value []graphChannelMessage `json:"value"`
}

type graphChannelMessage struct {
	ID        string                  `json:"id"`
	CreatedAt string                  `json:"createdDateTime"`
	From      *graphChannelMessageFrom `json:"from,omitempty"`
	Body      graphChannelMessageBody `json:"body"`
}

type graphChannelMessageFrom struct {
	User *graphChannelMessageUser `json:"user,omitempty"`
}

type graphChannelMessageUser struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
}

// channelMessageSummary is the simplified response returned to the caller.
type channelMessageSummary struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	From      string `json:"from"`
	Content   string `json:"content"`
}

// Execute lists recent messages from a Teams channel.
func (a *listChannelMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChannelMessagesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	path := fmt.Sprintf("/teams/%s/channels/%s/messages?$top=%d", params.TeamID, params.ChannelID, params.Top)

	var resp graphChannelMessagesResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]channelMessageSummary, len(resp.Value))
	for i, msg := range resp.Value {
		var from string
		if msg.From != nil && msg.From.User != nil {
			from = msg.From.User.DisplayName
		}
		summaries[i] = channelMessageSummary{
			ID:        msg.ID,
			CreatedAt: msg.CreatedAt,
			From:      from,
			Content:   msg.Body.Content,
		}
	}

	return connectors.JSONResult(summaries)
}
