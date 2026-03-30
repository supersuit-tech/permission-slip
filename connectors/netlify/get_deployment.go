package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getDeploymentAction struct {
	conn *NetlifyConnector
}

type getDeploymentParams struct {
	DeployID string `json:"deploy_id"`
}

func (p *getDeploymentParams) validate() error {
	if p.DeployID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deploy_id"}
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

	path := "/deploys/" + url.PathEscape(params.DeployID)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
