package square

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// moneySchema is the reusable JSON schema fragment for Square's Money object.
// Square represents all monetary amounts in the smallest currency unit
// (e.g., cents for USD). This is shared across create_order and create_payment.
var moneySchema = `{
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
			"description": "ISO 4217 currency code (e.g. USD, EUR, GBP)"
		}
	}
}`

// Manifest returns the connector's metadata manifest for DB auto-seeding.
// Includes full parameter JSON schemas for all 11 actions and configuration
// templates for common use cases.
func (c *SquareConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "square",
		Name:        "Square",
		Description: "Square integration for orders, payments, catalog, customers, bookings, refunds, invoices, and inventory",
		Actions: []connectors.ManifestAction{
			createOrderManifest(),
			createPaymentManifest(),
			listCatalogManifest(),
			createCustomerManifest(),
			createBookingManifest(),
			searchOrdersManifest(),
			issueRefundManifest(),
			updateCatalogItemManifest(),
			sendInvoiceManifest(),
			getInventoryManifest(),
			adjustInventoryManifest(),
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "square",
				AuthType:      "oauth2",
				OAuthProvider: "square",
				OAuthScopes: []string{
					"ORDERS_READ",
					"ORDERS_WRITE",
					"PAYMENTS_READ",
					"PAYMENTS_WRITE",
					"ITEMS_READ",
					"ITEMS_WRITE",
					"CUSTOMERS_READ",
					"CUSTOMERS_WRITE",
					"APPOINTMENTS_READ",
					"APPOINTMENTS_WRITE",
					"INVOICES_READ",
					"INVOICES_WRITE",
					"INVENTORY_READ",
					"INVENTORY_WRITE",
				},
			},
			{
				Service:         "square_api_key",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.squareup.com/docs/build-basics/access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_square_create_order",
				ActionType:  "square.create_order",
				Name:        "Create orders at a location",
				Description: "Agent can create orders at a specific Square location. Set location_id to lock to one location, or use \"*\" for any.",
				Parameters:  json.RawMessage(`{"location_id":"*","line_items":"*","customer_id":"*","note":"*"}`),
			},
			{
				ID:          "tpl_square_create_payment_cash",
				ActionType:  "square.create_payment",
				Name:        "Process cash payments only",
				Description: "Agent can process cash payments only (no card charges). Amount and order are agent-controlled. Requires human approval per payment.",
				Parameters:  json.RawMessage(`{"source_id":"CASH","amount_money":"*","order_id":"*","customer_id":"*","note":"*","reference_id":"*"}`),
			},
			{
				ID:          "tpl_square_create_payment",
				ActionType:  "square.create_payment",
				Name:        "Process payments (all sources)",
				Description: "Agent can process payments from any source including cards. WARNING: can charge real money. Requires human approval per payment.",
				Parameters:  json.RawMessage(`{"source_id":"*","amount_money":"*","order_id":"*","customer_id":"*","note":"*","reference_id":"*"}`),
			},
			{
				ID:          "tpl_square_list_catalog",
				ActionType:  "square.list_catalog",
				Name:        "Browse catalog (read-only)",
				Description: "Agent can browse the merchant's full catalog of items, categories, and modifiers.",
				Parameters:  json.RawMessage(`{"types":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_square_create_customer",
				ActionType:  "square.create_customer",
				Name:        "Create customer profiles",
				Description: "Agent can create customer records with any contact details.",
				Parameters:  json.RawMessage(`{"given_name":"*","family_name":"*","email_address":"*","phone_number":"*","company_name":"*","note":"*"}`),
			},
			{
				ID:          "tpl_square_create_booking",
				ActionType:  "square.create_booking",
				Name:        "Book appointments",
				Description: "Agent can schedule appointments at any location for any service. Requires human approval per booking.",
				Parameters:  json.RawMessage(`{"location_id":"*","customer_id":"*","start_at":"*","service_variation_id":"*","team_member_id":"*","customer_note":"*"}`),
			},
			{
				ID:          "tpl_square_search_orders",
				ActionType:  "square.search_orders",
				Name:        "Search orders (read-only)",
				Description: "Agent can search and filter orders across locations.",
				Parameters:  json.RawMessage(`{"location_ids":"*","query":"*","limit":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_square_issue_refund",
				ActionType:  "square.issue_refund",
				Name:        "Issue refunds",
				Description: "Agent can refund payments. WARNING: returns real money and is irreversible. Requires human approval per refund.",
				Parameters:  json.RawMessage(`{"payment_id":"*","amount_money":"*","reason":"*"}`),
			},
			{
				ID:          "tpl_square_update_catalog_item",
				ActionType:  "square.update_catalog_item",
				Name:        "Update catalog items",
				Description: "Agent can update product names, descriptions, and prices in the catalog.",
				Parameters:  json.RawMessage(`{"object_id":"*","name":"*","description":"*","variations":"*","version":"*"}`),
			},
			{
				ID:          "tpl_square_send_invoice",
				ActionType:  "square.send_invoice",
				Name:        "Send invoices",
				Description: "Agent can create and send invoices to customers. WARNING: sends real payment requests via email or SMS. Requires human approval per invoice.",
				Parameters:  json.RawMessage(`{"customer_id":"*","location_id":"*","line_items":"*","due_date":"*","delivery_method":"*","title":"*","note":"*"}`),
			},
			{
				ID:          "tpl_square_get_inventory",
				ActionType:  "square.get_inventory",
				Name:        "View inventory (read-only)",
				Description: "Agent can check inventory counts for catalog items across locations.",
				Parameters:  json.RawMessage(`{"catalog_object_ids":"*","location_ids":"*"}`),
			},
			{
				ID:          "tpl_square_adjust_inventory",
				ActionType:  "square.adjust_inventory",
				Name:        "Adjust inventory counts",
				Description: "Agent can adjust inventory quantities (e.g. receive stock, mark as sold). Changes are recoverable.",
				Parameters:  json.RawMessage(`{"catalog_object_id":"*","location_id":"*","quantity":"*","from_state":"*","to_state":"*"}`),
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
					"minimum": 0,
					"maximum": 500,
					"description": "Maximum orders per page (1-500). 0 or omit to use Square's default."
				},
				"cursor": {
					"type": "string",
					"description": "Pagination cursor from a previous search_orders response"
				}
			}
		}`)),
	}
}

func issueRefundManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.issue_refund",
		Name:        "Issue Refund",
		Description: "Refund a payment in full or partially. WARNING: returns real money and is irreversible. Omit amount_money for a full refund. Always double-check the payment ID and amount before submitting.",
		RiskLevel:   "high",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(fmt.Sprintf(`{
			"type": "object",
			"required": ["payment_id"],
			"additionalProperties": false,
			"properties": {
				"payment_id": {
					"type": "string",
					"description": "ID of the payment to refund (from square.create_payment or square.search_orders)"
				},
				"amount_money": %s,
				"reason": {
					"type": "string",
					"description": "Reason for the refund (shown on the receipt)"
				}
			}
		}`, moneySchema))),
	}
}

func updateCatalogItemManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.update_catalog_item",
		Name:        "Update Catalog Item",
		Description: "Update a catalog item's name, description, or pricing. Uses Square's upsert endpoint. Always include the version field (from list_catalog) to prevent overwriting concurrent changes — omitting it risks silent data loss.",
		RiskLevel:   "medium",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(fmt.Sprintf(`{
			"type": "object",
			"required": ["object_id"],
			"additionalProperties": false,
			"properties": {
				"object_id": {
					"type": "string",
					"description": "ID of the catalog item to update (from square.list_catalog)"
				},
				"name": {
					"type": "string",
					"description": "New display name for the item"
				},
				"description": {
					"type": "string",
					"description": "New description for the item"
				},
				"variations": {
					"type": "array",
					"description": "Item variations (sizes, colors, etc.) with pricing",
					"items": {
						"type": "object",
						"required": ["id"],
						"additionalProperties": false,
						"properties": {
							"id": {
								"type": "string",
								"description": "Variation ID (use existing ID to update, or #new-variation-id for new)"
							},
							"name": {
								"type": "string",
								"description": "Variation name (e.g. \"Small\", \"Regular\", \"Large\")"
							},
							"pricing_type": {
								"type": "string",
								"enum": ["FIXED_PRICING", "VARIABLE_PRICING"],
								"description": "FIXED_PRICING for set price, VARIABLE_PRICING for open amount"
							},
							"price_money": %s,
							"version": {
								"type": "integer",
								"description": "Current version of this variation (for conflict detection)"
							}
						}
					}
				},
				"version": {
					"type": "integer",
					"description": "Current version of the catalog object (for conflict detection). Get from list_catalog."
				}
			}
		}`, moneySchema))),
	}
}

func sendInvoiceManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.send_invoice",
		Name:        "Send Invoice",
		Description: "Create and send an invoice to a customer. WARNING: sends a real payment request that the customer will receive immediately. Creates an order, generates the invoice, and publishes it in one step.",
		RiskLevel:   "high",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(fmt.Sprintf(`{
			"type": "object",
			"required": ["customer_id", "location_id", "line_items", "due_date"],
			"additionalProperties": false,
			"properties": {
				"customer_id": {
					"type": "string",
					"description": "Square customer ID for the invoice recipient (from square.create_customer)"
				},
				"location_id": {
					"type": "string",
					"description": "Square location ID the invoice is issued from"
				},
				"line_items": {
					"type": "array",
					"minItems": 1,
					"description": "Items to include on the invoice",
					"items": {
						"type": "object",
						"required": ["description", "quantity", "base_price_money"],
						"additionalProperties": false,
						"properties": {
							"description": {
								"type": "string",
								"description": "Line item description (e.g. \"Web Design Services\")"
							},
							"quantity": {
								"type": "string",
								"description": "Quantity as a string (Square API requirement). Example: \"1\", \"2\""
							},
							"base_price_money": %s
						}
					}
				},
				"due_date": {
					"type": "string",
					"description": "Payment due date in YYYY-MM-DD format (e.g. \"2024-12-31\")"
				},
				"delivery_method": {
					"type": "string",
					"enum": ["EMAIL", "SMS", "SHARE_MANUALLY"],
					"description": "How to deliver the invoice. Default: EMAIL"
				},
				"title": {
					"type": "string",
					"description": "Invoice title (e.g. \"March 2024 Services\")"
				},
				"note": {
					"type": "string",
					"description": "Additional note included on the invoice"
				}
			}
		}`, moneySchema))),
	}
}

func getInventoryManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.get_inventory",
		Name:        "Get Inventory",
		Description: "Retrieve current inventory counts for one or more catalog items. Read-only — does not modify any data. Use this to check stock levels before adjusting inventory.",
		RiskLevel:   "low",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["catalog_object_ids"],
			"additionalProperties": false,
			"properties": {
				"catalog_object_ids": {
					"type": "array",
					"minItems": 1,
					"items": {"type": "string"},
					"description": "One or more catalog object IDs to retrieve inventory counts for"
				},
				"location_ids": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Filter counts to specific locations. Omit to get counts across all locations."
				}
			}
		}`)),
	}
}

func adjustInventoryManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "square.adjust_inventory",
		Name:        "Adjust Inventory",
		Description: "Adjust inventory counts for a catalog item at a location. Use to receive stock (NONE → IN_STOCK), record sales (IN_STOCK → SOLD), process returns (SOLD → RETURNED_BY_CUSTOMER), or any other state transition.",
		RiskLevel:   "medium",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["catalog_object_id", "location_id", "quantity", "from_state", "to_state"],
			"additionalProperties": false,
			"properties": {
				"catalog_object_id": {
					"type": "string",
					"description": "ID of the catalog item to adjust inventory for"
				},
				"location_id": {
					"type": "string",
					"description": "Square location ID where the inventory change occurs"
				},
				"quantity": {
					"type": "string",
					"description": "Quantity to adjust as a string (e.g. \"10\", \"5\")"
				},
				"from_state": {
					"type": "string",
					"enum": ["NONE", "IN_STOCK", "SOLD", "RETURNED_BY_CUSTOMER", "RESERVED_FOR_SALE", "SOLD_ONLINE", "ORDERED_FROM_VENDOR", "RECEIVED_FROM_VENDOR", "IN_TRANSIT_TO", "WASTE", "UNLINKED_RETURN", "COMPOSED", "DECOMPOSED", "SUPPORTED_BY_NEWER_VERSION"],
					"description": "Current inventory state"
				},
				"to_state": {
					"type": "string",
					"enum": ["NONE", "IN_STOCK", "SOLD", "RETURNED_BY_CUSTOMER", "RESERVED_FOR_SALE", "SOLD_ONLINE", "ORDERED_FROM_VENDOR", "RECEIVED_FROM_VENDOR", "IN_TRANSIT_TO", "WASTE", "UNLINKED_RETURN", "COMPOSED", "DECOMPOSED", "SUPPORTED_BY_NEWER_VERSION"],
					"description": "Target inventory state"
				}
			}
		}`)),
	}
}
