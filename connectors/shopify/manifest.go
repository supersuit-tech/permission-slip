package shopify

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *ShopifyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "shopify",
		Name:        "Shopify",
		Description: "Shopify integration for store management via the Admin REST API",
		LogoSVG:     logoSVG,
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
							"minimum": 1,
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
							"minimum": 1,
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
							"minimum": 1,
							"description": "The inventory item ID (from product variant)"
						},
						"location_id": {
							"type": "integer",
							"minimum": 1,
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
			{
				ActionType:  "shopify.fulfill_order",
				Name:        "Fulfill Order",
				Description: "Create a fulfillment for an order with optional tracking information. When notify_customer is true, Shopify sends a shipment notification email.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"order_id": {
							"type": "integer",
							"minimum": 1,
							"description": "The Shopify order ID to fulfill"
						},
						"tracking_number": {
							"type": "string",
							"description": "Shipment tracking number"
						},
						"tracking_company": {
							"type": "string",
							"description": "Shipping carrier name (e.g. UPS, FedEx, USPS)"
						},
						"tracking_url": {
							"type": "string",
							"format": "uri",
							"description": "URL for tracking the shipment"
						},
						"notify_customer": {
							"type": "boolean",
							"description": "Whether to send a shipment notification email to the customer"
						}
					},
					"required": ["order_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.cancel_order",
				Name:        "Cancel Order",
				Description: "Cancel an order. This action is irreversible and affects the customer experience. Use with caution.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"order_id": {
							"type": "integer",
							"minimum": 1,
							"description": "The Shopify order ID to cancel"
						},
						"reason": {
							"type": "string",
							"enum": ["customer", "fraud", "inventory", "declined", "other"],
							"description": "Reason for cancellation"
						},
						"restock": {
							"type": "boolean",
							"description": "Whether to restock the order's line items"
						},
						"email": {
							"type": "boolean",
							"description": "Whether to send a cancellation email to the customer"
						}
					},
					"required": ["order_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.update_product",
				Name:        "Update Product",
				Description: "Update an existing product's attributes such as title, description, vendor, tags, status, or variants",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"product_id": {
							"type": "integer",
							"minimum": 1,
							"description": "The Shopify product ID to update"
						},
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
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags"
						},
						"status": {
							"type": "string",
							"enum": ["active", "draft", "archived"],
							"description": "Product status"
						},
						"variants": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"id": {"type": "integer", "description": "Variant ID (required for updating existing variants)"},
									"price": {"type": "string", "description": "Variant price as decimal string"},
									"sku": {"type": "string", "description": "Unique SKU code"},
									"inventory_quantity": {"type": "integer", "description": "Inventory count"},
									"option1": {"type": "string", "description": "First option value"},
									"option2": {"type": "string", "description": "Second option value"},
									"option3": {"type": "string", "description": "Third option value"}
								}
							},
							"description": "Product variants to update or add"
						}
					},
					"required": ["product_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.create_collection",
				Name:        "Create Collection",
				Description: "Create a custom product collection for organizing products in the storefront",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"title": {
							"type": "string",
							"description": "Collection title"
						},
						"body_html": {
							"type": "string",
							"description": "Collection description in HTML"
						},
						"published": {
							"type": "boolean",
							"description": "Whether the collection is published to the storefront"
						},
						"sort_order": {
							"type": "string",
							"enum": ["alpha-asc", "alpha-desc", "best-selling", "created", "created-desc", "manual", "price-asc", "price-desc"],
							"description": "Default sort order for products in the collection"
						},
						"image": {
							"type": "object",
							"properties": {
								"src": {"type": "string", "format": "uri", "description": "Image URL"},
								"alt": {"type": "string", "description": "Image alt text"}
							},
							"description": "Collection image"
						}
					},
					"required": ["title"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.get_analytics",
				Name:        "Get Analytics",
				Description: "Retrieve shop analytics reports. Note: Analytics API access varies by Shopify plan — not all plans support it.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"since": {
							"type": "string",
							"format": "date-time",
							"description": "Show reports created at or after this date (ISO 8601)"
						},
						"until": {
							"type": "string",
							"format": "date-time",
							"description": "Show reports created at or before this date (ISO 8601)"
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return (e.g. id,name,shopify_ql)"
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.list_customers",
				Name:        "List Customers",
				Description: "List or search customers in the Shopify store",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query to filter customers by name, email, phone, or other fields"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of customers to return"
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields."
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.get_customer",
				Name:        "Get Customer",
				Description: "Get full details of a single customer by ID",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id"],
					"properties": {
						"customer_id": {
							"type": "integer",
							"description": "The numeric Shopify customer ID"
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.create_customer",
				Name:        "Create Customer",
				Description: "Create a new customer record in the Shopify store",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address"
						},
						"first_name": {
							"type": "string",
							"description": "Customer first name"
						},
						"last_name": {
							"type": "string",
							"description": "Customer last name"
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number in E.164 format (e.g. +15551234567)"
						},
						"note": {
							"type": "string",
							"description": "Internal note about the customer"
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated tags to attach to the customer"
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.list_products",
				Name:        "List Products",
				Description: "List products with optional filtering by status, type, or vendor",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"status": {
							"type": "string",
							"enum": ["active", "draft", "archived"],
							"description": "Filter by product status"
						},
						"product_type": {
							"type": "string",
							"description": "Filter by product type"
						},
						"vendor": {
							"type": "string",
							"description": "Filter by product vendor name"
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields."
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of products to return"
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.get_product",
				Name:        "Get Product",
				Description: "Get full details of a single product by ID",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["product_id"],
					"properties": {
						"product_id": {
							"type": "integer",
							"description": "The numeric Shopify product ID"
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields."
						}
					},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "shopify.create_draft_order",
				Name:        "Create Draft Order",
				Description: "Create a draft order with line items. Commonly used for B2B workflows and manual order creation. Does not charge the customer until completed.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["line_items"],
					"properties": {
						"line_items": {
							"type": "array",
							"minItems": 1,
							"items": {
								"type": "object",
								"required": ["quantity"],
								"properties": {
									"variant_id": {"type": "integer", "description": "Shopify variant ID"},
									"product_id": {"type": "integer", "description": "Shopify product ID"},
									"title": {"type": "string", "description": "Custom line item title"},
									"price": {"type": "string", "description": "Override price as decimal string"},
									"quantity": {"type": "integer", "minimum": 1, "description": "Number of units"}
								},
								"additionalProperties": false
							},
							"description": "Line items for the draft order"
						},
						"customer_id": {
							"type": "integer",
							"description": "Associate with an existing Shopify customer"
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email to associate with the draft order"
						},
						"note": {
							"type": "string",
							"description": "Internal note about the draft order"
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated tags to attach to the draft order"
						}
					},
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
			{
				ID:          "shopify-fulfill-order-with-tracking",
				ActionType:  "shopify.fulfill_order",
				Name:        "Fulfill order with tracking",
				Description: "Create a fulfillment with tracking info and notify the customer",
				Parameters:  json.RawMessage(`{"order_id":0,"tracking_number":"1Z999AA10123456784","tracking_company":"UPS","notify_customer":true}`),
			},
			{
				ID:          "shopify-fulfill-order-silent",
				ActionType:  "shopify.fulfill_order",
				Name:        "Fulfill order (no notification)",
				Description: "Create a fulfillment without sending a notification email to the customer",
				Parameters:  json.RawMessage(`{"order_id":0,"notify_customer":false}`),
			},
			{
				ID:          "shopify-cancel-order-customer-request",
				ActionType:  "shopify.cancel_order",
				Name:        "Cancel order (customer request)",
				Description: "Cancel an order due to customer request, restock items, and notify the customer",
				Parameters:  json.RawMessage(`{"order_id":0,"reason":"customer","restock":true,"email":true}`),
			},
			{
				ID:          "shopify-cancel-order-fraud",
				ActionType:  "shopify.cancel_order",
				Name:        "Cancel order (fraud)",
				Description: "Cancel a fraudulent order, restock items, and suppress notification to the customer",
				Parameters:  json.RawMessage(`{"order_id":0,"reason":"fraud","restock":true,"email":false}`),
			},
			{
				ID:          "shopify-update-product-status",
				ActionType:  "shopify.update_product",
				Name:        "Update product status",
				Description: "Change a product's status (e.g. draft to active, or active to archived)",
				Parameters:  json.RawMessage(`{"product_id":0,"status":"active"}`),
			},
			{
				ID:          "shopify-update-product-details",
				ActionType:  "shopify.update_product",
				Name:        "Update product details",
				Description: "Update a product's title, description, and tags",
				Parameters:  json.RawMessage(`{"product_id":0,"title":"","body_html":"","tags":""}`),
			},
			{
				ID:          "shopify-create-collection",
				ActionType:  "shopify.create_collection",
				Name:        "Create product collection",
				Description: "Create a new published collection with alphabetical sorting",
				Parameters:  json.RawMessage(`{"title":"New Collection","published":true,"sort_order":"alpha-asc"}`),
			},
			{
				ID:          "shopify-get-analytics",
				ActionType:  "shopify.get_analytics",
				Name:        "Get shop reports",
				Description: "Retrieve all available analytics reports",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}
}
