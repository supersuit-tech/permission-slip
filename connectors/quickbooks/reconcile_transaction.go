package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// reconcileTransactionAction implements connectors.Action for
// quickbooks.reconcile_transaction. It marks a bank deposit as reconciled
// by creating a Deposit object linked to an account.
type reconcileTransactionAction struct {
	conn *QuickBooksConnector
}

type reconcileTransactionParams struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	TxnDate     string  `json:"txn_date"`
	Description string  `json:"description"`
}

func (p *reconcileTransactionParams) validate() error {
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id"}
	}
	if p.Amount <= 0 {
		return &connectors.ValidationError{Message: "amount must be positive"}
	}
	if err := validateDate("txn_date", p.TxnDate); err != nil {
		return err
	}
	return nil
}

// Execute creates a Deposit in QuickBooks to reconcile a bank transaction.
func (a *reconcileTransactionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params reconcileTransactionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	line := map[string]any{
		"DetailType": "DepositLineDetail",
		"Amount":     params.Amount,
		"DepositLineDetail": map[string]any{
			"AccountRef": map[string]string{
				"value": params.AccountID,
			},
		},
	}
	if params.Description != "" {
		line["Description"] = params.Description
	}

	body := map[string]any{
		"DepositToAccountRef": map[string]string{
			"value": params.AccountID,
		},
		"Line": []map[string]any{line},
	}
	if params.TxnDate != "" {
		body["TxnDate"] = params.TxnDate
	}

	var resp map[string]any
	path := companyPath(req.Credentials) + "/deposit"
	if err := a.conn.doPost(ctx, req.Credentials, path, body, &resp); err != nil {
		return nil, err
	}

	deposit, ok := resp["Deposit"]
	if !ok {
		deposit = resp
	}

	return connectors.JSONResult(deposit)
}
