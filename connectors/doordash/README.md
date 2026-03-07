# DoorDash Drive Connector

The DoorDash Drive connector integrates Permission Slip with the [DoorDash Drive API](https://developer.doordash.com/en-US/docs/drive/reference/), a delivery-as-a-service platform. Drive is **not** consumer food ordering — it moves items from point A to point B using DoorDash's Dasher network.

## Connector ID

`doordash`

## Credentials

DoorDash Drive uses self-signed JWT tokens (HS256 with a `DD-JWT-V1` header). The connector generates a short-lived JWT (5 minutes) on each request.

| Key | Required | Description |
|-----|----------|-------------|
| `developer_id` | Yes | Your DoorDash developer ID |
| `key_id` | Yes | The key ID for your API key |
| `signing_secret` | Yes | The signing secret for JWT generation |

The credential `auth_type` in the database is `api_key`. All three values are found in the [DoorDash Developer Portal](https://developer.doordash.com/portal/integration/drive) under your Drive integration.

**Setup:** [DoorDash Drive Getting Started](https://developer.doordash.com/en-US/docs/drive/getting-started/)

**Test mode:** DoorDash provides sandbox credentials for development. Sandbox deliveries are simulated and don't dispatch real Dashers or charge money.

## Actions

| Action | Risk | Required Params | Description |
|--------|------|-----------------|-------------|
| `doordash.get_quote` | low | `pickup_address`, `dropoff_address`, `pickup_phone`, `dropoff_phone` | Get a delivery fee estimate and ETA. Optional: `order_value` (cents) |
| `doordash.create_delivery` | **high** | `pickup_address`, `pickup_phone`, `dropoff_address`, `dropoff_phone`, `dropoff_contact_given_name` | Create a delivery that dispatches a real Dasher. Optional: `pickup_business_name`, `pickup_instructions`, `dropoff_instructions`, `order_value`, `items[]` |
| `doordash.get_delivery` | low | `delivery_id` | Check delivery status and Dasher info |
| `doordash.cancel_delivery` | medium | `delivery_id` | Cancel an active delivery (may incur fees) |
| `doordash.list_deliveries` | low | *(none)* | List deliveries. Optional filters: `limit`, `starting_after` (cursor), `status` |

### Delivery Lifecycle

A delivery moves through these statuses:

```
created → confirmed → enroute_to_pickup → arrived_at_pickup → picked_up →
  enroute_to_dropoff → arrived_at_dropoff → delivered
```

A delivery can also be `cancelled` at most stages. If the Dasher cannot complete delivery, it may go to `enroute_to_return` → `returned`.

### Recommended Workflow

1. **Get a quote** (`get_quote`) — show the user the estimated fee and ETA
2. **User approves** — confirm they accept the cost
3. **Create delivery** (`create_delivery`) — dispatches a Dasher
4. **Track status** (`get_delivery`) — monitor progress
5. **Cancel if needed** (`cancel_delivery`) — before pickup to avoid fees

### Money Values

All monetary values (`order_value`, fees) are in **cents** (USD). For example, `2500` = $25.00.

## Configuration Templates

| Template | Action | Risk | Notes |
|----------|--------|------|-------|
| Get delivery quotes (read-only) | `get_quote` | Low | Estimate only, no Dasher dispatched |
| Create deliveries (requires approval) | `create_delivery` | **High** | Dispatches real Dashers, charges money |
| Track deliveries (read-only) | `get_delivery` | Low | Read-only status checks |
| Cancel deliveries | `cancel_delivery` | Medium | May incur cancellation fees |
| List deliveries (read-only) | `list_deliveries` | Low | Read-only listing with filters |

## Error Handling

The DoorDash API returns structured errors:

```json
{
  "field_errors": [{"field": "pickup_address", "error": "This field is required."}],
  "message": "Invalid request",
  "code": "validation_error"
}
```

The connector maps these to typed connector errors:

| HTTP Status | Connector Error | Notes |
|-------------|-----------------|-------|
| 400 | `ValidationError` | Field-level errors included in message |
| 401 | `AuthError` | Includes link to developer portal |
| 403 | `AuthError` | May indicate missing Drive API access |
| 404 | `ValidationError` | Delivery not found |
| 429 | `RateLimitError` | Back off and retry |
| 5xx | `ExternalError` | DoorDash server error |
| Timeout | `TimeoutError` | 30s default timeout |

## File Structure

```
connectors/doordash/
├── doordash.go              # DoorDashConnector, New(), JWT auth, shared HTTP lifecycle
├── manifest.go              # ManifestProvider: action schemas, credentials, templates
├── response.go              # DoorDash error parsing and typed error mapping
├── create_delivery.go       # doordash.create_delivery (high risk)
├── get_quote.go             # doordash.get_quote
├── get_delivery.go          # doordash.get_delivery
├── cancel_delivery.go       # doordash.cancel_delivery
├── list_deliveries.go       # doordash.list_deliveries
├── *_test.go                # Test files for each action
├── helpers_test.go          # Shared test helpers (validCreds)
└── README.md                # This file
```

## Testing

All tests use `httptest.NewServer` to mock the DoorDash API — no real API calls or credentials needed.

```bash
go test ./connectors/doordash/... -v

# With race detector
go test ./connectors/doordash/... -race
```
