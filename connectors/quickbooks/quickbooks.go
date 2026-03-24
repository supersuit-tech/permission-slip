// Package quickbooks implements the QuickBooks Online connector for the
// Permission Slip connector execution layer. It uses the QuickBooks Online
// REST API with plain net/http (no third-party SDK).
//
// QuickBooks Online uses JSON request/response bodies and OAuth 2.0 Bearer
// token authentication. The API is scoped to a "realm" (company ID).
package quickbooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://quickbooks.api.intuit.com"
	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"
	credKeyRealmID     = "realm_id"

	// defaultRetryAfter is used when QuickBooks returns a rate limit response
	// without a Retry-After header.
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes limits how much of a QuickBooks API response we'll read
	// into memory. 4 MB is generous for any QuickBooks response.
	maxResponseBytes = 4 << 20 // 4 MB

	// maxErrorMessageBytes caps the raw response body included in error messages.
	maxErrorMessageBytes = 512
)

// QuickBooksConnector owns the shared HTTP client and base URL used by all
// QuickBooks actions.
type QuickBooksConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a QuickBooksConnector with sensible defaults.
func New() *QuickBooksConnector {
	return &QuickBooksConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a QuickBooksConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *QuickBooksConnector {
	return &QuickBooksConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "quickbooks", matching the connectors.id in the database.
func (c *QuickBooksConnector) ID() string { return "quickbooks" }

// Actions returns the registered action handlers keyed by action_type.
func (c *QuickBooksConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"quickbooks.create_invoice":          &createInvoiceAction{conn: c},
		"quickbooks.record_payment":          &recordPaymentAction{conn: c},
		"quickbooks.create_expense":          &createExpenseAction{conn: c},
		"quickbooks.get_profit_loss":         &getProfitLossAction{conn: c},
		"quickbooks.get_balance_sheet":       &getBalanceSheetAction{conn: c},
		"quickbooks.reconcile_transaction":   &reconcileTransactionAction{conn: c},
		"quickbooks.create_customer":         &createCustomerAction{conn: c},
		"quickbooks.list_accounts":           &listAccountsAction{conn: c},
		"quickbooks.create_vendor":           &createVendorAction{conn: c},
		"quickbooks.create_bill":             &createBillAction{conn: c},
		"quickbooks.list_invoices":           &listInvoicesAction{conn: c},
		"quickbooks.list_customers":          &listCustomersAction{conn: c},
		"quickbooks.send_invoice":            &sendInvoiceAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token and realm_id.
func (c *QuickBooksConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	realmID, ok := creds.Get(credKeyRealmID)
	if !ok || realmID == "" {
		return &connectors.ValidationError{Message: "missing required credential: realm_id"}
	}
	return nil
}

// realmID extracts the realm_id from credentials. Callers should have already
// validated credentials before calling this.
func realmID(creds connectors.Credentials) string {
	id, _ := creds.Get(credKeyRealmID)
	return id
}

// companyPath returns the base API path for a company (realm).
func companyPath(creds connectors.Credentials) string {
	return "/v3/company/" + url.PathEscape(realmID(creds))
}

// doJSON is the shared request lifecycle for all QuickBooks actions. It sends a
// JSON request with Bearer auth, handles errors, and unmarshals the response.
func (c *QuickBooksConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, path string, reqBody any, respBody any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(data)
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("QuickBooks API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "QuickBooks API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("QuickBooks API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode QuickBooks API response",
			}
		}
	}

	return nil
}

// doGet is a convenience wrapper around doJSON for GET requests.
func (c *QuickBooksConnector) doGet(ctx context.Context, creds connectors.Credentials, path string, respBody any) error {
	return c.doJSON(ctx, creds, http.MethodGet, path, nil, respBody)
}

// doPost is a convenience wrapper around doJSON for POST requests.
func (c *QuickBooksConnector) doPost(ctx context.Context, creds connectors.Credentials, path string, reqBody any, respBody any) error {
	return c.doJSON(ctx, creds, http.MethodPost, path, reqBody, respBody)
}

// dateRe matches YYYY-MM-DD date format. Used to validate user-supplied dates
// before sending to the QuickBooks API so users get clear validation errors
// instead of confusing QBO responses.
var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// validateDate checks that a date string matches YYYY-MM-DD format.
// Returns nil for empty strings (optional dates).
func validateDate(field, value string) error {
	if value == "" {
		return nil
	}
	if !dateRe.MatchString(value) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must be in YYYY-MM-DD format (got %q)", field, value),
		}
	}
	return nil
}

// emailRe validates a basic email address format (local@domain.tld).
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]{2,}$`)

// validateEmail returns a ValidationError if the value is a non-empty string
// that does not look like a valid email address.
func validateEmail(field, value string) error {
	if value == "" {
		return nil
	}
	if !emailRe.MatchString(value) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must be a valid email address (got %q)", field, value),
		}
	}
	return nil
}

// qboIDRe validates that a QuickBooks entity ID contains only digits.
// QBO entity IDs (customer IDs, vendor IDs, etc.) are always positive integers.
var qboIDRe = regexp.MustCompile(`^\d+$`)

// validateQBOID returns a ValidationError if the given entity ID is non-empty
// and contains non-digit characters. This prevents query injection in
// QuickBooks SQL-like queries where IDs are interpolated into queries.
func validateQBOID(field, value string) error {
	if value == "" {
		return nil
	}
	if !qboIDRe.MatchString(value) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must be a numeric QuickBooks entity ID (got %q)", field, value),
		}
	}
	return nil
}

// escapeQBOString escapes single quotes for use inside QuickBooks SQL-like
// query string literals by doubling them (standard SQL escaping).
func escapeQBOString(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			result = append(result, '\'', '\'')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// escapeQBOLikeString escapes a string for use in a QuickBooks SQL-like LIKE
// clause. In addition to single-quote escaping, this also escapes wildcard
// characters (% and _) with a backslash so they are treated as literals.
// Use this instead of escapeQBOString when the value appears inside a LIKE pattern.
func escapeQBOLikeString(s string) string {
	result := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			result = append(result, '\'', '\'')
		case '%', '_', '\\':
			result = append(result, '\\', s[i])
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// truncate caps s at approximately maxLen bytes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	byteCount := 0
	for _, r := range s {
		runeLen := len(string(r))
		if byteCount+runeLen > maxLen {
			break
		}
		byteCount += runeLen
	}
	return s[:byteCount] + "..."
}
