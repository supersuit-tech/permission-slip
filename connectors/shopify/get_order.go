package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getOrderAction implements connectors.Action for shopify.get_order.
// It retrieves a single order by ID via GET /admin/api/2024-10/orders/{order_id}.json.
type getOrderAction struct {
	conn *ShopifyConnector
}

type getOrderParams struct {
	OrderID int64 `json:"order_id"`
}

func (p *getOrderParams) validate() error {
	if p.OrderID <= 0 {
		return &connectors.ValidationError{Message: "order_id must be a positive integer"}
	}
	return nil
}

func (a *getOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp struct {
		Order json.RawMessage `json:"order"`
	}
	path := fmt.Sprintf("/orders/%d.json", params.OrderID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
