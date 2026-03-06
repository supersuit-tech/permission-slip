// Package square implements the Square connector for the Permission Slip
// connector execution layer. It uses the Square REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package square

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
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
				Description: "Create an order (restaurant order, retail sale, etc.)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_id", "line_items"],
					"properties": {
						"location_id": {
							"type": "string",
							"description": "The ID of the business location for this order"
						},
						"line_items": {
							"type": "array",
							"description": "Line items to include in the order",
							"items": {
								"type": "object",
								"required": ["name", "quantity", "base_price_money"],
								"properties": {
									"name": {
										"type": "string",
										"description": "The name of the line item"
									},
									"quantity": {
										"type": "string",
										"description": "The quantity (as a string, per Square API)"
									},
									"base_price_money": {
										"type": "object",
										"required": ["amount", "currency"],
										"properties": {
											"amount": {
												"type": "integer",
												"description": "Amount in the smallest currency unit (e.g. cents for USD)"
											},
											"currency": {
												"type": "string",
												"description": "ISO 4217 currency code (e.g. USD)"
											}
										}
									}
								}
							}
						},
						"customer_id": {
							"type": "string",
							"description": "Optional Square customer ID to associate with the order"
						},
						"note": {
							"type": "string",
							"description": "Optional note to attach to the order"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_payment",
				Name:        "Create Payment",
				Description: "Process a payment. High risk — charges real money.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["source_id", "amount_money"],
					"properties": {
						"source_id": {
							"type": "string",
							"description": "Payment source: a payment token or \"CASH\" for cash payments"
						},
						"amount_money": {
							"type": "object",
							"required": ["amount", "currency"],
							"properties": {
								"amount": {
									"type": "integer",
									"description": "Amount in the smallest currency unit (e.g. cents for USD)"
								},
								"currency": {
									"type": "string",
									"description": "ISO 4217 currency code (e.g. USD)"
								}
							}
						},
						"order_id": {
							"type": "string",
							"description": "Optional order ID to associate with this payment"
						},
						"customer_id": {
							"type": "string",
							"description": "Optional Square customer ID"
						},
						"note": {
							"type": "string",
							"description": "Optional note for the payment"
						},
						"reference_id": {
							"type": "string",
							"description": "Optional external reference ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.list_catalog",
				Name:        "List Catalog",
				Description: "List menu items, products, and categories from the catalog",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"types": {
							"type": "string",
							"description": "Comma-separated catalog object types to filter (e.g. ITEM,CATEGORY,DISCOUNT,TAX,MODIFIER)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_customer",
				Name:        "Create Customer",
				Description: "Create a customer record",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["given_name"],
					"properties": {
						"given_name": {
							"type": "string",
							"description": "Customer's first name"
						},
						"family_name": {
							"type": "string",
							"description": "Customer's last name"
						},
						"email_address": {
							"type": "string",
							"description": "Customer's email address"
						},
						"phone_number": {
							"type": "string",
							"description": "Customer's phone number"
						},
						"company_name": {
							"type": "string",
							"description": "Customer's company name"
						},
						"note": {
							"type": "string",
							"description": "Optional note about the customer"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.create_booking",
				Name:        "Create Booking",
				Description: "Create an appointment booking (salon, spa, professional services)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_id", "start_at", "service_variation_id"],
					"properties": {
						"location_id": {
							"type": "string",
							"description": "The ID of the business location for this booking"
						},
						"customer_id": {
							"type": "string",
							"description": "Square customer ID for the booking"
						},
						"start_at": {
							"type": "string",
							"description": "Booking start time in RFC 3339 format"
						},
						"service_variation_id": {
							"type": "string",
							"description": "The ID of the catalog service variation to book"
						},
						"team_member_id": {
							"type": "string",
							"description": "Optional team member to assign the booking to"
						},
						"customer_note": {
							"type": "string",
							"description": "Optional note from the customer"
						}
					}
				}`)),
			},
			{
				ActionType:  "square.search_orders",
				Name:        "Search Orders",
				Description: "Search and filter orders across locations",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["location_ids"],
					"properties": {
						"location_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Location IDs to search across"
						},
						"query": {
							"type": "object",
							"description": "Search query with filters (state, date range, customer)"
						},
						"limit": {
							"type": "integer",
							"description": "Maximum number of orders to return (default 500)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
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
		return &connectors.ExternalError{Message: fmt.Sprintf("Square API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Square response: %v", err)}
		}
	}
	return nil
}
