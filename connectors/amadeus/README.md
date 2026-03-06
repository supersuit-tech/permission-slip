# Amadeus Connector

The Amadeus connector integrates Permission Slip with the [Amadeus Self-Service APIs](https://developers.amadeus.com/self-service) for flight search, hotel search, car rentals, and flight booking. It uses plain `net/http` — no third-party SDK.

## Connector ID

`amadeus`

## Authentication

Amadeus uses a **client credentials grant** (machine-to-machine OAuth). The connector exchanges `client_id` + `client_secret` for a short-lived bearer token via `POST /v1/security/oauth2/token`. This is NOT user-level OAuth — there's no redirect flow.

| Key | Required | Description |
|-----|----------|-------------|
| `client_id` | Yes | Amadeus API Key from the developer portal |
| `client_secret` | Yes | Amadeus API Secret from the developer portal |
| `environment` | No | `"test"` (default) or `"production"` — determines the base URL |

**Environments:**
- **Test:** `https://test.api.amadeus.com` — free tier with canned responses, great for development
- **Production:** `https://api.amadeus.com` — requires an approved Amadeus account

Tokens are cached per `client_id` and proactively refreshed 60 seconds before expiry (tokens last ~30 minutes). If a 401 response is received, the token is invalidated and the request is retried once with a fresh token.

**Getting credentials:** [Amadeus Self-Service API setup guide](https://developers.amadeus.com/get-started/get-started-with-self-service-apis-335)

## Actions

### `amadeus.search_airports`

Look up airports by name or IATA code — essential for building flight search params.

**Risk level:** low

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `keyword` | string | Yes | Airport name or IATA code (e.g., "San Francisco" or "SFO") |
| `subtype` | string | No | Filter by `AIRPORT` or `CITY` (defaults to both) |

**Amadeus API:** `GET /v1/reference-data/locations`

---

### `amadeus.search_flights`

Search flight offers between airports.

**Risk level:** low

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `origin` | string | Yes | — | Origin IATA code (e.g., "SFO") |
| `destination` | string | Yes | — | Destination IATA code (e.g., "LAX") |
| `departure_date` | string | Yes | — | Departure date (YYYY-MM-DD) |
| `return_date` | string | No | — | Return date for round trip (YYYY-MM-DD) |
| `adults` | integer | No | 1 | Number of adult travelers (1-9) |
| `cabin` | string | No | — | `ECONOMY`, `PREMIUM_ECONOMY`, `BUSINESS`, or `FIRST` |
| `nonstop` | boolean | No | false | Only show nonstop flights |
| `max_results` | integer | No | 10 | Maximum number of results (capped at 250) |

**Amadeus API:** `GET /v2/shopping/flight-offers`

---

### `amadeus.price_flight`

Confirm real-time pricing for a specific flight offer before booking.

**Risk level:** low

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `flight_offer` | object | Yes | Flight offer object from search results (max 100KB) |

**Amadeus API:** `POST /v1/shopping/flight-offers/pricing`

---

### `amadeus.book_flight`

Create a flight booking (PNR). **High risk — creates a real reservation.**

**Risk level:** high

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `flight_offer` | object | Yes | Priced flight offer object (max 100KB) |
| `travelers` | array | Yes | Array of traveler details (max 9) — see below |
| `payment_method_id` | string | Yes | Stored payment method ID (resolved server-side) |
| `remarks` | string | No | Booking remarks (max 500 chars) |

**Traveler object:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name.firstName` | string | Yes | Traveler's first/given name |
| `name.lastName` | string | Yes | Traveler's last/family name |
| `dateOfBirth` | string | Yes | Date of birth (YYYY-MM-DD) |
| `gender` | string | Yes | `MALE` or `FEMALE` |
| `contact.email` | string | Yes | Email address |
| `contact.phone` | string | Yes | Phone number |

**Payment:** The agent passes a `payment_method_id` and Permission Slip injects the real payment details server-side. The agent never sees raw card data.

**Amadeus API:** `POST /v1/booking/flight-orders`

---

### `amadeus.search_hotels`

Search hotel offers by city or geographic coordinates. Internally performs a two-step search:
1. Get hotel IDs for the location (up to 20)
2. Fetch offers with pricing for those hotels

**Risk level:** low

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `city_code` | string | Conditional | City IATA code (e.g., "PAR") — required if lat/lon not provided |
| `latitude` | string | Conditional | Latitude for geo search — required with longitude if city_code not provided |
| `longitude` | string | Conditional | Longitude for geo search |
| `check_in_date` | string | Yes | Check-in date (YYYY-MM-DD) |
| `check_out_date` | string | Yes | Check-out date (YYYY-MM-DD) |
| `adults` | integer | No | Number of adults (default 1) |
| `room_quantity` | integer | No | Number of rooms (default 1) |
| `ratings` | array | No | Star ratings to filter by (1-5) |
| `price_range` | string | No | Price range (e.g., "100-300") |
| `currency` | string | No | Currency code (e.g., "USD") |

**Amadeus API:** `GET /v1/reference-data/locations/hotels/by-city` → `GET /v3/shopping/hotel-offers`

---

### `amadeus.search_cars`

Search available rental car / transfer offers at a location.

**Risk level:** low

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `pickup_location` | string | Yes | — | Pickup IATA code |
| `pickup_date` | string | Yes | — | Pickup date/time |
| `dropoff_date` | string | Yes | — | Dropoff date/time |
| `dropoff_location` | string | No | pickup location | Dropoff IATA code |
| `provider` | string | No | — | Transfer type / provider filter |

**Amadeus API:** `POST /v2/shopping/transfer-offers`

## Flight Booking Flow

The recommended three-step flow for booking flights:

1. **Search** (`search_flights`) — find available flights
2. **Price** (`price_flight`) — confirm real-time pricing for a selected offer
3. **Book** (`book_flight`) — create the reservation

Each step validates the previous step's data. Pricing an offer before booking ensures the price hasn't changed since the search.

## Error Handling

| Amadeus Status | Connector Error | Description |
|----------------|-----------------|-------------|
| 401 | `AuthError` | Invalid/expired token (auto-retries once) |
| 403 | `AuthError` | Insufficient permissions (no retry) |
| 400 | `ValidationError` | Bad request parameters |
| 404 | `ValidationError` | Resource not found |
| 429 | `RateLimitError` | Rate limit exceeded (extracts `Retry-After`) |
| Other 5xx | `ExternalError` | Amadeus server error |

**Rate limits:** 1 request per 100ms on test environment, higher on production. 429 responses include a `Retry-After` header.

## Input Validation

All actions validate inputs before making API calls:

- **IATA codes** must be exactly 3 uppercase letters (e.g., "SFO")
- **Dates** must be valid YYYY-MM-DD format
- **Cabin classes** must be one of: ECONOMY, PREMIUM_ECONOMY, BUSINESS, FIRST
- **Genders** must be MALE or FEMALE
- **Flight offers** are capped at 100KB to prevent oversized payloads
- **Travelers** are capped at 9 per booking
- **Remarks** are capped at 500 characters

## File Structure

```
connectors/amadeus/
├── amadeus.go              # AmadeusConnector struct, New(), do(), token management
├── manifest.go             # Manifest() with JSON schemas, Actions() registration
├── validate.go             # Shared validation helpers and constants
├── response.go             # HTTP response → typed error mapping
├── search_airports.go      # amadeus.search_airports action
├── search_flights.go       # amadeus.search_flights action
├── price_flight.go         # amadeus.price_flight action
├── book_flight.go          # amadeus.book_flight action
├── search_hotels.go        # amadeus.search_hotels action (two-step)
├── search_cars.go          # amadeus.search_cars action
├── amadeus_test.go         # Connector-level tests
├── helpers_test.go         # Shared test helpers (validCreds, mock server)
├── search_airports_test.go # Search airports action tests
├── search_flights_test.go  # Search flights action tests
├── price_flight_test.go    # Price flight action tests
├── book_flight_test.go     # Book flight action tests
├── search_hotels_test.go   # Search hotels action tests
├── search_cars_test.go     # Search cars action tests
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Amadeus API — no real API calls are made. The test server intercepts the OAuth token endpoint and returns a fixed token.

```bash
go test ./connectors/amadeus/... -v
```

To run with the race detector:

```bash
go test ./connectors/amadeus/... -race
```
