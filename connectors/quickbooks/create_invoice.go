package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// maxInvoiceLineItems is the QuickBooks API limit for line items per invoice.
const maxInvoiceLineItems = 250

// createInvoiceAction implements connectors.Action for quickbooks.create_invoice.
type createInvoiceAction struct {
	conn *QuickBooksConnector
}

type invoiceLineItem struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Quantity    float64 `json:"quantity"`
}

type createInvoiceParams struct {
	CustomerID string            `json:"customer_id"`
	DueDate    string            `json:"due_date"`
	LineItems  []invoiceLineItem `json:"line_items"`
	EmailTo    string            `json:"email_to"`
}

func (p *createInvoiceParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	if err := validateDate("due_date", p.DueDate); err != nil {
		return err
	}
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "at least one line item is required"}
	}
	if len(p.LineItems) > maxInvoiceLineItems {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many line items: %d (max %d)", len(p.LineItems), maxInvoiceLineItems),
		}
	}
	for i, item := range p.LineItems {
		if item.Amount <= 0 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d].amount must be positive", i),
			}
		}
	}
	return nil
}

// Execute creates a QuickBooks invoice.
func (a *createInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build QuickBooks invoice request body.
	// QBO requires both the total line Amount (UnitPrice × Qty) and the
	// individual UnitPrice/Qty breakdown; omitting either causes a 400.
	lines := make([]map[string]any, 0, len(params.LineItems))
	for _, item := range params.LineItems {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		line := map[string]any{
			"DetailType": "SalesItemLineDetail",
			"Amount":     item.Amount * qty,
			"SalesItemLineDetail": map[string]any{
				"UnitPrice": item.Amount,
				"Qty":       qty,
			},
		}
		if item.Description != "" {
			line["Description"] = item.Description
		}
		lines = append(lines, line)
	}

	body := map[string]any{
		"CustomerRef": map[string]string{
			"value": params.CustomerID,
		},
		"Line": lines,
	}
	if params.DueDate != "" {
		body["DueDate"] = params.DueDate
	}
	if params.EmailTo != "" {
		body["BillEmail"] = map[string]string{
			"Address": params.EmailTo,
		}
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/invoice"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	// QBO wraps entity responses as {"Invoice": {...}}; extract the inner
	// object for a cleaner result. Fall back to the full response if the
	// wrapper key is absent (defensive against API version changes).
	invoice, ok := resp["Invoice"]
	if !ok {
		invoice = resp
	}

	return connectors.JSONResult(invoice)
}
