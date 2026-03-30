package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createInvoiceAction struct {
	conn *PayPalConnector
}

type createInvoiceParams struct {
	Invoice json.RawMessage `json:"invoice"`
}

func (a *createInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInvoiceParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	body, err := readJSONBody(params.Invoice, "invoice")
	if err != nil {
		return nil, err
	}
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, "/v2/invoicing/invoices", body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
