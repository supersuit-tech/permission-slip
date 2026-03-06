# Stripe Connector

The Stripe connector integrates Permission Slip with the [Stripe REST API](https://docs.stripe.com/api). It uses plain `net/http` — no third-party Stripe SDK.

**Key difference from other connectors:** Stripe uses `application/x-www-form-urlencoded` request bodies with bracket notation for nested objects (e.g., `metadata[key]=value`, `line_items[0][price]=price_...`). Responses are JSON as normal.

## Connector ID

`stripe`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A Stripe secret key (`sk_live_...`, `sk_test_...`) or restricted key (`rk_live_...`, `rk_test_...`). |

The credential `auth_type` in the database is `api_key`. Keys are stored encrypted in Supabase Vault and decrypted only at execution time. The connector validates that the key starts with a recognized prefix.

**Setup:** [Stripe API keys documentation](https://docs.stripe.com/keys)

**Test mode:** Keys starting with `sk_test_` or `rk_test_` are safe for development — they operate in Stripe's test mode and never touch real money.

## Actions

*Actions will be added in Phase 2. Each action will have its own file and test file.*

Planned actions (see [issue #90](https://github.com/supersuit-tech/permission-slip/issues/90)):

| Action | Risk | Description |
|--------|------|-------------|
| `stripe.create_customer` | low | Create a new customer record |
| `stripe.create_invoice` | medium | Create and optionally send an invoice |
| `stripe.issue_refund` | **high** | Refund a charge or payment intent |
| `stripe.list_subscriptions` | low | List subscriptions by customer/status |
| `stripe.create_payment_link` | medium | Create a shareable payment link |
| `stripe.get_balance` | low | Retrieve current account balance |

## Error Handling

The Stripe API returns structured errors in a JSON envelope:

```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "resource_missing",
    "message": "No such customer: 'cus_nonexistent'"
  }
}
```

The connector maps these to typed connector errors:

| Stripe Error Type | Connector Error | HTTP Response |
|-------------------|-----------------|---------------|
| `authentication_error` | `AuthError` | 502 Bad Gateway |
| `invalid_request_error` | `ValidationError` | 400 Bad Request |
| `rate_limit_error` (or HTTP 429) | `RateLimitError` | 429 Too Many Requests |
| `card_error` | `ExternalError` | 502 Bad Gateway |
| `api_error` | `ExternalError` | 502 Bad Gateway |
| HTTP 401 (no Stripe error type) | `AuthError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

Error messages include the Stripe error `code` when available (e.g., `card_declined`, `expired_card`) for easier debugging.

## Form Encoding

Unlike GitHub and Slack (which use JSON request bodies), Stripe requires `application/x-www-form-urlencoded` with bracket notation for nested structures:

```
# Flat values
email=test%40example.com&name=Test+User

# Nested objects (metadata)
metadata[order_id]=12345&metadata[source]=agent

# Arrays (line items)
line_items[0][price]=price_abc&line_items[0][quantity]=2
```

The `formEncode()` function handles this flattening. Phase 2 actions call it on their typed params before passing to `do()`.

## Idempotency

All POST endpoints support Stripe's `Idempotency-Key` header. The connector derives deterministic keys from a SHA-256 hash of the action type and raw parameters — this ensures the same request always produces the same key, so retries are safe.

The `doPost()` convenience method handles this automatically. Phase 2 actions should use `doPost()` instead of `do()` directly.

## API Version Pinning

The connector pins the Stripe API version via the `Stripe-Version` header (currently `2025-12-18.acacia`). This prevents breaking changes when Stripe releases new API versions. Update the `apiVersion` constant deliberately when ready to handle new response shapes.

## Adding a New Action

Each action lives in its own file. To add one (e.g., `stripe.create_customer`):

1. Create `connectors/stripe/create_customer.go` with a params struct, `validate()`, and an `Execute` method.
2. Parse and validate parameters from `json.RawMessage`.
3. Use `formEncode()` to flatten the params into Stripe's bracket notation.
4. Use `a.conn.doPost(ctx, creds, path, flatParams, &resp, actionType, rawParams)` for POST requests or `a.conn.doGet(ctx, creds, path, queryParams, &resp)` for GET requests.
5. Return `connectors.JSONResult(resp)` to wrap the response into an `ActionResult`.
6. Register the action in `Actions()` inside `stripe.go`.
7. Add the action to the `Manifest()` return value (Phase 3).
8. Add tests in `create_customer_test.go` using `httptest.NewServer` and `newForTest()`.

The `doPost`/`doGet` methods handle auth, form encoding, idempotency, error mapping, and response size limits. Each action file only contains parameter parsing, validation, and the Stripe endpoint path.

## File Structure

```
connectors/stripe/
├── stripe.go              # StripeConnector struct, New(), do(), doGet(), doPost(), formEncode()
├── response.go            # Stripe error parsing and typed error mapping
├── helpers_test.go        # Shared test helpers (validCreds)
├── stripe_test.go         # Scaffold tests (form encoding, do(), error mapping, idempotency)
└── README.md              # This file
```

*Phase 2 will add action files (e.g., `create_customer.go`, `create_customer_test.go`).*

## Testing

All tests use `httptest.NewServer` to mock the Stripe API — no real API calls or Stripe keys needed.

```bash
go test ./connectors/stripe/... -v

# With race detector
go test ./connectors/stripe/... -race
```
