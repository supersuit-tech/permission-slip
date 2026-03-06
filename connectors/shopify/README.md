# Shopify Connector

The Shopify connector integrates Permission Slip with the [Shopify Admin REST API](https://shopify.dev/docs/api/admin-rest). It uses plain `net/http` — no third-party Shopify SDK.

## Connector ID

`shopify`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `shop_domain` | Yes | The Shopify store subdomain (e.g., `mystore`) or full domain (e.g., `mystore.myshopify.com`). Custom domains are not supported. |
| `access_token` | Yes | A Shopify Admin API access token (e.g., `shpat_...`). See [Shopify docs](https://shopify.dev/docs/apps/build/authentication-authorization/access-tokens/generate-app-access-tokens-admin) for how to generate one. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

### Base URL

The connector dynamically constructs the Admin API base URL from `shop_domain`:

```
https://{shop_domain}.myshopify.com/admin/api/2024-10
```

Both bare subdomains (`mystore`) and full domains (`mystore.myshopify.com`) are accepted. Custom domains (e.g., `shop.example.com`) are rejected with a validation error.

## Actions

*No actions are registered yet — this is a Phase 1 scaffold. Phase 2 will add actions like product management, order management, etc.*

## Error Handling

The connector maps Shopify API responses to typed connector errors:

| Shopify Status | Connector Error | HTTP Response |
|----------------|-----------------|---------------|
| 401 | `AuthError` | 502 Bad Gateway |
| 403 | `AuthError` | 502 Bad Gateway |
| 404 | `ValidationError` | 400 Bad Request |
| 422 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |
| Context canceled | `TimeoutError` | 504 Gateway Timeout |

Shopify returns errors in multiple formats, all of which are parsed:

- `{"errors": "Not Found"}` — plain string
- `{"errors": {"title": ["can't be blank"]}}` — field-level validation errors
- `{"error": "Not Found"}` — singular key (used by some endpoints)

Rate limit responses include the `Retry-After` header value (defaults to 2s if missing).

## Adding a New Action

Each action lives in its own file. To add one (e.g., `shopify.create_product`):

1. Create `connectors/shopify/create_product.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, auth headers (`X-Shopify-Access-Token`), response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `shopify.go`.
5. Add the action to the `Manifest()` return value inside `shopify.go` — include a `ParametersSchema` (see below).
6. Add tests in `create_product_test.go` using `httptest.NewServer` and `newForTest()`.

The `do` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, error mapping) are handled once.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable:

```go
{
    ActionType:  "shopify.create_product",
    Name:        "Create Product",
    Description: "Create a new product in the Shopify store",
    RiskLevel:   "low",
    ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
        "type": "object",
        "required": ["title"],
        "properties": {
            "title": {
                "type": "string",
                "description": "Product title"
            },
            "body_html": {
                "type": "string",
                "description": "Product description (HTML)"
            },
            "vendor": {
                "type": "string",
                "description": "Product vendor name"
            }
        }
    }`)),
}
```

The schema supports standard JSON Schema properties: `type`, `description`, `required`, `enum`, and `default`. The frontend reads these to render rich parameter displays in the approval review modal.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `ShopifyConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/shopify/
├── shopify.go         # ShopifyConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── response.go        # Shared HTTP response → typed error mapping
├── shopify_test.go    # Connector-level tests
├── response_test.go   # Response parsing / error mapping tests
├── helpers_test.go    # Shared test helpers (validCreds)
└── README.md          # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Shopify API — no real API calls are made.

```bash
go test ./connectors/shopify/... -v
```
