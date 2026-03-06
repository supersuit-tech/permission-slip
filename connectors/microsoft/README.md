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

**Required OAuth scopes:** `Mail.Send`, `Mail.Read`, `Calendars.ReadWrite`, `Files.ReadWrite`, `Team.ReadBasic.All`, `Channel.ReadBasic.All`, `ChannelMessage.Send`, `ChannelMessage.Read.All`

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

### `microsoft.list_teams`

Lists the Microsoft Teams the authenticated user is a member of.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `top` | integer | No | `20` | Number of teams to return (1–50) |

**Response:**

```json
[
  {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "Engineering",
    "description": "Engineering team workspace",
    "visibility": "private"
  }
]
```

**Graph API:** `GET /me/joinedTeams` ([docs](https://learn.microsoft.com/en-us/graph/api/user-list-joinedteams))

---

### `microsoft.list_channels`

Lists channels in a Microsoft Teams team.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `team_id` | string | Yes | — | The ID of the team to list channels for |

**Response:**

```json
[
  {
    "id": "19:abc123@thread.tacv2",
    "name": "General",
    "description": "General discussion",
    "membership_type": "standard"
  }
]
```

**Graph API:** `GET /teams/{team-id}/channels` ([docs](https://learn.microsoft.com/en-us/graph/api/channel-list))

---

### `microsoft.send_channel_message`

Sends a message to a Microsoft Teams channel. Supports plain text and HTML (auto-detected). Optionally replies to an existing message thread.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `team_id` | string | Yes | — | The ID of the team |
| `channel_id` | string | Yes | — | The ID of the channel to post to |
| `message` | string | Yes | — | Message content (HTML or plain text — auto-detected) |
| `reply_to_message_id` | string | No | — | Message ID to reply to (creates a threaded reply) |

**Response:**

```json
{
  "status": "sent",
  "message_id": "1616990032035",
  "created_at": "2024-01-15T09:00:00Z"
}
```

**Graph API:** `POST /teams/{team-id}/channels/{channel-id}/messages` ([docs](https://learn.microsoft.com/en-us/graph/api/channel-post-messages))
When `reply_to_message_id` is provided: `POST /teams/{team-id}/channels/{channel-id}/messages/{message-id}/replies` ([docs](https://learn.microsoft.com/en-us/graph/api/chatmessage-post-replies))

---

### `microsoft.list_channel_messages`

Lists recent messages from a Microsoft Teams channel.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `team_id` | string | Yes | — | The ID of the team |
| `channel_id` | string | Yes | — | The ID of the channel to read messages from |
| `top` | integer | No | `20` | Number of messages to return (1–50) |

**Response:**

```json
[
  {
    "id": "1616990032035",
    "created_at": "2024-01-15T09:00:00Z",
    "from": "Jane Smith",
    "content": "Hello team!"
  }
]
```

**Graph API:** `GET /teams/{team-id}/channels/{channel-id}/messages` ([docs](https://learn.microsoft.com/en-us/graph/api/channel-list-messages))

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

### `microsoft.create_presentation`

Creates a new empty PowerPoint (.pptx) file in the user's OneDrive. The file is created using a minimal embedded PPTX template (~1.2 KB) uploaded via the OneDrive file upload endpoint.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `filename` | string | Yes | — | Name for the presentation (`.pptx` extension added if missing) |
| `folder_path` | string | No | `"/"` (root) | OneDrive folder path (e.g., `Documents/Presentations`) |

**Response:**

```json
{
  "item_id": "01NBRZAA...",
  "name": "Quarterly Report.pptx",
  "web_url": "https://onedrive.live.com/edit.aspx?id=...",
  "folder_path": "/Documents/Presentations"
}
```

**Graph API:** `PUT /me/drive/root:/{path}/{filename}.pptx:/content` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-put-content))

---

### `microsoft.list_presentations`

Searches for PowerPoint files (.pptx) in the user's OneDrive, optionally scoped to a folder.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder_path` | string | No | — | OneDrive folder path to search in (searches all files if omitted) |
| `top` | integer | No | `10` | Number of presentations to return (1–50) |

**Response:**

```json
[
  {
    "item_id": "01NBRZAA...",
    "name": "Q4 Review.pptx",
    "web_url": "https://onedrive.live.com/edit.aspx?id=...",
    "size": 1048576,
    "last_modified": "2024-03-15T14:30:00Z"
  }
]
```

**Graph API:** `GET /me/drive/root/search(q='.pptx')` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-search))

---

### `microsoft.get_presentation`

Gets metadata about a specific PowerPoint file by its OneDrive item ID.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `item_id` | string | Yes | — | OneDrive item ID of the presentation |

**Response:**

```json
{
  "item_id": "01NBRZAA...",
  "name": "Q4 Review.pptx",
  "web_url": "https://onedrive.live.com/edit.aspx?id=...",
  "size": 2048576,
  "last_modified_by": "Jane Smith",
  "last_modified": "2024-03-15T14:30:00Z"
}
```

**Graph API:** `GET /me/drive/items/{itemId}` ([docs](https://learn.microsoft.com/en-us/graph/api/driveitem-get))

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
2. Use `a.conn.doRequest(ctx, method, path, creds, body, &resp)` for JSON API calls — it handles JSON marshaling, auth headers, rate limiting, error mapping, and timeout detection. For binary file uploads, use `a.conn.doPutFileRequest(ctx, path, creds, fileBytes, &resp)`.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `microsoft.go`.
5. Add the action to the `Manifest()` return value inside `microsoft.go` — include a `ParametersSchema` and a template.
6. Add tests in `list_contacts_test.go` using `httptest.NewServer` and `newForTest()`.

Both `doRequest` and `doPutFileRequest` delegate to `executeAndHandleResponse` for shared response handling (rate limiting, error mapping, body parsing). Each action file only contains what's unique: parameter parsing, validation, request construction, and response shape.

**Security notes for user-supplied path segments:**
- Use `validateGraphID()` for opaque IDs interpolated into URL paths (rejects `/`, `\`, `?`, `#`, `%`, `..`)
- Use `validateFolderPath()` for folder paths (allows `/` for directory separators, rejects `..`, `?`, `#`, `%`, `\`)
- URL-encode path segments with `url.PathEscape()` or `escapePathSegments()` as defense-in-depth

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `MicrosoftConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/microsoft/
├── microsoft.go                    # MicrosoftConnector, Manifest(), doRequest(), doPutFileRequest(), executeAndHandleResponse()
├── types.go                        # Shared Microsoft Graph API types (graphEmailBody, graphMailAddress, etc.)
├── response.go                     # Graph API error response → typed connector error mapping
├── validation.go                   # Shared validation helpers (validateEmail, validateGraphID, escapePathSegments, etc.)
├── pptx_template.go                # Minimal embedded .pptx template for create_presentation
├── send_email.go                   # microsoft.send_email action
├── list_emails.go                  # microsoft.list_emails action
├── list_teams.go                   # microsoft.list_teams action
├── list_channels.go                # microsoft.list_channels action
├── send_channel_message.go         # microsoft.send_channel_message action
├── list_channel_messages.go        # microsoft.list_channel_messages action
├── create_calendar_event.go        # microsoft.create_calendar_event action
├── list_calendar_events.go         # microsoft.list_calendar_events action
├── create_presentation.go          # microsoft.create_presentation action
├── list_presentations.go           # microsoft.list_presentations action
├── get_presentation.go             # microsoft.get_presentation action
├── microsoft_test.go               # Connector-level tests
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_email_test.go              # Send email action tests
├── list_emails_test.go             # List emails action tests
├── list_teams_test.go              # List teams action tests
├── list_channels_test.go           # List channels action tests
├── send_channel_message_test.go    # Send channel message action tests
├── list_channel_messages_test.go   # List channel messages action tests
├── create_calendar_event_test.go   # Create calendar event action tests
├── list_calendar_events_test.go    # List calendar events action tests
├── create_presentation_test.go     # Create presentation action tests
├── list_presentations_test.go      # List presentations action tests
├── get_presentation_test.go        # Get presentation action tests
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Microsoft Graph API — no real API calls are made.

```bash
go test ./connectors/microsoft/... -v
```
