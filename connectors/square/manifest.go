package square

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// moneySchema is the reusable JSON schema fragment for Square's Money object.
// Square represents all monetary amounts in the smallest currency unit
// (e.g., cents for USD). This is shared across create_order and create_payment.
var moneySchema = fmt.Sprintf(`{
	"type": "object",
	"required": ["amount", "currency"],
	"additionalProperties": false,
	"properties": {
		"amount": {
			"type": "integer",
			"description": "Amount in smallest currency unit. For USD: cents. Example: $10.50 = 1050"
		},
		"currency": {
			"type": "string",
			"description": "ISO 4217 currency code (e.g. %q, %q, %q)"
		}
	}
}`, "USD", "EUR", "GBP")

// Manifest returns the connector's metadata manifest. Action metadata is
// declared here for DB seeding; the actual Action handlers are wired in
// Actions() as they are implemented in Phase 2.
func (c *SquareConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "square",
		Name:        "Square",
		Description: "Square integration for orders, payments, catalog, customers, and bookings",
		Actions: []connectors.ManifestAction{
			createOrderManifest(),
			createPaymentManifest(),
			listCatalogManifest(),
			createCustomerManifest(),
			createBookingManifest(),
			searchOrdersManifest(),
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

func createOrderManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.create_order",
		Name:        "Create Order",
		Description: "Create an order at a Square location. Use for restaurant orders, retail sales, or service invoices. Returns the order ID and total. Use square.list_catalog first to find valid item names and prices.",
		RiskLevel:   "medium",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(fmt.Sprintf(`{
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
							"base_price_money": %s
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
		}`, moneySchema))),
	}
}

func createPaymentManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.create_payment",
		Name:        "Create Payment",
		Description: "Process a payment. WARNING: This charges real money in production. Use source_id \"CASH\" for cash payments or a card nonce/token for card payments. Always double-check the amount before submitting.",
		RiskLevel:   "high",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(fmt.Sprintf(`{
			"type": "object",
			"required": ["source_id", "amount_money"],
			"additionalProperties": false,
			"properties": {
				"source_id": {
					"type": "string",
					"description": "Payment source: a card nonce from Square Web Payments SDK, a card-on-file ID, or \"CASH\" for cash payments. Use \"cnon:card-nonce-ok\" in sandbox."
				},
				"amount_money": %s,
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
		}`, moneySchema))),
	}
}

func listCatalogManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
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
	}
}

func createCustomerManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
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
	}
}

func createBookingManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
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
	}
}

func searchOrdersManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
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
	}
}
