package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getInvoiceAction struct {
	conn *PayPalConnector
}

type getInvoiceParams struct {
	InvoiceID string `json:"invoice_id"`
}

func (a *getInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getInvoiceParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("invoice_id", params.InvoiceID)
	if err != nil {
		return nil, err
	}
	path := "/v2/invoicing/invoices/" + seg
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
