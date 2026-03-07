# Square Connector

The Square connector integrates Permission Slip with the [Square REST API](https://developer.squareup.com/reference/square). It uses plain `net/http` — no third-party Square SDK.

Square powers 2M+ merchants across restaurants, retail, services, and appointments. This connector enables agents to create orders, process payments, issue refunds, manage catalog and inventory, create customer profiles, book appointments, search orders, and send invoices.

## Connector ID

`square`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | A Square access token (personal or application). Obtain from the [Square Developer Dashboard](https://developer.squareup.com/docs/build-basics/access-tokens). |
| `environment` | No | `"sandbox"` or `"production"` (default). Controls which Square API host is used. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

### Sandbox vs Production

Square provides a [free sandbox](https://developer.squareup.com/docs/devtools/sandbox/overview) with test credentials for development:

- **Sandbox:** `https://connect.squareupsandbox.com/v2`
- **Production:** `https://connect.squareup.com/v2`

Set `environment: "sandbox"` in the credential to use the sandbox API. Test card nonces like `"cnon:card-nonce-ok"` work in sandbox mode.

## Actions

### `square.create_order`

Creates an order at a Square location (restaurant order, retail sale, etc.).

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `location_id` | string | Yes | Square location ID |
| `line_items` | array | Yes | One or more items with name, quantity, and price |
| `customer_id` | string | No | Square customer ID to associate |
| `note` | string | No | Free-text note (visible to staff) |

**Square API:** `POST /v2/orders` ([docs](https://developer.squareup.com/reference/square/orders-api/create-order))

---

### `square.create_payment`

Processes a payment. **High risk — charges real money in production.**

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `source_id` | string | Yes | Card nonce, card-on-file ID, or `"CASH"` |
| `amount_money` | object | Yes | `{amount, currency}` — amount in cents |
| `order_id` | string | No | Link to an existing order |
| `customer_id` | string | No | Square customer ID |
| `note` | string | No | Note on the payment receipt |
| `reference_id` | string | No | External reference ID for reconciliation |

**Square API:** `POST /v2/payments` ([docs](https://developer.squareup.com/reference/square/payments-api/create-payment))

---

### `square.list_catalog`

Lists menu items, products, categories, and other catalog objects.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `types` | string | No | Comma-separated types: `ITEM`, `CATEGORY`, `DISCOUNT`, `TAX`, `MODIFIER`, `IMAGE` |
| `cursor` | string | No | Pagination cursor from a previous response |

**Square API:** `GET /v2/catalog/list` ([docs](https://developer.squareup.com/reference/square/catalog-api/list-catalog))

---

### `square.create_customer`

Creates a customer profile in the merchant's directory.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `given_name` | string | Yes | Customer's first name |
| `family_name` | string | No | Customer's last name |
| `email_address` | string | No | Email address |
| `phone_number` | string | No | Phone number (E.164 format preferred) |
| `company_name` | string | No | Company or business name |
| `note` | string | No | Internal note (not visible to customer) |

**Square API:** `POST /v2/customers` ([docs](https://developer.squareup.com/reference/square/customers-api/create-customer))

---

### `square.create_booking`

Schedules an appointment via Square Appointments.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `location_id` | string | Yes | Square location ID |
| `customer_id` | string | No | Customer ID for the booking |
| `start_at` | string | Yes | Start time (RFC 3339, e.g. `"2024-03-15T14:30:00Z"`) |
| `service_variation_id` | string | Yes | Catalog service variation ID |
| `team_member_id` | string | No | Staff member to assign |
| `customer_note` | string | No | Note from the customer |

**Square API:** `POST /v2/bookings` ([docs](https://developer.squareup.com/reference/square/bookings-api/create-booking))

---

### `square.search_orders`

Searches and filters orders across locations.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `location_ids` | array | Yes | Location IDs to search across |
| `query` | object | No | Search filters (state, date range, customer) |
| `limit` | integer | No | Max orders per page (1-500, default 500) |
| `cursor` | string | No | Pagination cursor |

**Square API:** `POST /v2/orders/search` ([docs](https://developer.squareup.com/reference/square/orders-api/search-orders))

---

### `square.issue_refund`

Refunds a payment in full or partially. **High risk — returns real money and is irreversible.**

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `payment_id` | string | Yes | ID of the payment to refund |
| `amount_money` | object | No | `{amount, currency}` — omit for full refund |
| `reason` | string | No | Reason for the refund (shown on receipt) |

**Square API:** `POST /v2/refunds` ([docs](https://developer.squareup.com/reference/square/refunds-api/refund-payment))

---

### `square.update_catalog_item`

Updates a catalog item's name, description, or pricing.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_id` | string | Yes | Catalog item ID (from `list_catalog`) |
| `name` | string | No | New display name |
| `description` | string | No | New description |
| `variations` | array | No | Variations with ID, name, pricing_type, price_money |
| `version` | integer | No | Current version for conflict detection (strongly recommended) |

At least one of `name`, `description`, or `variations` must be provided.

**Square API:** `POST /v2/catalog/object` ([docs](https://developer.squareup.com/reference/square/catalog-api/upsert-catalog-object))

---

### `square.send_invoice`

Creates and sends an invoice to a customer in a single atomic operation (creates order → creates invoice → publishes invoice). **High risk — sends a real payment request.**

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `customer_id` | string | Yes | Invoice recipient (from `create_customer`) |
| `location_id` | string | Yes | Location the invoice is issued from |
| `line_items` | array | Yes | Items with description, quantity, and `base_price_money` |
| `due_date` | string | Yes | Payment due date (`YYYY-MM-DD`) |
| `delivery_method` | string | No | `EMAIL` (default), `SMS`, or `SHARE_MANUALLY` |
| `title` | string | No | Invoice title |
| `note` | string | No | Additional note on the invoice |

Uses deterministic idempotency keys derived from the request parameters, so retries after partial failures are safe.

**Square API:** `POST /v2/invoices` + `POST /v2/invoices/{id}/publish` ([docs](https://developer.squareup.com/reference/square/invoices-api))

---

### `square.get_inventory`

Retrieves current inventory counts for catalog items. Read-only.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `catalog_object_ids` | array | Yes | Catalog object IDs to check (max 1000) |
| `location_ids` | array | No | Filter to specific locations |

**Square API:** `POST /v2/inventory/counts/batch-retrieve` ([docs](https://developer.squareup.com/reference/square/inventory-api/batch-retrieve-inventory-counts))

---

### `square.adjust_inventory`

Adjusts inventory counts for a catalog item at a location.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `catalog_object_id` | string | Yes | Catalog item ID |
| `location_id` | string | Yes | Location ID |
| `quantity` | string | Yes | Quantity as a positive number string |
| `from_state` | string | Yes | Current state (e.g. `NONE`, `IN_STOCK`, `SOLD`) |
| `to_state` | string | Yes | Target state (e.g. `IN_STOCK`, `SOLD`, `WASTE`) |

Uses `ADJUSTMENT` type for state transitions (from → to) and `PHYSICAL_COUNT` when from_state equals to_state (setting an absolute count).

**Square API:** `POST /v2/inventory/changes/batch-create` ([docs](https://developer.squareup.com/reference/square/inventory-api/batch-change-inventory))

## Money Amounts

Square represents all monetary amounts in the **smallest currency unit** (cents for USD). For example:

| Display Amount | API Value |
|---------------|-----------|
| $10.00 | `{"amount": 1000, "currency": "USD"}` |
| $25.50 | `{"amount": 2550, "currency": "USD"}` |
| €5.00 | `{"amount": 500, "currency": "EUR"}` |

## Idempotency

All Square write operations require an `idempotency_key`. The connector generates these automatically — callers don't need to provide them. Most actions use random UUIDs. The `send_invoice` action uses deterministic keys derived from request parameters (SHA-256), with per-step suffixes (`-order`, `-invoice`, `-publish`), so retries of the same request after partial failures are safe.

## Local Validation

All actions validate parameters locally before sending requests to Square. This catches common mistakes early with clear error messages instead of opaque API errors:

- **Payments:** `amount_money.amount` must be > 0; currency is required. Prevents zero-dollar or negative charges.
- **Refunds:** `payment_id` is required; if `amount_money` is provided, amount must be > 0 with currency.
- **Orders:** Line item `quantity` must be a positive integer string; `base_price_money.amount` must not be negative.
- **Bookings:** `start_at` must be valid RFC 3339 format.
- **Search orders:** `limit` must be 0-500 (0 uses Square's default); `query` must be a JSON object (not a string or array).
- **Catalog updates:** At least one of `name`, `description`, or `variations` must be provided. Variation prices must be >= 0.
- **Invoices:** `due_date` must be `YYYY-MM-DD` format; line item quantities must be positive integers; max 500 line items.
- **Inventory:** `catalog_object_ids` max 1000 items; `quantity` must be a positive number; `from_state`/`to_state` validated against Square's state enum.
- **All write actions:** Required fields are checked before the API call.

## Error Handling

The connector maps Square API responses to typed connector errors:

| Square Status | Square Category | Connector Error |
|---------------|-----------------|-----------------|
| 401 | any | `AuthError` |
| 403 | `AUTHENTICATION_ERROR` | `AuthError` |
| 403 | other | `ExternalError` |
| 400 | `INVALID_REQUEST_ERROR` | `ValidationError` |
| 429 | `RATE_LIMIT_ERROR` | `RateLimitError` |
| Other 4xx/5xx | any | `ExternalError` |
| Client timeout | — | `TimeoutError` |

Error messages include the Square error code and field name when available for faster debugging.

## Adding a New Action

Each action lives in its own file. To add one (e.g., `square.update_order`):

1. Create `connectors/square/update_order.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle.
3. For single-step write operations, include `"idempotency_key": newIdempotencyKey()` in the request body. For multi-step operations, use `deriveBaseKey(actionType, parameters)` with step suffixes (see `send_invoice.go`).
4. Return `connectors.JSONResult(respBody)` to wrap the response.
5. Register the action in `Actions()` inside `square.go`.
6. Add the action manifest in `manifest.go` — include a `ParametersSchema`.
7. Add tests in `update_order_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/square/
├── square.go              # SquareConnector struct, shared money type, New(), Actions(), do()
├── manifest.go            # Manifest() and all action schema declarations
├── response.go            # Square error envelope parsing and error type mapping
├── create_order.go        # square.create_order action
├── create_payment.go      # square.create_payment action (high risk)
├── list_catalog.go        # square.list_catalog action
├── create_customer.go     # square.create_customer action
├── create_booking.go      # square.create_booking action
├── search_orders.go       # square.search_orders action
├── issue_refund.go        # square.issue_refund action (high risk)
├── update_catalog_item.go # square.update_catalog_item action
├── send_invoice.go        # square.send_invoice action (high risk, multi-step)
├── get_inventory.go       # square.get_inventory action (read-only)
├── adjust_inventory.go    # square.adjust_inventory action
├── helpers_test.go        # Shared test helpers (validCreds, sandboxCreds)
├── square_test.go         # Connector-level tests (do, credentials, environment routing)
├── response_test.go       # Error message formatting and response checking tests
├── *_test.go              # Per-action test files (one per action)
└── README.md              # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Square API — no real API calls are made.

```bash
go test ./connectors/square/... -v
```

## API Version

The Square API version is pinned via the `Square-Version` header (currently `2024-01-18`). This prevents breaking changes from newer API versions. Update the `squareVersion` constant in `square.go` when upgrading.
