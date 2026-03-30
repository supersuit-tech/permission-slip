package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getInventoryAction implements connectors.Action for square.get_inventory.
// It retrieves inventory counts via POST /v2/inventory/counts/batch-retrieve.
// Read-only — does not modify any data.
type getInventoryAction struct {
	conn *SquareConnector
}

type getInventoryParams struct {
	CatalogObjectIDs []string `json:"catalog_object_ids"`
	LocationIDs      []string `json:"location_ids,omitempty"`
}

func (p *getInventoryParams) validate() error {
	if len(p.CatalogObjectIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: catalog_object_ids (must be a non-empty array)"}
	}
	if len(p.CatalogObjectIDs) > 1000 {
		return &connectors.ValidationError{Message: "catalog_object_ids exceeds maximum of 1000 items"}
	}
	return nil
}

func (a *getInventoryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getInventoryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"catalog_object_ids": params.CatalogObjectIDs,
	}
	if len(params.LocationIDs) > 0 {
		body["location_ids"] = params.LocationIDs
	}

	var resp struct {
		Counts json.RawMessage `json:"counts"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/inventory/counts/batch-retrieve", body, &resp); err != nil {
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
