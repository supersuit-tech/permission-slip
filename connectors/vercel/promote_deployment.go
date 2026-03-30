package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type promoteDeploymentAction struct {
	conn *VercelConnector
}

type promoteDeploymentParams struct {
	ProjectID    string `json:"project_id"`
	DeploymentID string `json:"deployment_id"`
	TeamID       string `json:"team_id"`
}

func (p *promoteDeploymentParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.DeploymentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deployment_id"}
	}
	return nil
}

func (a *promoteDeploymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params promoteDeploymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Vercel promotes a deployment by creating a new deployment alias pointing
	// to the given deployment ID via the project's production domain.
	// The v10 promote endpoint handles this atomically.
	body := map[string]string{
		"deploymentId": params.DeploymentID,
	}

	path := "/v10/projects/" + url.PathEscape(params.ProjectID) + "/promote"
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	if resp == nil || string(resp) == "null" {
		return connectors.JSONResult(map[string]string{
			"status":        "promoted",
			"deployment_id": params.DeploymentID,
			"project_id":    params.ProjectID,
		})
	}
	return connectors.JSONResult(resp)
}
