package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
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

func (p *listSubscriptionsParams) validate() error {
	if err := validateEnum(p.Status, "status", []string{
		"active", "past_due", "canceled", "unpaid", "trialing", "all",
	}); err != nil {
		return err
	}
	return validateListLimit(p.Limit)
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
	query["limit"] = fmt.Sprintf("%d", resolveLimit(params.Limit))

	var resp stripeListResponse
	if err := a.conn.doGet(ctx, req.Credentials, "/v1/subscriptions", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
