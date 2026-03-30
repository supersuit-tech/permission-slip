package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type triggerDeploymentAction struct {
	conn *NetlifyConnector
}

type triggerDeploymentParams struct {
	SiteID     string `json:"site_id"`
	Branch     string `json:"branch"`
	ClearCache bool   `json:"clear_cache"`
	Title      string `json:"title"`
}

func (p *triggerDeploymentParams) validate() error {
	if p.SiteID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: site_id"}
	}
	return nil
}

func (a *triggerDeploymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params triggerDeploymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{}
	if params.Branch != "" {
		body["branch"] = params.Branch
	}
	if params.ClearCache {
		body["clear_cache"] = true
	}
	if params.Title != "" {
		body["title"] = params.Title
	}

	path := "/sites/" + url.PathEscape(params.SiteID) + "/builds"

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
