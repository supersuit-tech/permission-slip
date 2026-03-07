package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getDeploymentAction struct {
	conn *VercelConnector
}

type getDeploymentParams struct {
	DeploymentID string `json:"deployment_id"`
	TeamID       string `json:"team_id"`
}

func (p *getDeploymentParams) validate() error {
	if p.DeploymentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deployment_id"}
	}
	return nil
}

func (a *getDeploymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDeploymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/v13/deployments/" + url.PathEscape(params.DeploymentID)
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
