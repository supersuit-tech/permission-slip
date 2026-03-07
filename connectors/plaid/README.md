# Plaid Connector

The Plaid connector integrates Permission Slip with the [Plaid API](https://plaid.com/docs/api/) for banking data, account balances, transactions, and identity verification. It uses plain `net/http` — no third-party Plaid SDK.

## Connector ID

`plaid`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `client_id` | Yes | Your Plaid client ID (minimum 20 characters). Found in the [Plaid Dashboard](https://dashboard.plaid.com/team/keys). |
| `secret` | Yes | Your Plaid secret key (minimum 20 characters). Use the sandbox secret for testing, production secret for live data. |
| `environment` | No | `"sandbox"` (default) or `"production"`. Controls which Plaid API endpoint is used. |

The credential `auth_type` in the database is `custom`. Plaid uses a non-standard auth pattern where `client_id` and `secret` are included in every JSON request body (not as headers). Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

### Environment Switching

The connector defaults to the Plaid **sandbox** environment (`https://sandbox.plaid.com`) for safety. To use production data, set the `environment` credential to `"production"` — the connector will route requests to `https://production.plaid.com`.

## API Version

The connector pins `Plaid-Version: 2020-09-14` on all requests to prevent breaking changes from Plaid API updates. Update the `plaidAPIVersion` constant in `plaid.go` when you're ready to adopt a newer version.

## Actions

### `plaid.create_link_token`

Creates a link token to initiate the Plaid Link bank connection flow.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `user_id` | string | Yes | A unique identifier for the user connecting their bank account |
| `products` | string[] | Yes | Plaid products to enable: `auth`, `transactions`, `identity`, `balance` |
| `country_codes` | string[] | No | Country codes (defaults to `["US"]`) |
| `language` | string | No | Language for the Link flow (defaults to `"en"`) |

**Response:**

```json
{
  "link_token": "link-sandbox-abc123...",
  "expiration": "2026-03-07T12:00:00Z",
  "request_id": "req123"
}
```

**Plaid API:** `POST /link/token/create` ([docs](https://plaid.com/docs/api/link/#linktokencreate))

---

### `plaid.get_accounts`

Gets account details (name, type, mask) for a connected bank.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_token` | string | Yes | The access token for the connected Item |
| `account_ids` | string[] | No | Filter to specific account IDs |

**Plaid API:** `POST /accounts/get` ([docs](https://plaid.com/docs/api/accounts/#accountsget))

---

### `plaid.get_balances`

Gets real-time or cached account balances for a connected bank.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_token` | string | Yes | The access token for the connected Item |
| `account_ids` | string[] | No | Filter to specific account IDs |

**Plaid API:** `POST /accounts/balance/get` ([docs](https://plaid.com/docs/api/products/balance/#accountsbalanceget))

---

### `plaid.list_transactions`

Lists transactions for a connected bank account within a date range.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_token` | string | Yes | The access token for the connected Item |
| `start_date` | string | Yes | Start date in `YYYY-MM-DD` format |
| `end_date` | string | Yes | End date in `YYYY-MM-DD` format |
| `account_ids` | string[] | No | Filter to specific account IDs |
| `count` | integer | No | Max transactions to return (1–500, default 100) |
| `offset` | integer | No | Number of transactions to skip (pagination) |

**Plaid API:** `POST /transactions/get` ([docs](https://plaid.com/docs/api/products/transactions/#transactionsget))

---

### `plaid.get_identity`

Gets account holder identity information (name, address, phone, email). This is PII and carries a **high** risk level.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_token` | string | Yes | The access token for the connected Item |
| `account_ids` | string[] | No | Filter to specific account IDs |

**Plaid API:** `POST /identity/get` ([docs](https://plaid.com/docs/api/products/identity/#identityget))

---

### `plaid.get_institution`

Gets details about a financial institution (name, supported products, etc.). This is public data and carries a **low** risk level.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `institution_id` | string | Yes | The Plaid institution ID (e.g. `ins_1`) |
| `country_codes` | string[] | No | Country codes (defaults to `["US"]`) |

**Plaid API:** `POST /institutions/get_by_id` ([docs](https://plaid.com/docs/api/institutions/#institutionsget_by_id))

## Architecture

The connector uses a shared `accessTokenAction` type for actions that share the same parameter pattern (access_token + optional account_ids): `get_accounts`, `get_balances`, and `get_identity`. Each is differentiated by its Plaid API path, configured in the `Actions()` map.

The `doPost` helper handles credential injection (copying the body map to avoid credential leakage), API version pinning, response size limits, timeout detection, and error mapping.

## Testing

```bash
go test ./connectors/plaid/... -v
```

All tests use `httptest.NewServer` to mock Plaid API responses. No real Plaid API calls are made in tests.
