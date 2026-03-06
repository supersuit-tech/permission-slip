# HubSpot Connector

The HubSpot connector integrates Permission Slip with the [HubSpot CRM API v3](https://developers.hubspot.com/docs/api/crm). It uses plain `net/http` — no third-party HubSpot SDK.

## Connector ID

`hubspot`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A HubSpot private app access token with appropriate scopes for the actions being executed. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

**Getting a token:** Create a [private app](https://developers.hubspot.com/docs/api/private-apps) in your HubSpot account. Select the scopes needed for your use case (see individual action docs below). Copy the access token — it starts with `pat-`.

## Actions

*Actions are added in Phase 2. The following are planned:*

### `hubspot.create_contact`

Creates a new contact in HubSpot CRM.

**Risk level:** low
**HubSpot API:** `POST /crm/v3/objects/contacts`
**Required scopes:** `crm.objects.contacts.write`

### `hubspot.update_contact`

Updates properties on an existing contact.

**Risk level:** low
**HubSpot API:** `PATCH /crm/v3/objects/contacts/{contact_id}`
**Required scopes:** `crm.objects.contacts.write`

### `hubspot.create_deal`

Creates a new deal in a pipeline.

**Risk level:** low
**HubSpot API:** `POST /crm/v3/objects/deals`
**Required scopes:** `crm.objects.deals.write`

### `hubspot.create_ticket`

Creates a support ticket.

**Risk level:** low
**HubSpot API:** `POST /crm/v3/objects/tickets`
**Required scopes:** `tickets`

### `hubspot.add_note`

Adds an engagement note to a CRM record (contact, deal, or ticket).

**Risk level:** low
**HubSpot API:** `POST /crm/v3/objects/notes` + association call
**Required scopes:** `crm.objects.contacts.write` (or the scope for the associated object type)

### `hubspot.search`

Searches CRM objects with filters.

**Risk level:** low
**HubSpot API:** `POST /crm/v3/objects/{object_type}/search`
**Required scopes:** `crm.objects.contacts.read`, `crm.objects.deals.read`, `crm.objects.companies.read`, or `tickets` (depending on object type)

## Error Handling

HubSpot returns structured error responses with a `category` field. The connector maps these to typed connector errors, with HTTP status code fallback for responses without a category:

| HubSpot Category / Status | Connector Error | HTTP Response |
|---------------------------|-----------------|---------------|
| `UNAUTHORIZED`, `INVALID_AUTHENTICATION`, `REVOKED_AUTHENTICATION` | `AuthError` | 502 Bad Gateway |
| 401, 403 (no category) | `AuthError` | 502 Bad Gateway |
| `RATE_LIMITS`, HTTP 429 | `RateLimitError` | 429 Too Many Requests |
| `VALIDATION_ERROR`, `INVALID_PARAMS`, `PROPERTY_DOESNT_EXIST`, `INVALID_EMAIL`, `CONTACT_EXISTS` | `ValidationError` | 400 Bad Request |
| `OBJECT_NOT_FOUND`, `RESOURCE_NOT_FOUND`, 404 | `ValidationError` | 400 Bad Request |
| 400, 422 (no category) | `ValidationError` | 400 Bad Request |
| Other categories / status codes | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

Error messages include HubSpot's `correlationId` when available, which is useful for troubleshooting with HubSpot support.

Rate limit: HubSpot allows 100 requests per 10 seconds per private app. The `RateLimitError` includes a `RetryAfter` duration parsed from the `Retry-After` header (defaults to 10s).

## Adding a New Action

Each action lives in its own file. To add one (e.g., `hubspot.create_contact`):

1. Create `connectors/hubspot/create_contact.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `hubspot.go`.
5. Add the action to the `Manifest()` return value inside `hubspot.go` — include a `ParametersSchema`.
6. Add tests in `create_contact_test.go` using `httptest.NewServer` and `newForTest()`.

The `do` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, error mapping, rate limiting) are handled once.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable:

```go
{
    ActionType:  "hubspot.create_contact",
    Name:        "Create Contact",
    Description: "Create a new contact in HubSpot CRM",
    RiskLevel:   "low",
    ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
        "type": "object",
        "required": ["email"],
        "properties": {
            "email": {
                "type": "string",
                "description": "Contact email address"
            },
            "firstname": {
                "type": "string",
                "description": "Contact first name"
            }
        }
    }`)),
}
```

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `HubSpotConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/hubspot/
├── hubspot.go           # HubSpotConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── response.go          # HubSpot error category → typed error mapping
├── hubspot_test.go      # Connector-level tests
├── helpers_test.go      # Shared test helpers (validCreds)
├── response_test.go     # Error mapping tests
└── README.md            # This file
```

## Testing

All tests use `httptest.NewServer` to mock the HubSpot API — no real API calls are made.

```bash
go test ./connectors/hubspot/... -v
```
