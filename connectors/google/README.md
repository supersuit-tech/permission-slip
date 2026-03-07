# Google Connector

The Google connector integrates Permission Slip with [Gmail](https://developers.google.com/gmail/api), [Google Calendar](https://developers.google.com/calendar/api), [Google Docs](https://developers.google.com/docs/api), and [Google Drive](https://developers.google.com/drive/api) APIs. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Google SDK.

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
| `documents` | `google.create_document`, `google.get_document`, `google.update_document` |
| `drive.readonly` | `google.list_documents` |

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

### `google.create_document`

Creates a new Google Doc with a title and optional initial body content.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `title` | string | Yes | Title of the new Google Doc |
| `body` | string | No | Optional initial body content (plain text) |

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
| `text` | string | Yes | — | Text to insert into the document |
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
| Create documents | `create_document` | Nothing — agent controls title and body |
| Create empty documents | `create_document` | `body` omitted — title only |
| Read any document | `get_document` | Nothing — agent can read any doc by ID |
| Edit any document | `update_document` | Nothing — agent controls all parameters |
| Search documents | `list_documents` | Nothing — agent controls query and count |

## Adding a New Action

Each action lives in its own file. To add one (e.g., `google.delete_calendar_event`):

1. Create `connectors/google/delete_calendar_event.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doJSON(ctx, creds, method, url, reqBody, &resp)` for the HTTP lifecycle — it handles JSON marshaling, Bearer auth, rate limiting, response size limits, and timeout detection.
3. Use `checkResponse()` (called automatically by `doJSON`) to map HTTP errors to typed connector errors.
4. Return `connectors.JSONResult(respBody)` to wrap the response into an `ActionResult`.
5. Register the action in `Actions()` inside `google.go`.
6. Add the action to the `Manifest()` return value inside `google.go` with a `ParametersSchema`.
7. Add tests in `delete_calendar_event_test.go` using `httptest.NewServer` and `newForTest()` (for Gmail/Calendar) or `newForTestDocs()` (for Docs/Drive).

## File Structure

```
connectors/google/
├── google.go                       # GoogleConnector struct, New(), Manifest(), doJSON(), ValidateCredentials()
├── docs_types.go                   # Shared Docs API types (batchUpdate request) and helpers (documentEditURL)
├── send_email.go                   # google.send_email action
├── list_emails.go                  # google.list_emails action
├── create_calendar_event.go        # google.create_calendar_event action
├── list_calendar_events.go         # google.list_calendar_events action
├── create_document.go              # google.create_document action
├── get_document.go                 # google.get_document action
├── update_document.go              # google.update_document action
├── list_documents.go               # google.list_documents action
├── google_test.go                  # Connector-level tests (ID, Actions, Manifest, ValidateCredentials)
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_email_test.go              # Send email action tests (including MIME injection, base64 encoding)
├── list_emails_test.go             # List emails action tests
├── create_calendar_event_test.go   # Create event tests (including time validation, URL encoding)
├── list_calendar_events_test.go    # List events action tests
├── create_document_test.go         # Create document tests (including partial failure handling)
├── get_document_test.go            # Get document tests (including plain text extraction)
├── update_document_test.go         # Update document tests (append and insert-at-index)
├── list_documents_test.go          # List documents tests (including query escaping)
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Google APIs — no real API calls are made. Tests pass the Go race detector (`-race` flag).

```bash
go test ./connectors/google/... -v
go test ./connectors/google/... -race  # verify no race conditions
```
