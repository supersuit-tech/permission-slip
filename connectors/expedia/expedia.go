// Package expedia implements the Expedia Rapid connector for the Permission
// Slip connector execution layer. It uses the Expedia Rapid API with
// SHA-512 hash signature authentication (api_key + secret + timestamp).
package expedia

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultBaseURL    = "https://api.ean.com"
	defaultTimeout    = 30 * time.Second
	defaultCustomerIP = "127.0.0.1"

	// maxResponseBytes caps the response body we read from Expedia to 10 MB.
	// Prevents a misbehaving upstream from exhausting server memory.
	maxResponseBytes = 10 * 1024 * 1024
)

// ExpediaConnector owns the shared HTTP client and base URL used by all
// Expedia Rapid actions. Actions hold a pointer back to the connector
// to access these shared resources.
type ExpediaConnector struct {
	client  *http.Client
	baseURL string
	// nowFunc is used to get the current unix timestamp for signature
	// generation. Defaults to time.Now; overridden in tests.
	nowFunc func() time.Time
}

// New creates an ExpediaConnector with sensible defaults (30s timeout,
// production base URL).
func New() *ExpediaConnector {
	return &ExpediaConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
		nowFunc: time.Now,
	}
}

// newForTest creates an ExpediaConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *ExpediaConnector {
	return &ExpediaConnector{
		client:  client,
		baseURL: baseURL,
		nowFunc: time.Now,
	}
}

// ID returns "expedia", matching the connectors.id in the database.
func (c *ExpediaConnector) ID() string { return "expedia" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ExpediaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"expedia.search_hotels":  &searchHotelsAction{conn: c},
		"expedia.get_hotel":      &getHotelAction{conn: c},
		"expedia.price_check":    &priceCheckAction{conn: c},
		"expedia.create_booking": &createBookingAction{conn: c},
		"expedia.cancel_booking": &cancelBookingAction{conn: c},
		"expedia.get_booking":    &getBookingAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key and secret, which are required for SHA-512 signature
// authentication with the Expedia Rapid API.
func (c *ExpediaConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	secret, ok := creds.Get("secret")
	if !ok || secret == "" {
		return &connectors.ValidationError{Message: "missing required credential: secret"}
	}
	return nil
}

// signature generates the SHA-512 hash signature for Expedia Rapid API
// authentication. The signature is SHA512(api_key + secret + unix_timestamp).
func (c *ExpediaConnector) signature(apiKey, secret string) (sig string, timestamp string) {
	ts := c.nowFunc().Unix()
	timestamp = strconv.FormatInt(ts, 10)

	h := sha512.New()
	h.Write([]byte(apiKey))
	h.Write([]byte(secret))
	h.Write([]byte(timestamp))
	sig = hex.EncodeToString(h.Sum(nil))

	return sig, timestamp
}

// do is the shared request lifecycle for all Expedia Rapid actions. It
// marshals reqBody as JSON, sends the request with signature auth headers,
// checks the response status, and unmarshals the response into respBody.
// Either reqBody or respBody may be nil.
//
// customerIP is the end-user's IP address, required by Expedia for fraud
// prevention. Pass defaultCustomerIP when the real IP is unavailable.
func (c *ExpediaConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, customerIP string, reqBody, respBody interface{}) error {
	_, err := c.doWithHeaders(ctx, creds, method, path, customerIP, reqBody, respBody)
	return err
}

// doWithHeaders is like do but also returns response headers. Used by actions
// that need to inspect headers (e.g., search_hotels reads the Link header
// for pagination).
func (c *ExpediaConnector) doWithHeaders(ctx context.Context, creds connectors.Credentials, method, path string, customerIP string, reqBody, respBody interface{}) (http.Header, error) {
	apiKey, _ := creds.Get("api_key")
	secret, _ := creds.Get("secret")
	if apiKey == "" || secret == "" {
		return nil, &connectors.ValidationError{Message: "api_key and secret credentials are required"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set Expedia Rapid signature auth header.
	sig, ts := c.signature(apiKey, secret)
	req.Header.Set("Authorization", fmt.Sprintf("EAN apikey=%s,signature=%s,timestamp=%s", apiKey, sig, ts))
	req.Header.Set("Accept", "application/json")
	if customerIP == "" {
		customerIP = defaultCustomerIP
	}
	req.Header.Set("Customer-Ip", customerIP)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("Expedia Rapid API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.CanceledError{Message: "Expedia Rapid API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Expedia Rapid API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing Expedia Rapid response: %v", err)}
		}
	}
	return resp.Header, nil
}
