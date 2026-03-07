package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validAccountTypes are the QuickBooks account types accepted for filtering.
var validAccountTypes = map[string]bool{
	"Bank":                true,
	"Accounts Receivable": true,
	"Other Current Asset": true,
	"Fixed Asset":         true,
	"Other Asset":         true,
	"Accounts Payable":    true,
	"Credit Card":         true,
	"Other Current Liability": true,
	"Long Term Liability": true,
	"Equity":              true,
	"Income":              true,
	"Cost of Goods Sold":  true,
	"Expense":             true,
	"Other Income":        true,
	"Other Expense":       true,
}

// listAccountsAction implements connectors.Action for quickbooks.list_accounts.
// It queries the Chart of Accounts using QuickBooks' query API.
type listAccountsAction struct {
	conn *QuickBooksConnector
}

type listAccountsParams struct {
	AccountType string `json:"account_type"`
	MaxResults  int    `json:"max_results"`
}

// Execute lists accounts (chart of accounts) from QuickBooks.
func (a *listAccountsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listAccountsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	query := "SELECT * FROM Account"
	if params.AccountType != "" {
		// Validate account type against known values to prevent query injection.
		if !validAccountTypes[params.AccountType] {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("invalid account_type %q; valid types: %s",
					params.AccountType, strings.Join(accountTypeNames(), ", ")),
			}
		}
		query += " WHERE AccountType = '" + params.AccountType + "'"
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

	// Extract QueryResponse.Account array for cleaner output.
	if qr, ok := resp["QueryResponse"].(map[string]any); ok {
		if accounts, ok := qr["Account"]; ok {
			return connectors.JSONResult(accounts)
		}
		// No accounts found — return empty array.
		return connectors.JSONResult([]any{})
	}

	return connectors.JSONResult(resp)
}

// accountTypeNames returns the valid account type names sorted for error messages.
func accountTypeNames() []string {
	names := make([]string, 0, len(validAccountTypes))
	for name := range validAccountTypes {
		names = append(names, name)
	}
	return names
}
