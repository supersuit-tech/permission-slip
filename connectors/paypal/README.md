# PayPal / Venmo connector

Integrates Permission Slip with the [PayPal REST APIs](https://developer.paypal.com/docs/api/overview/) using plain `net/http` (no PayPal SDK).

## Connector ID

`paypal`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | OAuth access token from **Log in with PayPal** (see [PayPal OAuth Setup](../../docs/oauth-setup.md#paypal-oauth-setup)). |
| `environment` | No | Omit or `live` → `https://api-m.paypal.com`. Set to `sandbox` → `https://api-m.sandbox.paypal.com` for REST calls. OAuth is configured for live PayPal endpoints; use a sandbox PayPal app if you need sandbox-only OAuth (see docs). |

Tokens are stored encrypted and decrypted only at execution time.

## Actions

| Action | Risk | Notes |
|--------|------|--------|
| `paypal.create_venmo_payout_batch` | high | `payout_batch` object per [Payouts API](https://developer.paypal.com/docs/api/payments.payouts-batch/v1/). Venmo: `recipient_wallet: VENMO`, `recipient_type` `PHONE` or `USER_HANDLE`. |
| `paypal.get_payout_batch` | low | GET batch by ID |
| `paypal.get_payout_item` | low | GET item by ID |
| `paypal.create_order` | medium | POST Orders v2 — use templates for a minimal example |
| `paypal.capture_order` | high | Capture after buyer approval |
| `paypal.get_order` | low | Poll order status |
| `paypal.create_invoice` | medium | Create draft invoice |
| `paypal.send_invoice` | medium | Send to customer |
| `paypal.get_invoice` | low | Fetch invoice |
| `paypal.remind_invoice` | medium | Payment reminder |
| `paypal.refund_capture` | high | Full or partial refund on a capture |

POST requests that create resources send a `PayPal-Request-Id` header derived from the action type and parameters (stable across retries).

## Templates

The manifest includes starter templates (minimal order, Venmo payout shape, read-only lookups) so admins can enable common flows without writing JSON from scratch.

## Further reading

- Issue context: PayPal/Venmo connector requirements ([permission-slip#113](https://github.com/supersuit-tech/permission-slip/issues/113))
- OAuth: [docs/oauth-setup.md](../../docs/oauth-setup.md#paypal-oauth-setup)
