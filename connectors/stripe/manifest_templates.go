package stripe

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

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
		// --- Catalog management ---
		{
			ID:          "tpl_stripe_create_product",
			ActionType:  "stripe.create_product",
			Name:        "Create products",
			Description: "Agent can create new product catalog entries with any name, description, and metadata.",
			Parameters:  json.RawMessage(`{"name":"*","description":"*","active":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_price_recurring",
			ActionType:  "stripe.create_price",
			Name:        "Create recurring prices",
			Description: "Agent can create recurring prices for subscription billing.",
			Parameters:  json.RawMessage(`{"currency":"*","product":"*","unit_amount":"*","recurring":"*","nickname":"*","active":"*","tax_behavior":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_price_one_time",
			ActionType:  "stripe.create_price",
			Name:        "Create one-time prices",
			Description: "Agent can create one-time prices. Recurring billing is blocked — use the recurring prices template to allow subscription pricing.",
			Parameters:  json.RawMessage(`{"currency":"*","product":"*","unit_amount":"*","nickname":"*","active":"*","tax_behavior":"*","metadata":"*"}`),
		},
		// --- Checkout ---
		{
			ID:          "tpl_stripe_create_checkout_session_subscription",
			ActionType:  "stripe.create_checkout_session",
			Name:        "Create subscription checkout sessions",
			Description: "Agent can create Checkout sessions for subscription plans. Mode is locked to \"subscription\".",
			Parameters:  json.RawMessage(`{"mode":"subscription","line_items":"*","success_url":"*","cancel_url":"*","customer":"*","customer_email":"*","allow_promotion_codes":"*","metadata":"*"}`),
		},
		{
			ID:          "tpl_stripe_create_checkout_session_payment",
			ActionType:  "stripe.create_checkout_session",
			Name:        "Create one-time payment checkout sessions",
			Description: "Agent can create Checkout sessions for one-time payments. Mode is locked to \"payment\".",
			Parameters:  json.RawMessage(`{"mode":"payment","line_items":"*","success_url":"*","cancel_url":"*","customer":"*","customer_email":"*","allow_promotion_codes":"*","metadata":"*"}`),
		},
		// --- Subscription management ---
		{
			ID:          "tpl_stripe_update_subscription",
			ActionType:  "stripe.update_subscription",
			Name:        "Update subscriptions",
			Description: "Agent can update subscriptions — change plans, quantities, add coupons, and manage proration.",
			Parameters:  json.RawMessage(`{"subscription_id":"*","items":"*","coupon":"*","proration_behavior":"*","trial_end":"*","cancel_at":"*","metadata":"*"}`),
		},
		// --- Read-only: customers, invoices, charges ---
		{
			ID:          "tpl_stripe_list_customers",
			ActionType:  "stripe.list_customers",
			Name:        "List customers",
			Description: "Agent can list customers and search by email. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"email":"*","limit":"*","starting_after":"*"}`),
		},
		{
			ID:          "tpl_stripe_get_customer",
			ActionType:  "stripe.get_customer",
			Name:        "Get customer by ID",
			Description: "Agent can retrieve any customer by ID. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"customer_id":"*"}`),
		},
		{
			ID:          "tpl_stripe_list_invoices",
			ActionType:  "stripe.list_invoices",
			Name:        "List invoices",
			Description: "Agent can list and filter invoices by customer or status. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"customer_id":"*","status":"*","limit":"*","starting_after":"*"}`),
		},
		{
			ID:          "tpl_stripe_list_charges",
			ActionType:  "stripe.list_charges",
			Name:        "List charges",
			Description: "Agent can list and filter charges by customer or payment intent. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"customer_id":"*","payment_intent_id":"*","limit":"*","starting_after":"*"}`),
		},
	}
}
