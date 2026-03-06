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

// validateMetadata checks that the metadata map does not exceed Stripe's
// 50-key limit. Returns nil if metadata is nil or within bounds.
func validateMetadata(metadata map[string]any) error {
	if len(metadata) > maxMetadataKeys {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many metadata keys: %d (max %d)", len(metadata), maxMetadataKeys),
		}
	}
	return nil
}

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

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *StripeConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "stripe",
		Name:        "Stripe",
		Description: "Stripe integration for payments, invoicing, and billing",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "stripe.create_customer",
				Name:        "Create Customer",
				Description: "Create a new customer record — foundational for all other Stripe operations",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["email"],
					"properties": {
						"email": {
							"type": "string",
							"description": "Customer email address"
						},
						"name": {
							"type": "string",
							"description": "Customer full name"
						},
						"description": {
							"type": "string",
							"description": "Free-form description of the customer"
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.create_invoice",
				Name:        "Create Invoice",
				Description: "Create and optionally send an invoice with line items",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id"],
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "Stripe customer ID (cus_...)"
						},
						"description": {
							"type": "string",
							"description": "Invoice description"
						},
						"due_date": {
							"type": "integer",
							"description": "Due date as Unix timestamp"
						},
						"auto_advance": {
							"type": "boolean",
							"default": true,
							"description": "Automatically finalize and send the invoice"
						},
						"currency": {
							"type": "string",
							"default": "usd",
							"description": "Three-letter ISO currency code (defaults to usd)"
						},
						"line_items": {
							"type": "array",
							"description": "Invoice line items",
							"items": {
								"type": "object",
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description"
									},
									"amount": {
										"type": "integer",
										"description": "Amount in cents (must be positive)"
									},
									"quantity": {
										"type": "integer",
										"description": "Quantity (defaults to 1)"
									}
								}
							}
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.issue_refund",
				Name:        "Issue Refund",
				Description: "Refund a charge or payment intent — high risk: moves real money",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"payment_intent_id": {
							"type": "string",
							"description": "Payment intent ID (pi_...) — provide this or charge_id"
						},
						"charge_id": {
							"type": "string",
							"description": "Charge ID (ch_...) — provide this or payment_intent_id"
						},
						"amount": {
							"type": "integer",
							"description": "Refund amount in cents (omit for full refund)"
						},
						"reason": {
							"type": "string",
							"enum": ["duplicate", "fraudulent", "requested_by_customer"],
							"description": "Reason for the refund"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.list_subscriptions",
				Name:        "List Subscriptions",
				Description: "List subscriptions, optionally filtered by customer or status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "Filter by Stripe customer ID (cus_...)"
						},
						"status": {
							"type": "string",
							"enum": ["active", "past_due", "canceled", "unpaid", "trialing", "all"],
							"description": "Filter by subscription status"
						},
						"price_id": {
							"type": "string",
							"description": "Filter by price ID (price_...)"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Number of subscriptions to return (default 10, max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.create_payment_link",
				Name:        "Create Payment Link",
				Description: "Create a shareable payment link for one-time or recurring purchases",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["line_items"],
					"properties": {
						"line_items": {
							"type": "array",
							"description": "Products to include in the payment link",
							"items": {
								"type": "object",
								"required": ["price_id", "quantity"],
								"properties": {
									"price_id": {
										"type": "string",
										"description": "Stripe price ID (price_...)"
									},
									"quantity": {
										"type": "integer",
										"description": "Quantity of the product"
									}
								}
							}
						},
						"after_completion": {
							"type": "string",
							"description": "Redirect URL after successful payment"
						},
						"allow_promotion_codes": {
							"type": "boolean",
							"description": "Allow customers to enter promotion codes"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.get_balance",
				Name:        "Get Balance",
				Description: "Retrieve the current account balance — useful for monitoring cash flow",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "stripe",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.stripe.com/keys",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_stripe_create_customers",
				ActionType:  "stripe.create_customer",
				Name:        "Create customers",
				Description: "Agent can create new customer records with any details.",
				Parameters:  json.RawMessage(`{"email":"*","name":"*","description":"*","phone":"*"}`),
			},
			{
				ID:          "tpl_stripe_create_invoices",
				ActionType:  "stripe.create_invoice",
				Name:        "Create invoices",
				Description: "Agent can create and send invoices for any customer.",
				Parameters:  json.RawMessage(`{"customer_id":"*","description":"*","line_items":"*"}`),
			},
			{
				ID:          "tpl_stripe_issue_refund_capped",
				ActionType:  "stripe.issue_refund",
				Name:        "Issue refunds up to $99.99",
				Description: "Agent can issue refunds up to 9999 cents ($99.99). Amount is constrained by pattern to prevent large refunds.",
				Parameters:  json.RawMessage(`{"payment_intent_id":"*","charge_id":"*","amount":{"$pattern":"^[1-9]\\d{0,3}$"},"reason":"*"}`),
			},
			{
				ID:          "tpl_stripe_list_subscriptions",
				ActionType:  "stripe.list_subscriptions",
				Name:        "List active subscriptions",
				Description: "Agent can list active subscriptions for any customer.",
				Parameters:  json.RawMessage(`{"customer_id":"*","status":"active","limit":"*"}`),
			},
			{
				ID:          "tpl_stripe_create_payment_links",
				ActionType:  "stripe.create_payment_link",
				Name:        "Create payment links",
				Description: "Agent can create shareable payment links for any products.",
				Parameters:  json.RawMessage(`{"line_items":"*","after_completion":"*","allow_promotion_codes":"*"}`),
			},
			{
				ID:          "tpl_stripe_get_balance",
				ActionType:  "stripe.get_balance",
				Name:        "Check account balance",
				Description: "Agent can retrieve the current Stripe account balance.",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}
}

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
