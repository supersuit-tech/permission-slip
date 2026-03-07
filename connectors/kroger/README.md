# Kroger Connector

The Kroger connector integrates Permission Slip with the [Kroger API v1](https://developer.kroger.com/). It uses plain `net/http` with OAuth 2.0 Bearer tokens — no third-party SDK.

Kroger is the largest US supermarket chain by revenue, operating ~2,800 stores across multiple banners (Kroger, Ralphs, Fred Meyer, Harris Teeter, etc.). This connector enables product search, store locator, and cart management.

## Connector ID

`kroger`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | OAuth 2.0 access token obtained through the platform's OAuth flow. |

The credential `auth_type` in the database is `oauth2`. The connector references the `kroger` OAuth provider, which is registered as a built-in provider. Users connect with a single click; self-hosted deployments can override credentials via BYOA.

### OAuth Configuration

| Setting | Value |
|---------|-------|
| Authorize URL | `https://api.kroger.com/v1/connect/oauth2/authorize` |
| Token URL | `https://api.kroger.com/v1/connect/oauth2/token` |
| Scopes | `product.compact`, `cart.basic:write` |
| Env vars | `KROGER_CLIENT_ID`, `KROGER_CLIENT_SECRET` |

Create an OAuth app at the [Kroger Developer Portal](https://developer.kroger.com/) and set the redirect URI to `https://<domain>/v1/oauth/kroger/callback`.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `kroger.search_products` | Search Products | low | Search products by keyword with optional location-specific pricing |
| `kroger.get_product` | Get Product Details | low | Get product details by ID (UPC) including nutrition, price, availability |
| `kroger.search_locations` | Search Locations | low | Find stores by zip code, coordinates, or chain banner |
| `kroger.add_to_cart` | Add to Cart | medium | Add items to the authenticated user's Kroger cart with optional fulfillment modality |

### Location IDs

Most queries benefit from a `location_id` parameter — prices and availability vary by store. Location IDs are returned by `kroger.search_locations` in the `locationId` field (e.g., `"01400376"`).

### Product IDs

Product IDs are UPC codes (e.g., `"0001111041700"`) — standard barcodes useful for cross-retailer matching.

## API Endpoints

| Action | Method | Endpoint |
|--------|--------|----------|
| search_products | GET | `/v1/products?filter.term=...` |
| get_product | GET | `/v1/products/{id}` |
| search_locations | GET | `/v1/locations?filter.zipCode.near=...` |
| add_to_cart | PUT | `/v1/cart/add` |

## Error Handling

The connector maps Kroger API responses to typed connector errors:

| Kroger Status | Connector Error | Description |
|---------------|-----------------|-------------|
| 401 | `AuthError` | Invalid or expired access token |
| 403 | `AuthError` | Insufficient scope or permissions |
| 429 | `RateLimitError` | Rate limit exceeded (includes `Retry-After`) |
| Other 4xx/5xx | `ExternalError` | API-level error with message from response |
| Client timeout | `TimeoutError` | Request exceeded 30-second deadline |

Kroger API errors return a JSON body with an `errors` array. The connector extracts the first error's `message` (or `reason`) for inclusion in the typed error. Raw error bodies are truncated to 512 characters in error messages to prevent leaking large payloads.

### Response Size Limit

All responses are capped at 10 MiB (`io.LimitReader`) to prevent memory exhaustion.

### Coordinate Validation

The `search_locations` action uses pointer types (`*float64`) for `lat` and `lon` to correctly distinguish "not provided" from the value `0`. Both `lat` and `lon` must be provided together — providing only one returns a `ValidationError`. Latitude must be in `[-90, 90]` and longitude in `[-180, 180]`.

## File Structure

```
connectors/kroger/
├── kroger.go               # KrogerConnector struct, New(), Actions(), ValidateCredentials(), do()
├── manifest.go             # Manifest() — metadata, actions, templates, OAuth credentials
├── response.go             # checkResponse() — HTTP status → typed error mapping
├── search_products.go      # kroger.search_products action
├── get_product.go          # kroger.get_product action
├── search_locations.go     # kroger.search_locations action
├── add_to_cart.go          # kroger.add_to_cart action
├── *_test.go               # Tests for each action + connector + response
├── helpers_test.go         # Shared test helpers (validCreds)
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Kroger API — no real API calls are made.

```bash
go test ./connectors/kroger/... -v
```
