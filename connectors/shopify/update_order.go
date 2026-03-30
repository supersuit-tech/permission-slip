package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateOrderAction implements connectors.Action for shopify.update_order.
// It updates order attributes via PUT /admin/api/2024-10/orders/{order_id}.json.
type updateOrderAction struct {
	conn *ShopifyConnector
}

// updateOrderParams maps the JSON parameters for the update_order action.
// Pointer fields (Note, Tags, Email) distinguish between "not provided" (nil)
// and "set to empty string" — both are valid for Shopify's PUT endpoint.
type updateOrderParams struct {
	OrderID         int64                  `json:"order_id"`
	Note            *string                `json:"note,omitempty"`
	Tags            *string                `json:"tags,omitempty"`
	Email           *string                `json:"email,omitempty"`
	ShippingAddress map[string]interface{} `json:"shipping_address,omitempty"`
}

func (p *updateOrderParams) validate() error {
	if p.OrderID <= 0 {
		return &connectors.ValidationError{Message: "order_id must be a positive integer"}
	}
	// At least one updatable field must be provided.
	if p.Note == nil && p.Tags == nil && p.Email == nil && p.ShippingAddress == nil {
		return &connectors.ValidationError{Message: "at least one field to update must be provided (note, tags, email, or shipping_address)"}
	}
	return nil
}

// Execute updates order attributes by sending only the provided fields to
// Shopify's PUT endpoint. Fields not included in params are left unchanged.
func (a *updateOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the order update body with only the provided fields.
	orderBody := map[string]interface{}{}
	if params.Note != nil {
		orderBody["note"] = *params.Note
	}
	if params.Tags != nil {
		orderBody["tags"] = *params.Tags
	}
	if params.Email != nil {
		orderBody["email"] = *params.Email
	}
	if params.ShippingAddress != nil {
		orderBody["shipping_address"] = params.ShippingAddress
	}

	reqBody := map[string]interface{}{
		"order": orderBody,
	}

	var resp struct {
		Order json.RawMessage `json:"order"`
	}
	path := fmt.Sprintf("/orders/%d.json", params.OrderID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
