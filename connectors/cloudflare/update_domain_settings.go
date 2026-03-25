package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type updateDomainSettingsAction struct {
	conn *CloudflareConnector
}

type updateDomainSettingsParams struct {
	AccountID string `json:"account_id"`
	Domain    string `json:"domain"`
	AutoRenew *bool  `json:"auto_renew"`
}

func (p *updateDomainSettingsParams) validate() error {
	if err := requirePathParam("account_id", p.AccountID); err != nil {
		return err
	}
	if err := requirePathParam("domain", p.Domain); err != nil {
		return err
	}
	if p.AutoRenew == nil {
		return &connectors.ValidationError{Message: "must specify at least one setting to update (auto_renew)"}
	}
	return nil
}

func (a *updateDomainSettingsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateDomainSettingsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.Domain,
	}
	if params.AutoRenew != nil {
		body["auto_renew"] = *params.AutoRenew
	}

	var result json.RawMessage
	path := fmt.Sprintf("/accounts/%s/registrar/domains/%s", params.AccountID, params.Domain)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, path, body, &result); err != nil {
		return nil, err
	}

	return connectors.JSONResult(result)
}
