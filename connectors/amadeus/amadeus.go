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

	// maxResponseBytes caps how much data we read from the Amadeus API.
	// Prevents a misbehaving server from exhausting memory.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// AmadeusConnector owns the shared HTTP client and base URL used by all
// Amadeus actions. Actions hold a pointer back to the connector to access
// these shared resources.
//
// Token caching is keyed by client_id so that different credential sets
// (e.g., different Amadeus accounts) don't share tokens.
type AmadeusConnector struct {
	client  *http.Client
	baseURL string

	mu     sync.Mutex
	tokens map[string]cachedToken // keyed by client_id
}

// cachedToken holds a cached OAuth2 access token and its expiry.
type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// New creates an AmadeusConnector with sensible defaults (30s timeout,
// test environment base URL). The base URL is resolved at request time
// from the "environment" credential field — "production" uses
// api.amadeus.com, anything else (including empty) uses the test
// environment. The baseURL field is only used as a fallback.
func New() *AmadeusConnector {
	return &AmadeusConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultTestBaseURL,
		tokens:  make(map[string]cachedToken),
	}
}

// newForTest creates an AmadeusConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *AmadeusConnector {
	return &AmadeusConnector{
		client:  client,
		baseURL: baseURL,
		tokens:  make(map[string]cachedToken),
	}
}

// resolveBaseURL returns the API base URL based on the "environment"
// credential field. "production" → api.amadeus.com, everything else →
// test.api.amadeus.com. If the connector was created with newForTest, the
// override base URL is always used.
func (c *AmadeusConnector) resolveBaseURL(creds connectors.Credentials) string {
	// Test constructors set a custom baseURL — always honor it.
	if c.baseURL != defaultTestBaseURL && c.baseURL != defaultProductionBaseURL {
		return c.baseURL
	}
	env, _ := creds.Get("environment")
	if env == "production" {
		return defaultProductionBaseURL
	}
	return defaultTestBaseURL
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
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "amadeus.search_airports",
				Name:        "Search Airports",
				Description: "Look up airports by name or IATA code",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Airport name or IATA code (e.g. 'San Francisco' or 'SFO')"},
						"subtype": {"type": "string", "enum": ["AIRPORT", "CITY"], "description": "Filter by location type"}
					},
					"required": ["keyword"]
				}`),
			},
			{
				ActionType:  "amadeus.search_flights",
				Name:        "Search Flights",
				Description: "Search flight offers between airports",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"origin": {"type": "string", "description": "Origin IATA code (e.g. 'SFO')"},
						"destination": {"type": "string", "description": "Destination IATA code (e.g. 'LAX')"},
						"departure_date": {"type": "string", "format": "date", "description": "Departure date (YYYY-MM-DD)"},
						"return_date": {"type": "string", "format": "date", "description": "Return date for round trip (YYYY-MM-DD)"},
						"adults": {"type": "integer", "minimum": 1, "maximum": 9, "default": 1, "description": "Number of adult travelers (1-9)"},
						"cabin": {"type": "string", "enum": ["ECONOMY", "PREMIUM_ECONOMY", "BUSINESS", "FIRST"], "description": "Cabin class"},
						"nonstop": {"type": "boolean", "default": false, "description": "Only show nonstop flights"},
						"max_results": {"type": "integer", "default": 10, "description": "Maximum number of results"}
					},
					"required": ["origin", "destination", "departure_date"]
				}`),
			},
			{
				ActionType:  "amadeus.price_flight",
				Name:        "Price Flight Offer",
				Description: "Confirm real-time pricing for a specific flight offer before booking",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"flight_offer": {"type": "object", "description": "Flight offer object from search results"}
					},
					"required": ["flight_offer"]
				}`),
			},
			{
				ActionType:  "amadeus.book_flight",
				Name:        "Book Flight",
				Description: "Create a flight booking (PNR). High risk — creates a real reservation.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"flight_offer": {"type": "object", "description": "Priced flight offer object"},
						"travelers": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"name": {"type": "string"},
									"dateOfBirth": {"type": "string", "format": "date"},
									"gender": {"type": "string", "enum": ["MALE", "FEMALE"]},
									"contact": {
										"type": "object",
										"properties": {
											"email": {"type": "string", "format": "email"},
											"phone": {"type": "string"}
										},
										"required": ["email", "phone"]
									}
								},
								"required": ["name", "dateOfBirth", "gender", "contact"]
							},
							"description": "Array of traveler details"
						},
						"payment_method_id": {"type": "string", "description": "Stored payment method ID"},
						"remarks": {"type": "string", "description": "Optional booking remarks"}
					},
					"required": ["flight_offer", "travelers", "payment_method_id"]
				}`),
			},
			{
				ActionType:  "amadeus.search_hotels",
				Name:        "Search Hotels",
				Description: "Search hotel offers by city or coordinates",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"city_code": {"type": "string", "description": "City IATA code (e.g. 'PAR')"},
						"latitude": {"type": "string", "description": "Latitude for geo search"},
						"longitude": {"type": "string", "description": "Longitude for geo search"},
						"check_in_date": {"type": "string", "format": "date", "description": "Check-in date (YYYY-MM-DD)"},
						"check_out_date": {"type": "string", "format": "date", "description": "Check-out date (YYYY-MM-DD)"},
						"adults": {"type": "integer", "default": 1, "description": "Number of adults"},
						"room_quantity": {"type": "integer", "default": 1, "description": "Number of rooms"},
						"ratings": {"type": "array", "items": {"type": "integer", "minimum": 1, "maximum": 5}, "description": "Hotel star ratings to filter by"},
						"price_range": {"type": "string", "description": "Price range (e.g. '100-300')"},
						"currency": {"type": "string", "description": "Currency code (e.g. 'USD')"}
					},
					"required": ["check_in_date", "check_out_date"]
				}`),
			},
			{
				ActionType:  "amadeus.search_cars",
				Name:        "Search Car Rentals",
				Description: "Search available rental cars at a location",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"pickup_location": {"type": "string", "description": "Pickup IATA code"},
						"pickup_date": {"type": "string", "description": "Pickup date/time"},
						"dropoff_date": {"type": "string", "description": "Dropoff date/time"},
						"dropoff_location": {"type": "string", "description": "Dropoff IATA code (defaults to pickup location)"},
						"provider": {"type": "string", "description": "Transfer type/provider filter"}
					},
					"required": ["pickup_location", "pickup_date", "dropoff_date"]
				}`),
			},
		},
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
func (c *AmadeusConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"amadeus.search_airports": &searchAirportsAction{conn: c},
		"amadeus.search_flights":  &searchFlightsAction{conn: c},
		"amadeus.price_flight":    &priceFlightAction{conn: c},
		"amadeus.book_flight":     &bookFlightAction{conn: c},
		"amadeus.search_hotels":   &searchHotelsAction{conn: c},
		"amadeus.search_cars":     &searchCarsAction{conn: c},
	}
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
// Tokens are cached per client_id so different credential sets don't
// share tokens. It uses the Amadeus client credentials grant:
// POST /v1/security/oauth2/token with grant_type=client_credentials.
func (c *AmadeusConnector) ensureToken(ctx context.Context, creds connectors.Credentials) (string, error) {
	clientID, _ := creds.Get("client_id")

	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid (with buffer).
	if cached, ok := c.tokens[clientID]; ok {
		if time.Now().Before(cached.expiresAt.Add(-tokenRefreshBuffer)) {
			return cached.accessToken, nil
		}
	}

	return c.fetchToken(ctx, creds, clientID)
}

// invalidateToken clears the cached token for a client_id, forcing a
// refresh on the next ensureToken call. Used when a 401 response
// suggests the token has been revoked or expired early.
func (c *AmadeusConnector) invalidateToken(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tokens, clientID)
}

// fetchToken performs the actual token exchange. Must be called with c.mu held.
func (c *AmadeusConnector) fetchToken(ctx context.Context, creds connectors.Credentials, clientID string) (string, error) {
	clientSecret, _ := creds.Get("client_secret")
	baseURL := c.resolveBaseURL(creds)

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/v1/security/oauth2/token",
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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

	c.tokens[clientID] = cachedToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return tokenResp.AccessToken, nil
}

// tokenResponse holds the fields we need from the Amadeus OAuth2 token
// endpoint response. Additional fields (type, username, scope, etc.) are
// ignored — json.Unmarshal silently skips them.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
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
//
// If the API returns 401 (token expired or revoked), do() invalidates
// the cached token and retries once with a fresh token.
func (c *AmadeusConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	// Pre-marshal the request body so we can replay it on retry.
	var payload []byte
	if reqBody != nil {
		var err error
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	statusCode, err := c.doOnce(ctx, creds, method, path, payload, respBody)

	// Only retry on 401 (expired/invalid token), not 403 or other auth
	// errors. A 403 means the credentials lack permission — a fresh
	// token won't help.
	if err != nil && statusCode == http.StatusUnauthorized {
		clientID, _ := creds.Get("client_id")
		c.invalidateToken(clientID)
		_, err = c.doOnce(ctx, creds, method, path, payload, respBody)
		return err
	}
	return err
}

// doOnce executes a single API request. payload is the pre-marshaled
// JSON body (nil for bodyless requests like GET). Returns the HTTP status
// code (0 if the request didn't complete) and any error.
func (c *AmadeusConnector) doOnce(ctx context.Context, creds connectors.Credentials, method, path string, payload []byte, respBody interface{}) (int, error) {
	token, err := c.ensureToken(ctx, creds)
	if err != nil {
		return 0, err
	}

	baseURL := c.resolveBaseURL(creds)

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return 0, &connectors.TimeoutError{Message: fmt.Sprintf("Amadeus API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return 0, &connectors.TimeoutError{Message: "Amadeus API request canceled"}
		}
		return 0, &connectors.ExternalError{Message: fmt.Sprintf("Amadeus API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return resp.StatusCode, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return resp.StatusCode, err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return resp.StatusCode, &connectors.ExternalError{Message: fmt.Sprintf("parsing Amadeus response: %v", err)}
		}
	}
	return resp.StatusCode, nil
}
