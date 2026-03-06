package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTeamsAction implements connectors.Action for microsoft.list_teams.
// It lists the teams the authenticated user is a member of via GET /me/joinedTeams.
type listTeamsAction struct {
	conn *MicrosoftConnector
}

// listTeamsParams is the user-facing parameter schema.
type listTeamsParams struct {
	Top int `json:"top"`
}

func (p *listTeamsParams) defaults() {
	if p.Top <= 0 {
		p.Top = 20
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

// graphTeamsResponse is the Microsoft Graph API response for listing joined teams.
type graphTeamsResponse struct {
	Value []graphTeam `json:"value"`
}

type graphTeam struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// teamSummary is the simplified response returned to the caller.
type teamSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Execute lists the teams the user has joined.
func (a *listTeamsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTeamsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.defaults()

	path := fmt.Sprintf("/me/joinedTeams?$top=%d&$select=id,displayName,description", params.Top)

	var resp graphTeamsResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]teamSummary, len(resp.Value))
	for i, t := range resp.Value {
		summaries[i] = teamSummary{
			ID:          t.ID,
			Name:        t.DisplayName,
			Description: t.Description,
		}
	}

	return connectors.JSONResult(summaries)
}
