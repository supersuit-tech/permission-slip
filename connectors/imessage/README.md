# iMessage Connector

Built-in connector that wraps [openclaw/imsg](https://github.com/openclaw/imsg) to read and send Messages.app conversations through Permission Slip's approval flow.

## Hard prerequisites

- **Same Apple ID** on the Mac (where `imsg` runs) and the user's iPhone — not a separate agent Apple ID.
- macOS 14+ with Messages.app signed in on the Mac.
- [imsg](https://github.com/openclaw/imsg): `brew install steipete/tap/imsg`
- **Full Disk Access** for reads (`chat.db`)
- **Automation** permission for sends (Messages.app)
- **Text Message Forwarding** enabled on the iPhone, pointed at this Mac — required for the Mac to see and send green-bubble SMS/MMS threads.
- **Messages in iCloud** recommended for full history sync across devices.

SMS relay routes through Apple's push servers (APNs), not the local Wi‑Fi network. Tailscale is neither required nor sufficient for SMS forwarding.

## Deployment topologies

1. **Mac host:** Run Permission Slip on the same Mac as `imsg`. Leave connector credentials at defaults.
2. **Linux + Mac gateway:** Run Permission Slip on Linux; set `remote_host` to an SSH alias that runs `imsg` on the Mac (e.g. `ssh -T messages-mac imsg`).

## Actions

| Action | Approval | Description |
|--------|----------|-------------|
| `imessage.list_chats` | No | Recent conversations with optional `unread_only` filter and per-chat `unread_count` (when imsg supports it) |
| `imessage.get_chat` | No | Single chat + participants |
| `imessage.read_history` | No | Messages in a chat (supports `since_guid` / `since_rowid`; per-message `is_read` / `date_read` when imsg supports it) |
| `imessage.search` | No | Search local history |
| `imessage.send_message` | Yes | Send text or file attachment |

## Unread state (read-only)

1. Call **`imessage.list_chats`** with `unread_only: true` to list chats that have unread inbound messages. Each chat includes `unread_count` when the installed imsg build supports it ([openclaw/imsg#151](https://github.com/openclaw/imsg/issues/151)).
2. Call **`imessage.read_history`** on a chat to inspect individual messages. Inbound rows may include `is_read` and `date_read` when imsg emits them.

Mark-as-read is intentionally unsupported — persisting read state requires IMCore injection (SIP off), which Permission Slip does not use.

## Send policy (service / fallback)

- **Default `service: auto`** — iMessage when possible, SMS fallback for new recipients or when replying in a green thread.
- **`no_sms_fallback: true`** — opt-in strict iMessage-only (no surprise SMS).
- **Approval-time disclosure** — `ResolveResourceDetails` probes the target chat and surfaces `delivery_disclosure` on the approval card (e.g. "Will send as SMS via relay" vs "Will send as iMessage").
- **Post-send verification** — after send, polls `message.send_status` and reports failed relay delivery instead of claiming success.
- **Idempotent retry** — pass `retry_guid` from a prior attempt; skips re-send when the message is already `sent` or `delivered`.

## Permission granularity

Send actions support `from` and `to` as arrays of typed handles (`phone` / `email`). Constraint metadata is resolved from `imsg account` (from), chat participants (to), and predicted delivery service for standing approval matching.

## Health check

Credential validation runs `chats.list` with `limit: 1` — a real read probe, not just a version check.
