// Package docusign implements the DocuSign connector for the Permission Slip
// connector execution layer. It uses the DocuSign eSignature REST API v2.1
// with plain net/http (no third-party SDK).
package docusign

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://demo.docusign.net/restapi/v2.1"
	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"
	credKeyAccountID   = "account_id"
	credKeyBaseURL     = "base_url" // optional — override for production or other regions

	// defaultRetryAfter is used when the DocuSign API returns a rate limit
	// response without a Retry-After header.
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the DocuSign API response body at 50 MB to prevent
	// memory exhaustion (signed PDFs can be large).
	maxResponseBytes = 50 << 20 // 50 MB
)

// DocuSignConnector owns the shared HTTP client and base URL used by all
// DocuSign actions.
type DocuSignConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a DocuSignConnector with sensible defaults.
func New() *DocuSignConnector {
	return &DocuSignConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a DocuSignConnector pointing at a test server.
func newForTest(client *http.Client, baseURL string) *DocuSignConnector {
	return &DocuSignConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "docusign", matching the connectors.id in the database.
func (c *DocuSignConnector) ID() string { return "docusign" }

// Actions returns the registered action handlers keyed by action_type.
func (c *DocuSignConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"docusign.create_envelope":   &createEnvelopeAction{conn: c},
		"docusign.send_envelope":     &sendEnvelopeAction{conn: c},
		"docusign.check_status":      &checkStatusAction{conn: c},
		"docusign.download_signed":   &downloadSignedAction{conn: c},
		"docusign.list_templates":    &listTemplatesAction{conn: c},
		"docusign.void_envelope":     &voidEnvelopeAction{conn: c},
		"docusign.update_recipients": &updateRecipientsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token and account_id.
func (c *DocuSignConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	accountID, ok := creds.Get(credKeyAccountID)
	if !ok || accountID == "" {
		return &connectors.ValidationError{Message: "missing required credential: account_id"}
	}
	return nil
}

// accountPath builds the account-scoped API path prefix with proper escaping.
func accountPath(accountID string) string {
	return "/accounts/" + url.PathEscape(accountID)
}

// validatable is implemented by all action parameter types.
type validatable interface {
	validate() error
}

// parseParams unmarshals JSON parameters, validates them, and extracts the
// account ID from credentials. This eliminates the repeated boilerplate across
// all action Execute methods.
func parseParams(req connectors.ActionRequest, params validatable) (string, error) {
	if err := json.Unmarshal(req.Parameters, params); err != nil {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return "", err
	}
	return requireAccountID(req.Credentials)
}

// requireAccountID extracts and validates the account_id credential.
// All actions require an account ID; silently ignoring a missing value would
// produce confusing 404s from DocuSign.
func requireAccountID(creds connectors.Credentials) (string, error) {
	accountID, ok := creds.Get(credKeyAccountID)
	if !ok || isBlank(accountID) {
		return "", &connectors.ValidationError{Message: "missing required credential: account_id"}
	}
	return accountID, nil
}

// isBlank returns true if s is empty or contains only whitespace.
// Used by validate() methods to reject whitespace-only required fields.
func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
