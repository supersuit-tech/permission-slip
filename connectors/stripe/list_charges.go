package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChargesAction implements connectors.Action for stripe.list_charges.
// It lists charges via GET /v1/charges, useful for reconciliation and reporting.
type listChargesAction struct {
	conn *StripeConnector
}

type listChargesParams struct {
	CustomerID      string `json:"customer_id"`
	PaymentIntentID string `json:"payment_intent_id"`
	Limit           int    `json:"limit"`
	StartingAfter   string `json:"starting_after"`
}

const (
	defaultChargeLimit = 10
	maxChargeLimit     = 100
)

func (p *listChargesParams) validate() error {
	if p.Limit < 0 || p.Limit > maxChargeLimit {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 0 and %d (0 uses default of %d)", maxChargeLimit, defaultChargeLimit),
		}
	}
	return nil
}

// Execute lists Stripe charges with optional filters.
func (a *listChargesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChargesParams
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
	if params.PaymentIntentID != "" {
		query["payment_intent"] = params.PaymentIntentID
	}
	if params.StartingAfter != "" {
		query["starting_after"] = params.StartingAfter
	}

	limit := params.Limit
	if limit == 0 {
		limit = defaultChargeLimit
	}
	query["limit"] = fmt.Sprintf("%d", limit)

	var resp struct {
		Data    json.RawMessage `json:"data"`
		HasMore bool            `json:"has_more"`
		Object  string          `json:"object"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/charges", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
