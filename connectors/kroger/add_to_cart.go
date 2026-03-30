package kroger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addToCartAction implements connectors.Action for kroger.add_to_cart.
// It adds items to a user's cart via PUT /v1/cart/add.
type addToCartAction struct {
	conn *KrogerConnector
}

type cartItem struct {
	UPC      string `json:"upc"`
	Quantity int    `json:"quantity"`
}

type addToCartParams struct {
	Items    []cartItem `json:"items"`
	Modality string     `json:"modality"`
}

func (p *addToCartParams) validate() error {
	if len(p.Items) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: items"}
	}
	if len(p.Items) > 25 {
		return &connectors.ValidationError{Message: "items cannot exceed 25 entries"}
	}
	if p.Modality != "" && p.Modality != "PICKUP" && p.Modality != "DELIVERY" {
		return &connectors.ValidationError{Message: fmt.Sprintf("modality must be PICKUP or DELIVERY (got %q)", p.Modality)}
	}
	for i, item := range p.Items {
		if item.UPC == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("items[%d]: missing required field: upc", i)}
		}
		if item.Quantity < 1 {
			return &connectors.ValidationError{Message: fmt.Sprintf("items[%d]: quantity must be at least 1", i)}
		}
	}
	return nil
}

// Execute adds items to the user's Kroger cart.
func (a *addToCartAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addToCartParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"items": params.Items,
	}
	if params.Modality != "" {
		body["modality"] = params.Modality
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, "/cart/add", body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"status":      "added",
		"items_count": len(params.Items),
	})
}
