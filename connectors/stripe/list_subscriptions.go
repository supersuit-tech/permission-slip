package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listSubscriptionsAction implements connectors.Action for stripe.list_subscriptions.
// It lists subscriptions via GET /v1/subscriptions.
type listSubscriptionsAction struct {
	conn *StripeConnector
}

type listSubscriptionsParams struct {
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	PriceID    string `json:"price_id"`
	Limit      int    `json:"limit"`
}

const (
	defaultSubscriptionLimit = 10
	maxSubscriptionLimit     = 100
)

func (p *listSubscriptionsParams) validate() error {
	validStatuses := map[string]bool{
		"active": true, "past_due": true, "canceled": true,
		"unpaid": true, "trialing": true, "all": true, "": true,
	}
	if !validStatuses[p.Status] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid status %q: must be one of active, past_due, canceled, unpaid, trialing, all", p.Status),
		}
	}
	if p.Limit < 0 || p.Limit > maxSubscriptionLimit {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 1 and %d", maxSubscriptionLimit),
		}
	}
	return nil
}

// Execute lists Stripe subscriptions with optional filters.
func (a *listSubscriptionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSubscriptionsParams
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
	if params.PriceID != "" {
		query["price"] = params.PriceID
	}

	limit := params.Limit
	if limit == 0 {
		limit = defaultSubscriptionLimit
	}
	query["limit"] = fmt.Sprintf("%d", limit)

	var resp struct {
		Data    json.RawMessage `json:"data"`
		HasMore bool            `json:"has_more"`
		Object  string          `json:"object"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/subscriptions", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
