package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// cancelOrderAction implements connectors.Action for shopify.cancel_order.
// It cancels an order via POST /admin/api/2024-10/orders/{order_id}/cancel.json.
// This is an irreversible, high-risk action.
type cancelOrderAction struct {
	conn *ShopifyConnector
}

// cancelOrderParams maps the JSON parameters for the cancel_order action.
type cancelOrderParams struct {
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
	Restock *bool  `json:"restock,omitempty"`
	Email   *bool  `json:"email,omitempty"`
}

// validCancelReasons are the cancellation reasons accepted by the Shopify Orders API.
// See: https://shopify.dev/docs/api/admin-rest/2024-10/resources/order#post-orders-order-id-cancel
var validCancelReasons = map[string]bool{
	"customer":  true,
	"fraud":     true,
	"inventory": true,
	"declined":  true,
	"other":     true,
}

func (p *cancelOrderParams) validate() error {
	if p.OrderID <= 0 {
		return &connectors.ValidationError{Message: "order_id must be a positive integer"}
	}
	if p.Reason != "" && !validCancelReasons[p.Reason] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid reason %q: must be one of %s", p.Reason, sortedKeys(validCancelReasons)),
		}
	}
	return nil
}

// Execute cancels a Shopify order. This action is irreversible.
func (a *cancelOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params cancelOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{}
	if params.Reason != "" {
		reqBody["reason"] = params.Reason
	}
	if params.Restock != nil {
		reqBody["restock"] = *params.Restock
	}
	if params.Email != nil {
		reqBody["email"] = *params.Email
	}

	var resp struct {
		Order json.RawMessage `json:"order"`
	}
	path := fmt.Sprintf("/orders/%d/cancel.json", params.OrderID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
