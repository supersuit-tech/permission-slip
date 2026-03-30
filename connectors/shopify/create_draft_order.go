package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createDraftOrderAction implements connectors.Action for shopify.create_draft_order.
// It creates a draft order via POST /admin/api/2024-10/draft_orders.json.
// Draft orders are commonly used for B2B workflows and manual order creation.
type createDraftOrderAction struct {
	conn *ShopifyConnector
}

// draftOrderLineItem represents a line item in a draft order.
type draftOrderLineItem struct {
	VariantID  int64   `json:"variant_id,omitempty"`
	ProductID  int64   `json:"product_id,omitempty"`
	Title      string  `json:"title,omitempty"`
	Price      string  `json:"price,omitempty"`
	Quantity   int     `json:"quantity"`
}

type createDraftOrderParams struct {
	LineItems  []draftOrderLineItem     `json:"line_items"`
	CustomerID int64                    `json:"customer_id,omitempty"`
	Email      string                   `json:"email,omitempty"`
	Note       string                   `json:"note,omitempty"`
	Tags       string                   `json:"tags,omitempty"`
}

func (p *createDraftOrderParams) validate() error {
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "at least one line item is required"}
	}
	for i, item := range p.LineItems {
		if item.Quantity <= 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].quantity must be a positive integer", i)}
		}
		if item.VariantID == 0 && item.ProductID == 0 && item.Title == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d] must include variant_id, product_id, or title", i)}
		}
		// Custom line items (title only, no variant_id/product_id) must have a price.
		if item.VariantID == 0 && item.ProductID == 0 && item.Title != "" && item.Price == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].price is required for custom line items (when using title without variant_id or product_id)", i)}
		}
	}
	return nil
}

// Execute creates a draft order in the Shopify store.
func (a *createDraftOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDraftOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build line items for the API request.
	lineItems := make([]map[string]interface{}, 0, len(params.LineItems))
	for _, item := range params.LineItems {
		li := map[string]interface{}{
			"quantity": item.Quantity,
		}
		if item.VariantID != 0 {
			li["variant_id"] = item.VariantID
		}
		if item.ProductID != 0 {
			li["product_id"] = item.ProductID
		}
		if item.Title != "" {
			li["title"] = item.Title
		}
		if item.Price != "" {
			li["price"] = item.Price
		}
		lineItems = append(lineItems, li)
	}

	draftOrder := map[string]interface{}{
		"line_items": lineItems,
	}
	if params.CustomerID != 0 {
		draftOrder["customer"] = map[string]interface{}{
			"id": params.CustomerID,
		}
	}
	if params.Email != "" {
		draftOrder["email"] = params.Email
	}
	if params.Note != "" {
		draftOrder["note"] = params.Note
	}
	if params.Tags != "" {
		draftOrder["tags"] = params.Tags
	}

	reqBody := map[string]interface{}{
		"draft_order": draftOrder,
	}

	var resp struct {
		DraftOrder json.RawMessage `json:"draft_order"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/draft_orders.json", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
