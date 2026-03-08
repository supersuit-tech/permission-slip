package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createProductAction implements connectors.Action for stripe.create_product.
// It creates a product in the Stripe catalog via POST /v1/products.
// Products are required before creating prices, subscriptions, or payment links.
type createProductAction struct {
	conn *StripeConnector
}

type createProductParams struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Active      *bool          `json:"active"`
	Metadata    map[string]any `json:"metadata"`
}

func (p *createProductParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe product and returns the created product data.
func (a *createProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.Name,
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if params.Active != nil {
		body["active"] = *params.Active
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/products", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
