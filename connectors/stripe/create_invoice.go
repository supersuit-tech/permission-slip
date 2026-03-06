package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createInvoiceAction implements connectors.Action for stripe.create_invoice.
// It creates an invoice via POST /v1/invoices, adds line items via
// POST /v1/invoiceitems, and optionally finalizes via POST /v1/invoices/{id}/finalize.
type createInvoiceAction struct {
	conn *StripeConnector
}

type invoiceLineItem struct {
	Description string `json:"description"`
	Amount      any    `json:"amount"`
	Quantity    any    `json:"quantity"`
}

type createInvoiceParams struct {
	CustomerID  string            `json:"customer_id"`
	Description string            `json:"description"`
	DueDate     any               `json:"due_date"`
	AutoAdvance *bool             `json:"auto_advance"`
	LineItems   []invoiceLineItem `json:"line_items"`
	Metadata    map[string]any    `json:"metadata"`
}

func (p *createInvoiceParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	return nil
}

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
	if params.DueDate != nil {
		invoiceBody["due_date"] = params.DueDate
	}
	if params.AutoAdvance != nil {
		invoiceBody["auto_advance"] = *params.AutoAdvance
	}
	if len(params.Metadata) > 0 {
		invoiceBody["metadata"] = params.Metadata
	}

	var invoiceResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	idempotencyKey := deriveIdempotencyKey(req.ActionType, req.Parameters)

	if err := a.conn.do(ctx, req.Credentials, "POST", "/v1/invoices", formEncode(invoiceBody), &invoiceResp, idempotencyKey); err != nil {
		return nil, err
	}

	// Step 2: Add line items.
	for i, item := range params.LineItems {
		itemBody := map[string]any{
			"invoice": invoiceResp.ID,
		}
		if item.Description != "" {
			itemBody["description"] = item.Description
		}
		if item.Amount != nil {
			itemBody["amount"] = item.Amount
		}
		if item.Quantity != nil {
			itemBody["quantity"] = item.Quantity
		}
		// Use the invoice currency — Stripe requires currency on invoice items.
		itemBody["currency"] = "usd"

		// Derive a unique idempotency key per line item using the index.
		itemKey := deriveIdempotencyKey(
			req.ActionType+"/invoiceitem",
			json.RawMessage(fmt.Sprintf(`{"invoice":"%s","index":%d}`, invoiceResp.ID, i)),
		)

		if err := a.conn.do(ctx, req.Credentials, "POST", "/v1/invoiceitems", formEncode(itemBody), nil, itemKey); err != nil {
			return nil, fmt.Errorf("adding line item %d: %w", i, err)
		}
	}

	// Step 3: Finalize the invoice if auto_advance is true (the default).
	shouldFinalize := params.AutoAdvance == nil || *params.AutoAdvance
	if shouldFinalize {
		finalizeKey := deriveIdempotencyKey(
			req.ActionType+"/finalize",
			json.RawMessage(fmt.Sprintf(`{"invoice":"%s"}`, invoiceResp.ID)),
		)

		var finalResp struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}

		path := fmt.Sprintf("/v1/invoices/%s/finalize", invoiceResp.ID)
		if err := a.conn.do(ctx, req.Credentials, "POST", path, nil, &finalResp, finalizeKey); err != nil {
			return nil, fmt.Errorf("finalizing invoice: %w", err)
		}

		return connectors.JSONResult(finalResp)
	}

	return connectors.JSONResult(invoiceResp)
}
