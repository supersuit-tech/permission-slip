// This file is split from stripe.go to keep the main connector file focused
// on struct, auth, and HTTP lifecycle. The manifest contains 6 action schemas
// and 8 templates (~270 lines of JSON Schema definitions) that would obscure
// the business logic if inlined in the connector file.
package stripe

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *StripeConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "stripe",
		Name:        "Stripe",
		Description: "Stripe integration for payments, invoicing, and billing management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "stripe.create_customer",
				Name:        "Create Customer",
				Description: "Create a new customer record — foundational for all other Stripe operations",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["email"],
					"additionalProperties": false,
					"properties": {
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address (e.g. \"billing@acme.com\")"
						},
						"name": {
							"type": "string",
							"description": "Customer full name or company name"
						},
						"description": {
							"type": "string",
							"description": "Free-form description of the customer"
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number in E.164 format (e.g. \"+14155551234\")"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys, values must be strings)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.create_invoice",
				Name:        "Create Invoice",
				Description: "Create and optionally send an invoice with line items",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id"],
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "Stripe customer ID (e.g. \"cus_ABC123\")"
						},
						"description": {
							"type": "string",
							"description": "Invoice memo or description shown to the customer"
						},
						"due_date": {
							"type": "integer",
							"description": "Due date as Unix timestamp (seconds since epoch)"
						},
						"auto_advance": {
							"type": "boolean",
							"default": true,
							"description": "When true, automatically finalize and send the invoice to the customer"
						},
						"currency": {
							"type": "string",
							"default": "usd",
							"description": "Three-letter ISO 4217 currency code (e.g. \"usd\", \"eur\", \"gbp\")"
						},
						"line_items": {
							"type": "array",
							"description": "Invoice line items — each becomes an InvoiceItem attached to the invoice",
							"items": {
								"type": "object",
								"additionalProperties": false,
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description shown on the invoice"
									},
									"amount": {
										"type": "integer",
										"minimum": 1,
										"description": "Amount in smallest currency unit (e.g. 1050 cents = $10.50)"
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"default": 1,
										"description": "Quantity (defaults to 1)"
									}
								}
							}
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.issue_refund",
				Name:        "Issue Refund",
				Description: "Refund a charge or payment intent — WARNING: this moves real money and cannot be undone",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"payment_intent_id": {
							"type": "string",
							"description": "Payment intent ID (e.g. \"pi_ABC123\") — provide this or charge_id"
						},
						"charge_id": {
							"type": "string",
							"description": "Charge ID (e.g. \"ch_ABC123\") — provide this or payment_intent_id"
						},
						"amount": {
							"type": "integer",
							"minimum": 1,
							"description": "Refund amount in cents for partial refund (e.g. 500 = $5.00). Omit for full refund"
						},
						"reason": {
							"type": "string",
							"enum": ["duplicate", "fraudulent", "requested_by_customer"],
							"description": "Reason for the refund — shown in the Stripe dashboard and receipts"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.list_subscriptions",
				Name:        "List Subscriptions",
				Description: "List subscriptions, optionally filtered by customer or status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "Filter by Stripe customer ID (e.g. \"cus_ABC123\")"
						},
						"status": {
							"type": "string",
							"enum": ["active", "past_due", "canceled", "unpaid", "trialing", "all"],
							"description": "Filter by subscription status (defaults to all non-canceled)"
						},
						"price_id": {
							"type": "string",
							"description": "Filter by price ID (e.g. \"price_ABC123\")"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Number of subscriptions to return (default 10, max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.create_payment_link",
				Name:        "Create Payment Link",
				Description: "Create a shareable payment link for one-time or recurring purchases",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["line_items"],
					"additionalProperties": false,
					"properties": {
						"line_items": {
							"type": "array",
							"minItems": 1,
							"description": "Products to include in the payment link (at least one required)",
							"items": {
								"type": "object",
								"required": ["price_id", "quantity"],
								"additionalProperties": false,
								"properties": {
									"price_id": {
										"type": "string",
										"description": "Stripe price ID (e.g. \"price_ABC123\") — must be a pre-created Price object"
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"description": "Quantity of the product"
									}
								}
							}
						},
						"after_completion": {
							"type": "string",
							"format": "uri",
							"description": "URL to redirect customers to after successful payment"
						},
						"allow_promotion_codes": {
							"type": "boolean",
							"description": "Allow customers to enter promotion codes at checkout"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.get_balance",
				Name:        "Get Balance",
				Description: "Retrieve the current account balance — useful for monitoring cash flow",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "stripe",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.stripe.com/keys",
			},
		},
		Templates: stripeTemplates(),
	}
}

// stripeTemplates returns configuration templates for common Stripe use cases.
// Templates offer different permission levels — from read-only balance checks
// to capped refunds — so administrators can grant agents the minimum access
// they need.
func stripeTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		// --- Read-only ---
		{
			ID:          "tpl_stripe_get_balance",
			ActionType:  "stripe.get_balance",
			Name:        "Check account balance",
			Description: "Agent can retrieve the current Stripe account balance. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{}`),
		},
		{
			ID:          "tpl_stripe_list_subscriptions_active",
			ActionType:  "stripe.list_subscriptions",
			Name:        "List active subscriptions",
			Description: "Agent can list active subscriptions for any customer. Status is locked to \"active\".",
			Parameters:  json.RawMessage(`{"customer_id":"*","status":"active","limit":"*"}`),
		},
		{
			ID:          "tpl_stripe_list_subscriptions_any",
			ActionType:  "stripe.list_subscriptions",
			Name:        "List subscriptions (any status)",
			Description: "Agent can list subscriptions in any status — active, past due, canceled, etc.",
			Parameters:  json.RawMessage(`{"customer_id":"*","status":"*","price_id":"*","limit":"*"}`),
		},
		// --- Write (low risk) ---
		{
			ID:          "tpl_stripe_create_customers",
			ActionType:  "stripe.create_customer",
			Name:        "Create customers",
			Description: "Agent can create new Stripe customer records with any details.",
			Parameters:  json.RawMessage(`{"email":"*","name":"*","description":"*","phone":"*"}`),
		},
		// --- Write (medium risk) ---
		{
			ID:          "tpl_stripe_create_invoices",
			ActionType:  "stripe.create_invoice",
			Name:        "Create invoices",
			Description: "Agent can create and send invoices for any customer with any line items.",
			Parameters:  json.RawMessage(`{"customer_id":"*","description":"*","line_items":"*","currency":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_payment_links",
			ActionType:  "stripe.create_payment_link",
			Name:        "Create payment links",
			Description: "Agent can create shareable payment links for any products.",
			Parameters:  json.RawMessage(`{"line_items":"*","after_completion":"*","allow_promotion_codes":"*"}`),
		},
		// --- Write (high risk) ---
		{
			ID:          "tpl_stripe_issue_refund_capped",
			ActionType:  "stripe.issue_refund",
			Name:        "Issue refunds up to $99.99",
			Description: "Agent can issue refunds capped at 9999 cents ($99.99). The amount is constrained by a regex pattern to prevent large refunds — requires human approval for anything $100+.",
			Parameters:  json.RawMessage(`{"payment_intent_id":"*","charge_id":"*","amount":{"$pattern":"^[1-9]\\d{0,3}$"},"reason":"*"}`),
		},
		{
			ID:          "tpl_stripe_issue_refund_full",
			ActionType:  "stripe.issue_refund",
			Name:        "Issue refunds (any amount)",
			Description: "Agent can issue refunds of any amount, including full refunds. High risk — use only for trusted agents with oversight.",
			Parameters:  json.RawMessage(`{"payment_intent_id":"*","charge_id":"*","amount":"*","reason":"*"}`),
		},
	}
}
