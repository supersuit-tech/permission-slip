package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getCustomerAction implements connectors.Action for shopify.get_customer.
// It retrieves a single customer by ID via GET /admin/api/2024-10/customers/{customer_id}.json.
type getCustomerAction struct {
	conn *ShopifyConnector
}

type getCustomerParams struct {
	CustomerID int64 `json:"customer_id"`
}

func (p *getCustomerParams) validate() error {
	if p.CustomerID <= 0 {
		return &connectors.ValidationError{Message: "customer_id must be a positive integer"}
	}
	return nil
}

// Execute retrieves a single customer by ID and returns the full customer details.
func (a *getCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp struct {
		Customer json.RawMessage `json:"customer"`
	}
	path := fmt.Sprintf("/customers/%d.json", params.CustomerID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
