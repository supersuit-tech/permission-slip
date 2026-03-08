package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listInvoicesAction implements connectors.Action for stripe.list_invoices.
// It lists invoices via GET /v1/invoices, useful for reconciliation and dashboards.
type listInvoicesAction struct {
	conn *StripeConnector
}

type listInvoicesParams struct {
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	Limit      int    `json:"limit"`
}

const (
	defaultInvoiceLimit = 10
	maxInvoiceLimit     = 100
)

func (p *listInvoicesParams) validate() error {
	if err := validateEnum(p.Status, "status", []string{
		"draft", "open", "paid", "uncollectible", "void",
	}); err != nil {
		return err
	}
	if p.Limit < 0 || p.Limit > maxInvoiceLimit {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 0 and %d (0 uses default of %d)", maxInvoiceLimit, defaultInvoiceLimit),
		}
	}
	return nil
}

// Execute lists Stripe invoices with optional filters.
func (a *listInvoicesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listInvoicesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := map[string]string{}
	if params.CustomerID != "" {
		query["customer"] = params.CustomerID
	}
	if params.Status != "" {
		query["status"] = params.Status
	}

	limit := params.Limit
	if limit == 0 {
		limit = defaultInvoiceLimit
	}
	query["limit"] = fmt.Sprintf("%d", limit)

	var resp struct {
		Data    json.RawMessage `json:"data"`
		HasMore bool            `json:"has_more"`
		Object  string          `json:"object"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/invoices", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
