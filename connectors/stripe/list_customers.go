package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

const (
	defaultCustomerLimit = 10
	maxCustomerLimit     = 100
)

func (p *listCustomersParams) validate() error {
	if p.Limit < 0 || p.Limit > maxCustomerLimit {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 0 and %d (0 uses default of %d)", maxCustomerLimit, defaultCustomerLimit),
		}
	}
	return nil
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

	limit := params.Limit
	if limit == 0 {
		limit = defaultCustomerLimit
	}
	query["limit"] = fmt.Sprintf("%d", limit)

	var resp struct {
		Data    json.RawMessage `json:"data"`
		HasMore bool            `json:"has_more"`
		Object  string          `json:"object"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/customers", query, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
