package figma

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listProjectsAction implements connectors.Action for figma.list_projects.
// It lists projects in a team via GET /teams/{team_id}/projects.
type listProjectsAction struct {
	conn *FigmaConnector
}

type listProjectsParams struct {
	TeamID string `json:"team_id"`
}

func (p *listProjectsParams) validate() error {
	p.TeamID = strings.TrimSpace(p.TeamID)
	if p.TeamID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: team_id"}
	}
	// Basic sanitization: team IDs should be numeric or alphanumeric.
	if strings.Contains(p.TeamID, "/") || strings.Contains(p.TeamID, "..") {
		return &connectors.ValidationError{Message: "team_id contains invalid characters"}
	}
	return nil
}

type listProjectsResponse struct {
	Name     string         `json:"name"`
	Projects []figmaProject `json:"projects"`
}

type figmaProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a *listProjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listProjectsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp listProjectsResponse
	path := fmt.Sprintf("/teams/%s/projects", url.PathEscape(params.TeamID))
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
