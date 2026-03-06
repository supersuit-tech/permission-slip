// Package stripe implements the Stripe connector for the Permission Slip
// connector execution layer. It uses the Stripe REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Stripe uses application/x-www-form-urlencoded request bodies (not JSON),
// with bracket notation for nested objects (e.g., metadata[key]=value,
// line_items[0][amount]=1000). Responses are JSON as normal.
package stripe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.stripe.com"
	defaultTimeout = 30 * time.Second
	credKeyAPIKey  = "api_key"

	// Stripe secret keys always start with one of these prefixes.
	liveKeyPrefix  = "sk_live_"
	testKeyPrefix  = "sk_test_"
	rTestKeyPrefix = "rk_test_"
	rLiveKeyPrefix = "rk_live_"

	// defaultRetryAfter is used when Stripe returns a rate limit response
	// without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes limits how much of a Stripe API response we'll read
	// into memory. 4 MB is generous for any Stripe response (list endpoints
	// return at most 100 objects). Prevents memory exhaustion from a
	// malicious or misconfigured upstream.
	maxResponseBytes = 4 << 20 // 4 MB

	// maxErrorMessageBytes caps the raw response body included in error
	// messages when Stripe returns non-JSON errors. Prevents oversized
	// log entries and potential data leakage.
	maxErrorMessageBytes = 512

	// apiVersion pins the Stripe API version. This prevents breaking changes
	// when Stripe releases new API versions. Update this deliberately when
	// you're ready to handle the new response shapes.
	// See https://docs.stripe.com/api/versioning
	apiVersion = "2025-12-18.acacia"

	// maxMetadataKeys is the Stripe limit on metadata key-value pairs.
	// Validating this client-side gives clearer error messages than
	// relying on the API to reject oversized metadata.
	maxMetadataKeys = 50
)

// validKeyPrefixes lists all recognized Stripe secret key prefixes.
var validKeyPrefixes = []string{liveKeyPrefix, testKeyPrefix, rLiveKeyPrefix, rTestKeyPrefix}

// StripeConnector owns the shared HTTP client and base URL used by all
// Stripe actions. Actions hold a pointer back to the connector to access
// these shared resources.
type StripeConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a StripeConnector with sensible defaults (30s timeout,
// https://api.stripe.com base URL).
func New() *StripeConnector {
	return &StripeConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a StripeConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *StripeConnector {
	return &StripeConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "stripe", matching the connectors.id in the database.
func (c *StripeConnector) ID() string { return "stripe" }

// Actions returns the registered action handlers keyed by action_type.
func (c *StripeConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"stripe.create_customer":     &createCustomerAction{conn: c},
		"stripe.create_invoice":      &createInvoiceAction{conn: c},
		"stripe.issue_refund":        &issueRefundAction{conn: c},
		"stripe.list_subscriptions":  &listSubscriptionsAction{conn: c},
		"stripe.create_payment_link": &createPaymentLinkAction{conn: c},
		"stripe.get_balance":         &getBalanceAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key with a recognized Stripe secret key prefix.
func (c *StripeConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	if !hasValidPrefix(key) {
		return &connectors.ValidationError{
			Message: "api_key must start with \"sk_live_\", \"sk_test_\", \"rk_live_\", or \"rk_test_\"",
		}
	}
	return nil
}

// hasValidPrefix reports whether key starts with a recognized Stripe
// secret key prefix.
func hasValidPrefix(key string) bool {
	for _, prefix := range validKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// do is the shared request lifecycle for all Stripe actions. It sends a
// request with Bearer auth, handles rate limiting, timeouts, and Stripe API
// errors, then unmarshals the JSON response into respBody.
//
// For POST/DELETE requests, params are form-encoded into the request body.
// For GET requests, params are appended as query parameters.
//
// When idempotencyKey is non-empty, it is sent as the Idempotency-Key header
// (only meaningful for POST requests).
func (c *StripeConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, params map[string]string, respBody any, idempotencyKey string) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}

	var body io.Reader
	fullURL := c.baseURL + path

	if method == http.MethodGet {
		if len(params) > 0 {
			fullURL += "?" + encodeParams(params)
		}
	} else if len(params) > 0 {
		body = strings.NewReader(encodeParams(params))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Stripe-Version", apiVersion)
	if method != http.MethodGet && len(params) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if idempotencyKey != "" && method == http.MethodPost {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Stripe API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Stripe API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Stripe API request failed: %v", err)}
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
				Message:    "failed to decode Stripe API response",
			}
		}
	}

	return nil
}

// doGet is a convenience wrapper around do() for GET requests.
// GET requests never need form bodies or idempotency keys.
func (c *StripeConnector) doGet(ctx context.Context, creds connectors.Credentials, path string, params map[string]string, respBody any) error {
	return c.do(ctx, creds, http.MethodGet, path, params, respBody, "")
}

// doPost is a convenience wrapper around do() for POST requests.
// It accepts the action type and raw parameters for automatic idempotency
// key derivation. Phase 2 actions should prefer this over do() directly.
func (c *StripeConnector) doPost(ctx context.Context, creds connectors.Credentials, path string, params map[string]string, respBody any, actionType string, rawParams json.RawMessage) error {
	idempotencyKey := deriveIdempotencyKey(actionType, rawParams)
	return c.do(ctx, creds, http.MethodPost, path, params, respBody, idempotencyKey)
}

// formEncode flattens a nested structure into Stripe's bracket-notation
// form encoding. It handles:
//   - Flat values: key=value
//   - Nested objects (map[string]any): metadata[key]=value
//   - Arrays ([]any): line_items[0][amount]=1000
//
// The input is expected to be the JSON-deserialized action parameters
// (map[string]any from json.Unmarshal). The output is a flat map of
// bracket-notated keys to string values, ready for encodeParams().
func formEncode(params map[string]any) map[string]string {
	result := make(map[string]string)
	flattenInto(result, "", params)
	return result
}

// flattenInto recursively flattens nested values into bracket-notated keys.
func flattenInto(dst map[string]string, prefix string, val any) {
	switch v := val.(type) {
	case map[string]any:
		for k, child := range v {
			key := k
			if prefix != "" {
				key = prefix + "[" + k + "]"
			}
			flattenInto(dst, key, child)
		}
	case []any:
		for i, child := range v {
			key := fmt.Sprintf("%s[%d]", prefix, i)
			flattenInto(dst, key, child)
		}
	case nil:
		// Skip nil values entirely.
	default:
		dst[prefix] = fmt.Sprintf("%v", v)
	}
}

// encodeParams encodes a flat key-value map into a URL-encoded query string.
// url.Values.Encode() sorts keys alphabetically, giving deterministic output
// (important for test assertions).
func encodeParams(params map[string]string) string {
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	return vals.Encode()
}

// truncate caps s at approximately maxLen bytes, appending "..." if
// truncated. It avoids splitting multi-byte UTF-8 characters by
// counting runes instead of raw bytes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Walk runes to find the last complete rune boundary within maxLen bytes.
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

// deriveIdempotencyKey produces a deterministic idempotency key from the
// action type and parameters. This ensures that retrying the same action
// with the same parameters produces the same key, which is the entire
// point of idempotency — a random UUID would defeat retry safety.
func deriveIdempotencyKey(actionType string, parameters json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(actionType))
	h.Write([]byte{0}) // separator
	h.Write(parameters)
	return hex.EncodeToString(h.Sum(nil))
}
