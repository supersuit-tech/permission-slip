// This file is split from stripe.go to keep the main connector file focused
// on struct, auth, and HTTP lifecycle. The manifest contains action schemas
// and templates that would obscure the business logic if inlined.
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
		Description: "Stripe integration for payments, invoicing, billing, subscriptions, and payouts",
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
			{
				ActionType:  "stripe.create_subscription",
				Name:        "Create Subscription",
				Description: "Create a recurring subscription for a customer — starts a billing cycle",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer", "items"],
					"additionalProperties": false,
					"properties": {
						"customer": {
							"type": "string",
							"description": "Stripe customer ID (e.g. \"cus_ABC123\")"
						},
						"items": {
							"type": "array",
							"minItems": 1,
							"description": "Subscription items — each references a price",
							"items": {
								"type": "object",
								"required": ["price"],
								"additionalProperties": false,
								"properties": {
									"price": {
										"type": "string",
										"description": "Stripe price ID (e.g. \"price_ABC123\")"
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"description": "Quantity for this item (defaults to 1)"
									}
								}
							}
						},
						"trial_period_days": {
							"type": "integer",
							"minimum": 0,
							"description": "Number of trial days before billing starts"
						},
						"payment_behavior": {
							"type": "string",
							"enum": ["default_incomplete", "error_if_incomplete", "allow_incomplete", "pending_if_incomplete"],
							"description": "How to handle payment failures (defaults to default_incomplete)"
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
				ActionType:  "stripe.cancel_subscription",
				Name:        "Cancel Subscription",
				Description: "Cancel an active subscription — WARNING: causes revenue loss and may impact customer",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["subscription_id"],
					"additionalProperties": false,
					"properties": {
						"subscription_id": {
							"type": "string",
							"description": "Stripe subscription ID (e.g. \"sub_ABC123\")"
						},
						"cancel_at_period_end": {
							"type": "boolean",
							"description": "When true, cancels at the end of the current billing period instead of immediately"
						},
						"proration_behavior": {
							"type": "string",
							"enum": ["create_prorations", "none", "always_invoice"],
							"description": "How to handle prorations for immediate cancellation"
						}
					}
				}`)),
			},
			{
				ActionType:  "stripe.create_coupon",
				Name:        "Create Coupon",
				Description: "Create a discount coupon — percent or fixed amount off, with duration controls",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["duration"],
					"additionalProperties": false,
					"properties": {
						"percent_off": {
							"type": "number",
							"exclusiveMinimum": 0,
							"maximum": 100,
							"description": "Percentage discount (e.g. 25.5 for 25.5% off) — provide this or amount_off"
						},
						"amount_off": {
							"type": "integer",
							"minimum": 1,
							"description": "Fixed discount in smallest currency unit (e.g. 500 = $5.00) — provide this or percent_off"
						},
						"currency": {
							"type": "string",
							"description": "Three-letter ISO 4217 currency code — required when using amount_off"
						},
						"duration": {
							"type": "string",
							"enum": ["once", "repeating", "forever"],
							"description": "How long the discount applies: once, repeating (requires duration_in_months), or forever"
						},
						"duration_in_months": {
							"type": "integer",
							"minimum": 1,
							"description": "Number of months the coupon applies — required when duration is \"repeating\""
						},
						"max_redemptions": {
							"type": "integer",
							"minimum": 1,
							"description": "Maximum number of times the coupon can be redeemed"
						},
						"name": {
							"type": "string",
							"description": "Display name for the coupon (shown on invoices)"
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
				ActionType:  "stripe.create_promotion_code",
				Name:        "Create Promotion Code",
				Description: "Create a shareable promotion code for an existing coupon",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["coupon"],
					"additionalProperties": false,
					"properties": {
						"coupon": {
							"type": "string",
							"description": "Stripe coupon ID to attach this promotion code to"
						},
						"code": {
							"type": "string",
							"description": "Customer-facing code (e.g. \"SUMMER25\") — auto-generated if omitted"
						},
						"max_redemptions": {
							"type": "integer",
							"minimum": 1,
							"description": "Maximum number of times this promotion code can be redeemed"
						},
						"expires_at": {
							"type": "integer",
							"description": "Expiration date as Unix timestamp (seconds since epoch)"
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
				ActionType:  "stripe.initiate_payout",
				Name:        "Initiate Payout",
				Description: "Trigger payout to connected bank account — WARNING: moves real money and cannot be undone",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["amount", "currency"],
					"additionalProperties": false,
					"properties": {
						"amount": {
							"type": "integer",
							"minimum": 1,
							"description": "Payout amount in smallest currency unit (e.g. 10000 = $100.00)"
						},
						"currency": {
							"type": "string",
							"description": "Three-letter ISO 4217 currency code (e.g. \"usd\")"
						},
						"description": {
							"type": "string",
							"description": "Internal description for the payout"
						},
						"destination": {
							"type": "string",
							"description": "Bank account or card ID to send funds to (uses default if omitted)"
						},
						"metadata": {
							"type": "object",
							"description": "Key-value pairs for storing additional information (max 50 keys)",
							"additionalProperties": { "type": "string" }
						}
					}
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
			Parameters:  json.RawMessage(`{"email":"*","name":"*","description":"*","phone":"*","metadata":"*"}`),
		},
		// --- Write (medium risk) ---
		{
			ID:          "tpl_stripe_create_invoices",
			ActionType:  "stripe.create_invoice",
			Name:        "Create invoices",
			Description: "Agent can create and send invoices for any customer with any line items.",
			Parameters:  json.RawMessage(`{"customer_id":"*","description":"*","line_items":"*","currency":"*","auto_advance":"*","due_date":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_payment_links",
			ActionType:  "stripe.create_payment_link",
			Name:        "Create payment links",
			Description: "Agent can create shareable payment links for any products.",
			Parameters:  json.RawMessage(`{"line_items":"*","after_completion":"*","allow_promotion_codes":"*","metadata":"*"}`),
		},
		// --- Write (medium risk — subscriptions & promotions) ---
		{
			ID:          "tpl_stripe_create_subscriptions",
			ActionType:  "stripe.create_subscription",
			Name:        "Create subscriptions",
			Description: "Agent can create recurring subscriptions for any customer and price, with optional trials and metadata.",
			Parameters:  json.RawMessage(`{"customer":"*","items":"*","trial_period_days":"*","payment_behavior":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_subscriptions_no_trial",
			ActionType:  "stripe.create_subscription",
			Name:        "Create subscriptions (no trials)",
			Description: "Agent can create subscriptions but cannot grant free trial periods — trial_period_days is locked to 0.",
			Parameters:  json.RawMessage(`{"customer":"*","items":"*","trial_period_days":0,"payment_behavior":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_coupons",
			ActionType:  "stripe.create_coupon",
			Name:        "Create coupons",
			Description: "Agent can create discount coupons with configurable percentage/amount, currency, duration, limits, and name.",
			Parameters:  json.RawMessage(`{"percent_off":"*","amount_off":"*","currency":"*","duration":"*","duration_in_months":"*","max_redemptions":"*","name":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_promotion_codes",
			ActionType:  "stripe.create_promotion_code",
			Name:        "Create promotion codes",
			Description: "Agent can create shareable promotion codes for any coupon.",
			Parameters:  json.RawMessage(`{"coupon":"*","code":"*","max_redemptions":"*","expires_at":"*","metadata":"*"}`),
		},
		// --- Write (high risk) ---
		{
			ID:          "tpl_stripe_cancel_subscription_end_of_period",
			ActionType:  "stripe.cancel_subscription",
			Name:        "Cancel subscriptions (end of period)",
			Description: "Agent can cancel subscriptions at the end of the current billing period only — safer than immediate cancellation.",
			Parameters:  json.RawMessage(`{"subscription_id":"*","cancel_at_period_end":true,"proration_behavior":"*"}`),
		},
		{
			ID:          "tpl_stripe_cancel_subscription_immediate",
			ActionType:  "stripe.cancel_subscription",
			Name:        "Cancel subscriptions (immediate)",
			Description: "Agent can cancel subscriptions immediately or at period end. High risk — use only for trusted agents.",
			Parameters:  json.RawMessage(`{"subscription_id":"*","cancel_at_period_end":"*","proration_behavior":"*"}`),
		},
		{
			ID:          "tpl_stripe_initiate_payout_default_dest",
			ActionType:  "stripe.initiate_payout",
			Name:        "Initiate payouts (default destination)",
			Description: "Agent can trigger payouts to the account's default bank. Destination is omitted so it cannot be overridden.",
			Parameters:  json.RawMessage(`{"amount":"*","currency":"*","description":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_initiate_payout_full",
			ActionType:  "stripe.initiate_payout",
			Name:        "Initiate payouts (any destination)",
			Description: "Agent can trigger payouts to any bank account or card. Highest risk — moves real money to bank accounts. Use only with strong oversight.",
			Parameters:  json.RawMessage(`{"amount":"*","currency":"*","description":"*","destination":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_issue_refund_partial_only",
			ActionType:  "stripe.issue_refund",
			Name:        "Issue partial refunds only",
			Description: "Agent can issue partial refunds (amount is required). Full refunds are blocked because amount cannot be omitted. Use the uncapped template for full refund capability.",
			Parameters:  json.RawMessage(`{"payment_intent_id":"*","charge_id":"*","amount":"*","reason":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_issue_refund_full",
			ActionType:  "stripe.issue_refund",
			Name:        "Issue refunds (any amount)",
			Description: "Agent can issue refunds of any amount, including full refunds (by omitting amount). High risk — use only for trusted agents with oversight.",
			Parameters:  json.RawMessage(`{"payment_intent_id":"*","charge_id":"*","amount":"*","reason":"*","metadata":"*"}`),
		},
	}
}
