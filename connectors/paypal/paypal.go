// Package paypal implements the PayPal / Venmo connector for the Permission Slip
// connector execution layer. It uses the PayPal REST API with plain net/http.
//
// API hosts: live uses https://api-m.paypal.com; sandbox uses
// https://api-m.sandbox.paypal.com. OAuth (built-in) uses PayPal's OpenID
// endpoints (www.paypal.com + api.paypal.com/token) — see oauth/providers/paypal.go.
package paypal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	liveAPIBaseURL    = "https://api-m.paypal.com"
	sandboxAPIBaseURL = "https://api-m.sandbox.paypal.com"

	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"
	credKeyEnvironment = "environment"

	defaultRetryAfter = 30 * time.Second

	maxResponseBytes     = 4 << 20 // 4 MiB
	maxErrorMessageChars = 512
	maxJSONBodyBytes     = 256 << 10 // 256 KiB cap for caller-supplied JSON bodies
)

// OAuthScopes are requested during Log in with PayPal. Kept in this package so
// the manifest and oauth/providers/paypal.go stay aligned.
var OAuthScopes = []string{
	"openid",
	"https://uri.paypal.com/services/payments/payment/authcapture",
	"https://uri.paypal.com/services/payments/realtimepayment",
	"https://uri.paypal.com/payments/payouts",
	"https://uri.paypal.com/services/invoicing",
	"https://uri.paypal.com/services/payments/refund",
}

// PayPalConnector owns the shared HTTP client used by all PayPal actions.
type PayPalConnector struct {
	client *http.Client
	// baseURL, when non-empty, overrides live/sandbox host selection (tests only).
	baseURL string
}

// New creates a PayPalConnector with sensible defaults.
func New() *PayPalConnector {
	return &PayPalConnector{
		client: &http.Client{
			Timeout: defaultTimeout,
			// Do not follow redirects — avoids leaking the Bearer token to an
			// unexpected host if upstream ever returns a 3xx (misconfig or attack).
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func newForTest(client *http.Client, baseURL string) *PayPalConnector {
	return &PayPalConnector{client: client, baseURL: strings.TrimSuffix(baseURL, "/")}
}

// ID returns "paypal", matching connectors.id in the database.
func (c *PayPalConnector) ID() string { return "paypal" }

// Actions returns registered action handlers keyed by action_type.
func (c *PayPalConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"paypal.create_venmo_payout_batch": &createVenmoPayoutBatchAction{conn: c},
		"paypal.get_payout_batch":          &getPayoutBatchAction{conn: c},
		"paypal.get_payout_item":           &getPayoutItemAction{conn: c},
		"paypal.create_order":              &createOrderAction{conn: c},
		"paypal.capture_order":             &captureOrderAction{conn: c},
		"paypal.get_order":                 &getOrderAction{conn: c},
		"paypal.create_invoice":            &createInvoiceAction{conn: c},
		"paypal.send_invoice":              &sendInvoiceAction{conn: c},
		"paypal.get_invoice":               &getInvoiceAction{conn: c},
		"paypal.remind_invoice":            &remindInvoiceAction{conn: c},
		"paypal.refund_capture":            &refundCaptureAction{conn: c},
	}
}

// ValidateCredentials ensures an OAuth access token is present. Optional
// environment selects the API host (live vs sandbox).
func (c *PayPalConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	tok, ok := creds.Get(credKeyAccessToken)
	if !ok || strings.TrimSpace(tok) == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token (connect via OAuth)"}
	}
	if env, ok := creds.Get(credKeyEnvironment); ok && strings.TrimSpace(env) != "" {
		env = strings.TrimSpace(env)
		if env != "live" && env != "sandbox" {
			return &connectors.ValidationError{Message: "environment must be \"live\" or \"sandbox\" when set"}
		}
	}
	return nil
}

func apiBaseURLForCreds(creds connectors.Credentials) string {
	if env, ok := creds.Get(credKeyEnvironment); ok && strings.TrimSpace(env) == "sandbox" {
		return sandboxAPIBaseURL
	}
	return liveAPIBaseURL
}

func (c *PayPalConnector) resolveAPIBase(creds connectors.Credentials) string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return apiBaseURLForCreds(creds)
}

// maxRequestIDLen is PayPal's maximum length for the PayPal-Request-Id header.
// See https://developer.paypal.com/api/rest/reference/idempotency/ — PayPal
// recommends UUID format (36 chars) because it fits the 38-character limit.
const maxRequestIDLen = 38

func deriveRequestID(actionType string, rawParams json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(actionType))
	h.Write([]byte{0})
	h.Write(rawParams)
	full := hex.EncodeToString(h.Sum(nil))
	if len(full) > maxRequestIDLen {
		return full[:maxRequestIDLen]
	}
	return full
}

func readJSONBody(raw json.RawMessage, fieldName string) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", fieldName)}
	}
	if len(raw) > maxJSONBodyBytes {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("%s exceeds maximum size (%d bytes)", fieldName, maxJSONBodyBytes)}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: %v", fieldName, err)}
	}
	if m == nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("%s must be a JSON object", fieldName)}
	}
	return m, nil
}

// doJSON sends a JSON request with Bearer auth and optional PayPal-Request-Id.
// The connector's HTTP client does not follow redirects (see New) so authorization
// headers are not replayed against an unexpected host after a 3xx.
func (c *PayPalConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, path string, requestBody any, respBody any, requestID string) error {
	tok, ok := creds.Get(credKeyAccessToken)
	if !ok || strings.TrimSpace(tok) == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	base := c.resolveAPIBase(creds)
	fullURL := base + path

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if requestID != "" && method == http.MethodPost {
		req.Header.Set("PayPal-Request-Id", requestID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("PayPal API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "PayPal API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("PayPal API request failed: %v", err)}
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
		if len(respBytes) == 0 {
			respBytes = []byte("null")
		}
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode PayPal API response",
			}
		}
	}
	return nil
}

// doJSONRaw sends JSON built from raw message; response returned as json.RawMessage.
func (c *PayPalConnector) doJSONRaw(ctx context.Context, creds connectors.Credentials, method, path string, requestBody map[string]any, requestID string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.doJSON(ctx, creds, method, path, requestBody, &raw, requestID); err != nil {
		return nil, err
	}
	return raw, nil
}
