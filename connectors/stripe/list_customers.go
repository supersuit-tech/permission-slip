package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listCustomersAction implements connectors.Action for stripe.list_customers.
// It lists/searches customers via GET /v1/customers.
// Useful for checking whether a customer already exists before creating a duplicate.
type listCustomersAction struct {
	conn *StripeConnector
}

type listCustomersParams struct {
	Email         string `json:"email"`
	Limit         int    `json:"limit"`
	StartingAfter string `json:"starting_after"`
}

func (p *listCustomersParams) validate() error {
	return validateListLimit(p.Limit)
}

// Execute lists Stripe customers with optional filters.
func (a *listCustomersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCustomersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := map[string]string{}
	if params.Email != "" {
		query["email"] = params.Email
	}
	if params.StartingAfter != "" {
		query["starting_after"] = params.StartingAfter
	}
	query["limit"] = fmt.Sprintf("%d", resolveLimit(params.Limit))

	var resp stripeListResponse
	if err := a.conn.doGet(ctx, req.Credentials, "/v1/customers", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
