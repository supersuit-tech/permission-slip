package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type sendInvoiceAction struct {
	conn *PayPalConnector
}

type sendInvoiceParams struct {
	InvoiceID string          `json:"invoice_id"`
	Body      json.RawMessage `json:"body"`
}

func (a *sendInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendInvoiceParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("invoice_id", params.InvoiceID)
	if err != nil {
		return nil, err
	}
	body, err := optionalJSONObject(params.Body, "body")
	if err != nil {
		return nil, err
	}
	path := "/v2/invoicing/invoices/" + seg + "/send"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
