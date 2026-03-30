package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// recordPaymentAction implements connectors.Action for quickbooks.record_payment.
type recordPaymentAction struct {
	conn *QuickBooksConnector
}

type recordPaymentParams struct {
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	InvoiceID  string  `json:"invoice_id"`
}

func (p *recordPaymentParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	if p.Amount <= 0 {
		return &connectors.ValidationError{Message: "amount must be positive"}
	}
	return nil
}

// Execute records a payment in QuickBooks.
func (a *recordPaymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params recordPaymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"CustomerRef": map[string]string{
			"value": params.CustomerID,
		},
		"TotalAmt": params.Amount,
	}

	// If an invoice ID is provided, link the payment to that invoice.
	if params.InvoiceID != "" {
		body["Line"] = []map[string]any{
			{
				"Amount": params.Amount,
				"LinkedTxn": []map[string]string{
					{
						"TxnId":   params.InvoiceID,
						"TxnType": "Invoice",
					},
				},
			},
		}
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/payment"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	payment, ok := resp["Payment"]
	if !ok {
		payment = resp
	}

	return connectors.JSONResult(payment)
}
