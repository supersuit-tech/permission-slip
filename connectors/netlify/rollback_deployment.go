package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type rollbackDeploymentAction struct {
	conn *NetlifyConnector
}

type rollbackDeploymentParams struct {
	SiteID   string `json:"site_id"`
	DeployID string `json:"deploy_id"`
}

func (p *rollbackDeploymentParams) validate() error {
	if p.SiteID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: site_id"}
	}
	if p.DeployID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deploy_id"}
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

	// Netlify "rollback" is publishing a previous deploy via POST /sites/{site_id}/deploys/{deploy_id}/restore
	path := "/sites/" + url.PathEscape(params.SiteID) + "/deploys/" + url.PathEscape(params.DeployID) + "/restore"

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
