package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getInvoiceAction struct {
	conn *PayPalConnector
}

type getInvoiceParams struct {
	InvoiceID string `json:"invoice_id"`
}

func (a *getInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("invoice_id", params.InvoiceID); err != nil {
		return nil, err
	}
	path := "/v2/invoicing/invoices/" + params.InvoiceID
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
