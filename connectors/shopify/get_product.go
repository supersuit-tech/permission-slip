package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getProductAction implements connectors.Action for shopify.get_product.
// It retrieves a single product by ID via GET /admin/api/2024-10/products/{product_id}.json.
type getProductAction struct {
	conn *ShopifyConnector
}

type getProductParams struct {
	ProductID int64  `json:"product_id"`
	Fields    string `json:"fields,omitempty"`
}

func (p *getProductParams) validate() error {
	if p.ProductID <= 0 {
		return &connectors.ValidationError{Message: "product_id must be a positive integer"}
	}
	return nil
}

// Execute retrieves a single product by ID and returns the full product details.
func (a *getProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/products/%d.json", params.ProductID)
	if params.Fields != "" {
		path += "?fields=" + url.QueryEscape(params.Fields)
	}

	var resp struct {
		Product json.RawMessage `json:"product"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
