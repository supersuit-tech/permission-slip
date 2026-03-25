package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteTunnelAction struct {
	conn *CloudflareConnector
}

type deleteTunnelParams struct {
	AccountID string `json:"account_id"`
	TunnelID  string `json:"tunnel_id"`
}

func (p *deleteTunnelParams) validate() error {
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id"}
	}
	if p.TunnelID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tunnel_id"}
	}
	if err := validatePathParam("account_id", p.AccountID); err != nil {
		return err
	}
	if err := validatePathParam("tunnel_id", p.TunnelID); err != nil {
		return err
	}
	return nil
}

func (a *deleteTunnelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteTunnelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", params.AccountID, params.TunnelID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "deleted", "tunnel_id": params.TunnelID})
}
