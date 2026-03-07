package plaid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// accessTokenParams is the shared parameter set for actions that operate on
// a connected bank account (identified by access_token) with optional
// account_id filtering. Used by get_accounts, get_balances, and get_identity.
type accessTokenParams struct {
	AccessToken string   `json:"access_token"`
	AccountIDs  []string `json:"account_ids"`
}

func (p *accessTokenParams) validate() error {
	if p.AccessToken == "" {
		return &connectors.ValidationError{Message: "missing required parameter: access_token"}
	}
	return nil
}

// accessTokenAction is a generic action that sends access_token (plus
// optional account_ids) to a Plaid endpoint and returns the raw JSON
// response. This eliminates duplication across get_accounts, get_balances,
// and get_identity, which all follow the same pattern.
type accessTokenAction struct {
	conn *PlaidConnector
	path string // e.g. "/accounts/get"
}

// Execute parses access_token params, builds the request body, and calls
// the configured Plaid endpoint.
func (a *accessTokenAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params accessTokenParams
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
	if err := a.conn.doPost(ctx, req.Credentials, a.path, body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
