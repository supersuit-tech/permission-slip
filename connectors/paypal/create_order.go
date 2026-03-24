package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createOrderAction struct {
	conn *PayPalConnector
}

type createOrderParams struct {
	Order json.RawMessage `json:"order"`
}

// Execute creates a checkout order via POST /v2/checkout/orders.
func (a *createOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	body, err := readJSONBody(params.Order, "order")
	if err != nil {
		return nil, err
	}
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, "/v2/checkout/orders", body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
