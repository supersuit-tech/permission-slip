package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listDeploymentsAction struct {
	conn *NetlifyConnector
}

type listDeploymentsParams struct {
	SiteID  string `json:"site_id"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

func (p *listDeploymentsParams) validate() error {
	if p.SiteID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: site_id"}
	}
	return nil
}

func (a *listDeploymentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDeploymentsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", params.PerPage))
	}

	path := "/sites/" + url.PathEscape(params.SiteID) + "/deploys"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
