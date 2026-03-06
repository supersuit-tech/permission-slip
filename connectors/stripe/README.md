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

All actions are implemented with full test coverage (see [issue #90](https://github.com/supersuit-tech/permission-slip/issues/90)).

| Action | Risk | Required Params | Description |
|--------|------|-----------------|-------------|
| `stripe.get_balance` | low | *(none)* | Retrieve current account balance (available + pending) |
| `stripe.create_customer` | low | `email` | Create a customer record. Optional: `name`, `description`, `phone`, `metadata` |
| `stripe.list_subscriptions` | low | *(none)* | List subscriptions. Optional filters: `customer_id`, `status`, `price_id`, `limit` (default 10, max 100) |
| `stripe.create_invoice` | medium | `customer_id` | Create an invoice, add line items, and optionally finalize. Optional: `description`, `due_date`, `currency` (default "usd"), `auto_advance`, `line_items[]`, `metadata` |
| `stripe.create_payment_link` | medium | `line_items[]` | Create a shareable payment link. Each item needs `price_id` + `quantity`. Optional: `after_completion` (redirect URL), `allow_promotion_codes`, `metadata` |
| `stripe.issue_refund` | **high** | `payment_intent_id` or `charge_id` | Refund a payment. Optional: `amount` (cents, omit for full refund), `reason`, `metadata`. Idempotency keys are mandatory. |

### Action Details

#### `stripe.create_invoice` — Multi-step Flow

This action performs up to 3 API calls:

1. **POST `/v1/invoices`** — creates the invoice
2. **POST `/v1/invoiceitems`** — adds each line item (one call per item)
3. **POST `/v1/invoices/{id}/finalize`** — finalizes the invoice (when `auto_advance` is true or unset)

If a line item or finalize step fails, the error includes the invoice ID for recovery (e.g., `"adding line item to invoice in_xxx: ..."`).

#### `stripe.issue_refund` — High Risk

Refunds move real money. The connector enforces:
- Exactly one of `payment_intent_id` or `charge_id` (not both, not neither)
- `reason` must be one of `duplicate`, `fraudulent`, `requested_by_customer`, or empty
- Deterministic idempotency keys prevent double-refunds on retries

### Validation Limits

All actions enforce these limits client-side for clear error messages:

- **Metadata:** max 50 key-value pairs, values must be strings (Stripe's limit). Non-string types (maps, arrays) are rejected client-side.
- **Invoice line items:** max 250 (Stripe's limit)
- **Payment link line items:** max 20 (Stripe's limit)
- **Payment link redirect URL:** must use HTTPS scheme (prevents open redirects via `javascript:`, `data:`, or `http:` schemes)
- **Subscription list limit:** 1–100 (default 10)

## Configuration Templates

The connector ships with 8 pre-built templates at different permission levels. Administrators can assign templates to agents to grant minimum-necessary access:

| Template | Action | Risk | Notes |
|----------|--------|------|-------|
| Check account balance | `get_balance` | Low | Read-only, no financial risk |
| List active subscriptions | `list_subscriptions` | Low | Status locked to "active" |
| List subscriptions (any status) | `list_subscriptions` | Low | All statuses including canceled |
| Create customers | `create_customer` | Low | Any customer details |
| Create invoices | `create_invoice` | Medium | Any customer, any line items |
| Create payment links | `create_payment_link` | Medium | Any products |
| Issue refunds up to $99.99 | `issue_refund` | **High** | Amount capped via `$pattern` regex (`^[1-9]\d{0,3}$` = max 9999 cents) |
| Issue refunds (any amount) | `issue_refund` | **High** | Uncapped — for trusted agents only |

Templates are defined in `manifest.go` and auto-seeded into the database on startup.

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

The `formEncode()` function handles this flattening. Actions call it on their typed params before passing to `do()`.

## Idempotency

All POST endpoints support Stripe's `Idempotency-Key` header. The connector derives deterministic keys from a SHA-256 hash of the action type and raw parameters — this ensures the same request always produces the same key, so retries are safe.

The `doPost()` convenience method handles this automatically. Actions should use `doPost()` instead of `do()` directly. For multi-step flows (like `create_invoice`), each step gets its own deterministic key derived from the step name + step-specific parameters.

## API Version Pinning

The connector pins the Stripe API version via the `Stripe-Version` header (currently `2025-12-18.acacia`). This prevents breaking changes when Stripe releases new API versions. Update the `apiVersion` constant deliberately when ready to handle new response shapes.

## Adding a New Action

Each action lives in its own file. To add one:

1. Create `connectors/stripe/<action_name>.go` with a params struct, `validate()`, and an `Execute` method.
2. Parse and validate parameters from `json.RawMessage`.
3. Use `formEncode()` to flatten the params into Stripe's bracket notation.
4. Use `a.conn.doPost(ctx, creds, path, flatParams, &resp, actionType, rawParams)` for POST requests or `a.conn.doGet(ctx, creds, path, queryParams, &resp)` for GET requests.
5. Return `connectors.JSONResult(resp)` to wrap the response into an `ActionResult`.
6. Register the action in `Actions()` inside `stripe.go`.
7. Add the action schema and any templates in `manifest.go` — the `TestManifest_ActionsMatchRegistered` test will catch any drift between `Actions()` and `Manifest()`.
8. Add tests in `<action_name>_test.go` using `httptest.NewServer` and `newForTest()`.

**Validation checklist for new actions:**
- Call `validateMetadata()` if accepting metadata
- Cap array parameters (line items, etc.) to prevent resource exhaustion
- Use `url.PathEscape()` for any user-supplied or API-returned IDs in URL paths
- Include `t.Parallel()` in all tests and protect shared state with `sync.Mutex`

The `doPost`/`doGet` methods handle auth, form encoding, idempotency, error mapping, and response size limits. Each action file only contains parameter parsing, validation, and the Stripe endpoint path.

## File Structure

```
connectors/stripe/
├── stripe.go                    # StripeConnector, New(), do(), doGet(), doPost(), formEncode(), validateMetadata()
├── manifest.go                  # ManifestProvider: action schemas, credentials, templates
├── response.go                  # Stripe error parsing and typed error mapping
├── create_customer.go           # stripe.create_customer action
├── create_invoice.go            # stripe.create_invoice action (multi-step: create → items → finalize)
├── issue_refund.go              # stripe.issue_refund action (high risk, idempotency-critical)
├── list_subscriptions.go        # stripe.list_subscriptions action
├── create_payment_link.go       # stripe.create_payment_link action
├── get_balance.go               # stripe.get_balance action
├── *_test.go                    # Corresponding test files for each action
├── helpers_test.go              # Shared test helpers (validCreds)
├── stripe_test.go               # Core tests (form encoding, do(), error mapping, idempotency, manifest validation)
└── README.md                    # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Stripe API — no real API calls or Stripe keys needed. Tests run with `-race` to catch data races.

```bash
go test ./connectors/stripe/... -v

# With race detector (recommended)
go test ./connectors/stripe/... -race
```
