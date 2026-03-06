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

### Excel Actions

All Excel actions operate on workbooks stored in OneDrive via the Microsoft Graph workbook API. They require the `Files.ReadWrite` OAuth scope.

**Obtaining `item_id`:** The `item_id` parameter is the OneDrive item ID of the `.xlsx` file. You can find it by browsing OneDrive via the Graph API (`GET /me/drive/root/children`) or by using the OneDrive search endpoint (`GET /me/drive/search(q='.xlsx')`). The ID looks like `01BYE5RZ6QN3ZWBTUFOFD3GSPGOHDJD36K`.

**Note on `excel_append_rows`:** This action operates on named [Excel tables](https://support.microsoft.com/en-us/office/create-and-format-tables-e81aa349-b006-4f8a-9806-5af9df0ac664), not raw ranges. The workbook must contain a table created via "Insert > Table" in Excel.

### `microsoft.excel_list_worksheets`

Lists all worksheets in an Excel workbook stored in OneDrive.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID of the Excel workbook |

**Response:**

```json
[
  {
    "id": "{00000000-0001-0000-0000-000000000000}",
    "name": "Sheet1",
    "position": 0,
    "visibility": "Visible"
  }
]
```

**Graph API:** `GET /me/drive/items/{itemId}/workbook/worksheets` ([docs](https://learn.microsoft.com/en-us/graph/api/workbook-list-worksheets))

---

### `microsoft.excel_read_range`

Reads cell values from a worksheet range in an Excel workbook.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID of the Excel workbook |
| `sheet_name` | string | Yes | — | Name of the worksheet to read from |
| `range` | string | Yes | — | Cell range to read (e.g., `A1:C10`) |

**Response:**

```json
{
  "address": "Sheet1!A1:B2",
  "values": [
    ["Name", "Age"],
    ["Alice", 30]
  ],
  "row_count": 2,
  "column_count": 2
}
```

**Graph API:** `GET /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}')` ([docs](https://learn.microsoft.com/en-us/graph/api/range-get))

---

### `microsoft.excel_write_range`

Writes cell values to a worksheet range in an Excel workbook.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID of the Excel workbook |
| `sheet_name` | string | Yes | — | Name of the worksheet to write to |
| `range` | string | Yes | — | Cell range to write (e.g., `A1:C3`) |
| `values` | any[][] | Yes | — | 2D array of cell values to write — all rows must have the same number of columns |

**Response:**

```json
{
  "address": "Sheet1!A1:B2",
  "values": [
    ["Name", "Age"],
    ["Alice", 30]
  ],
  "row_count": 2,
  "column_count": 2
}
```

**Graph API:** `PATCH /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}')` ([docs](https://learn.microsoft.com/en-us/graph/api/range-update))

---

### `microsoft.excel_append_rows`

Appends rows to a named table in an Excel workbook.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID of the Excel workbook |
| `table_name` | string | Yes | — | Name of the table to append rows to |
| `values` | any[][] | Yes | — | 2D array of row values to append — all rows must have the same number of columns |

**Response:**

```json
{
  "index": 5,
  "values": [
    ["Widget", 100, 9.99]
  ],
  "rows_added": 1
}
```

**Graph API:** `POST /me/drive/items/{itemId}/workbook/tables/{tableName}/rows` ([docs](https://learn.microsoft.com/en-us/graph/api/table-post-rows))

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

The `doRequest` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, rate limiting, error mapping) are handled once.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `MicrosoftConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/microsoft/
├── microsoft.go                  # MicrosoftConnector struct, New(), Manifest(), doRequest(), ValidateCredentials()
├── types.go                      # Shared Microsoft Graph API types (graphEmailBody, graphMailAddress, etc.)
├── response.go                   # Graph API error response → typed connector error mapping
├── validation.go                 # Shared validation helpers (validateEmail, detectContentType, validatePathSegment, validateValuesGrid)
├── excel_helpers.go              # Shared Excel helpers (excelWorkbookPath, newRangeResult, validateItemID)
├── send_email.go                 # microsoft.send_email action
├── list_emails.go                # microsoft.list_emails action
├── create_calendar_event.go      # microsoft.create_calendar_event action
├── list_calendar_events.go       # microsoft.list_calendar_events action
├── excel_list_worksheets.go      # microsoft.excel_list_worksheets action
├── excel_read.go                 # microsoft.excel_read_range action
├── excel_write.go                # microsoft.excel_write_range action
├── excel_append.go               # microsoft.excel_append_rows action
├── microsoft_test.go             # Connector-level tests
├── helpers_test.go               # Shared test helpers (validCreds)
├── validation_test.go            # Validation helper tests (path segment, values grid)
├── send_email_test.go            # Send email action tests
├── list_emails_test.go           # List emails action tests
├── create_calendar_event_test.go # Create calendar event action tests
├── list_calendar_events_test.go  # List calendar events action tests
├── excel_list_worksheets_test.go # Excel list worksheets action tests
├── excel_read_test.go            # Excel read range action tests
├── excel_write_test.go           # Excel write range action tests
├── excel_append_test.go          # Excel append rows action tests
└── README.md                     # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Microsoft Graph API — no real API calls are made.

```bash
go test ./connectors/microsoft/... -v
```
