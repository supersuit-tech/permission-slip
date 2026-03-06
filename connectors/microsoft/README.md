# Microsoft Connector

The Microsoft connector integrates Permission Slip with the [Microsoft Graph API](https://learn.microsoft.com/en-us/graph/overview). It uses plain `net/http` — no third-party Microsoft SDK.

## Connector ID

`microsoft`

## Credentials

This connector uses OAuth 2.0 — credentials are managed automatically by the platform's OAuth engine.

| Key | Source | Description |
|-----|--------|-------------|
| `access_token` | OAuth flow | A Microsoft Graph API access token, automatically provided by the platform after the user completes the OAuth consent flow. |

The credential `auth_type` in the database is `oauth2` with `oauth_provider: "microsoft"`. The platform handles the full OAuth lifecycle: redirect, token exchange, encrypted storage in Supabase Vault, and automatic refresh before expiry. The connector never touches OAuth code — it receives a valid access token in `Credentials` at execution time.

**Required OAuth scopes:** `Mail.Send`, `Mail.Read`, `Calendars.ReadWrite`, `Files.ReadWrite`

## Actions

### `microsoft.send_email`

Sends an email via Microsoft 365.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `to` | string[] | Yes | — | Recipient email addresses |
| `subject` | string | Yes | — | Email subject line |
| `body` | string | Yes | — | Email body (HTML or plain text — auto-detected) |
| `cc` | string[] | No | — | CC recipient email addresses |

**Response:**

```json
{
  "status": "sent"
}
```

**Graph API:** `POST /me/sendMail` ([docs](https://learn.microsoft.com/en-us/graph/api/user-sendmail))

---

### `microsoft.list_emails`

Lists recent emails from a mail folder.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder` | string | No | `"inbox"` | Mail folder (e.g., `inbox`, `sentitems`, `drafts`) |
| `top` | integer | No | `10` | Number of emails to return (1–50) |

**Response:**

```json
[
  {
    "id": "AAMkAD...",
    "subject": "Hello",
    "from": "sender@example.com",
    "to": ["recipient@example.com"],
    "received_at": "2024-01-15T09:00:00Z",
    "is_read": false,
    "preview": "Preview text...",
    "has_attachments": true
  }
]
```

**Graph API:** `GET /me/mailFolders/{folder}/messages` ([docs](https://learn.microsoft.com/en-us/graph/api/user-list-messages))

---

### `microsoft.create_calendar_event`

Creates a new event on the user's calendar.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `subject` | string | Yes | — | Event subject/title |
| `start` | string | Yes | — | Start date/time in ISO 8601 format (e.g., `2024-01-15T09:00:00`) |
| `end` | string | Yes | — | End date/time in ISO 8601 format (e.g., `2024-01-15T10:00:00`) |
| `time_zone` | string | No | `"UTC"` | Time zone (e.g., `America/New_York`) |
| `body` | string | No | — | Event body/description (HTML supported) |
| `attendees` | string[] | No | — | Attendee email addresses |
| `location` | string | No | — | Event location |

**Response:**

```json
{
  "id": "AAMkAD...",
  "subject": "Team Meeting",
  "start": "2024-01-15T09:00:00",
  "end": "2024-01-15T10:00:00",
  "web_link": "https://outlook.office365.com/calendar/item/..."
}
```

**Graph API:** `POST /me/events` ([docs](https://learn.microsoft.com/en-us/graph/api/user-post-events))

---

### `microsoft.list_calendar_events`

Lists upcoming events from the user's calendar.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `top` | integer | No | `10` | Number of events to return (1–50) |

**Response:**

```json
[
  {
    "id": "AAMkAD...",
    "subject": "Team Standup",
    "start": "2024-01-15T09:00:00",
    "end": "2024-01-15T09:30:00",
    "time_zone": "UTC",
    "location": "Zoom",
    "organizer": "manager@example.com",
    "web_link": "https://outlook.office365.com/calendar/item/...",
    "is_all_day": false
  }
]
```

**Graph API:** `GET /me/events` ([docs](https://learn.microsoft.com/en-us/graph/api/user-list-events))

---

### `microsoft.create_document`

Creates a new Word document in OneDrive via a simple upload.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `filename` | string | Yes | — | Document name (`.docx` appended if missing) |
| `folder_path` | string | No | root | OneDrive folder path (e.g., `Documents/Work`) |
| `content` | string | No | — | Initial plain-text content (max 4 MB) |

**Response:**

```json
{
  "id": "01BYE5RZ...",
  "name": "report.docx",
  "web_url": "https://onedrive.live.com/...",
  "created_date_time": "2024-01-15T09:00:00Z"
}
```

**Graph API:** `PUT /me/drive/root:/{path}:/content` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-put-content))

---

### `microsoft.get_document`

Gets metadata (and a temporary download URL) for a Word document in OneDrive.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID (returned by `create_document` or `list_documents`) |

**Response:**

```json
{
  "id": "01BYE5RZ...",
  "name": "report.docx",
  "web_url": "https://onedrive.live.com/...",
  "size": 12345,
  "created_date_time": "2024-01-15T09:00:00Z",
  "last_modified_date_time": "2024-01-16T10:00:00Z",
  "download_url": "https://download.example.com/..."
}
```

The `download_url` is a pre-authenticated temporary URL from `@microsoft.graph.downloadUrl` — it can be used to fetch the file content without additional auth.

**Graph API:** `GET /me/drive/items/{itemId}` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-get))

---

### `microsoft.update_document`

Replaces the content of an existing Word document in OneDrive.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID (returned by `create_document` or `list_documents`) |
| `content` | string | Yes | — | New document content (max 4 MB) |

**Response:**

```json
{
  "id": "01BYE5RZ...",
  "name": "report.docx",
  "web_url": "https://onedrive.live.com/...",
  "last_modified_date_time": "2024-01-16T10:00:00Z"
}
```

**Graph API:** `PUT /me/drive/items/{itemId}/content` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-put-content))

---

### `microsoft.list_documents`

Lists Word documents (`.docx`) from a OneDrive folder.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder_path` | string | No | root | OneDrive folder path (e.g., `Documents`) |
| `top` | integer | No | `10` | Number of documents to return (1–50) |

**Response:**

```json
{
  "documents": [
    {
      "id": "01BYE5RZ...",
      "name": "report.docx",
      "web_url": "https://onedrive.live.com/...",
      "size": 12345,
      "last_modified_date_time": "2024-01-16T10:00:00Z"
    }
  ]
}
```

Results are filtered server-side using `$filter=endswith(name,'.docx')` so only Word documents are returned.

**Graph API:** `GET /me/drive/root:/{path}:/children` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-list-children))

## Error Handling

The connector maps Microsoft Graph API responses to typed connector errors:

| Graph Status | Graph Error Code | Connector Error | HTTP Response |
|--------------|-----------------|-----------------|---------------|
| 401 | `InvalidAuthenticationToken` | `AuthError` | 502 Bad Gateway |
| 403 | `ErrorAccessDenied` | `AuthError` | 502 Bad Gateway |
| 429 | — | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | — | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | — | `TimeoutError` | 504 Gateway Timeout |

Rate limit responses include the `Retry-After` header value so callers know how long to wait (defaults to 30s if missing).

## Adding a New Action

Each action lives in its own file. To add one (e.g., `microsoft.list_contacts`):

1. Create `connectors/microsoft/list_contacts.go` with a params struct, `validate()` / `defaults()`, and an `Execute` method.
2. Use `a.conn.doRequest(ctx, method, path, creds, body, &resp)` for JSON APIs, or `a.conn.doUpload(ctx, method, path, creds, body, contentType, &resp)` for file uploads. Both share the same response handling via `handleResponse()`.
3. Use shared validators from `validation.go`: `validateItemID()`, `validateFolderPath()`, `escapeFolderPath()` for any OneDrive path params.
4. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
5. Register the action in `Actions()` inside `microsoft.go`.
6. Add the action to the `Manifest()` return value inside `microsoft.go` — include a `ParametersSchema` and a template.
7. Add tests in `list_contacts_test.go` using `httptest.NewServer` and `newForTest()`.

The `doRequest`/`doUpload` methods mean each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, rate limiting, error mapping) are handled once in `handleResponse()`.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `MicrosoftConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/microsoft/
├── microsoft.go                  # MicrosoftConnector struct, New(), Manifest(), doRequest(), doUpload(), handleResponse()
├── types.go                      # Shared Microsoft Graph API types (graphEmailBody, graphMailAddress, etc.)
├── response.go                   # Graph API error response → typed connector error mapping
├── validation.go                 # Shared validation helpers (validateEmail, validateItemID, validateFolderPath, etc.)
├── send_email.go                 # microsoft.send_email action
├── list_emails.go                # microsoft.list_emails action
├── create_calendar_event.go      # microsoft.create_calendar_event action
├── list_calendar_events.go       # microsoft.list_calendar_events action
├── create_document.go            # microsoft.create_document action (OneDrive)
├── get_document.go               # microsoft.get_document action (OneDrive)
├── update_document.go            # microsoft.update_document action (OneDrive)
├── list_documents.go             # microsoft.list_documents action (OneDrive)
├── microsoft_test.go             # Connector-level tests
├── helpers_test.go               # Shared test helpers (validCreds)
├── send_email_test.go            # Send email action tests
├── list_emails_test.go           # List emails action tests
├── create_calendar_event_test.go # Create calendar event action tests
├── list_calendar_events_test.go  # List calendar events action tests
├── create_document_test.go       # Create document action tests
├── get_document_test.go          # Get document action tests
├── update_document_test.go       # Update document action tests
├── list_documents_test.go        # List documents action tests
└── README.md                     # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Microsoft Graph API — no real API calls are made.

```bash
go test ./connectors/microsoft/... -v
```
