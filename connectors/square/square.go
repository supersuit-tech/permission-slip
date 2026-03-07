// Package square implements the Square connector for the Permission Slip
// connector execution layer. It uses the Square REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package square

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	productionBaseURL = "https://connect.squareup.com/v2"
	sandboxBaseURL    = "https://connect.squareupsandbox.com/v2"
	defaultTimeout    = 30 * time.Second

	// squareVersion is the API version sent via the Square-Version header.
	// Pinned to a stable date to avoid breaking changes.
	squareVersion = "2024-01-18"

	credKeyAccessToken = "access_token"
	credKeyEnvironment = "environment"
)

// money represents Square's Money object. Amount is in the smallest currency
// unit (e.g., cents for USD). Shared across create_order and create_payment.
type money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// SquareConnector owns the shared HTTP client and base URL used by all
// Square actions. Actions hold a pointer back to the connector to access
// these shared resources.
type SquareConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a SquareConnector with sensible defaults (30s timeout,
// production base URL).
func New() *SquareConnector {
	return &SquareConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: productionBaseURL,
	}
}

// newForTest creates a SquareConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SquareConnector {
	return &SquareConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "square", matching the connectors.id in the database.
func (c *SquareConnector) ID() string { return "square" }

// Actions returns the registered action handlers keyed by action_type.
func (c *SquareConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"square.create_order":        &createOrderAction{conn: c},
		"square.create_payment":      &createPaymentAction{conn: c},
		"square.list_catalog":        &listCatalogAction{conn: c},
		"square.create_customer":     &createCustomerAction{conn: c},
		"square.create_booking":      &createBookingAction{conn: c},
		"square.search_orders":       &searchOrdersAction{conn: c},
		"square.issue_refund":        &issueRefundAction{conn: c},
		"square.update_catalog_item": &updateCatalogItemAction{conn: c},
		"square.send_invoice":        &sendInvoiceAction{conn: c},
		"square.get_inventory":       &getInventoryAction{conn: c},
		"square.adjust_inventory":    &adjustInventoryAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token, which is required for all Square API calls.
// Optionally validates the environment field if present.
func (c *SquareConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	if env, ok := creds.Get(credKeyEnvironment); ok && env != "" {
		if env != "sandbox" && env != "production" {
			return &connectors.ValidationError{Message: "environment must be \"sandbox\" or \"production\""}
		}
	}
	return nil
}

// baseURLForCreds returns the appropriate base URL based on the environment
// credential. Falls back to the connector's configured baseURL (which
// supports test overrides via newForTest).
func (c *SquareConnector) baseURLForCreds(creds connectors.Credentials) string {
	// If baseURL was overridden (e.g., for tests), always use the override.
	if c.baseURL != productionBaseURL && c.baseURL != sandboxBaseURL {
		return c.baseURL
	}
	if env, ok := creds.Get(credKeyEnvironment); ok && env == "sandbox" {
		return sandboxBaseURL
	}
	return productionBaseURL
}

// newIdempotencyKey generates a new UUID v4 for use as an idempotency key.
// Square requires this on all write operations.
func newIdempotencyKey() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	// Set version 4 and variant bits per RFC 4122.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// do is the shared request lifecycle for all Square actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody. Either reqBody or
// respBody may be nil.
func (c *SquareConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	baseURL := c.baseURLForCreds(creds)
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Square-Version", squareVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Square API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Square API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Square API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Limit response reads to 5MB to guard against unexpectedly large payloads.
	const maxResponseSize = 5 << 20
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Square response: %v", err)}
		}
	}
	return nil
}
