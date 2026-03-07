# Walmart Connector

The Walmart connector integrates Permission Slip with the [Walmart Affiliate API](https://walmart.io/docs/affiliate/introduction). It uses plain `net/http` — no third-party SDK.

All actions are **read-only** (product search and lookup). The connector returns shoppable `addToCartUrl` links for purchase attribution.

## Connector ID

`walmart`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `consumer_id` | Yes | Walmart Affiliate API consumer ID from [developer.walmart.com](https://developer.walmart.com) |
| `key_version` | No | API key version (defaults to `"1"`) |
| `impact_id` | No | Impact/affiliate ID for link attribution (sent as `WM_CONSUMER.CHANNEL.TYPE` header) |

The credential `auth_type` in the database is `api_key`. Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

**Setup:** [Walmart Affiliate onboarding guide](https://walmart.io/docs/affiliate/onboarding-guide)

**Authentication:** Walmart uses header-based auth — `WM_CONSUMER.ID` and `WM_SEC.KEY_VERSION` — rather than Bearer tokens or API keys in the query string.

## Actions

| Action | Risk | Required Params | Description |
|--------|------|-----------------|-------------|
| `walmart.search_products` | low | `query` | Search products by keyword. Optional: `category_id`, `sort`, `order`, `start`, `limit` |
| `walmart.get_product` | low | `item_id` | Get product details including price, ratings, images, and add-to-cart URL |
| `walmart.get_taxonomy` | low | *(none)* | Browse the product category tree to discover category IDs |
| `walmart.get_trending` | low | *(none)* | Discover currently trending products on Walmart.com |

### Validation Limits

- **Search query:** max 500 characters
- **Search limit:** 1–25 per request (default 10)
- **Sort fields:** `relevance`, `price`, `title`, `bestseller`, `customerRating`, `new`
- **Order:** `asc` or `desc`
- **Item ID:** max 20 characters

### Response Examples

#### `walmart.search_products`

Returns the raw Walmart search response including:
- `totalResults` — total matching products
- `items[]` — array with `itemId`, `name`, `salePrice`, `customerRating`, `numReviews`, `availableOnline`, `addToCartUrl`, `thumbnailImage`

#### `walmart.get_product`

Returns full product details: pricing, availability, images, reviews, ratings, and a shoppable `addToCartUrl` for purchase attribution.

#### `walmart.get_taxonomy`

Returns the category tree with `id` and `name` for each category. Use category IDs to filter `search_products` results.

#### `walmart.get_trending`

Returns an array of currently trending product items.

## Configuration Templates

The connector ships with 6 pre-built templates. All are read-only (no purchase or write operations):

| Template | Action | Notes |
|----------|--------|-------|
| Search products | `search_products` | Unrestricted keyword search |
| Search for best deals | `search_products` | Sort locked to `price` ascending |
| Search top-rated products | `search_products` | Sort locked to `customerRating` descending |
| Get product details | `get_product` | Any item ID |
| Browse product categories | `get_taxonomy` | No parameters |
| View trending products | `get_trending` | No parameters |

Templates are defined in `manifest.go` and auto-seeded into the database on startup.

## Error Handling

The Walmart API returns errors in two formats:

```json
{"errors": [{"code": 4000, "message": "Invalid query parameter"}]}
```

```json
{"message": "Unauthorized"}
```

The connector maps these to typed connector errors:

| HTTP Status | Connector Error | HTTP Response |
|-------------|-----------------|---------------|
| 401 Unauthorized | `AuthError` | 502 Bad Gateway |
| 403 Forbidden | `AuthError` | 502 Bad Gateway |
| 400 Bad Request | `ValidationError` | 400 Bad Request |
| 404 Not Found | `ValidationError` | 400 Bad Request |
| 429 Too Many Requests | `RateLimitError` | 429 Too Many Requests |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |
| Other 5xx | `ExternalError` | 502 Bad Gateway |

Error messages include the Walmart error code when available (e.g., `"Walmart auth error: Unauthorized (code: 4010)"`) for easier debugging.

## API Version

The connector targets Walmart Affiliate API **v2** (`/api-proxy/service/affil/product/v2`). Update the `defaultBaseURL` constant when migrating to a new API version.

## Adding a New Action

Each action lives in its own file. To add one:

1. Create `connectors/walmart/<action_name>.go` with a params struct, `validate()`, and an `Execute` method.
2. Parse and validate parameters from `json.RawMessage`.
3. Build the URL path with query parameters using `url.Values`.
4. Call `a.conn.do(ctx, creds, http.MethodGet, path, &resp)` to make the request.
5. Return `connectors.JSONResult(resp)` to wrap the response into an `ActionResult`.
6. Register the action in `Actions()` inside `manifest.go`.
7. Add the action schema and any templates in `manifest.go` — the `TestManifest_ActionsMatchRegistered` test will catch drift between `Actions()` and `Manifest()`.
8. Add tests in `<action_name>_test.go` using `httptest.NewServer` and `newForTest()`.

The `do()` method handles auth headers, error mapping, response size limits, and timeout handling. Each action file only contains parameter parsing, validation, and the Walmart endpoint path.

## File Structure

```
connectors/walmart/
├── walmart.go                   # WalmartConnector, New(), do(), ValidateCredentials()
├── manifest.go                  # ManifestProvider: action schemas, credentials, templates, Actions()
├── response.go                  # Walmart error parsing and typed error mapping
├── search_products.go           # walmart.search_products action
├── get_product.go               # walmart.get_product action
├── get_taxonomy.go              # walmart.get_taxonomy action
├── get_trending.go              # walmart.get_trending action
├── *_test.go                    # Corresponding test files for each action
├── helpers_test.go              # Shared test helpers (validCreds)
├── walmart_test.go              # Core tests (headers, validation, manifest, error mapping)
└── README.md                    # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Walmart API — no real API calls or credentials needed. Tests run with `-race` to catch data races.

```bash
go test ./connectors/walmart/... -v

# With race detector (recommended)
go test ./connectors/walmart/... -race
```
