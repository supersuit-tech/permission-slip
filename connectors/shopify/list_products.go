package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listProductsAction implements connectors.Action for shopify.list_products.
// It lists products with optional filtering via GET /admin/api/2024-10/products.json.
type listProductsAction struct {
	conn *ShopifyConnector
}

type listProductsParams struct {
	Status      string `json:"status,omitempty"`
	ProductType string `json:"product_type,omitempty"`
	Vendor      string `json:"vendor,omitempty"`
	Fields      string `json:"fields,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

func (p *listProductsParams) validate() error {
	if p.Status != "" && !validProductStatuses[p.Status] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid status %q: must be one of %s", p.Status, sortedKeys(validProductStatuses))}
	}
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.Limit < 1 || p.Limit > 250 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 250, got %d", p.Limit)}
	}
	return nil
}

// Execute lists products from the Shopify store with optional filtering.
func (a *listProductsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listProductsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(params.Limit))
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.ProductType != "" {
		q.Set("product_type", params.ProductType)
	}
	if params.Vendor != "" {
		q.Set("vendor", params.Vendor)
	}
	if params.Fields != "" {
		q.Set("fields", params.Fields)
	}

	var resp struct {
		Products json.RawMessage `json:"products"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/products.json?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
