package paypal

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func paypalTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		{
			ID:          "tpl_paypal_get_order",
			ActionType:  "paypal.get_order",
			Name:        "Look up order status",
			Description: "Read-only: fetch a Checkout order by ID (e.g. after a buyer completes payment).",
			Parameters:  json.RawMessage(`{"order_id":"*"}`),
		},
		{
			ID:          "tpl_paypal_get_payout_batch",
			ActionType:  "paypal.get_payout_batch",
			Name:        "Check payout batch status",
			Description: "Read-only: poll batch payout processing status.",
			Parameters:  json.RawMessage(`{"payout_batch_id":"*"}`),
		},
		{
			ID:          "tpl_paypal_get_invoice",
			ActionType:  "paypal.get_invoice",
			Name:        "Look up invoice",
			Description: "Read-only: fetch invoice details and payment state.",
			Parameters:  json.RawMessage(`{"invoice_id":"*"}`),
		},
		{
			ID:          "tpl_paypal_create_order_minimal",
			ActionType:  "paypal.create_order",
			Name:        "Create order (CAPTURE, single amount)",
			Description: "Starter template: CAPTURE intent with one purchase unit in USD. Replace amount and return URLs before use.",
			Parameters: json.RawMessage(connectors.TrimIndent(`{
				"order": {
					"intent": "CAPTURE",
					"purchase_units": [
						{
							"amount": {
								"currency_code": "USD",
								"value": "10.00"
							}
						}
					],
					"payment_source": {
						"paypal": {
							"experience_context": {
								"return_url": "https://example.com/paypal/return",
								"cancel_url": "https://example.com/paypal/cancel"
							}
						}
					}
				}
			}`)),
		},
		{
			ID:          "tpl_paypal_venmo_payout_minimal",
			ActionType:  "paypal.create_venmo_payout_batch",
			Name:        "Venmo payout batch (template)",
			Description: "High risk: example payout_batch with recipient_wallet VENMO. Replace sender_batch_id, email_subject, recipient, and amount. Requires Payouts approval on the PayPal account.",
			Parameters: json.RawMessage(connectors.TrimIndent(`{
				"payout_batch": {
					"sender_batch_header": {
						"sender_batch_id": "batch_replace_me",
						"email_subject": "You have a payout"
					},
					"items": [
						{
							"recipient_type": "PHONE",
							"amount": {
								"value": "1.00",
								"currency": "USD"
							},
							"receiver": "+14155551234",
							"recipient_wallet": "VENMO",
							"note": "Payout via Permission Slip"
						}
					]
				}
			}`)),
		},
	}
}
