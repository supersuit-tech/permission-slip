package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("invoice_id", params.InvoiceID); err != nil {
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
	path := "/v2/invoicing/invoices/" + params.InvoiceID + "/send"
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, path, body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
