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

// Manifest returns the connector's metadata manifest. Action metadata is
// declared here for DB seeding; the actual Action handlers are wired in
// Actions() as they are implemented in Phase 2.
func (c *SquareConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "square",
		Name:        "Square",
		Description: "Square integration for orders, payments, catalog, customers, and bookings",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "square.create_order",
				Name:        "Create Order",
				Description: "Create an order at a Square location. Use for restaurant orders, retail sales, or service invoices. Returns the order ID and total. Use square.list_catalog first to find valid item names and prices.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_id", "line_items"],
					"additionalProperties": false,
					"properties": {
						"location_id": {
							"type": "string",
							"description": "Square location ID (e.g. \"L1234ABCD\"). Find via the Square Dashboard or API."
						},
						"line_items": {
							"type": "array",
							"minItems": 1,
							"description": "One or more items in the order",
							"items": {
								"type": "object",
								"required": ["name", "quantity", "base_price_money"],
								"additionalProperties": false,
								"properties": {
									"name": {
										"type": "string",
										"description": "Display name of the item (e.g. \"Latte\", \"T-Shirt\")"
									},
									"quantity": {
										"type": "string",
										"description": "Quantity as a string (Square API requirement). Example: \"1\", \"2\""
									},
									"base_price_money": {
										"type": "object",
										"required": ["amount", "currency"],
										"additionalProperties": false,
										"properties": {
											"amount": {
												"type": "integer",
												"description": "Price in smallest currency unit. For USD: cents. Example: $10.50 = 1050"
											},
											"currency": {
												"type": "string",
												"description": "ISO 4217 currency code (e.g. \"USD\", \"EUR\", \"GBP\")"
											}
										}
									}
								}
							}
						},
						"customer_id": {
							"type": "string",
							"description": "Square customer ID to link this order to a customer profile"
						},
						"note": {
							"type": "string",
							"description": "Free-text note attached to the order (visible to staff)"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_payment",
				Name:        "Create Payment",
				Description: "Process a payment. WARNING: This charges real money in production. Use source_id \"CASH\" for cash payments or a card nonce/token for card payments. Always double-check the amount before submitting.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["source_id", "amount_money"],
					"additionalProperties": false,
					"properties": {
						"source_id": {
							"type": "string",
							"description": "Payment source: a card nonce from Square Web Payments SDK, a card-on-file ID, or \"CASH\" for cash payments. Use \"cnon:card-nonce-ok\" in sandbox."
						},
						"amount_money": {
							"type": "object",
							"required": ["amount", "currency"],
							"additionalProperties": false,
							"properties": {
								"amount": {
									"type": "integer",
									"description": "Charge amount in smallest currency unit. For USD: cents. Example: $25.00 = 2500"
								},
								"currency": {
									"type": "string",
									"description": "ISO 4217 currency code (e.g. \"USD\")"
								}
							}
						},
						"order_id": {
							"type": "string",
							"description": "Link payment to an existing order (from square.create_order)"
						},
						"customer_id": {
							"type": "string",
							"description": "Square customer ID to associate with this payment"
						},
						"note": {
							"type": "string",
							"description": "Note displayed on the payment receipt"
						},
						"reference_id": {
							"type": "string",
							"description": "Your own external reference ID for reconciliation"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.list_catalog",
				Name:        "List Catalog",
				Description: "Browse the merchant's catalog of items, categories, discounts, taxes, and modifiers. Use this to discover what products are available before creating orders. Supports pagination for large catalogs.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"types": {
							"type": "string",
							"description": "Comma-separated object types: ITEM, CATEGORY, DISCOUNT, TAX, MODIFIER, IMAGE. Default: all types. Example: \"ITEM,CATEGORY\""
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous list_catalog response to fetch the next page"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_customer",
				Name:        "Create Customer",
				Description: "Create a customer profile in the merchant's directory. The customer ID can then be used with orders, payments, and bookings to build a purchase history.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["given_name"],
					"additionalProperties": false,
					"properties": {
						"given_name": {
							"type": "string",
							"description": "Customer's first name (required)"
						},
						"family_name": {
							"type": "string",
							"description": "Customer's last name"
						},
						"email_address": {
							"type": "string",
							"format": "email",
							"description": "Customer's email address"
						},
						"phone_number": {
							"type": "string",
							"description": "Customer's phone number (E.164 format preferred, e.g. \"+15551234567\")"
						},
						"company_name": {
							"type": "string",
							"description": "Customer's company or business name"
						},
						"note": {
							"type": "string",
							"description": "Internal note about the customer (not visible to the customer)"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_booking",
				Name:        "Create Booking",
				Description: "Schedule an appointment via Square Appointments. Use for salons, spas, consultations, or any service-based business. Requires a service variation ID from the catalog.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_id", "start_at", "service_variation_id"],
					"additionalProperties": false,
					"properties": {
						"location_id": {
							"type": "string",
							"description": "Square location ID where the appointment takes place"
						},
						"customer_id": {
							"type": "string",
							"description": "Square customer ID for the person being booked"
						},
						"start_at": {
							"type": "string",
							"format": "date-time",
							"description": "Appointment start time in RFC 3339 format (e.g. \"2024-03-15T14:30:00Z\")"
						},
						"service_variation_id": {
							"type": "string",
							"description": "Catalog service variation ID defining the service type and duration"
						},
						"team_member_id": {
							"type": "string",
							"description": "Specific staff member to assign (omit for any available)"
						},
						"customer_note": {
							"type": "string",
							"description": "Note from the customer about the appointment (e.g. special requests)"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.search_orders",
				Name:        "Search Orders",
				Description: "Search and filter orders across one or more locations. Filter by order state (OPEN, COMPLETED, CANCELED), date range, or customer. Returns up to 500 orders per page with cursor-based pagination.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_ids"],
					"additionalProperties": false,
					"properties": {
						"location_ids": {
							"type": "array",
							"minItems": 1,
							"items": {"type": "string"},
							"description": "One or more Square location IDs to search across"
						},
						"query": {
							"type": "object",
							"description": "Search filters: {\"filter\": {\"state_filter\": {\"states\": [\"OPEN\"]}, \"date_time_filter\": {\"closed_at\": {\"start_at\": \"...\", \"end_at\": \"...\"}}}}"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 500,
							"description": "Maximum orders to return per page (1-500, default 500)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous search_orders response"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "square",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.squareup.com/docs/build-basics/access-tokens",
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
// Empty in Phase 1 — actions are wired up in Phase 2.
func (c *SquareConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
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
