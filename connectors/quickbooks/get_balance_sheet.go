package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBalanceSheetAction implements connectors.Action for quickbooks.get_balance_sheet.
type getBalanceSheetAction struct {
	conn *QuickBooksConnector
}

type getBalanceSheetParams struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Execute retrieves the Balance Sheet report from QuickBooks.
func (a *getBalanceSheetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getBalanceSheetParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	path := companyPath(req.Credentials) + "/reports/BalanceSheet"
	q := url.Values{}
	if params.StartDate != "" {
		q.Set("start_date", params.StartDate)
	}
	if params.EndDate != "" {
		q.Set("end_date", params.EndDate)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	var resp map[string]any
	if err := a.conn.doGet(ctx, req.Credentials, path, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
