package plaid

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTransactionsAction implements connectors.Action for plaid.list_transactions.
// It retrieves transactions via POST /transactions/get.
type listTransactionsAction struct {
	conn *PlaidConnector
}

type listTransactionsParams struct {
	AccessToken string   `json:"access_token"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	AccountIDs  []string `json:"account_ids"`
	Count       *int     `json:"count,omitempty"`
	Offset      *int     `json:"offset,omitempty"`
}

// datePattern matches YYYY-MM-DD date format.
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

const maxTransactionCount = 500

func (p *listTransactionsParams) validate() error {
	if p.AccessToken == "" {
		return &connectors.ValidationError{Message: "missing required parameter: access_token"}
	}
	if p.StartDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_date"}
	}
	startNorm, err := connectors.NormalizePlaidDateParam(p.StartDate)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid start_date: %v", err)}
	}
	p.StartDate = startNorm
	if p.EndDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end_date"}
	}
	endNorm, err := connectors.NormalizePlaidDateParam(p.EndDate)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid end_date: %v", err)}
	}
	p.EndDate = endNorm
	if p.StartDate > p.EndDate {
		return &connectors.ValidationError{Message: "start_date must be before or equal to end_date"}
	}
	if p.Count != nil && (*p.Count < 1 || *p.Count > maxTransactionCount) {
		return &connectors.ValidationError{Message: fmt.Sprintf("count must be between 1 and %d", maxTransactionCount)}
	}
	if p.Offset != nil && *p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

// Execute retrieves transactions and returns the transaction data.
func (a *listTransactionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTransactionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"access_token": params.AccessToken,
		"start_date":   params.StartDate,
		"end_date":     params.EndDate,
	}

	options := map[string]any{}
	if len(params.AccountIDs) > 0 {
		options["account_ids"] = params.AccountIDs
	}
	if params.Count != nil {
		options["count"] = *params.Count
	}
	if params.Offset != nil {
		options["offset"] = *params.Offset
	}
	if len(options) > 0 {
		body["options"] = options
	}

	var resp json.RawMessage
	if err := a.conn.doPost(ctx, req.Credentials, "/transactions/get", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
