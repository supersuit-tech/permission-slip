package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createCustomerAction implements connectors.Action for quickbooks.create_customer.
type createCustomerAction struct {
	conn *QuickBooksConnector
}

type createCustomerParams struct {
	DisplayName  string `json:"display_name"`
	GivenName    string `json:"given_name"`
	FamilyName   string `json:"family_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	CompanyName  string `json:"company_name"`
}

func (p *createCustomerParams) validate() error {
	if p.DisplayName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: display_name"}
	}
	return nil
}

// Execute creates a customer record in QuickBooks.
func (a *createCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"DisplayName": params.DisplayName,
	}
	if params.GivenName != "" {
		body["GivenName"] = params.GivenName
	}
	if params.FamilyName != "" {
		body["FamilyName"] = params.FamilyName
	}
	if params.Email != "" {
		body["PrimaryEmailAddr"] = map[string]string{
			"Address": params.Email,
		}
	}
	if params.Phone != "" {
		body["PrimaryPhone"] = map[string]string{
			"FreeFormNumber": params.Phone,
		}
	}
	if params.CompanyName != "" {
		body["CompanyName"] = params.CompanyName
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/customer"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	customer, ok := resp["Customer"]
	if !ok {
		customer = resp
	}

	return connectors.JSONResult(customer)
}
