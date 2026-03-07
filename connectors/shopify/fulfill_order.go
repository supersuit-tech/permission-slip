package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// fulfillOrderAction implements connectors.Action for shopify.fulfill_order.
// It creates a fulfillment for an order via POST /admin/api/2024-10/orders/{order_id}/fulfillments.json.
type fulfillOrderAction struct {
	conn *ShopifyConnector
}

// fulfillOrderParams maps the JSON parameters for the fulfill_order action.
type fulfillOrderParams struct {
	OrderID         int64  `json:"order_id"`
	TrackingNumber  string `json:"tracking_number,omitempty"`
	TrackingCompany string `json:"tracking_company,omitempty"`
	TrackingURL     string `json:"tracking_url,omitempty"`
	NotifyCustomer  *bool  `json:"notify_customer,omitempty"`
}

func (p *fulfillOrderParams) validate() error {
	if p.OrderID <= 0 {
		return &connectors.ValidationError{Message: "order_id must be a positive integer"}
	}
	// Warn if tracking URL is provided without a tracking number — the URL
	// alone isn't very useful for customers.
	if p.TrackingURL != "" && p.TrackingNumber == "" {
		return &connectors.ValidationError{Message: "tracking_url requires tracking_number to be set"}
	}
	return nil
}

// Execute creates a fulfillment for a Shopify order with optional tracking info.
func (a *fulfillOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params fulfillOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	trackingInfo := map[string]interface{}{}
	if params.TrackingNumber != "" {
		trackingInfo["number"] = params.TrackingNumber
	}
	if params.TrackingCompany != "" {
		trackingInfo["company"] = params.TrackingCompany
	}
	if params.TrackingURL != "" {
		trackingInfo["url"] = params.TrackingURL
	}

	fulfillment := map[string]interface{}{}
	if len(trackingInfo) > 0 {
		fulfillment["tracking_info"] = trackingInfo
	}
	if params.NotifyCustomer != nil {
		fulfillment["notify_customer"] = *params.NotifyCustomer
	}

	reqBody := map[string]interface{}{
		"fulfillment": fulfillment,
	}

	var resp struct {
		Fulfillment json.RawMessage `json:"fulfillment"`
	}
	path := fmt.Sprintf("/orders/%d/fulfillments.json", params.OrderID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
