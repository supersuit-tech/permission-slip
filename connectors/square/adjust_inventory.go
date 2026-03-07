package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// adjustInventoryAction implements connectors.Action for square.adjust_inventory.
// It adjusts inventory counts via POST /v2/inventory/changes/batch-create.
// Medium risk — modifies inventory quantities but is recoverable.
type adjustInventoryAction struct {
	conn *SquareConnector
}

type adjustInventoryParams struct {
	CatalogObjectID string `json:"catalog_object_id"`
	LocationID      string `json:"location_id"`
	Quantity        string `json:"quantity"`
	FromState       string `json:"from_state"`
	ToState         string `json:"to_state"`
}

var validInventoryStates = map[string]bool{
	"IN_STOCK":                 true,
	"SOLD":                     true,
	"RETURNED_BY_CUSTOMER":     true,
	"RESERVED_FOR_SALE":        true,
	"SOLD_ONLINE":              true,
	"ORDERED_FROM_VENDOR":      true,
	"RECEIVED_FROM_VENDOR":     true,
	"IN_TRANSIT_TO":            true,
	"NONE":                     true,
	"WASTE":                    true,
	"UNLINKED_RETURN":          true,
	"COMPOSED":                 true,
	"DECOMPOSED":               true,
	"SUPPORTED_BY_NEWER_VERSION": true,
}

func (p *adjustInventoryParams) validate() error {
	if p.CatalogObjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: catalog_object_id"}
	}
	if p.LocationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: location_id"}
	}
	if p.Quantity == "" {
		return &connectors.ValidationError{Message: "missing required parameter: quantity"}
	}
	if qty, err := strconv.ParseFloat(p.Quantity, 64); err != nil || qty <= 0 {
		return &connectors.ValidationError{Message: "quantity must be a positive number string (e.g. \"10\")"}
	}
	if p.FromState == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from_state"}
	}
	if p.ToState == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to_state"}
	}
	if !validInventoryStates[p.FromState] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid from_state %q: must be a valid Square inventory state (e.g. IN_STOCK, SOLD, NONE)", p.FromState)}
	}
	if !validInventoryStates[p.ToState] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid to_state %q: must be a valid Square inventory state (e.g. IN_STOCK, SOLD, NONE)", p.ToState)}
	}
	return nil
}

func (a *adjustInventoryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params adjustInventoryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	occurredAt := time.Now().UTC().Format(time.RFC3339)

	// Use PHYSICAL_COUNT for same-state (setting absolute count) and
	// ADJUSTMENT for state transitions (moving between states).
	var change map[string]interface{}
	if params.FromState == params.ToState {
		change = map[string]interface{}{
			"type": "PHYSICAL_COUNT",
			"physical_count": map[string]interface{}{
				"catalog_object_id": params.CatalogObjectID,
				"location_id":      params.LocationID,
				"quantity":          params.Quantity,
				"state":             params.ToState,
				"occurred_at":       occurredAt,
			},
		}
	} else {
		change = map[string]interface{}{
			"type": "ADJUSTMENT",
			"adjustment": map[string]interface{}{
				"catalog_object_id": params.CatalogObjectID,
				"location_id":      params.LocationID,
				"quantity":          params.Quantity,
				"from_state":        params.FromState,
				"to_state":          params.ToState,
				"occurred_at":       occurredAt,
			},
		}
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"changes":         []map[string]interface{}{change},
	}

	var resp struct {
		Counts json.RawMessage `json:"counts"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/inventory/changes/batch-create", body, &resp); err != nil {
		return nil, err
	}

	counts := json.RawMessage(resp.Counts)
	if len(counts) == 0 || string(counts) == "null" {
		counts = json.RawMessage("[]")
	}

	return connectors.JSONResult(map[string]interface{}{
		"counts": counts,
	})
}
