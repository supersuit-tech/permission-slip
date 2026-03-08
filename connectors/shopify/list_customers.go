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

// listCustomersAction implements connectors.Action for shopify.list_customers.
// It lists or searches customers via GET /admin/api/2024-10/customers.json.
type listCustomersAction struct {
	conn *ShopifyConnector
}

type listCustomersParams struct {
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Fields string `json:"fields,omitempty"`
}

func (p *listCustomersParams) validate() error {
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.Limit < 1 || p.Limit > 250 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 250, got %d", p.Limit)}
	}
	return nil
}

// Execute lists or searches customers from the Shopify store.
func (a *listCustomersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCustomersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(params.Limit))
	if params.Query != "" {
		q.Set("query", params.Query)
	}
	if params.Fields != "" {
		q.Set("fields", params.Fields)
	}

	var resp struct {
		Customers json.RawMessage `json:"customers"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/customers.json?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
