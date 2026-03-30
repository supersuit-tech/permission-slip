package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type deleteTunnelAction struct {
	conn *CloudflareConnector
}

type deleteTunnelParams struct {
	AccountID string `json:"account_id"`
	TunnelID  string `json:"tunnel_id"`
}

func (p *deleteTunnelParams) validate() error {
	if err := requirePathParam("account_id", p.AccountID); err != nil {
		return err
	}
	if err := requirePathParam("tunnel_id", p.TunnelID); err != nil {
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
	if err := a.conn.doDelete(ctx, req.Credentials, path); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "deleted", "tunnel_id": params.TunnelID})
}
