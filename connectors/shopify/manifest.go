package shopify

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//
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
							"description": "Filter orders by status",
							"x-ui": {
								"widget": "select",
								"label": "Status"
							}
						},
						"financial_status": {
							"type": "string",
							"enum": ["paid", "unpaid", "partially_paid", "refunded", "authorized", "pending", "any"],
							"description": "Filter orders by financial status",
							"x-ui": {
								"widget": "select",
								"label": "Financial status"
							}
						},
						"created_at_min": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders created at or after this date (ISO 8601, e.g. 2024-01-01T00:00:00Z)",
							"x-ui": {
								"widget": "datetime",
								"label": "Created after",
								"help_text": "Only include orders created on or after this date and time",
								"datetime_range_pair": "created_at_max",
								"datetime_range_role": "lower"
							}
						},
						"created_at_max": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders created at or before this date (ISO 8601, e.g. 2024-12-31T23:59:59Z)",
							"x-ui": {
								"widget": "datetime",
								"label": "Created before",
								"help_text": "Only include orders created on or before this date and time",
								"datetime_range_pair": "created_at_min",
								"datetime_range_role": "upper"
							}
						},
						"updated_at_min": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders updated at or after this date (ISO 8601)",
							"x-ui": {
								"widget": "datetime",
								"label": "Updated after",
								"help_text": "Only include orders updated on or after this date and time",
								"datetime_range_pair": "updated_at_max",
								"datetime_range_role": "lower"
							}
						},
						"updated_at_max": {
							"type": "string",
							"format": "date-time",
							"description": "Show orders updated at or before this date (ISO 8601)",
							"x-ui": {
								"widget": "datetime",
								"label": "Updated before",
								"help_text": "Only include orders updated on or before this date and time",
								"datetime_range_pair": "updated_at_min",
								"datetime_range_role": "upper"
							}
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return (e.g. id,name,total_price). Omit to return all fields.",
							"x-ui": {
								"label": "Fields",
								"help_text": "Comma-separated field names to include"
							}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of orders to return",
							"x-ui": {
								"label": "Max results"
							}
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
							"description": "The Shopify order ID",
							"x-ui": {
								"label": "Order ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
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
							"description": "The Shopify order ID",
							"x-ui": {
								"label": "Order ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"note": {
							"type": "string",
							"description": "Internal order note visible to staff",
							"x-ui": {
								"widget": "textarea",
								"label": "Note"
							}
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags (e.g. \"vip,priority,reviewed\")",
							"x-ui": {
								"label": "Tags",
								"help_text": "Comma-separated tags"
							}
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "jane@example.com"
							}
						},
						"shipping_address": {
							"type": "object",
							"properties": {
								"address1": {
									"type": "string",
									"description": "Street address",
									"x-ui": {
										"label": "Address line 1",
										"placeholder": "123 Main St"
									}
								},
								"address2": {
									"type": "string",
									"description": "Apartment, suite, etc.",
									"x-ui": {
										"label": "Address line 2",
										"placeholder": "Apt 4B"
									}
								},
								"city": {
									"type": "string",
									"x-ui": {
										"label": "City",
										"placeholder": "New York"
									}
								},
								"province_code": {
									"type": "string",
									"description": "State/province code (e.g. NY, ON)",
									"x-ui": {
										"label": "State/province code",
										"placeholder": "NY"
									}
								},
								"country_code": {
									"type": "string",
									"description": "ISO 3166-1 alpha-2 country code (e.g. US, CA)",
									"x-ui": {
										"label": "Country code",
										"placeholder": "US"
									}
								},
								"zip": {
									"type": "string",
									"description": "Postal/ZIP code",
									"x-ui": {
										"label": "ZIP/Postal code",
										"placeholder": "10001"
									}
								},
								"phone": {
									"type": "string",
									"x-ui": {
										"label": "Phone",
										"placeholder": "+15551234567"
									}
								}
							},
							"description": "Shipping address fields to update",
							"x-ui": {
								"label": "Shipping address"
							}
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
							"description": "Product title",
							"x-ui": {
								"label": "Title",
								"placeholder": "Classic Cotton T-Shirt"
							}
						},
						"body_html": {
							"type": "string",
							"description": "Product description in HTML",
							"x-ui": {
								"widget": "textarea",
								"label": "Description (HTML)"
							}
						},
						"vendor": {
							"type": "string",
							"description": "Product vendor",
							"x-ui": {
								"label": "Vendor",
								"placeholder": "Acme Corp"
							}
						},
						"product_type": {
							"type": "string",
							"description": "Product type for categorization",
							"x-ui": {
								"label": "Product type",
								"placeholder": "Apparel"
							}
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags",
							"x-ui": {
								"label": "Tags",
								"help_text": "Comma-separated tags"
							}
						},
						"status": {
							"type": "string",
							"enum": ["active", "draft", "archived"],
							"description": "Product status (defaults to active if omitted)",
							"x-ui": {
								"widget": "select",
								"label": "Status"
							}
						},
						"variants": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"price": {
										"type": "string",
										"description": "Variant price as decimal string (e.g. \"19.99\")",
										"x-ui": {
											"label": "Price",
											"placeholder": "29.99",
											"help_text": "Price in store currency (e.g. 29.99 for $29.99)"
										}
									},
									"sku": {
										"type": "string",
										"description": "Unique SKU code",
										"x-ui": {
											"label": "SKU",
											"placeholder": "WIDGET-001"
										}
									},
									"inventory_quantity": {
										"type": "integer",
										"description": "Initial inventory count",
										"x-ui": {
											"label": "Quantity"
										}
									},
									"option1": {
										"type": "string",
										"description": "First option value (e.g. \"Small\")",
										"x-ui": {
											"label": "Option 1",
											"placeholder": "Small"
										}
									},
									"option2": {
										"type": "string",
										"description": "Second option value (e.g. \"Red\")",
										"x-ui": {
											"label": "Option 2",
											"placeholder": "Red"
										}
									},
									"option3": {
										"type": "string",
										"description": "Third option value",
										"x-ui": {
											"label": "Option 3",
											"placeholder": "Cotton"
										}
									}
								}
							},
							"description": "Product variants — each variant can have up to 3 options (e.g. size, color)",
							"x-ui": {
								"label": "Variants"
							}
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
							"description": "The inventory item ID (from product variant)",
							"x-ui": {
								"label": "Inventory item ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"location_id": {
							"type": "integer",
							"minimum": 1,
							"description": "The location ID where inventory is stored",
							"x-ui": {
								"label": "Location ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"available_adjustment": {
							"type": "integer",
							"description": "Inventory adjustment: positive to add, negative to subtract (e.g. -5 or +10; must be non-zero)",
							"x-ui": {
								"label": "Adjustment",
								"help_text": "Positive to add stock, negative to remove"
							}
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
							"description": "The discount code customers will enter (e.g. SUMMER10)",
							"x-ui": {
								"label": "Discount code",
								"placeholder": "SUMMER25"
							}
						},
						"value_type": {
							"type": "string",
							"enum": ["percentage", "fixed_amount"],
							"description": "Type of discount: percentage or fixed_amount",
							"x-ui": {
								"widget": "select",
								"label": "Discount type"
							}
						},
						"value": {
							"type": "string",
							"description": "Discount value as a negative decimal string (e.g. \"-10.0\" for 10% off, or \"-5.00\" for $5 off in shop currency)",
							"x-ui": {
								"label": "Discount value",
								"help_text": "Negative decimal string — e.g. '-10.0' for 10% or $10 off"
							}
						},
						"target_type": {
							"type": "string",
							"enum": ["line_item", "shipping_line"],
							"description": "What the discount applies to (defaults to line_item)",
							"x-ui": {
								"widget": "select",
								"label": "Target type"
							}
						},
						"starts_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the discount becomes active (ISO 8601, e.g. 2024-06-01T00:00:00Z)",
							"x-ui": {
								"widget": "datetime",
								"label": "Starts at",
								"help_text": "Date and time the discount becomes active"
							}
						},
						"ends_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the discount expires (ISO 8601). Omit for no expiration.",
							"x-ui": {
								"widget": "datetime",
								"label": "Ends at",
								"help_text": "Date and time the discount expires. Leave blank for no expiration."
							}
						},
						"usage_limit": {
							"type": "integer",
							"description": "Maximum total uses across all customers. Omit for unlimited.",
							"x-ui": {
								"label": "Usage limit"
							}
						},
						"applies_once_per_customer": {
							"type": "boolean",
							"description": "If true, each customer can use the code only once",
							"x-ui": {
								"widget": "toggle",
								"label": "Once per customer"
							}
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
							"description": "The Shopify order ID to fulfill",
							"x-ui": {
								"label": "Order ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"tracking_number": {
							"type": "string",
							"description": "Shipment tracking number",
							"x-ui": {
								"label": "Tracking number",
								"placeholder": "1Z999AA10123456784"
							}
						},
						"tracking_company": {
							"type": "string",
							"description": "Shipping carrier name (e.g. UPS, FedEx, USPS)",
							"x-ui": {
								"label": "Tracking company",
								"placeholder": "UPS"
							}
						},
						"tracking_url": {
							"type": "string",
							"format": "uri",
							"description": "URL for tracking the shipment",
							"x-ui": {
								"label": "Tracking URL",
								"placeholder": "https://www.ups.com/track?tracknum=1Z999AA10123456784"
							}
						},
						"notify_customer": {
							"type": "boolean",
							"description": "Whether to send a shipment notification email to the customer",
							"x-ui": {
								"widget": "toggle",
								"label": "Notify customer"
							}
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
							"description": "The Shopify order ID to cancel",
							"x-ui": {
								"label": "Order ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"reason": {
							"type": "string",
							"enum": ["customer", "fraud", "inventory", "declined", "other"],
							"description": "Reason for cancellation",
							"x-ui": {
								"widget": "select",
								"label": "Reason"
							}
						},
						"restock": {
							"type": "boolean",
							"description": "Whether to restock the order's line items",
							"x-ui": {
								"widget": "toggle",
								"label": "Restock items"
							}
						},
						"email": {
							"type": "boolean",
							"description": "Whether to send a cancellation email to the customer",
							"x-ui": {
								"widget": "toggle",
								"label": "Send cancellation email"
							}
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
							"description": "The Shopify product ID to update",
							"x-ui": {
								"label": "Product ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"title": {
							"type": "string",
							"description": "Product title",
							"x-ui": {
								"label": "Title",
								"placeholder": "Classic Cotton T-Shirt"
							}
						},
						"body_html": {
							"type": "string",
							"description": "Product description in HTML",
							"x-ui": {
								"widget": "textarea",
								"label": "Description (HTML)"
							}
						},
						"vendor": {
							"type": "string",
							"description": "Product vendor",
							"x-ui": {
								"label": "Vendor",
								"placeholder": "Acme Corp"
							}
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated list of tags",
							"x-ui": {
								"label": "Tags",
								"help_text": "Comma-separated tags"
							}
						},
						"status": {
							"type": "string",
							"enum": ["active", "draft", "archived"],
							"description": "Product status",
							"x-ui": {
								"widget": "select",
								"label": "Status"
							}
						},
						"variants": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"id": {
										"type": "integer",
										"description": "Variant ID (required for updating existing variants)",
										"x-ui": {
											"label": "Variant ID",
											"help_text": "Numeric Shopify ID — find in the admin URL or API response"
										}
									},
									"price": {
										"type": "string",
										"description": "Variant price as decimal string",
										"x-ui": {
											"label": "Price",
											"placeholder": "29.99",
											"help_text": "Price in store currency (e.g. 29.99 for $29.99)"
										}
									},
									"sku": {
										"type": "string",
										"description": "Unique SKU code",
										"x-ui": {
											"label": "SKU",
											"placeholder": "WIDGET-001"
										}
									},
									"inventory_quantity": {
										"type": "integer",
										"description": "Inventory count",
										"x-ui": {
											"label": "Quantity"
										}
									},
									"option1": {
										"type": "string",
										"description": "First option value",
										"x-ui": {
											"label": "Option 1",
											"placeholder": "Small"
										}
									},
									"option2": {
										"type": "string",
										"description": "Second option value",
										"x-ui": {
											"label": "Option 2",
											"placeholder": "Red"
										}
									},
									"option3": {
										"type": "string",
										"description": "Third option value",
										"x-ui": {
											"label": "Option 3",
											"placeholder": "Cotton"
										}
									}
								}
							},
							"description": "Product variants to update or add",
							"x-ui": {
								"label": "Variants"
							}
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
							"description": "Collection title",
							"x-ui": {
								"label": "Title",
								"placeholder": "Summer Collection"
							}
						},
						"body_html": {
							"type": "string",
							"description": "Collection description in HTML",
							"x-ui": {
								"widget": "textarea",
								"label": "Description (HTML)"
							}
						},
						"published": {
							"type": "boolean",
							"description": "Whether the collection is published to the storefront",
							"x-ui": {
								"widget": "toggle",
								"label": "Published"
							}
						},
						"sort_order": {
							"type": "string",
							"enum": ["alpha-asc", "alpha-desc", "best-selling", "created", "created-desc", "manual", "price-asc", "price-desc"],
							"description": "Default sort order for products in the collection",
							"x-ui": {
								"widget": "select",
								"label": "Sort order"
							}
						},
						"image": {
							"type": "object",
							"properties": {
								"src": {
									"type": "string",
									"format": "uri",
									"description": "Image URL",
									"x-ui": {
										"label": "Image URL",
										"placeholder": "https://example.com/image.png"
									}
								},
								"alt": {
									"type": "string",
									"description": "Image alt text",
									"x-ui": {
										"label": "Alt text",
										"placeholder": "Summer collection banner"
									}
								}
							},
							"description": "Collection image",
							"x-ui": {
								"label": "Image"
							}
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
							"description": "Show reports created at or after this date (ISO 8601)",
							"x-ui": {
								"widget": "datetime",
								"label": "Since",
								"help_text": "Only include reports created on or after this date and time"
							}
						},
						"until": {
							"type": "string",
							"format": "date-time",
							"description": "Show reports created at or before this date (ISO 8601)",
							"x-ui": {
								"widget": "datetime",
								"label": "Until",
								"help_text": "Only include reports created on or before this date and time"
							}
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return (e.g. id,name,shopify_ql)",
							"x-ui": {
								"label": "Fields",
								"help_text": "Comma-separated field names to include"
							}
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
							"description": "Search query to filter customers by name, email, phone, or other fields",
							"x-ui": {
								"label": "Search query",
								"placeholder": "email:jane@example.com"
							}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of customers to return",
							"x-ui": {
								"label": "Max results"
							}
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields.",
							"x-ui": {
								"label": "Fields",
								"help_text": "Comma-separated field names to include"
							}
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
							"description": "The numeric Shopify customer ID",
							"x-ui": {
								"label": "Customer ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
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
							"description": "Customer email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "jane@example.com"
							}
						},
						"first_name": {
							"type": "string",
							"description": "Customer first name",
							"x-ui": {
								"label": "First name",
								"placeholder": "Jane"
							}
						},
						"last_name": {
							"type": "string",
							"description": "Customer last name",
							"x-ui": {
								"label": "Last name",
								"placeholder": "Doe"
							}
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number in E.164 format (e.g. +15551234567)",
							"x-ui": {
								"label": "Phone",
								"placeholder": "+15551234567"
							}
						},
						"note": {
							"type": "string",
							"description": "Internal note about the customer",
							"x-ui": {
								"widget": "textarea",
								"label": "Note"
							}
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated tags to attach to the customer",
							"x-ui": {
								"label": "Tags",
								"help_text": "Comma-separated tags"
							}
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
							"description": "Filter by product status",
							"x-ui": {
								"widget": "select",
								"label": "Status"
							}
						},
						"product_type": {
							"type": "string",
							"description": "Filter by product type",
							"x-ui": {
								"label": "Product type",
								"placeholder": "Apparel"
							}
						},
						"vendor": {
							"type": "string",
							"description": "Filter by product vendor name",
							"x-ui": {
								"label": "Vendor",
								"placeholder": "Acme Corp"
							}
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields.",
							"x-ui": {
								"label": "Fields",
								"help_text": "Comma-separated field names to include"
							}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 250,
							"default": 50,
							"description": "Maximum number of products to return",
							"x-ui": {
								"label": "Max results"
							}
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
							"description": "The numeric Shopify product ID",
							"x-ui": {
								"label": "Product ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"fields": {
							"type": "string",
							"description": "Comma-separated list of fields to return. Omit for all fields.",
							"x-ui": {
								"label": "Fields",
								"help_text": "Comma-separated field names to include"
							}
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
									"variant_id": {
										"type": "integer",
										"description": "Shopify variant ID",
										"x-ui": {
											"label": "Variant ID",
											"help_text": "Numeric Shopify ID — find in the admin URL or API response"
										}
									},
									"product_id": {
										"type": "integer",
										"description": "Shopify product ID",
										"x-ui": {
											"label": "Product ID",
											"help_text": "Numeric Shopify ID — find in the admin URL or API response"
										}
									},
									"title": {
										"type": "string",
										"description": "Custom line item title",
										"x-ui": {
											"label": "Title",
											"placeholder": "Custom item"
										}
									},
									"price": {
										"type": "string",
										"description": "Override price as decimal string",
										"x-ui": {
											"label": "Price",
											"placeholder": "29.99",
											"help_text": "Price in store currency (e.g. 29.99 for $29.99)"
										}
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"description": "Number of units",
										"x-ui": {
											"label": "Quantity"
										}
									}
								},
								"additionalProperties": false
							},
							"description": "Line items for the draft order",
							"x-ui": {
								"label": "Line items"
							}
						},
						"customer_id": {
							"type": "integer",
							"description": "Associate with an existing Shopify customer",
							"x-ui": {
								"label": "Customer ID",
								"help_text": "Numeric Shopify ID — find in the admin URL or API response"
							}
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email to associate with the draft order",
							"x-ui": {
								"label": "Email",
								"placeholder": "jane@example.com"
							}
						},
						"note": {
							"type": "string",
							"description": "Internal note about the draft order",
							"x-ui": {
								"widget": "textarea",
								"label": "Note"
							}
						},
						"tags": {
							"type": "string",
							"description": "Comma-separated tags to attach to the draft order",
							"x-ui": {
								"label": "Tags",
								"help_text": "Comma-separated tags"
							}
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
