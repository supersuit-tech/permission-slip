package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getProfitLossAction implements connectors.Action for quickbooks.get_profit_loss.
type getProfitLossAction struct {
	conn *QuickBooksConnector
}

type getProfitLossParams struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Execute retrieves the Profit & Loss report from QuickBooks.
func (a *getProfitLossAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getProfitLossParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	if err := validateDate("start_date", params.StartDate); err != nil {
		return nil, err
	}
	if err := validateDate("end_date", params.EndDate); err != nil {
		return nil, err
	}

	path := companyPath(req.Credentials) + "/reports/ProfitAndLoss"
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
