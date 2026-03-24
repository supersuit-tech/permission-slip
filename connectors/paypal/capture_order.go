package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("order_id", params.OrderID); err != nil {
		return nil, err
	}
	var body map[string]any
	if len(params.Body) > 0 {
		var err error
		body, err = readJSONBody(params.Body, "body")
		if err != nil {
			return nil, err
		}
	}
	path := "/v2/checkout/orders/" + params.OrderID + "/capture"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
