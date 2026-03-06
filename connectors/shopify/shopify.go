// Package shopify implements the Shopify connector for the Permission Slip
// connector execution layer. It uses the Shopify Admin REST API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout = 30 * time.Second
	apiVersion     = "2024-10"

	credKeyAccessToken = "access_token"
	credKeyShopDomain  = "shop_domain"

	// defaultRetryAfter is used when the Shopify API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 2 * time.Second

	// maxResponseBodySize limits how much data we read from Shopify responses
	// to prevent memory exhaustion from unexpectedly large payloads.
	maxResponseBodySize = 10 * 1024 * 1024 // 10 MB
)

// validSubdomain matches valid Shopify store subdomains: lowercase alphanumeric
// and hyphens, not starting or ending with a hyphen.
var validSubdomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ShopifyConnector owns the shared HTTP client used by all Shopify actions.
// Unlike GitHub/Slack, the base URL is dynamic — it's derived from the
// shop_domain credential at request time.
type ShopifyConnector struct {
	client     *http.Client
	baseURLFn  func(creds connectors.Credentials) (string, error)
}

// New creates a ShopifyConnector with sensible defaults (30s timeout,
// dynamic base URL from shop_domain credential).
func New() *ShopifyConnector {
	return &ShopifyConnector{
		client:    &http.Client{Timeout: defaultTimeout},
		baseURLFn: shopBaseURL,
	}
}

// newForTest creates a ShopifyConnector that always uses the given base URL,
// ignoring the shop_domain credential. This lets tests point at httptest servers.
func newForTest(client *http.Client, baseURL string) *ShopifyConnector {
	return &ShopifyConnector{
		client:    client,
		baseURLFn: func(_ connectors.Credentials) (string, error) { return baseURL, nil },
	}
}

// shopBaseURL derives the Shopify Admin API base URL from credentials.
// Accepts either the bare subdomain ("mystore") or full domain
// ("mystore.myshopify.com") — anything else is rejected.
func shopBaseURL(creds connectors.Credentials) (string, error) {
	domain, ok := creds.Get(credKeyShopDomain)
	if !ok || domain == "" {
		return "", &connectors.ValidationError{Message: "shop_domain credential is missing or empty"}
	}

	// Strip trailing slash and whitespace.
	domain = strings.TrimRight(strings.TrimSpace(domain), "/")

	// Accept full domain or bare subdomain.
	shop := domain
	if strings.HasSuffix(domain, ".myshopify.com") {
		shop = strings.TrimSuffix(domain, ".myshopify.com")
	} else if strings.Contains(domain, ".") {
		return "", &connectors.ValidationError{
			Message: fmt.Sprintf("shop_domain must be a subdomain (e.g. \"mystore\") or full Shopify domain (e.g. \"mystore.myshopify.com\"), got %q", domain),
		}
	}

	if shop == "" {
		return "", &connectors.ValidationError{Message: "shop_domain resolved to an empty subdomain"}
	}

	// Normalize to lowercase — Shopify subdomains are case-insensitive
	// and the canonical form is lowercase.
	shop = strings.ToLower(shop)

	// Validate the subdomain contains only safe hostname characters to prevent
	// URL injection or SSRF via crafted shop_domain values.
	if !validSubdomain.MatchString(shop) {
		return "", &connectors.ValidationError{
			Message: fmt.Sprintf("shop_domain contains invalid characters: %q (expected lowercase alphanumeric and hyphens)", shop),
		}
	}

	return fmt.Sprintf("https://%s.myshopify.com/admin/api/%s", shop, apiVersion), nil
}

// ID returns "shopify", matching the connectors.id in the database.
func (c *ShopifyConnector) ID() string { return "shopify" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *ShopifyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "shopify",
		Name:        "Shopify",
		Description: "Shopify integration for store management via the Admin REST API",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "shopify.get_orders",
				Name:        "Get Orders",
				Description: "List or filter orders from the Shopify store",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"status": {
							"type": "string",
							"enum": ["open", "closed", "cancelled", "any"],
							"default": "open",
							"description": "Filter orders by status"
						},
						"financial_status": {
							"type": "string",
							"enum": ["paid", "unpaid", "partially_paid", "refunded", "authorized", "pending", "any"],
							"description": "Filter orders by financial status"
						},
						"created_at_min": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders created at or after this date (ISO 8601, e.g. 2024-01-01T00:00:00Z)"
						},
						"created_at_max": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders created at or before this date (ISO 8601, e.g. 2024-12-31T23:59:59Z)"
						},
						"updated_at_min": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders updated at or after this date (ISO 8601)"
						},
						"updated_at_max": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders updated at or before this date (ISO 8601)"
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return (e.g. id,name,total_price). Omit to return all fields."
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of orders to return"
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.get_order",
				Name:        "Get Order",
				Description: "Get full details of a single order by ID",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"order_id": {
							"type": "integer",
							"description": "The Shopify order ID"
						}
					},
					"required": ["order_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.update_order",
				Name:        "Update Order",
				Description: "Update order attributes such as notes, tags, email, or shipping address",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"order_id": {
							"type": "integer",
							"description": "The Shopify order ID"
						},
						"note": {
							"type": "string",
							"description": "Internal order note visible to staff"
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags (e.g. \"vip,priority,reviewed\")"
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address"
						},
						"shipping_address": {
							"type": "object",
							"properties": {
								"address1": {"type": "string", "description": "Street address"},
								"address2": {"type": "string", "description": "Apartment, suite, etc."},
								"city": {"type": "string"},
								"province_code": {"type": "string", "description": "State/province code (e.g. NY, ON)"},
								"country_code": {"type": "string", "description": "ISO 3166-1 alpha-2 country code (e.g. US, CA)"},
								"zip": {"type": "string", "description": "Postal/ZIP code"},
								"phone": {"type": "string"}
							},
							"description": "Shipping address fields to update"
						}
					},
					"required": ["order_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.create_product",
				Name:        "Create Product",
				Description: "Create a new product listing in the Shopify store",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"title": {
							"type": "string",
							"description": "Product title"
						},
						"body_html": {
							"type": "string",
							"description": "Product description in HTML"
						},
						"vendor": {
							"type": "string",
							"description": "Product vendor"
						},
						"product_type": {
							"type": "string",
							"description": "Product type for categorization"
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags"
						},
						"status": {
							"type": "string",
							"enum": ["active", "draft", "archived"],
							"description": "Product status (defaults to active if omitted)"
						},
						"variants": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"price": {"type": "string", "description": "Variant price as decimal string (e.g. \"19.99\")"},
									"sku": {"type": "string", "description": "Unique SKU code"},
									"inventory_quantity": {"type": "integer", "description": "Initial inventory count"},
									"option1": {"type": "string", "description": "First option value (e.g. \"Small\")"},
									"option2": {"type": "string", "description": "Second option value (e.g. \"Red\")"},
									"option3": {"type": "string", "description": "Third option value"}
								}
							},
							"description": "Product variants — each variant can have up to 3 options (e.g. size, color)"
						}
					},
					"required": ["title"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.update_inventory",
				Name:        "Update Inventory",
				Description: "Adjust inventory levels for a product variant at a location",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"inventory_item_id": {
							"type": "integer",
							"description": "The inventory item ID (from product variant)"
						},
						"location_id": {
							"type": "integer",
							"description": "The location ID where inventory is stored"
						},
						"available_adjustment": {
							"type": "integer",
							"description": "Inventory adjustment: positive to add, negative to subtract (e.g. -5 or +10; must be non-zero)"
						}
					},
					"required": ["inventory_item_id", "location_id", "available_adjustment"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.create_discount",
				Name:        "Create Discount",
				Description: "Create a discount code via a two-step flow: first creates a price rule, then attaches a discount code to it. If the second step fails, the price rule will exist without a code.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"code": {
							"type": "string",
							"description": "The discount code customers will enter (e.g. SUMMER10)"
						},
						"value_type": {
							"type": "string",
							"enum": ["percentage", "fixed_amount"],
							"description": "Type of discount: percentage or fixed_amount"
						},
						"value": {
							"type": "string",
							"description": "Discount value as a negative decimal string (e.g. \"-10.0\" for 10% off, or \"-5.00\" for $5 off in shop currency)"
						},
						"target_type": {
							"type": "string",
							"enum": ["line_item", "shipping_line"],
							"description": "What the discount applies to (defaults to line_item)"
						},
						"starts_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the discount becomes active (ISO 8601, e.g. 2024-06-01T00:00:00Z)"
						},
						"ends_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the discount expires (ISO 8601). Omit for no expiration."
						},
						"usage_limit": {
							"type": "integer",
							"description": "Maximum total uses across all customers. Omit for unlimited."
						},
						"applies_once_per_customer": {
							"type": "boolean",
							"description": "If true, each customer can use the code only once"
						}
					},
					"required": ["code", "value_type", "value", "starts_at"],
					"additionalProperties": false
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "shopify",
				AuthType:        "api_key",
				InstructionsURL: "https://shopify.dev/docs/apps/build/authentication-authorization/access-tokens/generate-app-access-tokens-admin",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "shopify-list-open-orders",
				ActionType:  "shopify.get_orders",
				Name:        "List open orders",
				Description: "Retrieve all currently open orders",
				Parameters:  json.RawMessage(`{"status":"open","limit":50}`),
			},
			{
				ID:          "shopify-list-recently-updated-orders",
				ActionType:  "shopify.get_orders",
				Name:        "List recently updated orders",
				Description: "Retrieve orders updated in the last 24 hours",
				Parameters:  json.RawMessage(`{"status":"any","updated_at_min":"2024-01-01T00:00:00Z","limit":50}`),
			},
			{
				ID:          "shopify-add-order-note",
				ActionType:  "shopify.update_order",
				Name:        "Add note to order",
				Description: "Add an internal note to an order for staff reference",
				Parameters:  json.RawMessage(`{"order_id":0,"note":"Agent processed — verified and approved"}`),
			},
			{
				ID:          "shopify-create-draft-product",
				ActionType:  "shopify.create_product",
				Name:        "Create draft product with variants",
				Description: "Create a new product in draft status with size variants",
				Parameters:  json.RawMessage(`{"title":"New Product","status":"draft","variants":[{"price":"19.99","sku":"NP-SM","option1":"Small"},{"price":"19.99","sku":"NP-LG","option1":"Large"}]}`),
			},
			{
				ID:          "shopify-adjust-inventory",
				ActionType:  "shopify.update_inventory",
				Name:        "Adjust inventory level",
				Description: "Add or subtract inventory for a product variant at a location",
				Parameters:  json.RawMessage(`{"inventory_item_id":0,"location_id":0,"available_adjustment":-1}`),
			},
			{
				ID:          "shopify-create-percentage-discount",
				ActionType:  "shopify.create_discount",
				Name:        "Create 10% discount code",
				Description: "Create a percentage-based discount code for all products",
				Parameters:  json.RawMessage(`{"code":"SAVE10","value_type":"percentage","value":"-10.0","starts_at":"2024-01-01T00:00:00Z"}`),
			},
			{
				ID:          "shopify-create-fixed-discount",
				ActionType:  "shopify.create_discount",
				Name:        "Create $5 off discount code",
				Description: "Create a fixed-amount discount code",
				Parameters:  json.RawMessage(`{"code":"SAVE5","value_type":"fixed_amount","value":"-5.00","starts_at":"2024-01-01T00:00:00Z"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *ShopifyConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"shopify.get_orders":      &getOrdersAction{conn: c},
		"shopify.get_order":       &getOrderAction{conn: c},
		"shopify.update_order":    &updateOrderAction{conn: c},
		"shopify.create_product":  &createProductAction{conn: c},
		"shopify.update_inventory": &updateInventoryAction{conn: c},
		"shopify.create_discount": &createDiscountAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain both a
// non-empty access_token and a valid shop_domain.
func (c *ShopifyConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	domain, ok := creds.Get(credKeyShopDomain)
	if !ok || domain == "" {
		return &connectors.ValidationError{Message: "missing required credential: shop_domain"}
	}

	// Validate the domain produces a valid base URL.
	if _, err := shopBaseURL(creds); err != nil {
		return err
	}

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}

	return nil
}

// do is the shared request lifecycle for all Shopify actions. It derives
// the base URL from credentials, marshals reqBody as JSON, sends the request
// with the X-Shopify-Access-Token header, checks the response status, and
// unmarshals the response into respBody.
func (c *ShopifyConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	baseURL, err := c.baseURLFn(creds)
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

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}
	req.Header.Set("X-Shopify-Access-Token", token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Shopify API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Shopify API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Shopify API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Shopify response: %v", err)}
		}
	}
	return nil
}
