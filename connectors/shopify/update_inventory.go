package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateInventoryAction implements connectors.Action for shopify.update_inventory.
// It adjusts inventory levels via POST /admin/api/2024-10/inventory_levels/adjust.json.
type updateInventoryAction struct {
	conn *ShopifyConnector
}

// updateInventoryParams maps the JSON parameters for the update_inventory action.
// All three fields are required. AvailableAdjustment is a relative delta, not
// an absolute value — positive adds stock, negative removes it.
type updateInventoryParams struct {
	InventoryItemID     int64 `json:"inventory_item_id"`
	LocationID          int64 `json:"location_id"`
	AvailableAdjustment int   `json:"available_adjustment"`
}

func (p *updateInventoryParams) validate() error {
	if p.InventoryItemID <= 0 {
		return &connectors.ValidationError{Message: "inventory_item_id must be a positive integer"}
	}
	if p.LocationID <= 0 {
		return &connectors.ValidationError{Message: "location_id must be a positive integer"}
	}
	if p.AvailableAdjustment == 0 {
		return &connectors.ValidationError{Message: "available_adjustment must be non-zero (positive to add inventory, negative to subtract)"}
	}
	return nil
}

// Execute adjusts inventory for an item at a location by the given delta.
func (a *updateInventoryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateInventoryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"inventory_item_id":    params.InventoryItemID,
		"location_id":          params.LocationID,
		"available_adjustment": params.AvailableAdjustment,
	}

	var resp struct {
		InventoryLevel json.RawMessage `json:"inventory_level"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/inventory_levels/adjust.json", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
