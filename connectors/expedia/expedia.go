// Package expedia implements the Expedia Rapid connector for the Permission
// Slip connector execution layer. It uses the Expedia Rapid API (formerly
// EAN/Expedia Affiliate Network) with HMAC-SHA512 signature authentication.
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

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL    = "https://api.ean.com"
	defaultTimeout    = 30 * time.Second
	defaultCustomerIP = "127.0.0.1"
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

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup. Actions are defined in Phase 2 and will be
// added to this manifest then.
func (c *ExpediaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "expedia",
		Name:        "Expedia Rapid",
		Description: "Expedia Rapid API integration for hotel search and booking",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "expedia.search_hotels",
				Name:        "Search Hotels",
				Description: "Search available hotels with pricing",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["checkin", "checkout", "occupancy"],
					"properties": {
						"checkin": {
							"type": "string",
							"description": "Check-in date (YYYY-MM-DD)"
						},
						"checkout": {
							"type": "string",
							"description": "Check-out date (YYYY-MM-DD)"
						},
						"region_id": {
							"type": "string",
							"description": "Expedia region ID to search in"
						},
						"latitude": {
							"type": "number",
							"description": "Latitude for location-based search"
						},
						"longitude": {
							"type": "number",
							"description": "Longitude for location-based search"
						},
						"occupancy": {
							"type": "string",
							"description": "Occupancy string (e.g. '2' for 2 adults, '2-0,4' for 2 adults + 1 child age 4)"
						},
						"currency": {
							"type": "string",
							"description": "Currency code (e.g. USD, EUR)"
						},
						"language": {
							"type": "string",
							"description": "Language code (e.g. en-US)"
						},
						"sort_by": {
							"type": "string",
							"enum": ["price", "distance", "rating"],
							"description": "Sort results by price, distance, or rating"
						},
						"star_rating": {
							"type": "array",
							"items": {"type": "integer"},
							"description": "Filter by star rating(s)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results to return"
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.get_hotel",
				Name:        "Get Hotel Details",
				Description: "Get full hotel details: photos, amenities, room types, policies, reviews summary",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["property_id"],
					"properties": {
						"property_id": {
							"type": "string",
							"description": "Expedia property ID"
						},
						"checkin": {
							"type": "string",
							"description": "Check-in date (YYYY-MM-DD) for rate information"
						},
						"checkout": {
							"type": "string",
							"description": "Check-out date (YYYY-MM-DD) for rate information"
						},
						"occupancy": {
							"type": "string",
							"description": "Occupancy string for rate information"
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.price_check",
				Name:        "Price Check",
				Description: "Confirm real-time pricing and availability before booking",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["room_id"],
					"properties": {
						"room_id": {
							"type": "string",
							"description": "Room ID from search results"
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.create_booking",
				Name:        "Create Booking",
				Description: "Book a hotel room. High risk — creates a real reservation and may charge payment.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["room_id", "given_name", "family_name", "email", "phone", "payment_method_id"],
					"properties": {
						"room_id": {
							"type": "string",
							"description": "Room ID from a successful price check"
						},
						"given_name": {
							"type": "string",
							"description": "Guest first name"
						},
						"family_name": {
							"type": "string",
							"description": "Guest last name"
						},
						"email": {
							"type": "string",
							"description": "Guest email address"
						},
						"phone": {
							"type": "string",
							"description": "Guest phone number"
						},
						"payment_method_id": {
							"type": "string",
							"description": "Stored payment method ID (resolved server-side)"
						},
						"special_request": {
							"type": "string",
							"description": "Special requests for the hotel"
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.cancel_booking",
				Name:        "Cancel Booking",
				Description: "Cancel a hotel booking — may incur cancellation fees depending on policy",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["itinerary_id", "room_id"],
					"properties": {
						"itinerary_id": {
							"type": "string",
							"description": "Itinerary ID from the booking"
						},
						"room_id": {
							"type": "string",
							"description": "Room ID within the itinerary to cancel"
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.get_booking",
				Name:        "Get Booking",
				Description: "Retrieve booking details and current status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["itinerary_id", "email"],
					"properties": {
						"itinerary_id": {
							"type": "string",
							"description": "Itinerary ID from the booking"
						},
						"email": {
							"type": "string",
							"description": "Email address used for the booking"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "expedia", AuthType: "api_key", InstructionsURL: "https://developers.expediagroup.com/docs/products/rapid/setup/getting-started"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_expedia_search_read_only",
				ActionType:  "expedia.search_hotels",
				Name:        "Search hotels (read-only)",
				Description: "Agent can search hotels for any dates, location, and occupancy. No booking capability.",
				Parameters:  json.RawMessage(`{"checkin":"*","checkout":"*","occupancy":"*","region_id":"*","latitude":"*","longitude":"*","currency":"*","language":"*","sort_by":"*","star_rating":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_expedia_get_hotel",
				ActionType:  "expedia.get_hotel",
				Name:        "View hotel details",
				Description: "Agent can view full details for any hotel property.",
				Parameters:  json.RawMessage(`{"property_id":"*","checkin":"*","checkout":"*","occupancy":"*"}`),
			},
			{
				ID:          "tpl_expedia_price_check",
				ActionType:  "expedia.price_check",
				Name:        "Check room pricing",
				Description: "Agent can confirm pricing and availability for any room.",
				Parameters:  json.RawMessage(`{"room_id":"*"}`),
			},
			{
				ID:          "tpl_expedia_create_booking",
				ActionType:  "expedia.create_booking",
				Name:        "Book hotel rooms",
				Description: "Agent can create hotel bookings. Requires human approval per booking.",
				Parameters:  json.RawMessage(`{"room_id":"*","given_name":"*","family_name":"*","email":"*","phone":"*","payment_method_id":"*","special_request":"*"}`),
			},
			{
				ID:          "tpl_expedia_cancel_booking",
				ActionType:  "expedia.cancel_booking",
				Name:        "Cancel bookings",
				Description: "Agent can cancel hotel bookings. Requires human approval per cancellation.",
				Parameters:  json.RawMessage(`{"itinerary_id":"*","room_id":"*"}`),
			},
			{
				ID:          "tpl_expedia_get_booking",
				ActionType:  "expedia.get_booking",
				Name:        "View booking details",
				Description: "Agent can retrieve booking details and status.",
				Parameters:  json.RawMessage(`{"itinerary_id":"*","email":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
// Phase 2 will add action implementations here.
func (c *ExpediaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		// Phase 2 actions will be registered here.
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key and secret, which are required for HMAC-SHA512 signature
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

// signature generates the HMAC-SHA512 signature for Expedia Rapid API
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
	apiKey, _ := creds.Get("api_key")
	secret, _ := creds.Get("secret")
	if apiKey == "" || secret == "" {
		return &connectors.ValidationError{Message: "api_key and secret credentials are required"}
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
			return &connectors.TimeoutError{Message: fmt.Sprintf("Expedia Rapid API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Expedia Rapid API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Expedia Rapid API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Expedia Rapid response: %v", err)}
		}
	}
	return nil
}
