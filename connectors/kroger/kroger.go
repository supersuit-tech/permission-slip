// Package kroger implements the Kroger connector for the Permission Slip
// connector execution layer. It uses the Kroger API v1 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package kroger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL    = "https://api.kroger.com/v1"
	defaultTimeout    = 30 * time.Second
	credKeyToken      = "access_token"
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps how much data we read from the Kroger API to
	// prevent OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// KrogerConnector owns the shared HTTP client and base URL used by all
// Kroger actions.
type KrogerConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a KrogerConnector with sensible defaults.
func New() *KrogerConnector {
	return &KrogerConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a KrogerConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *KrogerConnector {
	return &KrogerConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "kroger", matching the connectors.id in the database.
func (c *KrogerConnector) ID() string { return "kroger" }

// Actions returns the registered action handlers keyed by action_type.
func (c *KrogerConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"kroger.search_products":  &searchProductsAction{conn: c},
		"kroger.get_product":      &getProductAction{conn: c},
		"kroger.search_locations": &searchLocationsAction{conn: c},
		"kroger.add_to_cart":      &addToCartAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token.
func (c *KrogerConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// krogerErrorResponse is the Kroger API error envelope.
type krogerErrorResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"errors"`
}

// do is the shared request lifecycle for all Kroger actions. It sends the
// request with Bearer token auth, handles rate limiting and timeouts, and
// unmarshals the response into dest. reqBody may be nil for GET requests.
func (c *KrogerConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Kroger API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Kroger API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Kroger API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if dest != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Kroger API response",
			}
		}
	}

	return nil
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Kroger API error message.
	var kErr krogerErrorResponse
	msg := string(body)
	if json.Unmarshal(body, &kErr) == nil && len(kErr.Errors) > 0 {
		msg = kErr.Errors[0].Message
		if msg == "" {
			msg = kErr.Errors[0].Reason
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Kroger API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Kroger API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Kroger API permission error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Kroger API error: %s", msg)}
	}
}
