# Slack Connector

The Slack connector integrates Permission Slip with the [Slack Web API](https://api.slack.com/web). It uses plain `net/http` — no third-party Slack SDK.

## Connector ID

`slack`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `bot_token` | Yes | A Slack OAuth bot token (`xoxb-...`) with appropriate scopes for the actions being executed. |

The credential `auth_type` in the database is `custom`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time. The connector validates that the token starts with the `xoxb-` prefix.

## Actions

### `slack.send_message`

Sends a message to a Slack channel.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel name (e.g., `#general`) or ID (e.g., `C01234567`) |
| `message` | string | Yes | Message text (supports Slack mrkdwn formatting) |

**Response:**

```json
{
  "ts": "1234567890.123456",
  "channel": "C01234567"
}
```

**Slack API:** `POST /chat.postMessage` ([docs](https://api.slack.com/methods/chat.postMessage))

**Required bot token scopes:** `chat:write`

---

### `slack.create_channel`

Creates a new Slack channel.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Channel name (lowercase, no spaces, max 80 chars) |
| `is_private` | boolean | No | `false` | Create as private channel |

**Response:**

```json
{
  "id": "C09876543",
  "name": "new-channel"
}
```

**Slack API:** `POST /conversations.create` ([docs](https://api.slack.com/methods/conversations.create))

**Required bot token scopes:** `channels:manage` (public), `groups:write` (private)

---

### `slack.list_channels`

Lists Slack channels visible to the bot.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `types` | string | No | `public_channel` | Comma-separated channel types: `public_channel`, `private_channel`, `mpim`, `im` |
| `limit` | integer | No | `100` | Max channels to return (1–1000) |
| `cursor` | string | No | — | Pagination cursor from a previous response |

**Response:**

```json
{
  "channels": [
    {
      "id": "C01234567",
      "name": "general",
      "is_private": false,
      "topic": "General discussion",
      "purpose": "Company-wide announcements",
      "num_members": 42
    }
  ],
  "next_cursor": "dGVhbTpDMDI="
}
```

**Slack API:** `POST /conversations.list` ([docs](https://api.slack.com/methods/conversations.list))

**Required bot token scopes:** `channels:read` (public), `groups:read` (private)

---

### `slack.read_channel_messages`

Reads recent messages from a Slack channel.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `channel` | string | Yes | — | Channel ID (e.g., `C01234567`) — must be an ID, not a name |
| `limit` | integer | No | `20` | Max messages to return (1–1000) |
| `oldest` | string | No | — | Only messages after this Unix timestamp |
| `latest` | string | No | — | Only messages before this Unix timestamp |
| `cursor` | string | No | — | Pagination cursor from a previous response |

**Response:**

```json
{
  "messages": [
    {
      "user": "U001",
      "text": "Hello everyone!",
      "ts": "1678900000.000100",
      "thread_ts": "",
      "reply_count": 0
    }
  ],
  "has_more": true,
  "next_cursor": "bmV4dA=="
}
```

**Slack API:** `POST /conversations.history` ([docs](https://api.slack.com/methods/conversations.history))

**Required bot token scopes:** `channels:history` (public), `groups:history` (private)

---

### `slack.read_thread`

Reads replies in a Slack message thread.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `channel` | string | Yes | — | Channel ID containing the thread (e.g., `C01234567`) |
| `thread_ts` | string | Yes | — | Timestamp of the parent message (e.g., `1234567890.123456`) |
| `limit` | integer | No | `50` | Max replies to return (1–1000) |
| `cursor` | string | No | — | Pagination cursor from a previous response |

**Response:**

```json
{
  "messages": [
    {
      "user": "U002",
      "text": "Check out this thread",
      "ts": "1678900001.000200",
      "thread_ts": "1678900001.000200",
      "reply_count": 2
    },
    {
      "user": "U003",
      "text": "Great idea!",
      "ts": "1678900002.000300",
      "thread_ts": "1678900001.000200"
    }
  ],
  "has_more": false
}
```

**Slack API:** `POST /conversations.replies` ([docs](https://api.slack.com/methods/conversations.replies))

**Required bot token scopes:** `channels:history` (public), `groups:history` (private)

---

### `slack.schedule_message`

Schedules a message for future delivery to a Slack channel.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel name (e.g., `#general`) or ID (e.g., `C01234567`) |
| `message` | string | Yes | Message text (supports Slack mrkdwn formatting) |
| `post_at` | integer | Yes | Unix timestamp for when the message should be sent (must be in the future) |

**Response:**

```json
{
  "scheduled_message_id": "Q1234ABCD",
  "post_at": 1893456000,
  "channel": "C01234567"
}
```

**Slack API:** `POST /chat.scheduleMessage` ([docs](https://api.slack.com/methods/chat.scheduleMessage))

**Required bot token scopes:** `chat:write`

---

### `slack.set_topic`

Updates a Slack channel's topic.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel ID (e.g., `C01234567`) |
| `topic` | string | Yes | New channel topic (max 250 characters) |

**Response:**

```json
{
  "channel": "C01234567",
  "topic": "New topic text"
}
```

**Slack API:** `POST /conversations.setTopic` ([docs](https://api.slack.com/methods/conversations.setTopic))

**Required bot token scopes:** `channels:manage` (public), `groups:write` (private)

---

### `slack.invite_to_channel`

Invites one or more users to a Slack channel.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel ID (e.g., `C01234567`) |
| `users` | string | Yes | Comma-separated list of user IDs to invite (e.g., `U01234567,U09876543`) |

**Response:**

```json
{
  "channel": "C01234567",
  "channel_name": "general"
}
```

**Slack API:** `POST /conversations.invite` ([docs](https://api.slack.com/methods/conversations.invite))

**Required bot token scopes:** `channels:manage` (public), `groups:write` (private)

---

### `slack.upload_file`

Uploads a file to a Slack channel using Slack's v2 upload flow (3-step: get URL, upload content, complete).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `channel` | string | Yes | — | Channel ID (e.g., `C01234567`) |
| `filename` | string | Yes | — | Name of the file (e.g., `report.csv`) |
| `content` | string | Yes | — | File content as text (max 50 MB) |
| `title` | string | No | filename | Display title for the file |

**Response:**

```json
{
  "file_id": "F1234ABCD",
  "channel": "C01234567"
}
```

**Slack API:** `POST /files.getUploadURLExternal` + upload + `POST /files.completeUploadExternal` ([docs](https://api.slack.com/methods/files.getUploadURLExternal))

**Required bot token scopes:** `files:write`

**Security:** The upload URL returned by Slack is validated to ensure it points to a Slack-owned domain (`*.slack.com` or `*.slack-files.com`) over HTTPS, preventing SSRF.

---

### `slack.add_reaction`

Adds an emoji reaction to a Slack message.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel ID containing the message (e.g., `C01234567`) |
| `timestamp` | string | Yes | Timestamp of the message to react to (e.g., `1234567890.123456`) |
| `name` | string | Yes | Emoji name without colons (e.g., `thumbsup`, `white_check_mark`) — colons are stripped automatically |

**Response:**

```json
{
  "channel": "C01234567",
  "timestamp": "1234567890.123456",
  "reaction": "thumbsup"
}
```

**Slack API:** `POST /reactions.add` ([docs](https://api.slack.com/methods/reactions.add))

**Required bot token scopes:** `reactions:write`

---

### Channel ID Validation

The `channel` parameter on `read_channel_messages` and `read_thread` must be a Slack channel ID — not a channel name. Valid IDs start with `C` (public channels), `G` (private channels / group DMs), or `D` (direct messages). Passing a name like `#general` or `general` returns a `ValidationError` with a helpful hint before hitting the Slack API.

### Pagination Limits

The `limit` parameter on all list/read actions must be between 1 and 1000. Values outside this range are rejected with a `ValidationError` before making the API call.

## Error Handling

The Slack API returns HTTP 200 for most errors, with success/failure indicated by the `ok` field in the JSON response. The connector maps these to typed connector errors:

| Slack Error | Connector Error | HTTP Response |
|-------------|-----------------|---------------|
| `not_authed`, `invalid_auth`, `token_revoked`, `token_expired`, `account_inactive` | `AuthError` | 502 Bad Gateway |
| `missing_scope` | `AuthError` (with link to Slack app settings) | 502 Bad Gateway |
| `ratelimited` (or HTTP 429) | `RateLimitError` | 429 Too Many Requests |
| `channel_not_found`, `not_in_channel`, `is_archived` | `ExternalError` (user-friendly message) | 502 Bad Gateway |
| `already_reacted`, `already_in_channel`, `user_not_found`, etc. | `ExternalError` (user-friendly message) | 502 Bad Gateway |
| All other Slack errors | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline / cancellation | `TimeoutError` | 504 Gateway Timeout |

Rate limit responses include the `Retry-After` header value so callers know how long to wait (defaults to 30s if missing).

## Adding a New Action

Each action lives in its own file. To add one (e.g., `slack.invite_user`):

1. Create `connectors/slack/invite_user.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `parseAndValidate(req.Parameters, &params)` to unmarshal and validate params.
3. Use `a.conn.doPost(ctx, "method.name", creds, body, &resp)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, rate limiting, and timeout detection.
4. Check `resp.OK` and call `mapSlackError(resp.Error)` for Slack-level errors.
5. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
6. Register the action in `Actions()` inside `slack.go`.
7. Add the action to the `Manifest()` return value inside `manifest.go` — include a `ParametersSchema` (see below).
8. Add tests in `invite_user_test.go` using `httptest.NewServer` and `newForTest()`.

The `doPost` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, rate limiting, timeouts) are handled once.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable:

```go
{
    ActionType:  "slack.invite_user",
    Name:        "Invite User",
    Description: "Invite a user to a channel",
    RiskLevel:   "low",
    ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
        "type": "object",
        "required": ["channel", "user_id"],
        "properties": {
            "channel": {
                "type": "string",
                "description": "Channel ID (e.g. C01234567)"
            },
            "user_id": {
                "type": "string",
                "description": "Slack user ID to invite"
            }
        }
    }`)),
}
```

The schema supports standard JSON Schema properties: `type`, `description`, `required`, `enum`, and `default`. The frontend reads these to render rich parameter displays in the approval review modal.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `SlackConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value in `manifest.go` with a `ParametersSchema`.

## File Structure

```
connectors/slack/
├── slack.go                        # SlackConnector, New(), Actions(), doPost(), shared validators
├── manifest.go                     # Manifest() — action schemas, templates, credentials
├── messages.go                     # Shared types: slackMessage, messageSummary, messagesResponse
├── send_message.go                 # slack.send_message action
├── create_channel.go               # slack.create_channel action
├── list_channels.go                # slack.list_channels action
├── read_channel_messages.go        # slack.read_channel_messages action
├── read_thread.go                  # slack.read_thread action
├── schedule_message.go             # slack.schedule_message action
├── set_topic.go                    # slack.set_topic action
├── invite_to_channel.go            # slack.invite_to_channel action
├── upload_file.go                  # slack.upload_file action (v2 upload flow)
├── add_reaction.go                 # slack.add_reaction action
├── slack_test.go                   # Connector-level tests
├── helpers_test.go                 # Shared test helpers (validCreds)
├── send_message_test.go            # Send message action tests
├── create_channel_test.go          # Create channel action tests
├── list_channels_test.go           # List channels action tests
├── read_channel_messages_test.go   # Read channel messages action tests
├── read_thread_test.go             # Read thread action tests
├── schedule_message_test.go        # Schedule message action tests
├── set_topic_test.go               # Set topic action tests
├── invite_to_channel_test.go       # Invite to channel action tests
├── upload_file_test.go             # Upload file action tests
├── add_reaction_test.go            # Add reaction action tests
└── README.md                       # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Slack API — no real API calls are made.

```bash
go test ./connectors/slack/... -v
```
