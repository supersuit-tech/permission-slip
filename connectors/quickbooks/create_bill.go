package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxBillLineItems is the QuickBooks API limit for line items per bill.
const maxBillLineItems = 250

// createBillAction implements connectors.Action for quickbooks.create_bill.
type createBillAction struct {
	conn *QuickBooksConnector
}

type billLineItem struct {
	Description string  `json:"description,omitempty"`
	Amount      float64 `json:"amount"`
	AccountID   string  `json:"account_id,omitempty"`
}

type createBillParams struct {
	VendorID  string         `json:"vendor_id"`
	DueDate   string         `json:"due_date,omitempty"`
	TxnDate   string         `json:"txn_date,omitempty"`
	LineItems []billLineItem `json:"line_items"`
}

func (p *createBillParams) validate() error {
	if p.VendorID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: vendor_id"}
	}
	if err := validateDate("due_date", p.DueDate); err != nil {
		return err
	}
	if err := validateDate("txn_date", p.TxnDate); err != nil {
		return err
	}
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "at least one line item is required"}
	}
	if len(p.LineItems) > maxBillLineItems {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many line items: %d (max %d)", len(p.LineItems), maxBillLineItems),
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

// Execute creates a bill (accounts payable) in QuickBooks.
func (a *createBillAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBillParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	lines := make([]map[string]any, 0, len(params.LineItems))
	for _, item := range params.LineItems {
		line := map[string]any{
			"DetailType": "AccountBasedExpenseLineDetail",
			"Amount":     item.Amount,
			"AccountBasedExpenseLineDetail": map[string]any{},
		}
		if item.AccountID != "" {
			line["AccountBasedExpenseLineDetail"] = map[string]any{
				"AccountRef": map[string]string{
					"value": item.AccountID,
				},
			}
		}
		if item.Description != "" {
			line["Description"] = item.Description
		}
		lines = append(lines, line)
	}

	body := map[string]any{
		"VendorRef": map[string]string{
			"value": params.VendorID,
		},
		"Line": lines,
	}
	if params.DueDate != "" {
		body["DueDate"] = params.DueDate
	}
	if params.TxnDate != "" {
		body["TxnDate"] = params.TxnDate
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/bill"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	bill, ok := resp["Bill"]
	if !ok {
		bill = resp
	}

	return connectors.JSONResult(bill)
}
