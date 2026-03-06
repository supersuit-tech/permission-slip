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

### `microsoft.list_drive_files`

Lists files and folders in OneDrive.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder_path` | string | No | root | Relative folder path (e.g., `Documents/Work`) |
| `top` | integer | No | `10` | Number of items to return (1–50) |

**Response:**

```json
{
  "folder_path": "Documents",
  "items": [
    {
      "id": "abc123",
      "name": "report.docx",
      "type": "file",
      "size": 1024,
      "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      "web_url": "https://onedrive.live.com/...",
      "created_at": "2024-01-15T09:00:00Z",
      "modified_at": "2024-01-16T10:00:00Z"
    }
  ]
}
```

**Graph API:** `GET /me/drive/root/children` or `GET /me/drive/root:/{path}:/children` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-list-children))

**Security:** `folder_path` is validated to reject path traversal (`..`), backslashes, absolute paths, and URL-special characters (`?#%`).

---

### `microsoft.get_drive_file`

Gets file metadata and optionally downloads text content.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID |
| `include_content` | boolean | No | `false` | Download file content (text files only) |

**Response:**

```json
{
  "id": "abc123",
  "name": "notes.txt",
  "type": "file",
  "size": 256,
  "mime_type": "text/plain",
  "web_url": "https://onedrive.live.com/...",
  "created_at": "2024-01-15T09:00:00Z",
  "modified_at": "2024-01-16T10:00:00Z",
  "content": "File content here..."
}
```

**Graph API:** `GET /me/drive/items/{id}` for metadata, `GET /me/drive/items/{id}/content` for download ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-get))

**Security:** Only text MIME types can be downloaded (`text/*`, `application/json`, `application/xml`, etc.). Binary files and files with unknown MIME types are rejected. `item_id` is validated to reject path separators, traversal sequences, and URL-special characters.

---

### `microsoft.upload_drive_file`

Uploads or creates a file in OneDrive (max 4 MB).

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_path` | string | Yes | — | Relative file path (e.g., `Documents/report.txt`) |
| `content` | string | Yes | — | File content to upload (max 4 MB) |
| `conflict_behavior` | string | No | `"rename"` | Behavior when file exists: `rename`, `replace`, or `fail` |

**Response:**

```json
{
  "id": "abc123",
  "name": "report.txt",
  "size": 17,
  "web_url": "https://onedrive.live.com/...",
  "created_at": "2024-01-15T09:00:00Z",
  "modified_at": "2024-01-15T09:00:00Z"
}
```

**Graph API:** `PUT /me/drive/root:/{path}:/content` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-put-content))

**Security:** Content size is capped at 4 MB. `file_path` is validated to reject path traversal, backslashes, absolute paths, and URL-special characters.

---

### `microsoft.delete_drive_file`

Moves a file to the OneDrive recycle bin (recoverable — not permanent deletion).

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID to delete |

**Response:**

```json
{
  "status": "deleted",
  "item_id": "abc123",
  "message": "File moved to recycle bin and can be recovered"
}
```

**Graph API:** `DELETE /me/drive/items/{id}` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-delete))

**Security:** `item_id` is validated to reject path separators, traversal sequences, and URL-special characters. The delete operation moves the item to the recycle bin — it is recoverable and not a permanent deletion.

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
2. Use `a.conn.doRequest(ctx, method, path, creds, body, &resp)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, rate limiting, error mapping, and timeout detection.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `microsoft.go`.
5. Add the action to the `Manifest()` return value inside `microsoft.go` — include a `ParametersSchema`.
6. Add tests in `list_contacts_test.go` using `httptest.NewServer` and `newForTest()`.

Three request helpers are available depending on the content type:

- `doRequest` — JSON request/response (most actions)
- `doRequestRaw` — Returns raw string response (file content download)
- `doPutRaw` — Sends raw bytes, returns raw bytes (file upload)

All three share a common `executeRequest` lifecycle handler for auth, rate limiting, error mapping, and timeout detection. Each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `MicrosoftConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/microsoft/
├── microsoft.go                  # MicrosoftConnector struct, Manifest(), request helpers, ValidateCredentials()
├── types.go                      # Shared Microsoft Graph API types (email, calendar, drive)
├── response.go                   # Graph API error response → typed connector error mapping
├── validation.go                 # Shared validation helpers (validateEmail, detectContentType)
├── send_email.go                 # microsoft.send_email action
├── list_emails.go                # microsoft.list_emails action + path validation helpers
├── create_calendar_event.go      # microsoft.create_calendar_event action
├── list_calendar_events.go       # microsoft.list_calendar_events action
├── list_drive_files.go           # microsoft.list_drive_files action
├── get_drive_file.go             # microsoft.get_drive_file action + item ID validation
├── upload_drive_file.go          # microsoft.upload_drive_file action
├── delete_drive_file.go          # microsoft.delete_drive_file action
├── microsoft_test.go             # Connector-level tests
├── helpers_test.go               # Shared test helpers (validCreds)
├── *_test.go                     # Per-action test files
└── README.md                     # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Microsoft Graph API — no real API calls are made.

```bash
go test ./connectors/microsoft/... -v
```
