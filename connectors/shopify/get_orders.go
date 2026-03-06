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

// getOrdersAction implements connectors.Action for shopify.get_orders.
// It lists or filters orders via GET /admin/api/2024-10/orders.json.
type getOrdersAction struct {
	conn *ShopifyConnector
}

type getOrdersParams struct {
	Status          string `json:"status"`
	FinancialStatus string `json:"financial_status"`
	CreatedAtMin    string `json:"created_at_min"`
	CreatedAtMax    string `json:"created_at_max"`
	Limit           int    `json:"limit"`
}

var validOrderStatuses = map[string]bool{
	"open": true, "closed": true, "cancelled": true, "any": true,
}

var validFinancialStatuses = map[string]bool{
	"paid": true, "unpaid": true, "partially_paid": true, "refunded": true,
	"authorized": true, "pending": true, "any": true,
}

func (p *getOrdersParams) validate() error {
	if p.Status == "" {
		p.Status = "open"
	}
	if !validOrderStatuses[p.Status] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid status %q: must be open, closed, cancelled, or any", p.Status)}
	}
	if p.FinancialStatus != "" && !validFinancialStatuses[p.FinancialStatus] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid financial_status %q: must be paid, unpaid, partially_paid, refunded, authorized, pending, or any", p.FinancialStatus)}
	}
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.Limit < 1 || p.Limit > 250 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 250, got %d", p.Limit)}
	}
	return nil
}

func (a *getOrdersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getOrdersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("status", params.Status)
	q.Set("limit", strconv.Itoa(params.Limit))
	if params.FinancialStatus != "" {
		q.Set("financial_status", params.FinancialStatus)
	}
	if params.CreatedAtMin != "" {
		q.Set("created_at_min", params.CreatedAtMin)
	}
	if params.CreatedAtMax != "" {
		q.Set("created_at_max", params.CreatedAtMax)
	}

	var resp struct {
		Orders json.RawMessage `json:"orders"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/orders.json?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
