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

| Action | Risk | Description |
|--------|------|-------------|
| `hubspot.create_contact` | low | Create a new contact with email, name, phone, company, and custom properties |
| `hubspot.update_contact` | low | Update properties on an existing contact by ID |
| `hubspot.create_deal` | low | Create a deal in a pipeline with optional contact associations |
| `hubspot.create_ticket` | low | Create a support ticket in a pipeline |
| `hubspot.add_note` | low | Add an engagement note to a contact, deal, or ticket (two-step: create + associate) |
| `hubspot.search` | low | Search contacts, deals, tickets, or companies with property filters |

### `hubspot.create_contact`

Creates a new contact in HubSpot CRM.

**HubSpot API:** `POST /crm/v3/objects/contacts`
**Required scopes:** `crm.objects.contacts.write`

### `hubspot.update_contact`

Updates properties on an existing contact.

**HubSpot API:** `PATCH /crm/v3/objects/contacts/{contact_id}`
**Required scopes:** `crm.objects.contacts.write`

### `hubspot.create_deal`

Creates a new deal in a pipeline. Optionally associates the deal with contacts via the associations API.

**HubSpot API:** `POST /crm/v3/objects/deals` + optional `PUT /crm/v3/objects/deals/{id}/associations/contacts/{contact_id}/deal_to_contact`
**Required scopes:** `crm.objects.deals.write`

### `hubspot.create_ticket`

Creates a support ticket.

**HubSpot API:** `POST /crm/v3/objects/tickets`
**Required scopes:** `tickets`

### `hubspot.add_note`

Adds an engagement note to a CRM record (contact, deal, or ticket). This is a two-step flow: create the note, then associate it with the target object.

**HubSpot API:** `POST /crm/v3/objects/notes` + `PUT /crm/v3/objects/notes/{id}/associations/{type}/{object_id}/note_to_{type}`
**Required scopes:** `crm.objects.contacts.write` (or the scope for the associated object type)

### `hubspot.search`

Searches CRM objects with filters. Supports contacts, deals, tickets, and companies.

**HubSpot API:** `POST /crm/v3/objects/{object_type}/search`
**Required scopes:** `crm.objects.contacts.read`, `crm.objects.deals.read`, `crm.objects.companies.read`, or `tickets` (depending on object type)

**Supported operators:** `EQ`, `NEQ`, `LT`, `LTE`, `GT`, `GTE`, `CONTAINS_TOKEN`, `NOT_CONTAINS_TOKEN`, `HAS_PROPERTY`, `NOT_HAS_PROPERTY`, `BETWEEN`

**Limit:** Defaults to 10, capped at HubSpot's API maximum of 200.

### Key patterns

- **Property merging:** Actions like `create_contact`, `create_deal`, and `create_ticket` accept both explicit fields (e.g., `email`, `dealname`) and a catch-all `properties` map. Explicit fields take precedence over the properties map, so users can pass well-known fields naturally while still sending arbitrary HubSpot properties.

- **Two-step association:** `add_note` and `create_deal` (with `associated_contacts`) use a two-step flow — first create the object, then associate it. If the association call fails, the error includes the created object's ID so the user can recover.

- **Testable time:** The connector accepts a `nowFunc` field for deterministic timestamps in tests (used by `add_note` for `hs_timestamp`).

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

Each action lives in its own file. To add one (e.g., `hubspot.create_company`):

1. Create `connectors/hubspot/create_company.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `hubspot.go`.
5. Add the action to the `Manifest()` return value inside `hubspot.go` — include a `ParametersSchema`.
6. Add tests in `create_company_test.go` using `httptest.NewServer` and `newForTest()`.

HubSpot's CRM API follows a uniform pattern across object types — all use `/crm/v3/objects/{type}` for CRUD operations. The shared `hubspotObjectRequest` and `hubspotObjectResponse` types (defined in `create_contact.go`) can be reused for most CRM object actions.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `HubSpotConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/hubspot/
├── hubspot.go              # HubSpotConnector struct, New(), Manifest(), Actions(), do()
├── response.go             # HubSpot error category → typed error mapping
├── create_contact.go       # hubspot.create_contact + shared request/response types
├── update_contact.go       # hubspot.update_contact
├── create_deal.go          # hubspot.create_deal (with contact associations)
├── create_ticket.go        # hubspot.create_ticket
├── add_note.go             # hubspot.add_note (two-step create + associate)
├── search.go               # hubspot.search
├── hubspot_test.go         # Connector-level tests
├── response_test.go        # Error mapping tests
├── helpers_test.go         # Shared test helpers (validCreds)
├── *_test.go               # Per-action test files
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the HubSpot API — no real API calls are made.

```bash
go test ./connectors/hubspot/... -v
```
