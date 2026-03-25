package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type checkDomainAction struct {
	conn *CloudflareConnector
}

type checkDomainParams struct {
	AccountID string `json:"account_id"`
	Domain    string `json:"domain"`
}

func (p *checkDomainParams) validate() error {
	if err := requirePathParam("account_id", p.AccountID); err != nil {
		return err
	}
	if err := requirePathParam("domain", p.Domain); err != nil {
		return err
	}
	return nil
}

func (a *checkDomainAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params checkDomainParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var result json.RawMessage
	path := fmt.Sprintf("/accounts/%s/registrar/domains/%s", params.AccountID, params.Domain)
	if err := a.conn.doGet(ctx, req.Credentials, path, &result); err != nil {
		return nil, err
	}

	return connectors.JSONResult(result)
}
