package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createVendorAction implements connectors.Action for quickbooks.create_vendor.
type createVendorAction struct {
	conn *QuickBooksConnector
}

type createVendorParams struct {
	DisplayName string `json:"display_name"`
	GivenName   string `json:"given_name,omitempty"`
	FamilyName  string `json:"family_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

func (p *createVendorParams) validate() error {
	if p.DisplayName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: display_name"}
	}
	if err := validateEmail("email", p.Email); err != nil {
		return err
	}
	return nil
}

// Execute creates a vendor record in QuickBooks.
func (a *createVendorAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createVendorParams
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
	path := companyPath(req.Credentials) + "/vendor"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	vendor, ok := resp["Vendor"]
	if !ok {
		vendor = resp
	}

	return connectors.JSONResult(vendor)
}
