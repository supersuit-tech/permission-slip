// Package amadeus implements the Amadeus connector for the Permission Slip
// connector execution layer. It uses the Amadeus Self-Service APIs with
// client credentials grant (client_id + client_secret -> short-lived bearer
// token) for authentication. No third-party SDK — plain net/http.
package amadeus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTestBaseURL       = "https://test.api.amadeus.com"
	defaultProductionBaseURL = "https://api.amadeus.com"
	defaultTimeout           = 30 * time.Second

	// tokenRefreshBuffer is how long before expiry we proactively refresh.
	tokenRefreshBuffer = 60 * time.Second
)

// AmadeusConnector owns the shared HTTP client, base URL, and cached access
// token used by all Amadeus actions. Actions hold a pointer back to the
// connector to access these shared resources.
type AmadeusConnector struct {
	client  *http.Client
	baseURL string

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// New creates an AmadeusConnector with sensible defaults (30s timeout,
// test environment base URL).
func New() *AmadeusConnector {
	return &AmadeusConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultTestBaseURL,
	}
}

// newForTest creates an AmadeusConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *AmadeusConnector {
	return &AmadeusConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "amadeus", matching the connectors.id in the database.
func (c *AmadeusConnector) ID() string { return "amadeus" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *AmadeusConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "amadeus",
		Name:        "Amadeus",
		Description: "Amadeus travel APIs for flights, hotels, and car rentals",
		Actions:     []connectors.ManifestAction{},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "amadeus",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.amadeus.com/get-started/get-started-with-self-service-apis-335",
			},
		},
		Templates: []connectors.ManifestTemplate{},
	}
}

// Actions returns the registered action handlers keyed by action_type.
// Phase 1 has no actions — they are added in Phase 2.
func (c *AmadeusConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty client_id and client_secret, which are required for the
// Amadeus client credentials grant.
func (c *AmadeusConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	clientID, ok := creds.Get("client_id")
	if !ok || clientID == "" {
		return &connectors.ValidationError{Message: "missing required credential: client_id"}
	}
	clientSecret, ok := creds.Get("client_secret")
	if !ok || clientSecret == "" {
		return &connectors.ValidationError{Message: "missing required credential: client_secret"}
	}
	return nil
}

// ensureToken returns a valid access token, refreshing it if necessary.
// It uses the Amadeus client credentials grant: POST /v1/security/oauth2/token
// with grant_type=client_credentials, client_id, and client_secret as
// form-encoded body.
func (c *AmadeusConnector) ensureToken(ctx context.Context, creds connectors.Credentials) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid (with buffer).
	if c.token != "" && time.Now().Before(c.tokenExp.Add(-tokenRefreshBuffer)) {
		return c.token, nil
	}

	clientID, _ := creds.Get("client_id")
	clientSecret, _ := creds.Get("client_secret")

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/security/oauth2/token",
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return "", &connectors.TimeoutError{Message: fmt.Sprintf("Amadeus token request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return "", &connectors.TimeoutError{Message: "Amadeus token request canceled"}
		}
		return "", &connectors.ExternalError{Message: fmt.Sprintf("Amadeus token request failed: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("reading token response body: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return "", mapTokenError(resp.StatusCode, body)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("parsing token response: %v", err)}
	}

	if tokenResp.AccessToken == "" {
		return "", &connectors.ExternalError{Message: "Amadeus token response missing access_token"}
	}

	c.token = tokenResp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.token, nil
}

// tokenResponse is the Amadeus OAuth2 token endpoint response.
type tokenResponse struct {
	Type         string `json:"type"`
	Username     string `json:"username"`
	ApplicationName string `json:"application_name"`
	ClientID     string `json:"client_id"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}

// mapTokenError maps HTTP status codes from the token endpoint to typed errors.
func mapTokenError(statusCode int, body []byte) error {
	msg := string(body)
	var errResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.ErrorDescription != "" {
		msg = errResp.ErrorDescription
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Amadeus auth failed: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Amadeus token error: %s", msg),
		}
	}
}

// do is the shared request lifecycle for all Amadeus actions. It obtains
// a valid access token, sends the request with the Bearer header, checks
// the response status, and unmarshals the response into respBody. Either
// reqBody or respBody may be nil.
func (c *AmadeusConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	token, err := c.ensureToken(ctx, creds)
	if err != nil {
		return err
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
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Amadeus API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Amadeus API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Amadeus API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Amadeus response: %v", err)}
		}
	}
	return nil
}
