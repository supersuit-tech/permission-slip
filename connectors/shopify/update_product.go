package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateProductAction implements connectors.Action for shopify.update_product.
// It updates an existing product via PUT /admin/api/2024-10/products/{product_id}.json.
type updateProductAction struct {
	conn *ShopifyConnector
}

// updateProductParams maps the JSON parameters for the update_product action.
// Pointer fields distinguish between "not provided" (nil) and "set to empty string".
type updateProductParams struct {
	ProductID int64                    `json:"product_id"`
	Title     *string                  `json:"title,omitempty"`
	BodyHTML  *string                  `json:"body_html,omitempty"`
	Vendor    *string                  `json:"vendor,omitempty"`
	Tags      *string                  `json:"tags,omitempty"`
	Status    *string                  `json:"status,omitempty"`
	Variants  []map[string]interface{} `json:"variants,omitempty"`
}

func (p *updateProductParams) validate() error {
	if p.ProductID <= 0 {
		return &connectors.ValidationError{Message: "product_id must be a positive integer"}
	}
	// Treat an empty variants slice the same as not provided — sending an
	// empty array to Shopify is almost certainly unintended.
	hasVariants := len(p.Variants) > 0
	if p.Title == nil && p.BodyHTML == nil && p.Vendor == nil && p.Tags == nil && p.Status == nil && !hasVariants {
		return &connectors.ValidationError{Message: "at least one field to update must be provided"}
	}
	if p.Status != nil {
		if *p.Status == "" {
			return &connectors.ValidationError{Message: "status cannot be empty: must be active, draft, or archived"}
		}
		if !validProductStatuses[*p.Status] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid status %q: must be active, draft, or archived", *p.Status),
			}
		}
	}
	return nil
}

// Execute updates an existing product in the Shopify store.
// Only provided fields are included in the request body.
func (a *updateProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	product := map[string]interface{}{}
	if params.Title != nil {
		product["title"] = *params.Title
	}
	if params.BodyHTML != nil {
		product["body_html"] = *params.BodyHTML
	}
	if params.Vendor != nil {
		product["vendor"] = *params.Vendor
	}
	if params.Tags != nil {
		product["tags"] = *params.Tags
	}
	if params.Status != nil {
		product["status"] = *params.Status
	}
	if len(params.Variants) > 0 {
		product["variants"] = params.Variants
	}

	reqBody := map[string]interface{}{
		"product": product,
	}

	var resp struct {
		Product json.RawMessage `json:"product"`
	}
	path := fmt.Sprintf("/products/%d.json", params.ProductID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
