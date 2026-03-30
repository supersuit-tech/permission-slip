package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listCustomersAction implements connectors.Action for quickbooks.list_customers.
// It queries customer records using QuickBooks' SQL-like query API.
type listCustomersAction struct {
	conn *QuickBooksConnector
}

type listQBCustomersParams struct {
	DisplayName string `json:"display_name,omitempty"`
	MaxResults  int    `json:"max_results,omitempty"`
}

// Execute lists customer records from QuickBooks.
func (a *listCustomersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listQBCustomersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if params.MaxResults < 0 {
		return nil, &connectors.ValidationError{Message: "max_results must be a non-negative integer"}
	}

	query := "SELECT * FROM Customer WHERE Active = true"
	if params.DisplayName != "" {
		query += " AND DisplayName LIKE '%" + escapeQBOLikeString(params.DisplayName) + "%'"
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}
	if maxResults > 1000 {
		maxResults = 1000
	}
	query += fmt.Sprintf(" MAXRESULTS %d", maxResults)

	path := companyPath(req.Credentials) + "/query?query=" + url.QueryEscape(query)

	var resp map[string]any
	if err := a.conn.doGet(ctx, req.Credentials, path, &resp); err != nil {
		return nil, err
	}

	if qr, ok := resp["QueryResponse"].(map[string]any); ok {
		if customers, ok := qr["Customer"]; ok {
			return connectors.JSONResult(customers)
		}
		return connectors.JSONResult([]any{})
	}

	return connectors.JSONResult(resp)
}
