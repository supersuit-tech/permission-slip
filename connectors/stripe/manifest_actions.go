package stripe

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// stripeActions returns the JSON Schema definitions for all Stripe actions.
// Each entry describes the parameters an agent may supply for that action,
// validated against the schema before execution.
func stripeActions() []connectors.ManifestAction {
	return []connectors.ManifestAction{
		{
			ActionType:  "stripe.create_customer",
			Name:        "Create Customer",
			Description: "Create a new customer record — foundational for all other Stripe operations",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["email"],
				"additionalProperties": false,
				"x-ui": {
					"order": ["email", "name", "phone", "description", "metadata"]
				},
				"properties": {
					"email": {
						"type": "string",
						"format": "email",
						"description": "Customer email address (e.g. \"billing@acme.com\")",
						"x-ui": { "label": "Email Address", "placeholder": "billing@acme.com", "help_text": "Primary contact email used for invoices and receipts" }
					},
					"name": {
						"type": "string",
						"description": "Customer full name or company name",
						"x-ui": { "label": "Full Name", "placeholder": "Acme Inc." }
					},
					"description": {
						"type": "string",
						"description": "Free-form description of the customer",
						"x-ui": { "label": "Description", "placeholder": "Enterprise client, annual billing", "widget": "textarea" }
					},
					"phone": {
						"type": "string",
						"description": "Customer phone number in E.164 format (e.g. \"+14155551234\")",
						"x-ui": { "label": "Phone Number", "placeholder": "+14155551234", "help_text": "Use E.164 format with country code" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys, values must be strings)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Add custom key-value pairs (e.g. internal_id, account_manager)" }
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
				"x-ui": {
					"groups": [
						{ "id": "billing", "label": "Billing" },
						{ "id": "options", "label": "Options", "collapsed": true }
					],
					"order": ["customer_id", "currency", "description", "due_date", "line_items", "auto_advance", "metadata"]
				},
				"properties": {
					"customer_id": {
						"type": "string",
						"description": "Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123", "group": "billing" }
					},
					"description": {
						"type": "string",
						"description": "Invoice memo or description shown to the customer",
						"x-ui": { "label": "Memo", "placeholder": "Invoice for March 2026 services", "widget": "textarea", "group": "billing" }
					},
					"due_date": {
						"type": "integer",
						"description": "Due date as Unix timestamp (seconds since epoch)",
						"x-ui": { "label": "Due Date", "group": "billing" }
					},
					"auto_advance": {
						"type": "boolean",
						"default": true,
						"description": "When true, automatically finalize and send the invoice to the customer",
						"x-ui": { "widget": "toggle", "label": "Auto-send invoice", "group": "options" }
					},
					"currency": {
						"type": "string",
						"default": "usd",
						"description": "Three-letter ISO 4217 currency code (e.g. \"usd\", \"eur\", \"gbp\")",
						"x-ui": { "label": "Currency", "group": "billing" }
					},
					"line_items": {
						"type": "array",
						"description": "Invoice line items — each becomes an InvoiceItem attached to the invoice",
						"x-ui": { "label": "Line Items", "group": "billing" },
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
						"additionalProperties": { "type": "string" },
						"x-ui": { "group": "options" }
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
				"x-ui": {
					"order": ["payment_intent_id", "charge_id", "amount", "reason", "metadata"]
				},
				"properties": {
					"payment_intent_id": {
						"type": "string",
						"description": "Payment intent ID (e.g. \"pi_ABC123\") — provide this or charge_id",
						"x-ui": { "label": "Payment Intent", "placeholder": "pi_ABC123" }
					},
					"charge_id": {
						"type": "string",
						"description": "Charge ID (e.g. \"ch_ABC123\") — provide this or payment_intent_id",
						"x-ui": { "label": "Charge", "placeholder": "ch_ABC123" }
					},
					"amount": {
						"type": "integer",
						"minimum": 1,
						"description": "Refund amount in cents for partial refund (e.g. 500 = $5.00). Omit for full refund",
						"x-ui": { "label": "Amount", "help_text": "In smallest currency unit (e.g. 1050 cents = $10.50)" }
					},
					"reason": {
						"type": "string",
						"enum": ["duplicate", "fraudulent", "requested_by_customer"],
						"description": "Reason for the refund — shown in the Stripe dashboard and receipts",
						"x-ui": { "label": "Reason", "widget": "select" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
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
				"x-ui": {
					"order": ["customer_id", "status", "price_id", "limit"]
				},
				"properties": {
					"customer_id": {
						"type": "string",
						"description": "Filter by Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123" }
					},
					"status": {
						"type": "string",
						"enum": ["active", "past_due", "canceled", "unpaid", "trialing", "all"],
						"description": "Filter by subscription status (defaults to all non-canceled)",
						"x-ui": { "label": "Status", "widget": "select" }
					},
					"price_id": {
						"type": "string",
						"description": "Filter by price ID (e.g. \"price_ABC123\")",
						"x-ui": { "label": "Price", "placeholder": "price_ABC123" }
					},
					"limit": {
						"type": "integer",
						"default": 10,
						"minimum": 1,
						"maximum": 100,
						"description": "Number of subscriptions to return (default 10, max 100)",
						"x-ui": { "label": "Limit" }
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
				"x-ui": {
					"order": ["line_items", "after_completion", "allow_promotion_codes", "metadata"]
				},
				"properties": {
					"line_items": {
						"type": "array",
						"minItems": 1,
						"description": "Products to include in the payment link (at least one required)",
						"x-ui": { "label": "Line Items" },
						"items": {
							"type": "object",
							"required": ["price_id", "quantity"],
							"additionalProperties": false,
							"properties": {
								"price_id": {
									"type": "string",
									"description": "Stripe price ID (e.g. \"price_ABC123\") — must be a pre-created Price object",
									"x-ui": { "label": "Price", "placeholder": "price_ABC123" }
								},
								"quantity": {
									"type": "integer",
									"minimum": 1,
									"description": "Quantity of the product",
									"x-ui": { "label": "Quantity" }
								}
							}
						}
					},
					"after_completion": {
						"type": "string",
						"format": "uri",
						"description": "URL to redirect customers to after successful payment",
						"x-ui": { "label": "Redirect URL", "placeholder": "https://example.com/success" }
					},
					"allow_promotion_codes": {
						"type": "boolean",
						"description": "Allow customers to enter promotion codes at checkout",
						"x-ui": { "label": "Allow Promo Codes", "widget": "toggle" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
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
				"x-ui": {
					"order": ["customer", "items", "trial_period_days", "payment_behavior", "metadata"]
				},
				"properties": {
					"customer": {
						"type": "string",
						"description": "Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123" }
					},
					"items": {
						"type": "array",
						"minItems": 1,
						"description": "Subscription items — each references a price",
						"x-ui": { "label": "Subscription Items" },
						"items": {
							"type": "object",
							"required": ["price"],
							"additionalProperties": false,
							"properties": {
								"price": {
									"type": "string",
									"description": "Stripe price ID (e.g. \"price_ABC123\")",
									"x-ui": { "label": "Price", "placeholder": "price_ABC123" }
								},
								"quantity": {
									"type": "integer",
									"minimum": 1,
									"description": "Quantity for this item (defaults to 1)",
									"x-ui": { "label": "Quantity" }
								}
							}
						}
					},
					"trial_period_days": {
						"type": "integer",
						"minimum": 0,
						"description": "Number of trial days before billing starts",
						"x-ui": { "label": "Trial Period (days)" }
					},
					"payment_behavior": {
						"type": "string",
						"enum": ["default_incomplete", "error_if_incomplete", "allow_incomplete", "pending_if_incomplete"],
						"description": "How to handle payment failures (defaults to default_incomplete)",
						"x-ui": { "label": "Payment Behavior", "widget": "select" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
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
				"x-ui": {
					"order": ["subscription_id", "cancel_at_period_end", "proration_behavior"]
				},
				"properties": {
					"subscription_id": {
						"type": "string",
						"description": "Stripe subscription ID (e.g. \"sub_ABC123\")",
						"x-ui": { "label": "Subscription", "placeholder": "sub_ABC123" }
					},
					"cancel_at_period_end": {
						"type": "boolean",
						"description": "When true, cancels at the end of the current billing period instead of immediately",
						"x-ui": { "label": "Cancel at Period End", "widget": "toggle" }
					},
					"proration_behavior": {
						"type": "string",
						"enum": ["create_prorations", "none", "always_invoice"],
						"description": "How to handle prorations for immediate cancellation",
						"x-ui": { "label": "Proration Behavior", "widget": "select" }
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
				"x-ui": {
					"order": ["name", "percent_off", "amount_off", "currency", "duration", "duration_in_months", "max_redemptions", "metadata"]
				},
				"properties": {
					"percent_off": {
						"type": "number",
						"exclusiveMinimum": 0,
						"maximum": 100,
						"description": "Percentage discount (e.g. 25.5 for 25.5% off) — provide this or amount_off",
						"x-ui": { "label": "Percent Off" }
					},
					"amount_off": {
						"type": "integer",
						"minimum": 1,
						"description": "Fixed discount in smallest currency unit (e.g. 500 = $5.00) — provide this or percent_off",
						"x-ui": { "label": "Amount Off", "help_text": "In smallest currency unit (e.g. 1050 cents = $10.50)" }
					},
					"currency": {
						"type": "string",
						"description": "Three-letter ISO 4217 currency code — required when using amount_off",
						"x-ui": { "label": "Currency", "placeholder": "usd" }
					},
					"duration": {
						"type": "string",
						"enum": ["once", "repeating", "forever"],
						"description": "How long the discount applies: once, repeating (requires duration_in_months), or forever",
						"x-ui": { "label": "Duration", "widget": "select" }
					},
					"duration_in_months": {
						"type": "integer",
						"minimum": 1,
						"description": "Number of months the coupon applies — required when duration is \"repeating\"",
						"x-ui": { "label": "Duration (months)" }
					},
					"max_redemptions": {
						"type": "integer",
						"minimum": 1,
						"description": "Maximum number of times the coupon can be redeemed",
						"x-ui": { "label": "Max Redemptions" }
					},
					"name": {
						"type": "string",
						"description": "Display name for the coupon (shown on invoices)",
						"x-ui": { "label": "Coupon Name", "placeholder": "Summer Sale 25% Off" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
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
				"x-ui": {
					"order": ["coupon", "code", "max_redemptions", "expires_at", "metadata"]
				},
				"properties": {
					"coupon": {
						"type": "string",
						"description": "Stripe coupon ID to attach this promotion code to",
						"x-ui": { "label": "Coupon", "placeholder": "coupon_ABC123" }
					},
					"code": {
						"type": "string",
						"description": "Customer-facing code (e.g. \"SUMMER25\") — auto-generated if omitted",
						"x-ui": { "label": "Promo Code", "placeholder": "SUMMER25" }
					},
					"max_redemptions": {
						"type": "integer",
						"minimum": 1,
						"description": "Maximum number of times this promotion code can be redeemed",
						"x-ui": { "label": "Max Redemptions" }
					},
					"expires_at": {
						"type": "integer",
						"description": "Expiration date as Unix timestamp (seconds since epoch)",
						"x-ui": { "label": "Expires At" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
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
				"x-ui": {
					"order": ["amount", "currency", "description", "destination", "metadata"]
				},
				"properties": {
					"amount": {
						"type": "integer",
						"minimum": 1,
						"description": "Payout amount in smallest currency unit (e.g. 10000 = $100.00)",
						"x-ui": { "label": "Amount", "help_text": "In smallest currency unit (e.g. 1050 cents = $10.50)" }
					},
					"currency": {
						"type": "string",
						"description": "Three-letter ISO 4217 currency code (e.g. \"usd\")",
						"x-ui": { "label": "Currency", "placeholder": "usd" }
					},
					"description": {
						"type": "string",
						"description": "Internal description for the payout",
						"x-ui": { "label": "Description", "widget": "textarea" }
					},
					"destination": {
						"type": "string",
						"description": "Bank account or card ID to send funds to (uses default if omitted)",
						"x-ui": { "label": "Destination", "placeholder": "ba_ABC123" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.create_product",
			Name:        "Create Product",
			Description: "Create a product in the Stripe catalog — required before creating prices or subscriptions",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["name"],
				"additionalProperties": false,
				"x-ui": {
					"order": ["name", "description", "active", "metadata"]
				},
				"properties": {
					"name": {
						"type": "string",
						"description": "Product name displayed to customers (e.g. \"Pro Plan\")",
						"x-ui": { "label": "Product Name", "placeholder": "Pro Plan" }
					},
					"description": {
						"type": "string",
						"description": "Optional product description",
						"x-ui": { "label": "Description", "widget": "textarea" }
					},
					"active": {
						"type": "boolean",
						"description": "Whether the product is available for purchase (defaults to true)",
						"x-ui": { "label": "Active", "widget": "toggle" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.create_price",
			Name:        "Create Price",
			Description: "Create a price for a product — one-time or recurring, required before creating subscriptions",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["currency", "product", "unit_amount"],
				"additionalProperties": false,
				"x-ui": {
					"order": ["product", "unit_amount", "currency", "recurring", "nickname", "active", "tax_behavior", "metadata"]
				},
				"properties": {
					"currency": {
						"type": "string",
						"description": "Three-letter ISO 4217 currency code (e.g. \"usd\")",
						"x-ui": { "label": "Currency", "placeholder": "usd" }
					},
					"product": {
						"type": "string",
						"description": "Stripe product ID (e.g. \"prod_ABC123\")",
						"x-ui": { "label": "Product", "placeholder": "prod_ABC123" }
					},
					"unit_amount": {
						"type": "integer",
						"minimum": 0,
						"description": "Price amount in smallest currency unit (e.g. 2000 = $20.00)",
						"x-ui": { "label": "Unit Amount", "help_text": "In smallest currency unit (e.g. 1050 cents = $10.50)" }
					},
					"recurring": {
						"type": "object",
						"description": "Makes this a recurring price for subscriptions",
						"required": ["interval"],
						"additionalProperties": false,
						"x-ui": { "label": "Recurring" },
						"properties": {
							"interval": {
								"type": "string",
								"enum": ["day", "week", "month", "year"],
								"description": "Billing interval",
								"x-ui": { "label": "Interval", "widget": "select" }
							},
							"interval_count": {
								"type": "integer",
								"minimum": 1,
								"description": "Number of intervals between billings (e.g. 3 with month = quarterly)",
								"x-ui": { "label": "Interval Count" }
							}
						}
					},
					"nickname": {
						"type": "string",
						"description": "Internal label for this price (not shown to customers)",
						"x-ui": { "label": "Nickname", "placeholder": "Monthly Pro" }
					},
					"active": {
						"type": "boolean",
						"description": "Whether the price is available for new subscriptions (defaults to true)",
						"x-ui": { "label": "Active", "widget": "toggle" }
					},
					"tax_behavior": {
						"type": "string",
						"enum": ["inclusive", "exclusive", "unspecified"],
						"description": "Tax treatment: inclusive (tax in price), exclusive (tax added on top), or unspecified (inherits from product). Required when using Stripe Tax.",
						"x-ui": { "label": "Tax Behavior", "widget": "select" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.create_checkout_session",
			Name:        "Create Checkout Session",
			Description: "Create a Stripe Checkout session — the most common redirect-based payment flow for SaaS",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["mode", "line_items"],
				"additionalProperties": false,
				"x-ui": {
					"groups": [
						{ "id": "products", "label": "Products" },
						{ "id": "session", "label": "Session Options" },
						{ "id": "customer", "label": "Customer", "collapsed": true }
					],
					"order": ["mode", "line_items", "success_url", "cancel_url", "customer", "customer_email", "allow_promotion_codes", "metadata"]
				},
				"properties": {
					"mode": {
						"type": "string",
						"enum": ["payment", "subscription", "setup"],
						"description": "Checkout mode: payment (one-time), subscription (recurring), or setup (save payment method)",
						"x-ui": { "widget": "select", "label": "Checkout Mode", "group": "session" }
					},
					"line_items": {
						"type": "array",
						"minItems": 1,
						"maxItems": 20,
						"description": "Products to include in the checkout session",
						"x-ui": { "label": "Line Items", "group": "products" },
						"items": {
							"type": "object",
							"required": ["price", "quantity"],
							"additionalProperties": false,
							"properties": {
								"price": {
									"type": "string",
									"description": "Stripe price ID (e.g. \"price_ABC123\")"
								},
								"quantity": {
									"type": "integer",
									"minimum": 1,
									"description": "Quantity of this item"
								}
							}
						}
					},
					"success_url": {
						"type": "string",
						"format": "uri",
						"description": "https URL to redirect to after successful payment (must use https)",
						"x-ui": { "label": "Success URL", "placeholder": "https://example.com/success", "group": "session" }
					},
					"cancel_url": {
						"type": "string",
						"format": "uri",
						"description": "https URL to redirect to if customer cancels checkout (must use https)",
						"x-ui": { "label": "Cancel URL", "placeholder": "https://example.com/cancel", "group": "session" }
					},
					"customer": {
						"type": "string",
						"description": "Stripe customer ID to associate with this session (mutually exclusive with customer_email)",
						"x-ui": { "label": "Customer ID", "placeholder": "cus_ABC123", "group": "customer" }
					},
					"customer_email": {
						"type": "string",
						"format": "email",
						"description": "Pre-fill the customer email field (mutually exclusive with customer)",
						"x-ui": { "label": "Customer Email", "placeholder": "billing@acme.com", "group": "customer" }
					},
					"allow_promotion_codes": {
						"type": "boolean",
						"description": "Allow customers to enter promotion codes at checkout",
						"x-ui": { "widget": "toggle", "label": "Allow Promo Codes", "group": "session" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "group": "session" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.update_subscription",
			Name:        "Update Subscription",
			Description: "Upgrade, downgrade, or modify an existing subscription — change plans, quantity, or add coupons",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["subscription_id"],
				"additionalProperties": false,
				"x-ui": {
					"order": ["subscription_id", "items", "coupon", "proration_behavior", "trial_end", "cancel_at", "metadata"]
				},
				"properties": {
					"subscription_id": {
						"type": "string",
						"description": "Stripe subscription ID (e.g. \"sub_ABC123\")",
						"x-ui": { "label": "Subscription", "placeholder": "sub_ABC123" }
					},
					"items": {
						"type": "array",
						"description": "Updated subscription items — include id to modify existing items, or just price to add new ones",
						"x-ui": { "label": "Subscription Items" },
						"items": {
							"type": "object",
							"additionalProperties": false,
							"properties": {
								"id": {
									"type": "string",
									"description": "Subscription item ID (e.g. \"si_ABC123\") — required to update or delete an existing item",
									"x-ui": { "label": "Item ID", "placeholder": "si_ABC123" }
								},
								"price": {
									"type": "string",
									"description": "New price ID — use to switch plans or add a new item",
									"x-ui": { "label": "Price", "placeholder": "price_ABC123" }
								},
								"quantity": {
									"type": "integer",
									"minimum": 1,
									"description": "Updated quantity for this item",
									"x-ui": { "label": "Quantity" }
								},
								"deleted": {
									"type": "boolean",
									"description": "Set to true to remove this item from the subscription",
									"x-ui": { "label": "Delete Item", "widget": "toggle" }
								}
							}
						}
					},
					"coupon": {
						"type": "string",
						"description": "Coupon ID to apply a discount to the subscription",
						"x-ui": { "label": "Coupon", "placeholder": "coupon_ABC123" }
					},
					"proration_behavior": {
						"type": "string",
						"enum": ["create_prorations", "none", "always_invoice"],
						"description": "How to handle prorations when changing plans mid-cycle",
						"x-ui": { "label": "Proration Behavior", "widget": "select" }
					},
					"trial_end": {
						"type": "string",
						"description": "Unix timestamp to set a new trial end date, or \"now\" to end the trial immediately and start billing",
						"x-ui": { "label": "Trial End" }
					},
					"cancel_at": {
						"type": "integer",
						"minimum": 1,
						"description": "Unix timestamp to schedule a future cancellation (must be positive/in the future).",
						"x-ui": { "label": "Cancel At" }
					},
					"metadata": {
						"type": "object",
						"description": "Key-value pairs for storing additional information (max 50 keys)",
						"additionalProperties": { "type": "string" },
						"x-ui": { "help_text": "Custom key-value pairs (max 50 keys)" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.list_customers",
			Name:        "List Customers",
			Description: "List or search customers — use to check for existing customers before creating duplicates",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"additionalProperties": false,
				"x-ui": {
					"order": ["email", "limit", "starting_after"]
				},
				"properties": {
					"email": {
						"type": "string",
						"format": "email",
						"description": "Filter by exact email address",
						"x-ui": { "label": "Email", "placeholder": "billing@acme.com" }
					},
					"limit": {
						"type": "integer",
						"default": 10,
						"minimum": 1,
						"maximum": 100,
						"description": "Number of customers to return (default 10, max 100)",
						"x-ui": { "label": "Limit" }
					},
					"starting_after": {
						"type": "string",
						"description": "Cursor for pagination — the customer ID of the last item from the previous page (from has_more response)",
						"x-ui": { "hidden": true }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.get_customer",
			Name:        "Get Customer",
			Description: "Retrieve a single customer by ID",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["customer_id"],
				"additionalProperties": false,
				"x-ui": {
					"order": ["customer_id"]
				},
				"properties": {
					"customer_id": {
						"type": "string",
						"description": "Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123" }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.list_invoices",
			Name:        "List Invoices",
			Description: "List invoices with optional filtering by customer or status — useful for reconciliation and dashboards",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"additionalProperties": false,
				"x-ui": {
					"order": ["customer_id", "status", "limit", "starting_after"]
				},
				"properties": {
					"customer_id": {
						"type": "string",
						"description": "Filter by Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123" }
					},
					"status": {
						"type": "string",
						"enum": ["draft", "open", "paid", "uncollectible", "void"],
						"description": "Filter by invoice status",
						"x-ui": { "label": "Status", "widget": "select" }
					},
					"limit": {
						"type": "integer",
						"default": 10,
						"minimum": 1,
						"maximum": 100,
						"description": "Number of invoices to return (default 10, max 100)",
						"x-ui": { "label": "Limit" }
					},
					"starting_after": {
						"type": "string",
						"description": "Cursor for pagination — the invoice ID of the last item from the previous page (from has_more response)",
						"x-ui": { "hidden": true }
					}
				}
			}`)),
		},
		{
			ActionType:  "stripe.list_charges",
			Name:        "List Charges",
			Description: "List charges with optional filtering by customer or payment intent — useful for reconciliation and reporting",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"additionalProperties": false,
				"x-ui": {
					"order": ["customer_id", "payment_intent_id", "limit", "starting_after"]
				},
				"properties": {
					"customer_id": {
						"type": "string",
						"description": "Filter by Stripe customer ID (e.g. \"cus_ABC123\")",
						"x-ui": { "label": "Customer", "placeholder": "cus_ABC123" }
					},
					"payment_intent_id": {
						"type": "string",
						"description": "Filter by payment intent ID (e.g. \"pi_ABC123\")",
						"x-ui": { "label": "Payment Intent", "placeholder": "pi_ABC123" }
					},
					"limit": {
						"type": "integer",
						"default": 10,
						"minimum": 1,
						"maximum": 100,
						"description": "Number of charges to return (default 10, max 100)",
						"x-ui": { "label": "Limit" }
					},
					"starting_after": {
						"type": "string",
						"description": "Cursor for pagination — the charge ID of the last item from the previous page (from has_more response)",
						"x-ui": { "hidden": true }
					}
				}
			}`)),
		},
	}
}
