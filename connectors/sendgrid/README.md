# SendGrid Connector

The SendGrid connector integrates Permission Slip with the [SendGrid v3 API](https://docs.sendgrid.com/api-reference). It uses plain `net/http` with Bearer token auth and JSON request bodies — no third-party SendGrid SDK.

## Connector ID

`sendgrid`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | SendGrid API key — starts with `SG.` and is typically 69+ characters. Create one at [SendGrid API Keys](https://docs.sendgrid.com/ui/account-and-settings/api-keys). |

The credential `auth_type` in the database is `api_key`. Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

### Authentication

All API requests use Bearer token authentication with the API key in the `Authorization` header.

### Required API Key Permissions

The API key needs these scopes depending on which actions you enable:

| Scope | Actions |
|-------|---------|
| Marketing > Single Sends | `send_campaign`, `schedule_campaign`, `get_campaign_stats` |
| Marketing > Contacts | `add_to_list`, `remove_from_list`, `list_lists` |
| Marketing > Segments | `list_segments` |
| Templates | `create_template` |
| Sender Verification | `list_senders` |

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `sendgrid.send_campaign` | Send Email Campaign | high | Create and immediately send a single send email campaign |
| `sendgrid.schedule_campaign` | Schedule Email Campaign | high | Create and schedule a single send campaign for future delivery |
| `sendgrid.add_to_list` | Add Subscriber to List | medium | Add a contact to a marketing contact list |
| `sendgrid.remove_from_list` | Remove Subscriber from List | medium | Remove a contact from a contact list |
| `sendgrid.create_template` | Create Email Template | medium | Create a dynamic transactional email template |
| `sendgrid.get_campaign_stats` | Get Campaign Stats | low | Get analytics for a campaign (opens, clicks, bounces) |
| `sendgrid.list_segments` | List Segments | low | List all contact segments in the account |
| `sendgrid.list_senders` | List Verified Senders | low | List verified sender identities (find sender_id for campaigns) |
| `sendgrid.list_lists` | List Contact Lists | low | List contact lists with subscriber counts (find list_id values) |

### Risk Levels

- **High:** `send_campaign`, `schedule_campaign` — sends to potentially thousands of recipients. The blast radius of a bad email is large, so these always require approval.
- **Medium:** `add_to_list`, `remove_from_list`, `create_template` — modifies audience data or content.
- **Low:** `get_campaign_stats`, `list_segments`, `list_senders`, `list_lists` — read-only operations with no side effects.

### Discovery Actions

Before creating campaigns, agents typically need to discover available resources:
- **`list_senders`** — find `sender_id` values (required for campaigns)
- **`list_lists`** — find `list_id` values (required for campaigns and subscriber management)
- **`list_segments`** — find segments for targeted campaigns

These are all low-risk, read-only actions that help agents work autonomously without requiring users to look up IDs manually.

### Typical Agent Workflow

1. Call `list_senders` to find the verified sender identity to use
2. Call `list_lists` to find the target audience list(s)
3. Draft the campaign content (subject, HTML/plain text body)
4. Call `send_campaign` or `schedule_campaign` (requires human approval due to high risk)
5. After sending, call `get_campaign_stats` to monitor delivery and engagement

### Campaign Sending

Campaign sending uses a two-step process:
1. Create a single send via `POST /marketing/singlesends`
2. Schedule it via `PUT /marketing/singlesends/{id}/schedule` (with `"now"` for immediate send or a future ISO 8601 timestamp)

Both steps happen atomically within a single action execution.

### Async Operations

Some SendGrid operations are asynchronous:
- **`add_to_list`** returns a `job_id` — the contact import runs in the background
- **`remove_from_list`** returns a `job_id` — the removal runs in the background

Both operations return immediately with `"status": "accepted"`. The actual processing happens asynchronously on SendGrid's side.

### Email Validation

The `add_to_list` action validates email addresses with a basic pattern check (`user@domain.tld`) before making the API call. This catches obviously invalid addresses early.

## Templates

Pre-configured templates for common setups:

| Template | Description | Safety |
|----------|-------------|--------|
| Send email campaign | Agent chooses all parameters | Unrestricted |
| **Send campaign to specific list** | Locks recipient list + sender | **Recommended** — prevents wrong-audience sends |
| Schedule email campaign | Agent chooses all parameters | Unrestricted |
| Add subscriber to list | Agent can add to any list | Unrestricted |
| **Add to specific list** | Locks the target list | **Recommended** — prevents cross-list additions |

The locked-list templates are the recommended starting point — they let agents draft content freely while preventing the most dangerous mistake (sending to the wrong audience).

## API Endpoints

| Action | Method | Endpoint |
|--------|--------|----------|
| send_campaign | POST + PUT | `/marketing/singlesends` then `/marketing/singlesends/{id}/schedule` |
| schedule_campaign | POST + PUT | `/marketing/singlesends` then `/marketing/singlesends/{id}/schedule` |
| add_to_list | PUT | `/marketing/contacts` |
| remove_from_list | DELETE | `/marketing/lists/{list_id}/contacts?contact_ids={id}` |
| create_template | POST | `/templates` |
| get_campaign_stats | GET | `/marketing/singlesends/{id}` |
| list_segments | GET | `/marketing/segments/2.0` |
| list_senders | GET | `/verified_senders` |
| list_lists | GET | `/marketing/lists` |

All endpoints use `application/json` request and response bodies. Dynamic path segments and query parameters are escaped via `url.PathEscape` / `url.QueryEscape` to prevent injection.

## Error Handling

The connector maps SendGrid API responses to typed connector errors:

| SendGrid Status | Connector Error | HTTP Response |
|-----------------|-----------------|---------------|
| 400 | `ValidationError` | 400 Bad Request |
| 401 | `AuthError` | 502 Bad Gateway |
| 403 | `AuthError` | 502 Bad Gateway |
| 404 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

SendGrid error responses include an `errors` array with `message` and optional `field` values. The connector extracts the first error message for display. Raw response bodies are truncated to 512 characters in error messages.

Rate limit responses include the `Retry-After` header value when available.

### Response Size Limit

All responses are capped at 1 MiB (`io.LimitReader`) to prevent memory exhaustion.

## File Structure

```
connectors/sendgrid/
├── sendgrid.go              # SendGridConnector struct, New(), Actions(), ValidateCredentials(), doJSON()
├── manifest.go              # Manifest() — action definitions, credentials, templates
├── campaign.go              # Shared campaignFields validation + buildSingleSendBody()
├── response.go              # checkResponse() — HTTP status → typed error mapping
├── send_campaign.go         # sendgrid.send_campaign action
├── schedule_campaign.go     # sendgrid.schedule_campaign action
├── add_to_list.go           # sendgrid.add_to_list action
├── remove_from_list.go      # sendgrid.remove_from_list action
├── create_template.go       # sendgrid.create_template action
├── get_campaign_stats.go    # sendgrid.get_campaign_stats action
├── list_segments.go         # sendgrid.list_segments action
├── list_senders.go          # sendgrid.list_senders action
├── list_lists.go            # sendgrid.list_lists action
├── *_test.go                # Tests for each action + connector + response
├── helpers_test.go          # Shared test helpers (validCreds, testAPIKey)
└── README.md                # This file
```

### Architecture

- **`sendgrid.go`** — Connector setup, HTTP client, credential validation, and the shared `doJSON()` method
- **`manifest.go`** — All action metadata, parameter schemas, credential requirements, and templates (separated for maintainability)
- **`campaign.go`** — Shared `campaignFields` struct and validation used by both `send_campaign` and `schedule_campaign`, plus the `buildSingleSendBody()` helper
- **`response.go`** — Centralized HTTP status code → typed error mapping
- **Action files** — One file per action, each containing the action struct, parameter validation, and Execute method

## Testing

All tests use `httptest.NewServer` to mock the SendGrid API — no real API calls are made.

```bash
go test ./connectors/sendgrid/... -v
```
