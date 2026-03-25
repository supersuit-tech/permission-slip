package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type updateTunnelConfigAction struct {
	conn *CloudflareConnector
}

type updateTunnelConfigParams struct {
	AccountID string          `json:"account_id"`
	TunnelID  string          `json:"tunnel_id"`
	Config    json.RawMessage `json:"config"`
}

func (p *updateTunnelConfigParams) validate() error {
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id"}
	}
	if p.TunnelID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tunnel_id"}
	}
	if len(p.Config) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: config"}
	}
	return nil
}

func (a *updateTunnelConfigAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateTunnelConfigParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"config": json.RawMessage(params.Config),
	}

	var config json.RawMessage
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", params.AccountID, params.TunnelID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, path, body, &config); err != nil {
		return nil, err
	}

	return connectors.JSONResult(config)
}
