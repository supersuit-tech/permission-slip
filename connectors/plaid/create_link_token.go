package plaid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createLinkTokenAction implements connectors.Action for plaid.create_link_token.
// It creates a link token via POST /link/token/create.
type createLinkTokenAction struct {
	conn *PlaidConnector
}

type createLinkTokenParams struct {
	UserID       string   `json:"user_id"`
	Products     []string `json:"products"`
	CountryCodes []string `json:"country_codes"`
	Language     string   `json:"language"`
}

// validProducts lists the Plaid products accepted by this connector.
var validProducts = map[string]bool{
	"auth":         true,
	"transactions": true,
	"identity":     true,
	"balance":      true,
}

func (p *createLinkTokenParams) validate() error {
	if p.UserID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: user_id"}
	}
	if len(p.Products) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: products"}
	}
	for _, prod := range p.Products {
		if !validProducts[prod] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid product %q; must be one of: auth, transactions, identity, balance", prod),
			}
		}
	}
	return nil
}

// Execute creates a link token and returns the token metadata.
func (a *createLinkTokenAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createLinkTokenParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	countryCodes := params.CountryCodes
	if len(countryCodes) == 0 {
		countryCodes = []string{"US"}
	}
	language := params.Language
	if language == "" {
		language = "en"
	}

	body := map[string]any{
		"user": map[string]any{
			"client_user_id": params.UserID,
		},
		"client_name":   "Permission Slip",
		"products":      params.Products,
		"country_codes": countryCodes,
		"language":      language,
	}

	var resp struct {
		LinkToken  string `json:"link_token"`
		Expiration string `json:"expiration"`
		RequestID  string `json:"request_id"`
	}
	if err := a.conn.doPost(ctx, req.Credentials, "/link/token/create", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
