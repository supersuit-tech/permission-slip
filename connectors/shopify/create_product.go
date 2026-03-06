package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createProductAction implements connectors.Action for shopify.create_product.
// It creates a new product via POST /admin/api/2024-10/products.json.
type createProductAction struct {
	conn *ShopifyConnector
}

// createProductParams maps the JSON parameters for the create_product action.
// Only Title is required; all other fields are optional.
type createProductParams struct {
	Title       string                   `json:"title"`
	BodyHTML    string                   `json:"body_html,omitempty"`
	Vendor      string                   `json:"vendor,omitempty"`
	ProductType string                   `json:"product_type,omitempty"`
	Tags        string                   `json:"tags,omitempty"`
	Status      string                   `json:"status,omitempty"`
	Variants    []map[string]interface{} `json:"variants,omitempty"`
}

// validProductStatuses are the product visibility states accepted by the Shopify Products API.
// See: https://shopify.dev/docs/api/admin-rest/2024-10/resources/product#post-products
var validProductStatuses = map[string]bool{
	"active": true, "draft": true, "archived": true,
}

func (p *createProductParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.Status != "" && !validProductStatuses[p.Status] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid status %q: must be active, draft, or archived", p.Status)}
	}
	return nil
}

// Execute creates a new product in the Shopify store with the given attributes.
// Only provided fields are included in the request body.
func (a *createProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the product body.
	product := map[string]interface{}{
		"title": params.Title,
	}
	if params.BodyHTML != "" {
		product["body_html"] = params.BodyHTML
	}
	if params.Vendor != "" {
		product["vendor"] = params.Vendor
	}
	if params.ProductType != "" {
		product["product_type"] = params.ProductType
	}
	if params.Tags != "" {
		product["tags"] = params.Tags
	}
	if params.Status != "" {
		product["status"] = params.Status
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
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/products.json", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
