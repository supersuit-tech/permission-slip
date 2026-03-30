package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type captureOrderAction struct {
	conn *PayPalConnector
}

type captureOrderParams struct {
	OrderID string          `json:"order_id"`
	Body    json.RawMessage `json:"body"`
}

// Execute captures an approved order via POST /v2/checkout/orders/{id}/capture.
func (a *captureOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params captureOrderParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("order_id", params.OrderID)
	if err != nil {
		return nil, err
	}
	body, err := optionalJSONObject(params.Body, "body")
	if err != nil {
		return nil, err
	}
	path := "/v2/checkout/orders/" + seg + "/capture"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
