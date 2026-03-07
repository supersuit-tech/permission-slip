package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createExpenseAction implements connectors.Action for quickbooks.create_expense.
// In QuickBooks, expenses are represented as Purchase objects with PaymentType "Cash".
type createExpenseAction struct {
	conn *QuickBooksConnector
}

type expenseLineItem struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	AccountID   string  `json:"account_id"`
}

type createExpenseParams struct {
	AccountID   string            `json:"account_id"`
	PaymentType string            `json:"payment_type"`
	Lines       []expenseLineItem `json:"lines"`
	VendorID    string            `json:"vendor_id"`
	TxnDate     string            `json:"txn_date"`
}

func (p *createExpenseParams) validate() error {
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id"}
	}
	if len(p.Lines) == 0 {
		return &connectors.ValidationError{Message: "at least one line is required"}
	}
	for i, line := range p.Lines {
		if line.Amount <= 0 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("lines[%d].amount must be positive", i),
			}
		}
	}
	return nil
}

// Execute creates a Purchase (expense) in QuickBooks.
func (a *createExpenseAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createExpenseParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	lines := make([]map[string]any, 0, len(params.Lines))
	for _, line := range params.Lines {
		l := map[string]any{
			"DetailType": "AccountBasedExpenseLineDetail",
			"Amount":     line.Amount,
			"AccountBasedExpenseLineDetail": map[string]any{
				"AccountRef": map[string]string{
					"value": line.AccountID,
				},
			},
		}
		if line.Description != "" {
			l["Description"] = line.Description
		}
		lines = append(lines, l)
	}

	paymentType := params.PaymentType
	if paymentType == "" {
		paymentType = "Cash"
	}

	body := map[string]any{
		"AccountRef": map[string]string{
			"value": params.AccountID,
		},
		"PaymentType": paymentType,
		"Line":        lines,
	}

	if params.VendorID != "" {
		body["EntityRef"] = map[string]any{
			"value": params.VendorID,
			"type":  "Vendor",
		}
	}
	if params.TxnDate != "" {
		body["TxnDate"] = params.TxnDate
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/purchase"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	purchase, ok := resp["Purchase"]
	if !ok {
		purchase = resp
	}

	return connectors.JSONResult(purchase)
}
