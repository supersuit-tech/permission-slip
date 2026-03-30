package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateCatalogItemAction implements connectors.Action for
// square.update_catalog_item. It upserts a catalog object via
// POST /v2/catalog/object. Medium risk — modifies product data but is
// recoverable.
type updateCatalogItemAction struct {
	conn *SquareConnector
}

type catalogVariation struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	PricingType    string `json:"pricing_type,omitempty"`
	PriceMoney     *money `json:"price_money,omitempty"`
	Version        *int64 `json:"version,omitempty"`
}

type updateCatalogItemParams struct {
	ObjectID    string             `json:"object_id"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	Variations  []catalogVariation `json:"variations,omitempty"`
	Version     *int64             `json:"version,omitempty"`
}

func (p *updateCatalogItemParams) validate() error {
	if p.ObjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: object_id"}
	}
	if p.Name == "" && p.Description == "" && len(p.Variations) == 0 {
		return &connectors.ValidationError{Message: "at least one of name, description, or variations must be provided"}
	}
	for i, v := range p.Variations {
		if v.ID == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("variations[%d].id is required", i)}
		}
		if v.PriceMoney != nil {
			if v.PriceMoney.Amount < 0 {
				return &connectors.ValidationError{Message: fmt.Sprintf("variations[%d].price_money.amount must be >= 0", i)}
			}
			if v.PriceMoney.Currency == "" {
				return &connectors.ValidationError{Message: fmt.Sprintf("variations[%d].price_money.currency is required", i)}
			}
		}
	}
	return nil
}

func (a *updateCatalogItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCatalogItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the item_data object with only provided fields.
	itemData := map[string]interface{}{}
	if params.Name != "" {
		itemData["name"] = params.Name
	}
	if params.Description != "" {
		itemData["description"] = params.Description
	}

	// Build variations for the upsert if provided.
	if len(params.Variations) > 0 {
		variations := make([]map[string]interface{}, len(params.Variations))
		for i, v := range params.Variations {
			varData := map[string]interface{}{}
			if v.Name != "" {
				varData["name"] = v.Name
			}
			if v.PricingType != "" {
				varData["pricing_type"] = v.PricingType
			}
			if v.PriceMoney != nil {
				varData["price_money"] = v.PriceMoney
			}

			variation := map[string]interface{}{
				"type":                    "ITEM_VARIATION",
				"id":                      v.ID,
				"item_variation_data":     varData,
			}
			if v.Version != nil {
				variation["version"] = *v.Version
			}
			variations[i] = variation
		}
		itemData["variations"] = variations
	}

	catalogObject := map[string]interface{}{
		"type":      "ITEM",
		"id":        params.ObjectID,
		"item_data": itemData,
	}
	if params.Version != nil {
		catalogObject["version"] = *params.Version
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"object":          catalogObject,
	}

	var resp struct {
		CatalogObject json.RawMessage `json:"catalog_object"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/catalog/object", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.CatalogObject))
}
