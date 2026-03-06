package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createInvoiceAction implements connectors.Action for stripe.create_invoice.
// It creates an invoice, adds line items, and optionally finalizes it.
// This is a multi-step flow: POST /v1/invoices → POST /v1/invoiceitems (per item)
// → optionally POST /v1/invoices/{id}/finalize.
type createInvoiceAction struct {
	conn *StripeConnector
}

type invoiceLineItem struct {
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	Quantity    int64  `json:"quantity"`
}

type createInvoiceParams struct {
	CustomerID  string            `json:"customer_id"`
	Description string            `json:"description"`
	DueDate     int64             `json:"due_date"`
	AutoAdvance *bool             `json:"auto_advance"`
	LineItems   []invoiceLineItem `json:"line_items"`
	Metadata    map[string]any    `json:"metadata"`
}

func (p *createInvoiceParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
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

// Execute creates a Stripe invoice with optional line items and finalization.
func (a *createInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the invoice.
	invoiceBody := map[string]any{
		"customer": params.CustomerID,
	}
	if params.Description != "" {
		invoiceBody["description"] = params.Description
	}
	if params.DueDate != 0 {
		invoiceBody["due_date"] = params.DueDate
	}
	if params.AutoAdvance != nil {
		invoiceBody["auto_advance"] = *params.AutoAdvance
	}
	if params.Metadata != nil {
		invoiceBody["metadata"] = params.Metadata
	}

	var invoiceResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/invoices", formEncode(invoiceBody), &invoiceResp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	// Step 2: Add line items.
	for _, item := range params.LineItems {
		itemBody := map[string]any{
			"invoice":  invoiceResp.ID,
			"amount":   item.Amount,
			"currency": "usd",
		}
		if item.Description != "" {
			itemBody["description"] = item.Description
		}
		if item.Quantity > 0 {
			itemBody["quantity"] = item.Quantity
		}

		// Each invoice item gets its own idempotency key derived from the
		// invoice ID + item index to ensure retry safety.
		itemParams, _ := json.Marshal(item)
		itemKey := deriveIdempotencyKey(req.ActionType+".item."+invoiceResp.ID, itemParams)

		if err := a.conn.do(ctx, req.Credentials, "POST", "/v1/invoiceitems", formEncode(itemBody), nil, itemKey); err != nil {
			return nil, err
		}
	}

	// Step 3: Optionally finalize the invoice.
	// auto_advance defaults to true — finalize when true or unset.
	if params.AutoAdvance == nil || *params.AutoAdvance {
		var finalResp struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		finalizeKey := deriveIdempotencyKey(req.ActionType+".finalize."+invoiceResp.ID, req.Parameters)
		if err := a.conn.do(ctx, req.Credentials, "POST", "/v1/invoices/"+invoiceResp.ID+"/finalize", nil, &finalResp, finalizeKey); err != nil {
			return nil, err
		}
		return connectors.JSONResult(finalResp)
	}

	return connectors.JSONResult(invoiceResp)
}
