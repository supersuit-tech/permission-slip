# Google Connector

The Google connector integrates Permission Slip with [Gmail](https://developers.google.com/gmail/api), [Google Calendar](https://developers.google.com/calendar/api), and [Google Slides](https://developers.google.com/slides/api) APIs. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Google SDK.

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
| `presentations` | `google.create_presentation`, `google.get_presentation`, `google.add_slide` |

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

## Adding a New Action

Each action lives in its own file. To add one (e.g., `google.delete_calendar_event`):

1. Create `connectors/google/delete_calendar_event.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doJSON(ctx, creds, method, url, reqBody, &resp)` for the HTTP lifecycle — it handles JSON marshaling, Bearer auth, rate limiting, response size limits, and timeout detection.
3. Use `checkResponse()` (called automatically by `doJSON`) to map HTTP errors to typed connector errors.
4. Return `connectors.JSONResult(respBody)` to wrap the response into an `ActionResult`.
5. Register the action in `Actions()` inside `google.go`.
6. Add the action to the `Manifest()` return value inside `google.go` with a `ParametersSchema`.
7. Add tests in `delete_calendar_event_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/google/
├── google.go                       # GoogleConnector struct, New(), Manifest(), doJSON(), ValidateCredentials()
├── send_email.go                   # google.send_email action
├── list_emails.go                  # google.list_emails action
├── create_calendar_event.go        # google.create_calendar_event action
├── list_calendar_events.go         # google.list_calendar_events action
├── create_presentation.go          # google.create_presentation action
├── get_presentation.go             # google.get_presentation action
├── add_slide.go                    # google.add_slide action (via batchUpdate)
├── slides_helpers.go               # Shared helpers for Slides actions (presentationURL)
├── google_test.go                  # Connector-level tests (ID, Actions, Manifest, ValidateCredentials)
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_email_test.go              # Send email action tests (including MIME injection, base64 encoding)
├── list_emails_test.go             # List emails action tests
├── create_calendar_event_test.go   # Create event tests (including time validation, URL encoding)
├── list_calendar_events_test.go    # List events action tests
├── create_presentation_test.go     # Create presentation tests
├── get_presentation_test.go        # Get presentation tests (including URL encoding)
├── add_slide_test.go               # Add slide tests (layout validation, insertion index)
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Google APIs — no real API calls are made. Tests pass the Go race detector (`-race` flag).

```bash
go test ./connectors/google/... -v
go test ./connectors/google/... -race  # verify no race conditions
```
