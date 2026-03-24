package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type refundCaptureAction struct {
	conn *PayPalConnector
}

type refundCaptureParams struct {
	CaptureID string          `json:"capture_id"`
	Body      json.RawMessage `json:"body"`
}

// Execute refunds a captured payment via POST /v2/payments/captures/{id}/refund.
func (a *refundCaptureAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params refundCaptureParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("capture_id", params.CaptureID); err != nil {
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
	path := "/v2/payments/captures/" + params.CaptureID + "/refund"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
