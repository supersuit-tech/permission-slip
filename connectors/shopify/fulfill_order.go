package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// fulfillOrderAction implements connectors.Action for shopify.fulfill_order.
// It creates a fulfillment via the fulfillment-order REST API:
//  1. GET /admin/api/{version}/orders/{order_id}/fulfillment_orders.json
//  2. POST /admin/api/{version}/fulfillments.json with line_items_by_fulfillment_order
//
// The order-scoped POST /orders/{id}/fulfillments.json endpoint was removed in
// API version 2022-07 and 404s on current Shopify versions.
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

// fulfillmentOrder is the subset of Shopify's FulfillmentOrder resource needed
// to decide whether (and from which location) we can create a fulfillment.
type fulfillmentOrder struct {
	ID                 int64    `json:"id"`
	Status             string   `json:"status"`
	AssignedLocationID int64    `json:"assigned_location_id"`
	SupportedActions   []string `json:"supported_actions"`
}

func (fo fulfillmentOrder) canCreateFulfillment() bool {
	if len(fo.SupportedActions) == 0 {
		return fo.Status == "open" || fo.Status == "in_progress"
	}
	for _, action := range fo.SupportedActions {
		if action == "create_fulfillment" {
			return true
		}
	}
	return false
}

// Execute creates a fulfillment for a Shopify order with optional tracking info.
// Fulfillment orders at the same location are combined into one fulfillment;
// orders split across locations produce one fulfillment per location.
func (a *fulfillOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params fulfillOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var foResp struct {
		FulfillmentOrders []fulfillmentOrder `json:"fulfillment_orders"`
	}
	listPath := fmt.Sprintf("/orders/%d/fulfillment_orders.json", params.OrderID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, listPath, nil, &foResp); err != nil {
		return nil, err
	}

	byLocation := map[int64][]fulfillmentOrder{}
	var locationOrder []int64
	for _, fo := range foResp.FulfillmentOrders {
		if !fo.canCreateFulfillment() {
			continue
		}
		if _, seen := byLocation[fo.AssignedLocationID]; !seen {
			locationOrder = append(locationOrder, fo.AssignedLocationID)
		}
		byLocation[fo.AssignedLocationID] = append(byLocation[fo.AssignedLocationID], fo)
	}
	if len(locationOrder) == 0 {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("order %d has no fulfillable fulfillment orders (already fulfilled, on hold, or missing fulfillment-order access scopes)", params.OrderID),
		}
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

	created := make([]json.RawMessage, 0, len(locationOrder))
	for _, locID := range locationOrder {
		fos := byLocation[locID]
		lineItems := make([]map[string]interface{}, 0, len(fos))
		for _, fo := range fos {
			lineItems = append(lineItems, map[string]interface{}{
				"fulfillment_order_id": fo.ID,
			})
		}

		fulfillment := map[string]interface{}{
			"line_items_by_fulfillment_order": lineItems,
		}
		if len(trackingInfo) > 0 {
			fulfillment["tracking_info"] = trackingInfo
		}
		if params.NotifyCustomer != nil {
			fulfillment["notify_customer"] = *params.NotifyCustomer
		}

		var createResp struct {
			Fulfillment json.RawMessage `json:"fulfillment"`
		}
		reqBody := map[string]interface{}{"fulfillment": fulfillment}
		if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/fulfillments.json", reqBody, &createResp); err != nil {
			if len(created) > 0 {
				return nil, fmt.Errorf("fulfillment created for %d location(s) but failed for location %d: %w",
					len(created), locID, err)
			}
			return nil, err
		}
		if len(createResp.Fulfillment) > 0 {
			created = append(created, createResp.Fulfillment)
		}
	}

	if len(created) == 1 {
		return connectors.JSONResult(struct {
			Fulfillment json.RawMessage `json:"fulfillment"`
		}{Fulfillment: created[0]})
	}
	return connectors.JSONResult(struct {
		Fulfillments []json.RawMessage `json:"fulfillments"`
	}{Fulfillments: created})
}
