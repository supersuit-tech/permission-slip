package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getCustomerAction implements connectors.Action for stripe.get_customer.
// It retrieves a single customer by ID via GET /v1/customers/{id}.
type getCustomerAction struct {
	conn *StripeConnector
}

type getCustomerParams struct {
	CustomerID string `json:"customer_id"`
}

func (p *getCustomerParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	return nil
}

// Execute retrieves a Stripe customer by ID.
func (a *getCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	escapedID := url.PathEscape(params.CustomerID)

	var resp struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Phone       string `json:"phone"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/customers/"+escapedID, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
