package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listInvoicesAction implements connectors.Action for stripe.list_invoices.
// It lists invoices via GET /v1/invoices, useful for reconciliation and dashboards.
type listInvoicesAction struct {
	conn *StripeConnector
}

type listInvoicesParams struct {
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	Limit         int    `json:"limit"`
	StartingAfter string `json:"starting_after"`
}

func (p *listInvoicesParams) validate() error {
	if err := validateEnum(p.Status, "status", []string{
		"draft", "open", "paid", "uncollectible", "void",
	}); err != nil {
		return err
	}
	return validateListLimit(p.Limit)
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
	if params.StartingAfter != "" {
		query["starting_after"] = params.StartingAfter
	}
	query["limit"] = fmt.Sprintf("%d", resolveLimit(params.Limit))

	var resp stripeListResponse
	if err := a.conn.doGet(ctx, req.Credentials, "/v1/invoices", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
