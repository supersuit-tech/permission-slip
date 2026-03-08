package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listInvoicesAction implements connectors.Action for quickbooks.list_invoices.
// It queries invoices using QuickBooks' SQL-like query API.
type listInvoicesAction struct {
	conn *QuickBooksConnector
}

type listInvoicesParams struct {
	CustomerID string `json:"customer_id,omitempty"`
	StartDate  string `json:"start_date,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

func (p *listInvoicesParams) validate() error {
	if err := validateQBOID("customer_id", p.CustomerID); err != nil {
		return err
	}
	if err := validateDate("start_date", p.StartDate); err != nil {
		return err
	}
	if err := validateDate("end_date", p.EndDate); err != nil {
		return err
	}
	if p.MaxResults < 0 {
		return &connectors.ValidationError{Message: "max_results must be a non-negative integer"}
	}
	return nil
}

// Execute lists invoices from QuickBooks with optional filtering.
func (a *listInvoicesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listInvoicesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var conditions []string
	if params.CustomerID != "" {
		conditions = append(conditions, "CustomerRef = '"+escapeQBOString(params.CustomerID)+"'")
	}
	if params.StartDate != "" {
		conditions = append(conditions, "TxnDate >= '"+params.StartDate+"'")
	}
	if params.EndDate != "" {
		conditions = append(conditions, "TxnDate <= '"+params.EndDate+"'")
	}

	query := "SELECT * FROM Invoice"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
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
		if invoices, ok := qr["Invoice"]; ok {
			return connectors.JSONResult(invoices)
		}
		return connectors.JSONResult([]any{})
	}

	return connectors.JSONResult(resp)
}
