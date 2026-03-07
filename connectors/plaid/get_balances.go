package plaid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBalancesAction implements connectors.Action for plaid.get_balances.
// It retrieves account balances via POST /accounts/balance/get.
type getBalancesAction struct {
	conn *PlaidConnector
}

type getBalancesParams struct {
	AccessToken string   `json:"access_token"`
	AccountIDs  []string `json:"account_ids"`
}

func (p *getBalancesParams) validate() error {
	if p.AccessToken == "" {
		return &connectors.ValidationError{Message: "missing required parameter: access_token"}
	}
	return nil
}

// Execute retrieves account balances and returns the balance data.
func (a *getBalancesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getBalancesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"access_token": params.AccessToken,
	}
	if len(params.AccountIDs) > 0 {
		body["options"] = map[string]any{
			"account_ids": params.AccountIDs,
		}
	}

	var resp json.RawMessage
	if err := a.conn.doPost(ctx, req.Credentials, "/accounts/balance/get", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
