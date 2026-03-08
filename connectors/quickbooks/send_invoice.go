package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendInvoiceAction implements connectors.Action for quickbooks.send_invoice.
// It emails an invoice to a customer via POST /v3/company/{realmId}/invoice/{invoiceId}/send.
type sendInvoiceAction struct {
	conn *QuickBooksConnector
}

type sendInvoiceParams struct {
	InvoiceID string `json:"invoice_id"`
	EmailTo   string `json:"email_to,omitempty"`
}

func (p *sendInvoiceParams) validate() error {
	if p.InvoiceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: invoice_id"}
	}
	return nil
}

// Execute sends an invoice email to the customer via QuickBooks.
func (a *sendInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := companyPath(req.Credentials) + "/invoice/" + params.InvoiceID + "/send"
	if params.EmailTo != "" {
		path += "?sendTo=" + params.EmailTo
	}

	var resp map[string]any
	if err := a.conn.doPost(ctx, req.Credentials, path, nil, &resp); err != nil {
		return nil, err
	}

	invoice, ok := resp["Invoice"]
	if !ok {
		invoice = resp
	}

	return connectors.JSONResult(invoice)
}
