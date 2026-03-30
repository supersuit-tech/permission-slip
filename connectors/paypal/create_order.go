package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	if err := parseParams(req, &params); err != nil {
		return nil, err
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
