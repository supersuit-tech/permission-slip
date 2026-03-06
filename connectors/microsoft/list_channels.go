package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChannelsAction implements connectors.Action for microsoft.list_channels.
// It lists channels in a team via GET /teams/{team-id}/channels.
type listChannelsAction struct {
	conn *MicrosoftConnector
}

// listChannelsParams is the user-facing parameter schema.
type listChannelsParams struct {
	TeamID string `json:"team_id"`
}

func (p *listChannelsParams) validate() error {
	if p.TeamID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: team_id"}
	}
	return nil
}

// graphChannelsResponse is the Microsoft Graph API response for listing channels.
type graphChannelsResponse struct {
	Value []graphChannel `json:"value"`
}

type graphChannel struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// channelSummary is the simplified response returned to the caller.
type channelSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Execute lists channels for the specified team.
func (a *listChannelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChannelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/teams/%s/channels?$select=id,displayName,description", params.TeamID)

	var resp graphChannelsResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]channelSummary, len(resp.Value))
	for i, ch := range resp.Value {
		summaries[i] = channelSummary{
			ID:          ch.ID,
			Name:        ch.DisplayName,
			Description: ch.Description,
		}
	}

	return connectors.JSONResult(summaries)
}
