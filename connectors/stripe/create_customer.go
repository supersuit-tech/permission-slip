package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCustomerAction implements connectors.Action for stripe.create_customer.
// It creates a new customer via POST /v1/customers.
type createCustomerAction struct {
	conn *StripeConnector
}

type createCustomerParams struct {
	Email       string         `json:"email"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Phone       string         `json:"phone"`
	Metadata    map[string]any `json:"metadata"`
}

func (p *createCustomerParams) validate() error {
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe customer and returns the created customer data.
func (a *createCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"email": params.Email,
	}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if params.Phone != "" {
		body["phone"] = params.Phone
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Phone       string `json:"phone"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/customers", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
