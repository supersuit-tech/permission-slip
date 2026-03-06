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
	Limit      any    `json:"limit"`
}

func (p *listSubscriptionsParams) validate() error {
	if p.Status != "" {
		switch p.Status {
		case "active", "past_due", "canceled", "all":
			// valid
		default:
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid status %q: must be one of active, past_due, canceled, all", p.Status),
			}
		}
	}
	return nil
}

func (a *listSubscriptionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSubscriptionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := map[string]any{}
	if params.CustomerID != "" {
		query["customer"] = params.CustomerID
	}
	if params.Status != "" {
		query["status"] = params.Status
	}
	if params.PriceID != "" {
		query["price"] = params.PriceID
	}
	if params.Limit != nil {
		query["limit"] = params.Limit
	} else {
		query["limit"] = "10"
	}

	flat := formEncode(query)

	var resp struct {
		Data    json.RawMessage `json:"data"`
		HasMore bool            `json:"has_more"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/subscriptions", flat, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
