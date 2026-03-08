package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getCustomerAction implements connectors.Action for square.get_customer.
// It retrieves a customer by ID via GET /v2/customers/{customer_id}.
type getCustomerAction struct {
	conn *SquareConnector
}

type getSquareCustomerParams struct {
	CustomerID string `json:"customer_id"`
}

func (p *getSquareCustomerParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	return nil
}

// Execute retrieves a single customer by ID from Square.
func (a *getCustomerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getSquareCustomerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp struct {
		Customer json.RawMessage `json:"customer"`
	}
	path := "/customers/" + params.CustomerID
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
