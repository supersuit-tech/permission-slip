package paypal

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func paypalActions() []connectors.ManifestAction {
	return []connectors.ManifestAction{
		{
			ActionType:  "paypal.create_venmo_payout_batch",
			Name:        "Create Venmo / PayPal payout batch",
			Description: "POST /v1/payments/payouts — create a batch payout. For Venmo, set recipient_wallet to VENMO and use recipient_type PHONE or USER_HANDLE per PayPal docs. Requires Payouts product approval for live.",
			RiskLevel:   "high",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["payout_batch"],
				"additionalProperties": false,
				"properties": {
					"payout_batch": {
						"type": "object",
						"description": "PayPal payout batch JSON (sender_batch_header, items[]). For Venmo recipients use recipient_wallet: VENMO with recipient_type PHONE (E.164) or USER_HANDLE (Venmo username). Docs: https://developer.paypal.com/docs/api/payments.payouts-batch/v1/",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.get_payout_batch",
			Name:        "Get payout batch status",
			Description: "GET /v1/payments/payouts/{payout_batch_id}",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["payout_batch_id"],
				"additionalProperties": false,
				"properties": {
					"payout_batch_id": {
						"type": "string",
						"description": "PayPal payout batch ID"
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.get_payout_item",
			Name:        "Get payout item status",
			Description: "GET /v1/payments/payouts-item/{payout_item_id}",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["payout_item_id"],
				"additionalProperties": false,
				"properties": {
					"payout_item_id": {
						"type": "string",
						"description": "PayPal payout item ID"
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.create_order",
			Name:        "Create Checkout order",
			Description: "POST /v2/checkout/orders — create an order (e.g. with payment_source.paypal for Venmo-capable checkout). Body must match PayPal Orders v2 schema.",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["order"],
				"additionalProperties": false,
				"properties": {
					"order": {
						"type": "object",
						"description": "Order create payload (intent, purchase_units, optional payment_source.paypal.experience_context with https return_url/cancel_url). Docs: https://developer.paypal.com/docs/api/orders/v2/",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.capture_order",
			Name:        "Capture Checkout order",
			Description: "POST /v2/checkout/orders/{order_id}/capture — capture after buyer approval. Optional body for payment_source or other overrides.",
			RiskLevel:   "high",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["order_id"],
				"additionalProperties": false,
				"properties": {
					"order_id": {
						"type": "string",
						"description": "PayPal order ID"
					},
					"body": {
						"type": "object",
						"description": "Optional JSON body for the capture request",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.get_order",
			Name:        "Get Checkout order",
			Description: "GET /v2/checkout/orders/{order_id}",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["order_id"],
				"additionalProperties": false,
				"properties": {
					"order_id": {
						"type": "string",
						"description": "PayPal order ID"
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.create_invoice",
			Name:        "Create invoice",
			Description: "POST /v2/invoicing/invoices — create a draft invoice per Invoicing API schema.",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["invoice"],
				"additionalProperties": false,
				"properties": {
					"invoice": {
						"type": "object",
						"description": "Invoice object per PayPal Invoicing v2",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.send_invoice",
			Name:        "Send invoice",
			Description: "POST /v2/invoicing/invoices/{invoice_id}/send",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["invoice_id"],
				"additionalProperties": false,
				"properties": {
					"invoice_id": {
						"type": "string",
						"description": "PayPal invoice ID"
					},
					"body": {
						"type": "object",
						"description": "Optional send options (e.g. additional_recipients)",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.get_invoice",
			Name:        "Get invoice",
			Description: "GET /v2/invoicing/invoices/{invoice_id}",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["invoice_id"],
				"additionalProperties": false,
				"properties": {
					"invoice_id": {
						"type": "string",
						"description": "PayPal invoice ID"
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.remind_invoice",
			Name:        "Remind invoice",
			Description: "POST /v2/invoicing/invoices/{invoice_id}/remind",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["invoice_id"],
				"additionalProperties": false,
				"properties": {
					"invoice_id": {
						"type": "string",
						"description": "PayPal invoice ID"
					},
					"body": {
						"type": "object",
						"description": "Optional reminder payload",
						"additionalProperties": true
					}
				}
			}`)),
		},
		{
			ActionType:  "paypal.refund_capture",
			Name:        "Refund capture",
			Description: "POST /v2/payments/captures/{capture_id}/refund — full or partial refund. Optional body with amount object.",
			RiskLevel:   "high",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["capture_id"],
				"additionalProperties": false,
				"properties": {
					"capture_id": {
						"type": "string",
						"description": "PayPal capture ID from an completed order/capture"
					},
					"body": {
						"type": "object",
						"description": "Optional refund body (e.g. amount for partial refund)",
						"additionalProperties": true
					}
				}
			}`)),
		},
	}
}
