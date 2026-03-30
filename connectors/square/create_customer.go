package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createCustomerAction implements connectors.Action for square.create_customer.
// It creates a customer record via POST /v2/customers.
type createCustomerAction struct {
	conn *SquareConnector
}

type createCustomerParams struct {
	GivenName    string `json:"given_name"`
	FamilyName   string `json:"family_name,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	CompanyName  string `json:"company_name,omitempty"`
	Note         string `json:"note,omitempty"`
}

func (p *createCustomerParams) validate() error {
	if p.GivenName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: given_name"}
	}
	return nil
}

func (a *createCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"given_name":      params.GivenName,
	}
	if params.FamilyName != "" {
		body["family_name"] = params.FamilyName
	}
	if params.EmailAddress != "" {
		body["email_address"] = params.EmailAddress
	}
	if params.PhoneNumber != "" {
		body["phone_number"] = params.PhoneNumber
	}
	if params.CompanyName != "" {
		body["company_name"] = params.CompanyName
	}
	if params.Note != "" {
		body["note"] = params.Note
	}

	var resp struct {
		Customer json.RawMessage `json:"customer"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/customers", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.Customer))
}
