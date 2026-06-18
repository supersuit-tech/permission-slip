# iMessage Connector

Built-in connector that wraps [openclaw/imsg](https://github.com/openclaw/imsg) to read and send iMessages through Permission Slip's approval flow.

## Requirements

- macOS 14+ with Messages.app signed in (on the host where `imsg` runs)
- [imsg](https://github.com/openclaw/imsg): `brew install steipete/tap/imsg`
- **Full Disk Access** for reads (`chat.db`)
- **Automation** permission for sends (Messages.app)

## Deployment topologies

1. **Mac host:** Run Permission Slip on the same Mac as `imsg`. Leave connector credentials at defaults.
2. **Linux + Mac gateway:** Run Permission Slip on Linux; set `remote_host` to an SSH alias that runs `imsg` on the Mac (e.g. `ssh -T messages-mac imsg`).

## Actions

| Action | Approval | Description |
|--------|----------|-------------|
| `imessage.list_chats` | No | Recent conversations |
| `imessage.get_chat` | No | Single chat + participants |
| `imessage.read_history` | No | Messages in a chat (supports `since_guid` / `since_rowid`) |
| `imessage.search` | No | Search local history |
| `imessage.send_message` | Yes | Send text or file attachment |

## Permission granularity

Send actions support `from` and `to` as arrays of typed handles (`phone` / `email`). Constraint metadata is resolved from `imsg account` (from) and chat participants (to) for standing approval matching.

Default send policy: `service: imessage` with `no_sms_fallback: true` (iMessage-only, no surprise SMS).

## Health check

Credential validation runs `chats.list` with `limit: 1` — a real read probe, not just a version check.
