package paypal

import (
	"context"
	"encoding/json"
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
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("capture_id", params.CaptureID)
	if err != nil {
		return nil, err
	}
	body, err := optionalJSONObject(params.Body, "body")
	if err != nil {
		return nil, err
	}
	path := "/v2/payments/captures/" + seg + "/refund"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
