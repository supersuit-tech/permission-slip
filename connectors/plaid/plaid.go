// Package plaid implements the Plaid connector for the Permission Slip
// connector execution layer. It uses the Plaid REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	sandboxBaseURL    = "https://sandbox.plaid.com"
	productionBaseURL = "https://production.plaid.com"
	defaultTimeout    = 30 * time.Second

	// plaidAPIVersion pins the Plaid API version to prevent breaking changes
	// when Plaid releases new versions. Update deliberately after testing.
	// See https://plaid.com/docs/api/versioning/
	plaidAPIVersion = "2020-09-14"

	credKeyClientID    = "client_id"
	credKeySecret      = "secret"
	credKeyEnvironment = "environment"

	// defaultRetryAfter is used when Plaid returns a rate limit response
	// without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBody prevents memory exhaustion from a malicious or
	// misconfigured Plaid API. 1 MiB is well above any legitimate response
	// (account lists are sparse, transaction lists page at 500 items).
	maxResponseBody = 1 << 20 // 1 MiB

	// clientIDMinLen is the minimum length for a Plaid client_id.
	clientIDMinLen = 20
	// secretMinLen is the minimum length for a Plaid secret.
	secretMinLen = 20
)

// PlaidConnector owns the shared HTTP client and base URL used by all
// Plaid actions.
type PlaidConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a PlaidConnector with sensible defaults (30s timeout,
// sandbox base URL). The actual API URL used at request time depends on
// the "environment" credential — see baseURLForCreds.
func New() *PlaidConnector {
	return &PlaidConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: sandboxBaseURL,
	}
}

// newForTest creates a PlaidConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *PlaidConnector {
	return &PlaidConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "plaid", matching the connectors.id in the database.
func (c *PlaidConnector) ID() string { return "plaid" }

// Actions returns the registered action handlers keyed by action_type.
func (c *PlaidConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"plaid.create_link_token":  &createLinkTokenAction{conn: c},
		"plaid.get_balances":       &accessTokenAction{conn: c, path: "/accounts/balance/get"},
		"plaid.list_transactions":  &listTransactionsAction{conn: c},
		"plaid.get_accounts":       &accessTokenAction{conn: c, path: "/accounts/get"},
		"plaid.get_identity":       &accessTokenAction{conn: c, path: "/identity/get"},
		"plaid.get_institution":    &getInstitutionAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid client_id, secret, and optional environment.
func (c *PlaidConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	clientID, ok := creds.Get(credKeyClientID)
	if !ok || clientID == "" {
		return &connectors.ValidationError{Message: "missing required credential: client_id"}
	}
	if len(clientID) < clientIDMinLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("client_id must be at least %d characters", clientIDMinLen)}
	}

	secret, ok := creds.Get(credKeySecret)
	if !ok || secret == "" {
		return &connectors.ValidationError{Message: "missing required credential: secret"}
	}
	if len(secret) < secretMinLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("secret must be at least %d characters", secretMinLen)}
	}

	if env, ok := creds.Get(credKeyEnvironment); ok && env != "" {
		if env != "sandbox" && env != "production" {
			return &connectors.ValidationError{Message: "environment must be \"sandbox\" or \"production\""}
		}
	}
	return nil
}

// baseURLForCreds returns the appropriate base URL based on the environment
// credential. Defaults to sandbox for safety. Falls back to the connector's
// configured baseURL for test overrides via newForTest.
func (c *PlaidConnector) baseURLForCreds(creds connectors.Credentials) string {
	// If baseURL was overridden (e.g., for tests), always use the override.
	if c.baseURL != sandboxBaseURL && c.baseURL != productionBaseURL {
		return c.baseURL
	}
	if env, ok := creds.Get(credKeyEnvironment); ok && env == "production" {
		return productionBaseURL
	}
	return sandboxBaseURL
}

// doPost sends a JSON POST request to the Plaid API. Plaid's API uses
// JSON request bodies with client_id and secret included in the body
// (not as headers or basic auth).
func (c *PlaidConnector) doPost(ctx context.Context, creds connectors.Credentials, path string, body map[string]any, respBody any) error {
	clientID, _ := creds.Get(credKeyClientID)
	secret, _ := creds.Get(credKeySecret)

	// Copy the body map before injecting credentials so the caller's map
	// is never mutated with sensitive values (prevents accidental credential
	// leakage if the caller logs or reuses the map).
	reqBody := make(map[string]any, len(body)+2)
	for k, v := range body {
		reqBody[k] = v
	}
	reqBody["client_id"] = clientID
	reqBody["secret"] = secret

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	reqURL := c.baseURLForCreds(creds) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Plaid-Version", plaidAPIVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Plaid API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Plaid API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Plaid response: %v", err)}
		}
	}
	return nil
}
