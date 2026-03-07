# Google Connector

The Google connector integrates Permission Slip with [Gmail](https://developers.google.com/gmail/api), [Google Calendar](https://developers.google.com/calendar/api), and [Google Sheets](https://developers.google.com/sheets/api) APIs. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Google SDK.

## Connector ID

`google`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 access token provided automatically by the platform's OAuth infrastructure. |

The credential `auth_type` is `oauth2` with `oauth_provider` set to `google` (a built-in provider). The platform handles the full OAuth flow, token storage, and automatic refresh — the connector just receives a valid access token at execution time.

**OAuth scopes requested:**

| Scope | Used by |
|-------|---------|
| `gmail.send` | `google.send_email` |
| `gmail.readonly` | `google.list_emails` |
| `calendar.events` | `google.create_calendar_event`, `google.list_calendar_events` |
| `spreadsheets` | `google.sheets_read_range`, `google.sheets_write_range`, `google.sheets_append_rows`, `google.sheets_list_sheets` |

Scopes follow the principle of least privilege — `calendar.events` (event-level access) is used instead of the broader `calendar` scope (full calendar management).

## Actions

### `google.send_email`

Sends an email via the Gmail API.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `to` | string | Yes | Recipient email address |
| `subject` | string | Yes | Email subject line |
| `body` | string | Yes | Email body (plain text) |

**Response:**

```json
{
  "id": "18abc123def",
  "thread_id": "18abc123def"
}
```

**Gmail API:** `POST /gmail/v1/users/me/messages/send` ([docs](https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send))

**Security notes:**
- The `to` and `subject` fields are validated to reject newline characters (`\r`, `\n`) to prevent MIME header injection in the RFC 2822 message.
- The message is encoded with unpadded base64url (`base64.RawURLEncoding`) per the Gmail API specification.

---

### `google.list_emails`

Lists recent emails from the Gmail inbox with metadata.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | No | — | Gmail search query (e.g., `from:user@example.com is:unread`) |
| `max_results` | integer | No | `10` | Maximum number of emails to return (1-100) |

**Response:**

```json
{
  "emails": [
    {
      "id": "18abc123def",
      "from": "sender@example.com",
      "to": "you@example.com",
      "subject": "Hello",
      "snippet": "Preview of the email body...",
      "date": "Mon, 15 Jan 2024 09:00:00 -0500"
    }
  ],
  "total_estimate": 42
}
```

**Gmail API:** `GET /gmail/v1/users/me/messages` + `GET /gmail/v1/users/me/messages/{id}` ([docs](https://developers.google.com/gmail/api/reference/rest/v1/users.messages/list))

The action first lists message IDs matching the query, then fetches metadata (From, To, Subject, Date headers and snippet) for each message.

---

### `google.create_calendar_event`

Creates a new event on Google Calendar.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `summary` | string | Yes | — | Event title |
| `description` | string | No | — | Event description |
| `start_time` | string | Yes | — | Start time in RFC 3339 format (e.g., `2024-01-15T09:00:00-05:00`) |
| `end_time` | string | Yes | — | End time in RFC 3339 format (must be after `start_time`) |
| `attendees` | string[] | No | — | List of attendee email addresses |
| `calendar_id` | string | No | `primary` | Calendar ID (defaults to `primary`) |

**Response:**

```json
{
  "id": "abc123event",
  "html_link": "https://calendar.google.com/event?eid=abc123",
  "status": "confirmed"
}
```

**Calendar API:** `POST /calendars/{calendarId}/events` ([docs](https://developers.google.com/calendar/api/v3/reference/events/insert))

**Validation:**
- `start_time` and `end_time` must be valid RFC 3339 timestamps.
- `end_time` must be strictly after `start_time` — equal or earlier times are rejected with a clear validation error.
- `calendar_id` is URL-encoded in the API path to safely handle IDs containing special characters (e.g., `user@group.calendar.google.com`).

---

### `google.list_calendar_events`

Lists upcoming events from Google Calendar.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `calendar_id` | string | No | `primary` | Calendar ID |
| `max_results` | integer | No | `10` | Maximum number of events to return (1-250) |
| `time_min` | string | No | now | Lower bound for event start time (RFC 3339). Defaults to current time. |
| `time_max` | string | No | — | Upper bound for event start time (RFC 3339) |

**Response:**

```json
{
  "events": [
    {
      "id": "event123",
      "summary": "Team Standup",
      "description": "Daily sync",
      "start_time": "2024-01-15T09:00:00-05:00",
      "end_time": "2024-01-15T09:30:00-05:00",
      "status": "confirmed",
      "html_link": "https://calendar.google.com/event?eid=...",
      "attendees": ["alice@example.com", "bob@example.com"]
    }
  ]
}
```

**Calendar API:** `GET /calendars/{calendarId}/events` ([docs](https://developers.google.com/calendar/api/v3/reference/events/list))

Events are returned as single instances (recurring events expanded) ordered by start time. All-day events use a date string (e.g., `2024-01-15`) instead of a full RFC 3339 timestamp.

---

### `google.sheets_read_range`

Reads cell values from a specified range in a Google Sheets spreadsheet.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `spreadsheet_id` | string | Yes | The ID of the spreadsheet to read from |
| `range` | string | Yes | The A1 notation range to read (e.g., `Sheet1!A1:D10`) |

**Response:**

```json
{
  "range": "Sheet1!A1:D3",
  "values": [
    ["Name", "Age", "City"],
    ["Alice", 30, "NYC"]
  ]
}
```

**Sheets API:** `GET /v4/spreadsheets/{id}/values/{range}` ([docs](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/get))

---

### `google.sheets_write_range`

Writes cell values to a specified range in a Google Sheets spreadsheet.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `spreadsheet_id` | string | Yes | The ID of the spreadsheet to write to |
| `range` | string | Yes | The A1 notation range to write (e.g., `Sheet1!A1:D3`) |
| `values` | any[][] | Yes | 2D array of cell values to write (rows of columns) |

**Response:**

```json
{
  "updated_range": "Sheet1!A1:C2",
  "updated_rows": 2,
  "updated_columns": 3,
  "updated_cells": 6
}
```

**Sheets API:** `PUT /v4/spreadsheets/{id}/values/{range}?valueInputOption=USER_ENTERED` ([docs](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/update))

Values are interpreted as if the user typed them into the UI (`USER_ENTERED`), so formulas and number formats are applied automatically.

---

### `google.sheets_append_rows`

Appends rows to a sheet or table in a Google Sheets spreadsheet.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `spreadsheet_id` | string | Yes | The ID of the spreadsheet to append to |
| `range` | string | Yes | The A1 notation of the range to search for a table to append to (e.g., `Sheet1`) |
| `values` | any[][] | Yes | 2D array of row values to append (rows of columns) |

**Response:**

```json
{
  "updated_range": "Sheet1!A4:C5",
  "updated_rows": 2,
  "updated_columns": 3,
  "updated_cells": 6
}
```

**Sheets API:** `POST /v4/spreadsheets/{id}/values/{range}:append?valueInputOption=USER_ENTERED` ([docs](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/append))

The Sheets API finds the last row with data in the specified range and appends rows after it.

---

### `google.sheets_list_sheets`

Lists all worksheets (tabs) in a Google Sheets spreadsheet.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `spreadsheet_id` | string | Yes | The ID of the spreadsheet |

**Response:**

```json
{
  "sheets": [
    {
      "sheet_id": 0,
      "title": "Sheet1",
      "index": 0,
      "sheet_type": "GRID",
      "row_count": 1000,
      "column_count": 26
    },
    {
      "sheet_id": 123456,
      "title": "Data",
      "index": 1,
      "sheet_type": "GRID",
      "row_count": 500,
      "column_count": 10
    }
  ]
}
```

**Sheets API:** `GET /v4/spreadsheets/{id}?fields=sheets.properties` ([docs](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets/get))

The response includes `row_count` and `column_count` from the sheet's grid properties, which can be used to determine sheet dimensions before read/write operations.

## Error Handling

The connector maps Google API HTTP status codes to typed connector errors:

| HTTP Status | Connector Error | Meaning |
|-------------|-----------------|---------|
| 401 Unauthorized | `AuthError` | Access token is invalid or expired |
| 403 Forbidden | `AuthError` | Missing required OAuth scope or permission |
| 429 Too Many Requests | `RateLimitError` | Google API quota exceeded (includes `Retry-After`) |
| Other 4xx/5xx | `ExternalError` | General API error (error message extracted from response) |
| Client timeout | `TimeoutError` | Request didn't complete within 30s |

Response bodies are capped at 10 MB via `io.LimitReader` to prevent out-of-memory from unexpectedly large responses.

## Templates

The connector ships with constrained templates that demonstrate parameter locking:

| Template | Action | What's locked |
|----------|--------|---------------|
| Send emails freely | `send_email` | Nothing — agent controls all parameters |
| Send email to specific recipient | `send_email` | `to` is locked to a placeholder; admin sets the real recipient |
| Search emails | `list_emails` | Nothing — agent controls query and count |
| List unread emails | `list_emails` | `query` locked to `is:unread` |
| Create calendar events | `create_calendar_event` | Nothing — agent controls all parameters |
| Create personal calendar events | `create_calendar_event` | `calendar_id` locked to `primary`, no attendees |
| List calendar events | `list_calendar_events` | Nothing — agent controls all parameters |
| Read from specific spreadsheet | `sheets_read_range` | `spreadsheet_id` locked; agent chooses range |
| Write to specific spreadsheet | `sheets_write_range` | `spreadsheet_id` locked; agent chooses range and values |
| Append to specific spreadsheet | `sheets_append_rows` | `spreadsheet_id` locked; agent chooses range and values |
| Read from any spreadsheet | `sheets_read_range` | Nothing — agent controls all parameters |
| List worksheets in any spreadsheet | `sheets_list_sheets` | Nothing — agent controls spreadsheet |

## Adding a New Action

Each action lives in its own file. To add one (e.g., `google.delete_calendar_event`):

1. Create `connectors/google/delete_calendar_event.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doJSON(ctx, creds, method, url, reqBody, &resp)` for the HTTP lifecycle — it handles JSON marshaling, Bearer auth, rate limiting, response size limits, and timeout detection.
3. Use `checkResponse()` (called automatically by `doJSON`) to map HTTP errors to typed connector errors.
4. Return `connectors.JSONResult(respBody)` to wrap the response into an `ActionResult`.
5. Register the action in `Actions()` inside `google.go`.
6. Add the action to the `Manifest()` return value inside `manifest.go` with a `ParametersSchema`.
7. Add tests in `delete_calendar_event_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/google/
├── google.go                       # GoogleConnector struct, New(), Actions(), doJSON(), ValidateCredentials()
├── manifest.go                     # Manifest() — connector metadata, action schemas, templates
├── send_email.go                   # google.send_email action
├── list_emails.go                  # google.list_emails action
├── create_calendar_event.go        # google.create_calendar_event action
├── list_calendar_events.go         # google.list_calendar_events action
├── sheets_read.go                  # google.sheets_read_range action
├── sheets_write.go                 # google.sheets_write_range action
├── sheets_append.go                # google.sheets_append_rows action
├── sheets_list.go                  # google.sheets_list_sheets action
├── sheets_helpers.go               # Shared validation helpers for Sheets actions
├── google_test.go                  # Connector-level tests (ID, Actions, Manifest, ValidateCredentials)
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_email_test.go              # Send email action tests (including MIME injection, base64 encoding)
├── list_emails_test.go             # List emails action tests
├── create_calendar_event_test.go   # Create event tests (including time validation, URL encoding)
├── list_calendar_events_test.go    # List events action tests
├── sheets_read_test.go             # Sheets read range tests
├── sheets_write_test.go            # Sheets write range tests
├── sheets_append_test.go           # Sheets append rows tests
├── sheets_list_test.go             # Sheets list worksheets tests
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Google APIs — no real API calls are made. Tests pass the Go race detector (`-race` flag).

```bash
go test ./connectors/google/... -v
go test ./connectors/google/... -race  # verify no race conditions
```
