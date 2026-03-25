package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listTunnelsAction struct {
	conn *CloudflareConnector
}

type listTunnelsParams struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	IsDeleted bool   `json:"is_deleted"`
}

func (p *listTunnelsParams) validate() error {
	return requirePathParam("account_id", p.AccountID)
}

func (a *listTunnelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTunnelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.Name != "" {
		q.Set("name", params.Name)
	}
	if params.IsDeleted {
		q.Set("is_deleted", "true")
	}

	path := fmt.Sprintf("/accounts/%s/cfd_tunnel", params.AccountID)
	if qs := q.Encode(); qs != "" {
		path += "?" + qs
	}

	var tunnels []json.RawMessage
	if err := a.conn.doGet(ctx, req.Credentials, path, &tunnels); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{"tunnels": tunnels})
}
