# Expedia Rapid Connector

The Expedia Rapid connector integrates Permission Slip with the [Expedia Rapid API](https://developers.expediagroup.com/docs/products/rapid) for hotel search, pricing, and booking. It uses plain `net/http` with HMAC-SHA512 signature authentication — no third-party SDK.

## Connector ID

`expedia`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | Expedia Rapid API key from your [EPS Rapid account](https://developers.expediagroup.com/). |
| `secret` | Yes | Shared secret for HMAC-SHA512 signature generation. |

The credential `auth_type` in the database is `api_key`. Both values are stored encrypted in Supabase Vault and decrypted only at execution time.

### Authentication

Expedia Rapid uses signature-based authentication. Each request includes an `Authorization` header in the format:

```
EAN apikey={api_key},signature={signature},timestamp={timestamp}
```

The signature is `SHA512(api_key + secret + unix_timestamp)`.

## Actions

### `expedia.search_hotels`

Search available hotels with pricing for given dates and occupancy.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `checkin` | string | Yes | — | Check-in date (YYYY-MM-DD) |
| `checkout` | string | Yes | — | Check-out date (YYYY-MM-DD) |
| `occupancy` | string | Yes | — | Occupancy string (e.g. `"2"` for 2 adults, `"2-0,4"` for 2 adults + 1 child age 4) |
| `region_id` | string | No | — | Expedia region ID to search in |
| `latitude` | number | No | — | Latitude for location-based search |
| `longitude` | number | No | — | Longitude for location-based search |
| `currency` | string | No | — | Currency code (e.g. USD, EUR) |
| `language` | string | No | — | Language code (e.g. en-US) |
| `sort_by` | string | No | — | Sort by `price`, `distance`, or `rating` |
| `star_rating` | integer[] | No | — | Filter by star rating(s) |
| `limit` | integer | No | 20 | Maximum number of results |

---

### `expedia.get_hotel`

Get full hotel details: photos, amenities, room types, policies, reviews summary.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `property_id` | string | Yes | Expedia property ID |
| `checkin` | string | No | Check-in date (YYYY-MM-DD) for rate information |
| `checkout` | string | No | Check-out date (YYYY-MM-DD) for rate information |
| `occupancy` | string | No | Occupancy string for rate information |

---

### `expedia.price_check`

Confirm real-time pricing and availability before booking.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `room_id` | string | Yes | Room ID from search results |

---

### `expedia.create_booking`

Book a hotel room. Creates a real reservation and may charge payment.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `room_id` | string | Yes | Room ID from a successful price check |
| `given_name` | string | Yes | Guest first name |
| `family_name` | string | Yes | Guest last name |
| `email` | string | Yes | Guest email address |
| `phone` | string | Yes | Guest phone number |
| `payment_method_id` | string | Yes | Stored payment method ID (resolved server-side) |
| `special_request` | string | No | Special requests for the hotel |

---

### `expedia.cancel_booking`

Cancel a hotel booking — may incur cancellation fees depending on policy.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `itinerary_id` | string | Yes | Itinerary ID from the booking |
| `room_id` | string | Yes | Room ID within the itinerary to cancel |

---

### `expedia.get_booking`

Retrieve booking details and current status.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `itinerary_id` | string | Yes | Itinerary ID from the booking |
| `email` | string | Yes | Email address used for the booking |

## Error Handling

The connector maps Expedia Rapid API responses to typed connector errors:

| Expedia Status | Connector Error | HTTP Response |
|----------------|-----------------|---------------|
| 400 | `ValidationError` | 400 Bad Request |
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline / canceled | `TimeoutError` | 504 Gateway Timeout |

Expedia returns errors as `{"type": "...", "message": "..."}`. The connector extracts the `message` field for error details when available.

## Adding a New Action

Each action lives in its own file. To add one (e.g., `expedia.search_hotels`):

1. Create `connectors/expedia/search_hotels.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, signature auth headers, `Customer-Ip`, response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `expedia.go`.
5. Add the action to the `Manifest()` return value inside `expedia.go` — include a `ParametersSchema`.
6. Add tests in `search_hotels_test.go` using `httptest.NewServer` and `newForTest()`.

The `do` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (signature auth, Content-Type, Customer-Ip, error mapping) are handled once.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable:

```go
{
    ActionType:  "expedia.search_hotels",
    Name:        "Search Hotels",
    Description: "Search available hotels with pricing",
    RiskLevel:   "low",
    ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
        "type": "object",
        "required": ["checkin", "checkout", "occupancy"],
        "properties": {
            "checkin": {
                "type": "string",
                "description": "Check-in date (YYYY-MM-DD)"
            }
        }
    }`)),
}
```

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `ExpediaConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/expedia/
├── expedia.go          # ExpediaConnector struct, New(), Manifest(), do(), ValidateCredentials(), signature()
├── response.go         # Shared HTTP response → typed error mapping
├── expedia_test.go     # Connector-level tests (signature, do(), credentials, manifest)
├── helpers_test.go     # Shared test helpers (validCreds)
└── README.md           # This file
```

Action files (Phase 2) will follow the pattern `<action>.go` + `<action>_test.go`.

## Testing

All tests use `httptest.NewServer` to mock the Expedia Rapid API — no real API calls are made.

```bash
go test ./connectors/expedia/... -v
```
