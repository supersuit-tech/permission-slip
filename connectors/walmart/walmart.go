// Package walmart implements the Walmart connector for the Permission Slip
// connector execution layer. It uses the Walmart Affiliate API with
// Consumer ID + RSA-SHA256 request signing. No third-party SDK — plain net/http.
//
// Every request is signed using the consumer's RSA private key per
// https://walmart.io/docs/affiliate/onboarding-guide. The signature
// covers {consumerID}\n{timestamp}\n{keyVersion}\n and has a 3-minute TTL.
package walmart

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// defaultBaseURL pins the Walmart Affiliate API version (currently v2).
	// Update deliberately when ready to handle new response shapes.
	// See https://walmart.io/docs/affiliate/introduction
	defaultBaseURL = "https://developer.api.walmart.com/api-proxy/service/affil/product/v2"
	defaultTimeout = 30 * time.Second

	credKeyConsumerID  = "consumer_id"
	credKeyPrivateKey  = "private_key"
	credKeyKeyVersion  = "key_version"
	credKeyImpactID    = "impact_id"

	// maxResponseBytes prevents memory exhaustion from a malicious or
	// misconfigured upstream. 10 MB is generous for any Walmart search response.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// WalmartConnector owns the shared HTTP client and base URL used by all
// Walmart actions. Actions hold a pointer back to the connector.
type WalmartConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a WalmartConnector with sensible defaults (30s timeout,
// production base URL).
func New() *WalmartConnector {
	return &WalmartConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a WalmartConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *WalmartConnector {
	return &WalmartConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "walmart", matching the connectors.id in the database.
func (c *WalmartConnector) ID() string { return "walmart" }

// ValidateCredentials checks that the provided credentials contain a
// non-empty consumer_id and private_key (PEM-encoded RSA private key).
// key_version defaults to "1" if omitted. impact_id is optional.
func (c *WalmartConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	consumerID, ok := creds.Get(credKeyConsumerID)
	if !ok || consumerID == "" {
		return &connectors.ValidationError{Message: "missing required credential: consumer_id"}
	}
	privateKey, ok := creds.Get(credKeyPrivateKey)
	if !ok || privateKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: private_key (PEM-encoded RSA private key)"}
	}
	if _, err := parsePrivateKey(privateKey); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid private_key: %v", err)}
	}
	return nil
}

// do is the shared request lifecycle for all Walmart actions. It sends
// the request with the required Walmart API headers, checks the response
// status, and unmarshals the response into respBody.
func (c *WalmartConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, respBody interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	consumerID, _ := creds.Get(credKeyConsumerID)
	if consumerID == "" {
		return &connectors.ValidationError{Message: "consumer_id credential is missing or empty"}
	}
	req.Header.Set("WM_CONSUMER.ID", consumerID)

	keyVersion, ok := creds.Get(credKeyKeyVersion)
	if !ok || keyVersion == "" {
		keyVersion = "1"
	}
	req.Header.Set("WM_SEC.KEY_VERSION", keyVersion)

	// RSA-SHA256 request signing (required by Walmart Affiliate API).
	// The signature covers "{consumerID}\n{timestamp}\n{keyVersion}\n"
	// and has a 3-minute TTL.
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	req.Header.Set("WM_CONSUMER.INTIMESTAMP", timestamp)

	privateKeyPEM, ok := creds.Get(credKeyPrivateKey)
	if !ok || privateKeyPEM == "" {
		return &connectors.ValidationError{Message: "private_key credential is missing — required for request signing"}
	}
	sig, err := signRequest(consumerID, timestamp, keyVersion, privateKeyPEM)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("failed to sign request: %v", err)}
	}
	req.Header.Set("WM_SEC.AUTH_SIGNATURE", sig)

	// Impact ID for affiliate attribution (optional).
	if impactID, ok := creds.Get(credKeyImpactID); ok && impactID != "" {
		req.Header.Set("WM_CONSUMER.CHANNEL.TYPE", impactID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Walmart API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Walmart API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Walmart API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Walmart response: %v", err)}
		}
	}
	return nil
}

// parsePrivateKey decodes a PEM-encoded RSA private key (PKCS#1 or PKCS#8).
func parsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	// Try PKCS#8 first, then PKCS#1.
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not RSA")
		}
		return rsaKey, nil
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse as PKCS#1 or PKCS#8: %v", err)
	}
	return rsaKey, nil
}

// signRequest generates the WM_SEC.AUTH_SIGNATURE header value.
// The signature is SHA256WithRSA over "{consumerID}\n{timestamp}\n{keyVersion}\n".
func signRequest(consumerID, timestamp, keyVersion, pemData string) (string, error) {
	key, err := parsePrivateKey(pemData)
	if err != nil {
		return "", err
	}

	message := consumerID + "\n" + timestamp + "\n" + keyVersion + "\n"
	hash := sha256.Sum256([]byte(message))

	sig, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("signing failed: %v", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// Actions returns the registered action handlers keyed by action_type.
func (c *WalmartConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"walmart.search_products": &searchProductsAction{conn: c},
		"walmart.get_product":     &getProductAction{conn: c},
		"walmart.get_taxonomy":    &getTaxonomyAction{conn: c},
		"walmart.get_trending":    &getTrendingAction{conn: c},
	}
}
