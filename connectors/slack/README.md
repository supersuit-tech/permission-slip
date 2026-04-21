# Slack Connector

The Slack connector integrates Permission Slip with the [Slack Web API](https://api.slack.com/web). It uses plain `net/http` — no third-party Slack SDK.

## Connector ID

`slack`

## Credentials

Slack uses **user OAuth only** (Slack user token `xoxp-...`). There is no workspace bot install or manual bot token path.

### OAuth

| Key | Source | Description |
|-----|--------|-------------|
| `access_token` | OAuth connection | User token from Slack OAuth v2 (`authed_user.access_token`), stored as the connection primary secret. |

The credential `auth_type` is `oauth2` with provider `slack`. Users connect via **Settings > Connected Accounts > Connect Slack**. The platform handles token exchange, encrypted storage, and refresh when Slack issues a refresh token.

### Credential resolution

At execution time, the connector uses the OAuth `access_token` (Bearer) for all Web API calls.

### Re-authorization after scope changes

If you previously connected with the old bot+user dual-token flow, disconnect and
connect again (or wait for the connection status `needs_reauth` after deploy). The
platform stores a single user token per connection; bot-only primary tokens are no
longer supported.

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

Lists Slack channels visible to the bot. When listing private channel types (`private_channel`, `mpim`, `im`), results are filtered to channels the executing user belongs to.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `types` | string | No | `public_channel,private_channel,mpim,im` | Comma-separated channel types: `public_channel`, `private_channel`, `mpim`, `im`. Defaults to all types when user email is available; falls back to `public_channel` only when no email is set. |
| `limit` | integer | No | `100` | Max channels to return (1–1000) |
| `cursor` | string | No | — | Pagination cursor from a previous response |
| `exclude_archived` | boolean | No | `true` | Exclude archived channels from results |

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
    },
    {
      "id": "D09876543",
      "user": "U001",
      "is_private": true,
      "num_members": 0
    }
  ],
  "next_cursor": "dGVhbTpDMDI="
}
```

> **Note:** IM channels (DMs) don't have a `name` field. Instead, they include a `user` field with the other participant's Slack user ID.

**Slack API:** `POST /conversations.list` ([docs](https://api.slack.com/methods/conversations.list))

**Required bot token scopes:** `channels:read` (public), `groups:read` (private), `im:read` (DMs), `mpim:read` (group DMs)

---

### `slack.read_channel_messages`

Reads recent messages from a Slack channel. For private channels, DMs, and group DMs, verifies the executing user is a member before allowing access.

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

#### Unread messages (user token)

To answer “do I have unread Slack messages?” or load only unread content, use this flow (no extra OAuth scopes beyond the connector’s user token):

1. Call **`slack.list_unread`** — returns conversations where `unread_count` &gt; 0, with `last_read_ts` and a short `latest_message_preview` when Slack provides `latest`.
2. For a given conversation, call **`slack.read_channel_messages`** with **`oldest` = `last_read_ts`** from that row (Slack accepts the same timestamp format as in `list_unread`). That limits history to messages after the user’s read cursor instead of guessing from “recent” history alone.
3. Optionally **`slack.mark_read`** with **`channel_id`** and an explicit **`ts`** — the Slack message timestamp of the last message you surfaced to the user. **`ts` is required** (there is no default) so the agent cannot clear unreads it never showed.

---

### `slack.list_unread`

Lists conversations where the authorizing user has unread messages. Uses `users.conversations` to enumerate conversations, then `conversations.info` per channel for `last_read`, `unread_count_display`, and `latest`. Requires a Permission Slip profile email that matches the Slack account (same identity model as other private/DM flows).

**Slack API:** `POST /users.conversations`, `GET /conversations.info`

---

### `slack.mark_read`

Sets the read cursor for a conversation (`conversations.mark`). **`channel_id`** and **`ts`** are both required.

**Slack API:** `POST /conversations.mark`

---

### `slack.read_thread`

Reads replies in a Slack message thread. For private channels, DMs, and group DMs, verifies the executing user is a member of the parent channel before allowing access.

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

### `slack.send_dm`

Sends a direct message to a Slack user. Opens (or reuses) a DM channel with the user, then posts the message. Self-DMs are supported — passing your own user ID opens a "note to self" channel.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `user_id` | string | Yes | Slack user ID (e.g., `U01234567`) — must start with `U` or `W` |
| `message` | string | Yes | Message text (supports Slack mrkdwn formatting) |

**Response:**

```json
{
  "ts": "1234567890.123456",
  "channel": "D09876543"
}
```

**Slack API:** `POST /conversations.open` + `POST /chat.postMessage` ([docs](https://api.slack.com/methods/conversations.open))

**Required bot token scopes:** `im:write`, `mpim:write`, `chat:write`

---

### `slack.update_message`

Edits an existing message in a Slack channel. Bots can only edit their own messages.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel ID (e.g., `C01234567`) |
| `ts` | string | Yes | Timestamp of the message to update (e.g., `1234567890.123456`) |
| `message` | string | Yes | New message text (supports Slack mrkdwn formatting) |

**Response:**

```json
{
  "ts": "1234567890.123456",
  "channel": "C01234567"
}
```

**Slack API:** `POST /chat.update` ([docs](https://api.slack.com/methods/chat.update))

**Required bot token scopes:** `chat:write`

---

### `slack.delete_message`

Deletes a message from a Slack channel. Bots can only delete their own messages.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel ID (e.g., `C01234567`) |
| `ts` | string | Yes | Timestamp of the message to delete (e.g., `1234567890.123456`) |

**Response:**

```json
{
  "ts": "1234567890.123456",
  "channel": "C01234567"
}
```

**Slack API:** `POST /chat.delete` ([docs](https://api.slack.com/methods/chat.delete))

**Required bot token scopes:** `chat:write`

---

### `slack.list_users`

Lists workspace users visible to the bot, with cursor-based pagination.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `limit` | integer | No | `100` | Max users to return (1–1000) |
| `cursor` | string | No | — | Pagination cursor from a previous response |

**Response:**

```json
{
  "users": [
    {
      "id": "U001",
      "name": "jdoe",
      "real_name": "John Doe",
      "display_name": "John",
      "email": "john@example.com",
      "is_bot": false,
      "is_admin": false
    }
  ],
  "next_cursor": "dGVhbTpDMDI="
}
```

**Slack API:** `POST /users.list` ([docs](https://api.slack.com/methods/users.list))

**Required bot token scopes:** `users:read` (add `users:read.email` for email addresses)

---

### `slack.search_messages`

Searches messages across Slack channels. Requires the executing user to have a verified Slack identity (email match). Results are post-filtered to only include matches from channels the user belongs to. **Requires a user token** (`xoxp-`) with the granular `search:read.*` scopes (`search:read.public`, `search:read.private`, `search:read.im`, `search:read.mpim`, `search:read.files`) — bot tokens (`xoxb-`) do not support this endpoint.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search query (supports Slack modifiers like `in:#channel`, `from:@user`) |
| `count` | integer | No | `20` | Max results per page (1–100) |
| `page` | integer | No | `1` | Page number for pagination (1-indexed) |
| `sort` | string | No | `score` | Sort order: `score` (relevance) or `timestamp` |

**Response:**

```json
{
  "matches": [
    {
      "channel_id": "C001",
      "channel_name": "engineering",
      "user": "U001",
      "username": "jdoe",
      "text": "Deploying v2.0 now",
      "ts": "1234567890.123456",
      "permalink": "https://team.slack.com/archives/C001/p1234567890123456"
    }
  ],
  "total": 42,
  "page": 1,
  "pages": 3
}
```

**Slack API:** `POST /search.messages` ([docs](https://api.slack.com/methods/search.messages))

**Required scopes:** `search:read.public`, `search:read.private`, `search:read.im`, `search:read.mpim`, `search:read.files` (user token only)

> **Note:** This action will return a `missing_scope` error when invoked with a bot token. To use it, the OAuth flow must persist the user's access token (the `authed_user.access_token` field from Slack's OAuth v2 response).

---

### Access Control (Private Channels, DMs, Group DMs)

Read actions (`read_channel_messages`, `read_thread`, `list_channels`, `search_messages`) enforce per-user access control to prevent users from reading private content they don't belong to. The bot token has workspace-wide scopes, so without this check any Permission Slip user could read any DM or private channel.

**How it works:**

1. The executing user's Permission Slip profile email is resolved from the database.
2. For private channels, DMs (`D`-prefixed), and group DMs (`G`-prefixed), the email is mapped to a Slack user ID via `users.lookupByEmail`.
3. Channel membership is verified via `conversations.members`.
4. Public channels (`C`-prefixed, `is_private: false`) are accessible without verification.

**Requirements:**

- The user must have an email set on their Permission Slip profile.
- The email must match their Slack workspace account.
- The bot token must have the `users:read.email` scope for email-based lookups.

**Behavior by action:**

| Action | Access check |
|--------|-------------|
| `read_channel_messages` | Verifies membership before fetching history |
| `read_thread` | Verifies membership in the parent channel |
| `list_channels` | Filters results to channels the user belongs to (when listing private types) |
| `search_messages` | Requires email; post-filters results by channel membership |

If access is denied, the action returns a `ValidationError` with a message describing the channel type and what the user needs to do (e.g., add an email to their profile).

---

### Channel ID Validation

The `channel` parameter on `read_channel_messages` and `read_thread` must be a Slack channel ID — not a channel name. Valid IDs start with `C` (public channels), `G` (private channels / group DMs), or `D` (direct messages). Passing a name like `#general` or `general` returns a `ValidationError` with a helpful hint before hitting the Slack API.

### User ID Validation

The `user_id` parameter on `send_dm` must be a Slack user ID starting with `U` or `W`. Passing a username or email returns a `ValidationError` with a helpful hint.

### Message Timestamp Validation

The `ts` / `timestamp` parameters on `update_message`, `delete_message`, and `add_reaction` must be a valid Slack timestamp in `<seconds>.<microseconds>` format (e.g., `1234567890.123456`). Non-numeric values or missing dot separators are rejected with a `ValidationError`.

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
| `message_not_found`, `cant_update_message`, `cant_delete_message`, `edit_window_closed` | `ExternalError` (user-friendly message) | 502 Bad Gateway |
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
├── channel_membership.go           # Per-user access control: verifyChannelAccess, hasChannelAccess, lookupSlackUserByEmail
├── messages.go                     # Shared types: slackMessage, messageSummary, messagesResponse
├── send_message.go                 # slack.send_message action
├── create_channel.go               # slack.create_channel action
├── list_channels.go                # slack.list_channels action
├── read_channel_messages.go        # slack.read_channel_messages action
├── conversation_info.go            # conversations.info helper (unread + shared)
├── list_unread.go                  # slack.list_unread action
├── mark_read.go                    # slack.mark_read action
├── read_thread.go                  # slack.read_thread action
├── schedule_message.go             # slack.schedule_message action
├── set_topic.go                    # slack.set_topic action
├── invite_to_channel.go            # slack.invite_to_channel action
├── upload_file.go                  # slack.upload_file action (v2 upload flow)
├── add_reaction.go                 # slack.add_reaction action
├── send_dm.go                      # slack.send_dm action (conversations.open + chat.postMessage)
├── update_message.go               # slack.update_message action
├── delete_message.go               # slack.delete_message action
├── list_users.go                   # slack.list_users action
├── search_messages.go              # slack.search_messages action (requires user token)
├── slack_test.go                   # Connector-level tests + validator tests
├── helpers_test.go                 # Shared test helpers (validCreds)
├── *_test.go                       # Per-action test files (one per action)
└── README.md                       # This file
```

## Required OAuth Scopes (Complete Reference)

When connecting via OAuth, scopes are requested automatically. If you're creating a Slack app manually (bot token flow), add all the scopes below to your app's **Bot Token Scopes** in the [Slack App Dashboard](https://api.slack.com/apps).

**Bot token scopes (`xoxb-`):**

| Scope | Purpose |
|-------|---------|
| `channels:history` | Read messages in public channels |
| `channels:join` | Join public channels |
| `channels:manage` | Create/archive/rename public channels |
| `channels:read` | List public channels |
| `chat:write` | Send messages, update/delete bot messages |
| `files:write` | Upload files |
| `groups:history` | Read messages in private channels |
| `groups:read` | List private channels |
| `im:history` | Read DM messages |
| `im:read` | List DM channels |
| `im:write` | Open DM conversations |
| `mpim:history` | Read group DM messages |
| `mpim:read` | List group DM channels |
| `mpim:write` | Open group DM conversations |
| `reactions:write` | Add emoji reactions |
| `users:read` | List workspace users |
| `users:read.email` | Access user email addresses (needed for identity verification) |

**User token scopes (`xoxp-`)** — only needed for `search_messages`:

| Scope | Purpose |
|-------|---------|
| `search:read.public` | Search public channel messages |
| `search:read.private` | Search private channel messages |
| `search:read.im` | Search DM messages |
| `search:read.mpim` | Search group DM messages |
| `search:read.files` | Search shared files |

## Testing

All tests use `httptest.NewServer` to mock the Slack API — no real API calls are made.

```bash
go test ./connectors/slack/... -v
```
