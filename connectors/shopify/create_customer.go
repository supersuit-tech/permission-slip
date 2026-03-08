package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCustomerAction implements connectors.Action for shopify.create_customer.
// It creates a new customer via POST /admin/api/2024-10/customers.json.
type createCustomerAction struct {
	conn *ShopifyConnector
}

type createCustomerParams struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Note      string `json:"note,omitempty"`
	Tags      string `json:"tags,omitempty"`
}

func (p *createCustomerParams) validate() error {
	if p.Email == "" && p.FirstName == "" && p.LastName == "" && p.Phone == "" {
		return &connectors.ValidationError{Message: "at least one of email, first_name, last_name, or phone is required"}
	}
	return nil
}

// Execute creates a new customer in the Shopify store.
func (a *createCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	customer := map[string]interface{}{}
	if params.Email != "" {
		customer["email"] = params.Email
	}
	if params.FirstName != "" {
		customer["first_name"] = params.FirstName
	}
	if params.LastName != "" {
		customer["last_name"] = params.LastName
	}
	if params.Phone != "" {
		customer["phone"] = params.Phone
	}
	if params.Note != "" {
		customer["note"] = params.Note
	}
	if params.Tags != "" {
		customer["tags"] = params.Tags
	}

	reqBody := map[string]interface{}{
		"customer": customer,
	}

	var resp struct {
		Customer json.RawMessage `json:"customer"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/customers.json", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
