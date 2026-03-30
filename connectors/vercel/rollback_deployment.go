package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type rollbackDeploymentAction struct {
	conn *VercelConnector
}

type rollbackDeploymentParams struct {
	ProjectID    string `json:"project_id"`
	DeploymentID string `json:"deployment_id"`
	TeamID       string `json:"team_id"`
}

func (p *rollbackDeploymentParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.DeploymentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deployment_id"}
	}
	return nil
}

func (a *rollbackDeploymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params rollbackDeploymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]string{
		"deploymentId": params.DeploymentID,
	}

	path := "/v9/projects/" + url.PathEscape(params.ProjectID) + "/rollback"
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	// Vercel rollback returns empty body on success (200 OK).
	if resp == nil || string(resp) == "null" {
		return connectors.JSONResult(map[string]string{
			"status":        "rollback_initiated",
			"deployment_id": params.DeploymentID,
			"project_id":    params.ProjectID,
		})
	}
	return connectors.JSONResult(resp)
}
