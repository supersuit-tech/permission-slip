package plaid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getIdentityAction implements connectors.Action for plaid.get_identity.
// It retrieves account holder identity via POST /identity/get.
type getIdentityAction struct {
	conn *PlaidConnector
}

type getIdentityParams struct {
	AccessToken string   `json:"access_token"`
	AccountIDs  []string `json:"account_ids"`
}

func (p *getIdentityParams) validate() error {
	if p.AccessToken == "" {
		return &connectors.ValidationError{Message: "missing required parameter: access_token"}
	}
	return nil
}

// Execute retrieves identity information and returns the identity data.
func (a *getIdentityAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getIdentityParams
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
	if err := a.conn.doPost(ctx, req.Credentials, "/identity/get", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
