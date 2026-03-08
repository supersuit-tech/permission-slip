# Discord Connector

Discord integration for community management and messaging. Uses the
[Discord REST API v10](https://discord.com/developers/docs/reference)
with a plain `net/http` client (no third-party SDK).

## Authentication

### Bot token (required for actions)

All connector actions (send messages, manage roles, ban/kick, etc.) use a
**bot token** via Discord's `Bot` authorization header. This is the only
credential needed for action execution.

| Key         | Description                          |
|-------------|--------------------------------------|
| `bot_token` | Discord bot token (from the [Developer Portal](https://discord.com/developers/applications)) |

### OAuth provider (bot authorization flow)

Discord is also registered as a **built-in OAuth provider** (`discord`) in the
platform's OAuth registry. This enables the standard bot authorization flow —
users can install the bot into their servers via the OAuth consent screen. The
OAuth access token is **not** used for executing connector actions (Discord bot
actions require `Bot` auth, not `Bearer` auth).

OAuth scopes: `bot`, `guilds`

To enable the OAuth flow, set `DISCORD_CLIENT_ID` and `DISCORD_CLIENT_SECRET`
environment variables (from the Discord Developer Portal > OAuth2 section).

### Setup steps

1. Go to <https://discord.com/developers/applications> and create a new application.
2. Navigate to **Bot** in the sidebar.
3. Click **Reset Token** (or **Copy** if shown) to get the bot token.
4. Under **Privileged Gateway Intents**, enable **Server Members Intent** if you plan to use `list_members`.
5. Go to **OAuth2 > URL Generator**, select the `bot` scope, choose the permissions your actions require (see table below), then use the generated URL to invite the bot to your server.
6. Paste the bot token into Permission Slip as the `bot_token` credential.

## Actions

| Action | Risk | Description | Required Bot Permission |
|--------|------|-------------|------------------------|
| `discord.send_message` | low | Send a message to a channel | Send Messages |
| `discord.create_channel` | medium | Create a text, voice, or category channel | Manage Channels |
| `discord.manage_roles` | medium | Assign or remove a role from a member | Manage Roles |
| `discord.create_event` | medium | Create a scheduled guild event | Manage Events |
| `discord.ban_user` | high | Ban a user from a guild | Ban Members |
| `discord.kick_user` | high | Kick a user from a guild | Kick Members |
| `discord.pin_message` | medium | Pin a message in a channel | Manage Messages |
| `discord.unpin_message` | medium | Unpin a message in a channel | Manage Messages |
| `discord.list_channels` | low | List channels in a guild | View Channels |
| `discord.list_members` | low | List members of a guild | Server Members Intent |
| `discord.create_thread` | low | Create a thread in a channel | Create Public Threads |
| `discord.list_roles` | low | List roles in a guild | (none) |

## Parameter constraints

All ID parameters (channel, guild, user, role, message) are **snowflakes** --
numeric strings, typically 17-20 digits (e.g. `1234567890123456789`). The
connector validates that IDs contain only digits.

| Parameter | Constraint |
|-----------|-----------|
| Message `content` | 1-2000 characters |
| Channel `name` | 2-100 characters, lowercase, no spaces (`^[a-z0-9_-]+$`) |
| Channel `topic` | Up to 1024 characters |
| Event `name` | 1-100 characters |
| Event `description` | Up to 1000 characters |
| Thread `name` | 1-100 characters |
| Thread `auto_archive_duration` | One of: 0 (server default), 60, 1440, 4320, 10080 (minutes) |
| `entity_type` | 1 (stage), 2 (voice), or 3 (external) |
| Ban `delete_message_seconds` | 0 - 604800 (0 = none, 604800 = 7 days) |
| List `limit` | 1 - 1000 (default 100) |

## Error handling

The connector maps Discord API error codes to actionable messages:

| Discord Code | Meaning | Connector Error |
|-------------|---------|-----------------|
| 10003 | Unknown Channel | ExternalError -- "channel not found, verify the ID" |
| 10004 | Unknown Guild | ExternalError -- "guild not found, verify the ID" |
| 10007 | Unknown Member | ExternalError -- "member not found in guild" |
| 10008 | Unknown Message | ExternalError -- "message not found in channel" |
| 10011 | Unknown Role | ExternalError -- "role not found, use list_roles" |
| 30003 | Max pins | ExternalError -- "50 pinned messages limit reached" |
| 40001 | Unauthorized | AuthError -- "verify bot token" |
| 50001 | Missing Access | AuthError -- "bot missing guild/channel access" |
| 50013 | Missing Permissions | AuthError -- "check bot role permissions" |
| 50028 | Invalid Role | ExternalError -- "role hierarchy issue" |
| 50035 | Invalid Form Body | ValidationError -- parameter format issue |
| HTTP 401 | Invalid token | AuthError -- "regenerate token" |
| HTTP 403 | Forbidden | AuthError -- "check bot permissions" |
| HTTP 429 | Rate limited | RateLimitError with Retry-After |

## Response examples

### send_message
```json
{"id": "1234567890", "channel_id": "9876543210"}
```

### create_channel
```json
{"id": "1234567890", "name": "new-channel", "type": 0}
```

### manage_roles
```json
{"status": "success", "action": "assign", "user_id": "111", "role_id": "222"}
```

### create_event
```json
{"id": "1234567890", "name": "Team Standup"}
```

### ban_user / kick_user
```json
{"status": "banned", "user_id": "111"}
```

### pin_message / unpin_message
```json
{"status": "pinned", "message_id": "111", "channel_id": "222"}
```

### list_channels
```json
{
  "channels": [
    {"id": "100", "name": "general", "type": 0, "position": 0}
  ]
}
```

### list_members
```json
{
  "members": [
    {"user_id": "100", "username": "alice", "nick": "Alice", "roles": ["200"], "joined_at": "2025-01-01T00:00:00Z"}
  ]
}
```

### list_roles
```json
{
  "roles": [
    {"id": "100", "name": "@everyone", "color": 0, "position": 0, "managed": false, "mentionable": false},
    {"id": "200", "name": "Moderator", "color": 3447003, "position": 1, "managed": false, "mentionable": true}
  ]
}
```

### create_thread
```json
{"id": "1234567890", "name": "Discussion Thread"}
```

## File structure

```
connectors/discord/
  discord.go              Main connector: manifest, HTTP client, error mapping
  send_message.go         discord.send_message action
  create_channel.go       discord.create_channel action
  manage_roles.go         discord.manage_roles action
  create_event.go         discord.create_event action
  ban_user.go             discord.ban_user action
  kick_user.go            discord.kick_user action
  pin_message.go          discord.pin_message / discord.unpin_message actions
  list_channels.go        discord.list_channels action
  list_members.go         discord.list_members action
  create_thread.go        discord.create_thread action
  list_roles.go           discord.list_roles action
  helpers_test.go         Shared test helpers (validCreds)
  *_test.go               Tests for each action
  README.md               This file
```

## Testing

```bash
go test ./connectors/discord/... -v
```

All tests use `httptest.NewServer` to mock the Discord API -- no real
Discord credentials or network access required.
