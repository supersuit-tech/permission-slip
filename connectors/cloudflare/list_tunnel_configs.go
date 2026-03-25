package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listTunnelConfigsAction struct {
	conn *CloudflareConnector
}

type listTunnelConfigsParams struct {
	AccountID string `json:"account_id"`
	TunnelID  string `json:"tunnel_id"`
}

func (p *listTunnelConfigsParams) validate() error {
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

func (a *listTunnelConfigsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTunnelConfigsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var config json.RawMessage
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", params.AccountID, params.TunnelID)
	if err := a.conn.doGet(ctx, req.Credentials, path, &config); err != nil {
		return nil, err
	}

	return connectors.JSONResult(config)
}
