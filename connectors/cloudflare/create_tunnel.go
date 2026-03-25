package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createTunnelAction struct {
	conn *CloudflareConnector
}

type createTunnelParams struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	TunnelSecret string `json:"tunnel_secret"`
}

func (p *createTunnelParams) validate() error {
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.TunnelSecret == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tunnel_secret"}
	}
	return nil
}

func (a *createTunnelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTunnelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name":          params.Name,
		"tunnel_secret": params.TunnelSecret,
	}

	var tunnel json.RawMessage
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel", params.AccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, path, body, &tunnel); err != nil {
		return nil, err
	}

	return connectors.JSONResult(tunnel)
}
