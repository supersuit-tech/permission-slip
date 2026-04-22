# Google Connector

The Google connector integrates Permission Slip with [Gmail](https://developers.google.com/gmail/api), [Google Calendar](https://developers.google.com/calendar/api), [Google Slides](https://developers.google.com/slides/api), [Google Sheets](https://developers.google.com/sheets/api), [Google Docs](https://developers.google.com/docs/api), [Google Chat](https://developers.google.com/chat/api), and [Google Drive](https://developers.google.com/drive/api) APIs. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Google SDK.

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
| `gmail.readonly` | `google.list_emails`, `google.read_email` |
| `calendar.events` | `google.create_calendar_event`, `google.list_calendar_events`, `google.create_meeting` |
| `presentations` | `google.create_presentation`, `google.get_presentation`, `google.add_slide` |
| `spreadsheets` | `google.sheets_read_range`, `google.sheets_write_range`, `google.sheets_append_rows`, `google.sheets_list_sheets` |
| `documents` | `google.create_document`, `google.get_document`, `google.update_document` |
| `chat.spaces.readonly` | `google.list_chat_spaces` |
| `chat.messages.create` | `google.send_chat_message` |
| `drive` | `google.list_drive_files`, `google.get_drive_file`, `google.upload_drive_file`, `google.delete_drive_file`, `google.list_documents`, `google.search_drive`, `google.create_drive_folder` |
| `calendar.events` (also used for updates/deletes) | `google.update_calendar_event`, `google.delete_calendar_event` |
| `gmail.send` + `gmail.readonly` (both required) | `google.send_email_reply` |
| `gmail.modify` | `google.archive_email` |

Scopes follow the principle of least privilege — `calendar.events` (event-level access) is used instead of the broader `calendar` scope (full calendar management). The `drive` scope grants full Drive access; a narrower scope like `drive.file` would only allow access to files created by this app, which is too restrictive for file browsing and reading.

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
      "thread_id": "18abc123def",
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

The action first lists message IDs matching the query, then fetches metadata (From, To, Subject, Date headers and snippet) for each message. Each email includes a `thread_id` to enable seamless list → read → reply workflows.

---

### `google.read_email`

Fetches a single email by message ID and returns the full body, headers, labels, and attachment metadata.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `message_id` | string | Yes | The Gmail message ID (from `list_emails` or other source) |

**Response:**

```json
{
  "id": "18abc123def",
  "thread_id": "thread123",
  "from": "sender@example.com",
  "to": "recipient@example.com",
  "cc": "cc@example.com",
  "subject": "Project Update",
  "date": "Mon, 14 Mar 2026 10:00:00 -0500",
  "snippet": "Here is the latest update on...",
  "labels": ["INBOX", "UNREAD"],
  "content_type": "text/plain",
  "body": "Full email body text...",
  "attachments": [
    {
      "filename": "report.pdf",
      "mime_type": "application/pdf",
      "size": 12345,
      "part_id": "1",
      "attachment_id": "ANGjdJ8..."
    }
  ]
}
```

**Gmail API:** `GET /gmail/v1/users/me/messages/{id}?format=full` ([docs](https://developers.google.com/gmail/api/reference/rest/v1/users.messages/get))

**Typical workflow:** `list_emails` → `read_email` (using the `id` from the list) → `send_email_reply` (using `thread_id` and `id`).

**Body extraction:**
- For multipart messages, `text/plain` is preferred over `text/html`.
- The MIME part tree is walked recursively (depth-limited to 20) to find the best text body.
- Bodies exceeding 1 MB are truncated at a valid UTF-8 boundary with a `[truncated]` marker.

**Attachment metadata:**
- Filenames are extracted from `Content-Disposition` headers, falling back to `Content-Type` `name=` parameter.
- RFC 5987 extended filenames (`filename*=UTF-8''encoded%20name`) are supported and take priority per RFC 6266.
- Only UTF-8 encoded filenames are decoded; other charsets are skipped to avoid producing invalid strings.
- The `attachment_id` field can be used with `GET /gmail/v1/users/me/messages/{messageId}/attachments/{attachmentId}` to download content.

**Security notes:**
- Header names are matched case-insensitively per RFC 5322.
- MIME tree recursion is capped at depth 20 to prevent stack overflow from crafted deeply-nested messages.
- Base64url decoding tries raw (no padding) first since that's the Gmail API format, with padded fallback.

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

### `google.create_presentation`

Creates a new Google Slides presentation.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `title` | string | Yes | Title of the new presentation |

**Response:**

```json
{
  "presentation_id": "1BxiMVs0XRA5nFMdKvNZL4mZHlAGaSqWi",
  "title": "Q1 Review",
  "url": "https://docs.google.com/presentation/d/1BxiMVs0XRA5nFMdKvNZL4mZHlAGaSqWi/edit"
}
```

**Slides API:** `POST /v1/presentations` ([docs](https://developers.google.com/slides/api/reference/rest/v1/presentations/create))

---

### `google.get_presentation`

Retrieves metadata about an existing Google Slides presentation, including slide count and individual slide IDs.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `presentation_id` | string | Yes | The ID of the presentation to retrieve |

**Response:**

```json
{
  "presentation_id": "1BxiMVs0XRA5nFMdKvNZL4mZHlAGaSqWi",
  "title": "Q1 Review",
  "url": "https://docs.google.com/presentation/d/1BxiMVs0XRA5nFMdKvNZL4mZHlAGaSqWi/edit",
  "slide_count": 3,
  "slides": ["slide-001", "slide-002", "slide-003"]
}
```

**Slides API:** `GET /v1/presentations/{presentationId}` ([docs](https://developers.google.com/slides/api/reference/rest/v1/presentations/get))

---

### `google.add_slide`

Adds a new slide to an existing presentation using the Slides API's `batchUpdate` endpoint.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `presentation_id` | string | Yes | — | The ID of the presentation to add a slide to |
| `layout` | string | No | `BLANK` | Predefined slide layout (see below) |
| `insertion_index` | integer | No | end | Position to insert the slide at (0-indexed) |

**Supported layouts:** `BLANK`, `TITLE`, `TITLE_AND_BODY`, `TITLE_ONLY`, `SECTION_HEADER`, `SECTION_TITLE_AND_DESCRIPTION`, `ONE_COLUMN_TEXT`, `MAIN_POINT`, `BIG_NUMBER`, `CAPTION_ONLY`, `TITLE_AND_TWO_COLUMNS`

**Response:**

```json
{
  "slide_id": "g123abc",
  "presentation_id": "1BxiMVs0XRA5nFMdKvNZL4mZHlAGaSqWi"
}
```

**Slides API:** `POST /v1/presentations/{presentationId}:batchUpdate` ([docs](https://developers.google.com/slides/api/reference/rest/v1/presentations/batchUpdate))

The `batchUpdate` pattern is the standard Slides API approach for mutations and is extensible for future text/image insertion actions.

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

---

### `google.create_document`

Creates a new Google Doc with a title and optional initial document body text.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `title` | string | Yes | Title of the new Google Doc |
| `content` | string | No | Optional initial document body (plain text); same field name as `google.update_document` |
| `body` | string | No | Deprecated — use `content` instead |

**Response:**

```json
{
  "document_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "title": "Meeting Notes",
  "document_url": "https://docs.google.com/document/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms/edit"
}
```

**Docs API:** `POST /v1/documents` ([docs](https://developers.google.com/docs/api/reference/rest/v1/documents/create))

**Implementation notes:**
- Document creation and body insertion are two separate API calls. If the create succeeds but body insertion fails, the response includes a `warning` field with the error details and the document info (to avoid orphaning).
- Document IDs are URL-encoded in all constructed URLs.

---

### `google.get_document`

Retrieves the content and metadata of a Google Doc by document ID. Returns plain text content extracted from the Docs API structural content (paragraphs and text runs).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `document_id` | string | Yes | The ID of the Google Doc (e.g., `1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms`) |

**Response:**

```json
{
  "document_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "title": "Meeting Notes",
  "body_text": "Full plain text content of the document...",
  "document_url": "https://docs.google.com/document/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms/edit",
  "word_count": 42
}
```

**Docs API:** `GET /v1/documents/{documentId}` ([docs](https://developers.google.com/docs/api/reference/rest/v1/documents/get))

**Implementation notes:**
- The Google Docs API returns structural content (paragraphs, text runs, inline objects). This action extracts plain text only — formatting, images, tables, and other rich content are not preserved.
- `word_count` is a whitespace-separated count of the extracted text.

---

### `google.update_document`

Appends or inserts text into an existing Google Doc using the Docs API batchUpdate.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `document_id` | string | Yes | — | The ID of the Google Doc to update |
| `content` | string | Yes | — | Text to insert into the document |
| `text` | string | — | — | Deprecated — use `content` instead |
| `index` | integer | No | end | Character index to insert at (1-based). Defaults to end of document. |

**Response:**

```json
{
  "document_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "status": "updated"
}
```

**Docs API:** `POST /v1/documents/{documentId}:batchUpdate` ([docs](https://developers.google.com/docs/api/reference/rest/v1/documents/batchUpdate))

**Validation:**
- `index` must be >= 1 when provided (Google Docs uses 1-based indexing where index 1 is the start of the document body).
- When `index` is omitted (or 0), text is appended to the end of the document using `endOfSegmentLocation`.

---

### `google.list_documents`

Searches and lists Google Docs from Drive, filtered by the Google Docs MIME type.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | No | — | Search query to filter documents by name |
| `max_results` | integer | No | `10` | Maximum number of documents to return (1-100) |

**Response:**

```json
{
  "documents": [
    {
      "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
      "name": "Meeting Notes",
      "created_time": "2024-01-15T09:00:00.000Z",
      "modified_time": "2024-01-16T10:00:00.000Z",
      "web_view_link": "https://docs.google.com/document/d/1Bxi.../edit"
    }
  ],
  "count": 1
}
```

**Drive API:** `GET /drive/v3/files` ([docs](https://developers.google.com/drive/api/v3/reference/files/list))

**Security notes:**
- The `query` parameter is escaped (single quotes and backslashes) before being inserted into the Drive query string to prevent query injection.
- Results are ordered by `modifiedTime desc` and capped at 100 items.
- Trashed documents are automatically excluded.

---

### `google.send_chat_message`

Sends a message to a Google Chat space.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `space_name` | string | Yes | The resource name of the space (e.g., `spaces/AAAA1234`) |
| `text` | string | Yes | The message text |

**Response:**

```json
{
  "name": "spaces/AAAA1234/messages/msg-001",
  "space": "spaces/AAAA1234",
  "thread": "spaces/AAAA1234/threads/thread-001",
  "create_time": "2024-01-15T09:00:00Z"
}
```

**Chat API:** `POST /v1/{parent}/messages` ([docs](https://developers.google.com/chat/api/reference/rest/v1/spaces.messages/create))

**Security notes:**
- `space_name` is validated to start with `spaces/` and cannot contain `..` segments, preventing path traversal attacks.

---

### `google.list_chat_spaces`

Lists Google Chat spaces accessible to the authenticated user.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page_size` | integer | No | `20` | Maximum number of spaces to return (1-100) |
| `filter` | string | No | — | Optional filter query (e.g., `spaceType = "SPACE"` to list only named spaces) |

**Response:**

```json
{
  "spaces": [
    {
      "name": "spaces/AAAA1234",
      "display_name": "Engineering",
      "type": "ROOM",
      "space_type": "SPACE"
    }
  ]
}
```

**Chat API:** `GET /v1/spaces` ([docs](https://developers.google.com/chat/api/reference/rest/v1/spaces/list))

---

### `google.create_meeting`

Creates a Google Calendar event with an auto-generated Google Meet conference link.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `summary` | string | Yes | — | Meeting title |
| `description` | string | No | — | Meeting description |
| `start_time` | string | Yes | — | Start time in RFC 3339 format (e.g., `2024-01-15T09:00:00-05:00`) |
| `end_time` | string | Yes | — | End time in RFC 3339 format (must be after `start_time`) |
| `attendees` | string[] | No | — | List of attendee email addresses |
| `calendar_id` | string | No | `primary` | Calendar ID (defaults to `primary`) |

**Response:**

```json
{
  "id": "event-123",
  "html_link": "https://calendar.google.com/event?eid=123",
  "status": "confirmed",
  "meet_link": "https://meet.google.com/abc-defg-hij"
}
```

**Calendar API:** `POST /calendars/{calendarId}/events?conferenceDataVersion=1` ([docs](https://developers.google.com/calendar/api/v3/reference/events/insert))

**Implementation details:**
- Uses `conferenceDataVersion=1` and `conferenceSolutionKey.type=hangoutsMeet` to request automatic Meet link generation.
- The `requestId` is derived deterministically from the meeting summary and start time (SHA-256 hash), making the request idempotent — creating the same meeting twice returns the same conference link.
- The `meet_link` field is only present when Google successfully attaches conference data to the event.
- Validation rules match `google.create_calendar_event` (RFC 3339 times, end after start).

---

### `google.list_drive_files`

Lists or searches files in Google Drive.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | No | — | Google Drive search query (e.g., `name contains 'report'`) |
| `max_results` | integer | No | `10` | Maximum number of files to return (1-100) |
| `folder_id` | string | No | — | Folder ID to filter by (lists files within that folder) |
| `order_by` | string | No | relevance | Sort order (e.g., `modifiedTime desc`, `name`) |

**Response:**

```json
{
  "files": [
    {
      "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
      "name": "Q4 Report",
      "mime_type": "application/vnd.google-apps.document",
      "modified_time": "2024-01-15T10:00:00.000Z",
      "size": "",
      "web_view_link": "https://docs.google.com/document/d/1Bxi.../edit"
    }
  ],
  "count": 1
}
```

**Drive API:** `GET /drive/v3/files` ([docs](https://developers.google.com/drive/api/reference/rest/v3/files/list))

Trashed files are automatically excluded. Google Workspace files (Docs, Sheets, Slides) have no `size` — they report an empty string.

---

### `google.get_drive_file`

Gets file metadata and optionally downloads content from Google Drive.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_id` | string | Yes | — | The ID of the file to retrieve |
| `include_content` | boolean | No | `false` | Whether to include file content |

**Response (metadata only):**

```json
{
  "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "name": "Q4 Report",
  "mime_type": "application/vnd.google-apps.document",
  "modified_time": "2024-01-15T10:00:00.000Z",
  "web_view_link": "https://docs.google.com/document/d/1Bxi.../edit"
}
```

**Response (with content):**

```json
{
  "id": "...",
  "name": "Q4 Report",
  "mime_type": "application/vnd.google-apps.document",
  "content": "The full text content of the document..."
}
```

**Drive API:** `GET /drive/v3/files/{id}` for metadata, `GET /drive/v3/files/{id}/export` for Workspace content, `GET /drive/v3/files/{id}?alt=media` for regular files ([docs](https://developers.google.com/drive/api/reference/rest/v3/files/get))

**Content export behavior:**
- **Google Docs** → exported as `text/plain`
- **Google Sheets** → exported as `text/csv`
- **Google Slides** → exported as `text/plain`
- **Text files** (text/\*, application/json, etc.) → downloaded directly
- **Binary files** (images, PDFs, etc.) → content skipped, `content_skipped_reason` field explains why

---

### `google.upload_drive_file`

Creates and uploads a text file to Google Drive.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | File name |
| `content` | string | Yes | — | File content (text, max 4 MB) |
| `mime_type` | string | No | `text/plain` | MIME type of the file |
| `folder_id` | string | No | — | Parent folder ID |

**Response:**

```json
{
  "id": "1newFileId123",
  "name": "report.txt",
  "web_view_link": "https://drive.google.com/file/d/1newFileId123/view"
}
```

**Drive API:** `POST /upload/drive/v3/files?uploadType=multipart` ([docs](https://developers.google.com/drive/api/guides/manage-uploads#multipart))

Uses multipart upload with JSON metadata in the first part and file content in the second. Content is capped at 4 MB to prevent oversized payloads.

---

### `google.update_calendar_event`

Updates an existing Google Calendar event using a partial update (PATCH). Only fields that are explicitly provided are changed — omitted fields retain their current values.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `event_id` | string | Yes | — | The ID of the event to update |
| `calendar_id` | string | No | `primary` | Calendar ID containing the event |
| `summary` | string | No | — | New event title |
| `description` | string | No | — | New event description |
| `start_time` | string | No | — | New start time in RFC 3339 format (must be provided together with `end_time`) |
| `end_time` | string | No | — | New end time in RFC 3339 format (must be provided together with `start_time`) |
| `attendees` | string[] | No | — | Replace the attendee list with these email addresses |
| `clear_attendees` | boolean | No | `false` | Remove all attendees (mutually exclusive with `attendees`) |
| `location` | string | No | — | New event location |

At least one update field must be provided. `start_time` and `end_time` must always be provided together.

**Response:**

```json
{
  "id": "event-123",
  "summary": "Updated Title",
  "html_link": "https://calendar.google.com/event?eid=event-123",
  "status": "confirmed",
  "updated": "2024-01-15T10:00:00Z"
}
```

**Calendar API:** `PATCH /calendars/{calendarId}/events/{eventId}` ([docs](https://developers.google.com/calendar/api/v3/reference/events/patch))

**Implementation notes:**
- Uses `PATCH` (partial update) via a `map[string]any` request body so that only provided fields are sent. A struct with `omitempty` tags cannot distinguish "not provided" from "intentionally empty" for the attendees list.
- `clear_attendees: true` sends an explicit empty `attendees: []` array to remove all attendees from the event.
- `calendar_id` is URL-encoded in the API path to safely handle IDs with special characters (e.g., `group@calendar.google.com`).

---

### `google.delete_calendar_event`

Deletes a Google Calendar event by ID.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `event_id` | string | Yes | — | The ID of the event to delete |
| `calendar_id` | string | No | `primary` | Calendar ID containing the event |

**Response:**

```json
{
  "event_id": "event-123",
  "calendar_id": "primary",
  "status": "deleted"
}
```

**Calendar API:** `DELETE /calendars/{calendarId}/events/{eventId}` ([docs](https://developers.google.com/calendar/api/v3/reference/events/delete))

The Google Calendar API returns HTTP 204 No Content on success. The connector synthesizes a response with `status: "deleted"` and the IDs for confirmation.

---

### `google.search_drive`

Searches Google Drive files by name, full-text content, file type, or folder scope. Results are ordered by `modifiedTime desc`.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | No | — | Search text (matched against file name and full-text content) |
| `file_type` | string | No | — | Filter by file type: `document`, `spreadsheet`, `presentation`, `pdf`, `folder`, `image`, `video`, `audio` |
| `folder_id` | string | No | — | Limit results to files within this folder (alphanumeric Drive ID) |
| `max_results` | integer | No | `10` | Maximum number of results (1-100) |

At least one of `query`, `file_type`, or `folder_id` must be provided. Trashed files are excluded automatically.

**Response:**

```json
{
  "files": [
    {
      "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
      "name": "Q4 Report",
      "mime_type": "application/vnd.google-apps.document",
      "modified_time": "2024-01-15T10:00:00.000Z",
      "size": "",
      "web_view_link": "https://docs.google.com/document/d/1Bxi.../edit"
    }
  ],
  "count": 1
}
```

**Drive API:** `GET /drive/v3/files?q=...` ([docs](https://developers.google.com/drive/api/reference/rest/v3/files/list))

**Security notes:**
- The `query` value is escaped (backslashes and single quotes) before insertion into the Drive query string to prevent query injection.
- The `folder_id` is validated with an alphanumeric allowlist to prevent path traversal.
- Image, video, and audio types use a `mimeType contains 'prefix/'` clause (prefix match) since they encompass many subtypes; all other types use an exact `mimeType = '...'` match.
- The sorted `driveFileTypeNames` slice produces deterministic validation error messages regardless of Go map iteration order.

---

### `google.create_drive_folder`

Creates a folder in Google Drive.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Folder name (max 255 characters) |
| `parent_id` | string | No | — | Parent folder ID (alphanumeric Drive ID). Omit to create in the root of My Drive. |

**Response:**

```json
{
  "id": "1folderIdAbc123",
  "name": "Project Files",
  "web_view_link": "https://drive.google.com/drive/folders/1folderIdAbc123"
}
```

**Drive API:** `POST /drive/v3/files` with `mimeType: application/vnd.google-apps.folder` ([docs](https://developers.google.com/drive/api/reference/rest/v3/files/create))

**Validation:**
- `name` is required and may not exceed 255 characters (Google Drive allows up to 32,767, but a practical limit prevents oversized API requests).
- `parent_id`, when provided, must match the alphanumeric Drive ID pattern to prevent injection attacks.

---

### `google.send_email_reply`

Sends a reply to an existing Gmail thread. Fetches the original message headers (From, Subject, Message-ID) and sends a new message in the same thread with correct `In-Reply-To` and `References` headers per RFC 2822.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `thread_id` | string | Yes | The Gmail thread ID to reply to |
| `message_id` | string | Yes | The specific message ID to reply to (used to fetch headers) |
| `body` | string | Yes | Reply body text (plain text) |

**Response:**

```json
{
  "id": "reply-message-id",
  "thread_id": "thread123",
  "subject": "Re: Original Subject",
  "to": "original-sender@example.com"
}
```

**Gmail API:** `GET /gmail/v1/users/me/messages/{id}?format=metadata` (fetch headers) + `POST /gmail/v1/users/me/messages/send` (send reply) ([docs](https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send))

**Security notes:**
- `message_id` is verified to belong to `thread_id` before sending — mismatches return a validation error.
- Gmail-sourced header values (From, Subject, Message-ID) are stripped of CR (`\r`) and LF (`\n`) characters before being written to the RFC 2822 message to prevent MIME header injection. Unlike user-supplied values (which are rejected outright), Gmail-provided values are silently sanitized.
- Both `Message-Id` and `Message-ID` header casings are requested and parsed case-insensitively to handle variations across email providers.
- Subject is automatically prefixed with `Re: ` if not already present (case-insensitive check).

---

### `google.delete_drive_file`

Moves a file to trash in Google Drive (soft delete — not permanent).

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file_id` | string | Yes | The ID of the file to move to trash |

**Response:**

```json
{
  "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "name": "old-report.txt",
  "trashed": true
}
```

**Drive API:** `PATCH /drive/v3/files/{id}` with `{trashed: true}` ([docs](https://developers.google.com/drive/api/reference/rest/v3/files/update))

**Security notes:**
- This is a **soft delete only** — files are moved to the Google Drive trash and can be recovered by the user. The permanent delete endpoint is intentionally not exposed.
- File IDs are validated with an allowlist pattern (alphanumeric, hyphens, underscores) to prevent query injection and path traversal.

---

### `google.archive_email`

Archives a Gmail thread by removing the INBOX label from all messages in the thread. Archived emails remain accessible via search and All Mail — this matches Gmail's built-in Archive button behavior.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `thread_id` | string | Yes | The Gmail thread ID to archive (obtained from `list_emails` `thread_id` field) |

**Response:**

```json
{
  "thread_id": "18abc123def"
}
```

**Gmail API:** `POST /gmail/v1/users/me/threads/{id}/modify` ([docs](https://developers.google.com/gmail/api/reference/rest/v1/users.threads/modify))

**Implementation notes:**
- Uses `threads.modify` (not `messages.modify`) to atomically remove `INBOX` from all messages in the thread, matching Gmail's native Archive behavior.
- Thread IDs are URL-encoded via `url.PathEscape` to prevent path traversal.

---

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
| Create presentations | `create_presentation` | Nothing — agent controls title |
| View presentations | `get_presentation` | Nothing — agent controls presentation ID |
| Add slides to presentations | `add_slide` | Nothing — agent controls all parameters |
| Read from specific spreadsheet | `sheets_read_range` | `spreadsheet_id` locked; agent chooses range |
| Write to specific spreadsheet | `sheets_write_range` | `spreadsheet_id` locked; agent chooses range and values |
| Append to specific spreadsheet | `sheets_append_rows` | `spreadsheet_id` locked; agent chooses range and values |
| Read from any spreadsheet | `sheets_read_range` | Nothing — agent controls all parameters |
| List worksheets in any spreadsheet | `sheets_list_sheets` | Nothing — agent controls spreadsheet |
| Create documents | `create_document` | Nothing — agent controls title and content |
| Create empty documents | `create_document` | `content` omitted — title only |
| Read any document | `get_document` | Nothing — agent can read any doc by ID |
| Edit any document | `update_document` | Nothing — agent controls all parameters |
| Search documents | `list_documents` | Nothing — agent controls query and count |
| Send chat messages | `send_chat_message` | Nothing — agent controls space and text |
| Send message to specific space | `send_chat_message` | `space_name` locked to a placeholder; admin sets the real space |
| List chat spaces | `list_chat_spaces` | Nothing — agent controls page size and filter |
| Create meetings with Meet link | `create_meeting` | Nothing — agent controls all parameters |
| Create personal meetings | `create_meeting` | `calendar_id` locked to `primary`, no attendees |
| Browse Drive files | `list_drive_files` | Nothing — agent controls query, folder, and sort |
| Read Drive files | `get_drive_file` | Nothing — agent can read metadata and content |
| View Drive file metadata | `get_drive_file` | `include_content` locked to `false` (metadata only) |
| Upload files to Drive | `upload_drive_file` | Nothing — agent controls name, content, and destination |
| Upload files to specific folder | `upload_drive_file` | `folder_id` locked to a specific folder |
| Trash Drive files | `delete_drive_file` | Nothing — agent can trash any file |
| Update calendar events | `update_calendar_event` | Nothing — agent can update summary, time, attendees, location |
| Reschedule calendar events | `update_calendar_event` | `summary`, `description`, `attendees`, `location` omitted — time changes only |
| Delete calendar events | `delete_calendar_event` | Nothing — agent can delete events from any calendar |
| Search Drive files | `search_drive` | Nothing — agent controls query, type, folder |
| Search Drive within folder | `search_drive` | `folder_id` locked to a specific folder |
| Create Drive folders | `create_drive_folder` | Nothing — agent controls name and parent |
| Read any email | `read_email` | Nothing — agent controls message ID |
| Reply to emails | `send_email_reply` | Nothing — agent controls thread, message, and body |
| Archive emails | `archive_email` | Nothing — agent can archive any thread |

## Adding a New Action

Each action lives in its own file. To add one (e.g., `google.delete_calendar_event`):

1. Create `connectors/google/delete_calendar_event.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doJSON(ctx, creds, method, url, reqBody, &resp)` for JSON API calls — it handles marshaling, Bearer auth, rate limiting, response size limits, and timeout detection. For non-JSON responses (e.g., file downloads), use `a.conn.doRawGet()`.
3. Use `checkResponse()` (called automatically by `doJSON` and `doRawGet`) to map HTTP errors to typed connector errors. Use `wrapHTTPError()` when making custom HTTP requests (e.g., multipart upload).
4. Return `connectors.JSONResult(respBody)` to wrap the response into an `ActionResult`.
5. Validate user-provided IDs (file IDs, folder IDs) with `isValidDriveID()` to prevent injection attacks.
6. Register the action in `Actions()` inside `google.go`.
7. Add the action to the `Manifest()` return value inside `manifest.go` with a `ParametersSchema`.
8. Add tests in `delete_calendar_event_test.go` using `httptest.NewServer` and the appropriate per-service helper: `newCalendarForTest()`, `newGmailForTest()`, `newDriveForTest()`, `newForTestDocs()`, `newForTestWithSlides()`, `newForTestWithChat()`, or `newForTest()` for mixed Gmail+Calendar tests. Use `validCreds()` from `helpers_test.go` for credentials and `connectors.IsValidationError()` / `connectors.IsExternalError()` for error type assertions.

## File Structure

```
connectors/google/
├── google.go                       # GoogleConnector struct, New(), Actions(), doJSON(), doRawGet(), wrapHTTPError(), ValidateCredentials()
├── manifest.go                     # Manifest() — 28 action schemas and 40+ templates
├── docs_types.go                   # Shared Docs API types (batchUpdate request) and helpers (documentEditURL)
├── email_helpers.go                # buildGmailRaw() — shared RFC 2822 message builder used by send_email and send_email_reply
├── send_email.go                   # google.send_email action
├── list_emails.go                  # google.list_emails action
├── read_email.go                   # google.read_email action (full body, headers, attachment metadata, MIME tree walking)
├── send_email_reply.go             # google.send_email_reply action (fetches headers, strips injection chars, sends reply)
├── archive_email.go               # google.archive_email action (thread-level archive via threads.modify)
├── create_calendar_event.go        # google.create_calendar_event action
├── list_calendar_events.go         # google.list_calendar_events action
├── update_calendar_event.go        # google.update_calendar_event action (PATCH with map[string]any for explicit empty arrays)
├── delete_calendar_event.go        # google.delete_calendar_event action
├── create_presentation.go          # google.create_presentation action
├── get_presentation.go             # google.get_presentation action
├── add_slide.go                    # google.add_slide action (via batchUpdate)
├── slides_helpers.go               # Shared helpers for Slides actions (presentationURL)
├── sheets_read.go                  # google.sheets_read_range action
├── sheets_write.go                 # google.sheets_write_range action
├── sheets_append.go                # google.sheets_append_rows action
├── sheets_list.go                  # google.sheets_list_sheets action
├── sheets_helpers.go               # Shared validation helpers for Sheets actions
├── create_document.go              # google.create_document action
├── get_document.go                 # google.get_document action
├── update_document.go              # google.update_document action
├── list_documents.go               # google.list_documents action
├── send_chat_message.go            # google.send_chat_message action
├── list_chat_spaces.go             # google.list_chat_spaces action
├── create_meeting.go               # google.create_meeting action (Calendar + Meet)
├── calendar_helpers.go             # Shared calendar validation (time range, attendees)
├── list_drive_files.go             # google.list_drive_files action + shared isValidDriveID()
├── get_drive_file.go               # google.get_drive_file action (metadata + content export)
├── upload_drive_file.go            # google.upload_drive_file action (multipart upload)
├── delete_drive_file.go            # google.delete_drive_file action (soft delete via trash)
├── search_drive.go                 # google.search_drive action (name/fullText/type/folder search)
├── create_drive_folder.go          # google.create_drive_folder action
├── google_test.go                  # Connector-level tests (ID, Actions, Manifest, ValidateCredentials)
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_email_test.go              # Send email action tests (including MIME injection, base64 encoding)
├── list_emails_test.go             # List emails action tests
├── read_email_test.go              # Read email tests (MIME parsing, attachments, RFC 5987, depth limit, edge cases)
├── send_email_reply_test.go        # Send email reply tests (thread validation, header injection, Re: prefix)
├── create_calendar_event_test.go   # Create event tests (including time validation, URL encoding)
├── list_calendar_events_test.go    # List events action tests
├── update_calendar_event_test.go   # Update event tests (partial update, clear_attendees, conflict validation)
├── delete_calendar_event_test.go   # Delete event tests
├── create_presentation_test.go     # Create presentation tests
├── get_presentation_test.go        # Get presentation tests (including URL encoding)
├── add_slide_test.go               # Add slide tests (layout validation, insertion index)
├── sheets_read_test.go             # Sheets read range tests
├── sheets_write_test.go            # Sheets write range tests
├── sheets_append_test.go           # Sheets append rows tests
├── sheets_list_test.go             # Sheets list worksheets tests
├── sheets_helpers_test.go          # Sheets validation helpers tests
├── create_document_test.go         # Create document tests (including partial failure handling)
├── get_document_test.go            # Get document tests (including plain text extraction)
├── update_document_test.go         # Update document tests (append and insert-at-index)
├── list_documents_test.go          # List documents tests (including query escaping)
├── send_chat_message_test.go       # Send chat message tests (including path traversal validation)
├── list_chat_spaces_test.go        # List chat spaces tests (including page size clamping)
├── create_meeting_test.go          # Create meeting tests (including Meet link extraction)
├── list_drive_files_test.go        # List Drive files tests (including query injection prevention)
├── get_drive_file_test.go          # Get Drive file tests (metadata, content export, binary skip)
├── upload_drive_file_test.go       # Upload tests (multipart, size limit, folder targeting)
├── delete_drive_file_test.go       # Delete tests (soft delete, ID validation, rate limiting)
├── search_drive_test.go            # Search Drive tests (query escaping, type filter, deterministic errors)
├── create_drive_folder_test.go     # Create folder tests (name validation, parent ID)
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Google APIs — no real API calls are made. Tests pass the Go race detector (`-race` flag).

```bash
go test ./connectors/google/... -v
go test ./connectors/google/... -race  # verify no race conditions
```
