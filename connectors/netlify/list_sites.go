package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listSitesAction struct {
	conn *NetlifyConnector
}

type listSitesParams struct {
	Filter  string `json:"filter"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

func (a *listSitesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSitesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	q := url.Values{}
	if params.Filter != "" {
		q.Set("filter", params.Filter)
	}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", params.PerPage))
	}

	path := "/sites"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
