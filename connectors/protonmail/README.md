# Proton Mail Connector

The Proton Mail connector integrates Permission Slip with [Proton Mail](https://proton.me/mail) via [Proton Mail Bridge](https://proton.me/mail/bridge). It uses standard IMAP/SMTP protocols to communicate with Bridge, which handles Proton's end-to-end encryption transparently.

> **Important:** This connector requires Proton Mail Bridge running locally. It is designed for self-hosted, single-user setups only — not suitable for multi-tenant or cloud-hosted deployments.

## Connector ID

`protonmail`

## Prerequisites

1. **Paid Proton Mail plan** — Bridge only works with paid plans (Mail Plus, Proton Unlimited, etc.)
2. **Proton Mail Bridge installed and running** — [Download Bridge](https://proton.me/mail/bridge)
3. **Bridge password generated** — This is distinct from your Proton account password. Find it in Bridge settings under your account.

## Credentials

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `username` | Yes | — | Your Proton Mail email address (e.g., `user@proton.me`) |
| `password` | Yes | — | Bridge-generated password (NOT your Proton account password) |
| `smtp_host` | No | `127.0.0.1` | SMTP server host |
| `smtp_port` | No | `1025` | SMTP server port |
| `imap_host` | No | `127.0.0.1` | IMAP server host |
| `imap_port` | No | `1143` | IMAP server port |

The credential `auth_type` in the database is `custom`. Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

### Getting Your Bridge Password

1. Open Proton Mail Bridge
2. Click on your account
3. Under "Mailbox details", copy the password shown — this is your Bridge-generated password
4. The IMAP and SMTP ports are also shown in the same section

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `protonmail.send_email` | Send Email | high | Send an email via SMTP |
| `protonmail.read_inbox` | Read Inbox | low | Fetch recent emails from a mailbox folder |
| `protonmail.search_emails` | Search Emails | low | Search emails by subject, sender, or date range |
| `protonmail.read_email` | Read Email | low | Fetch a specific email by sequence number with full body |

### `protonmail.send_email`

Sends an email via SMTP through Proton Mail Bridge.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `to` | string[] | Yes | — | Recipient email addresses |
| `cc` | string[] | No | — | CC recipient email addresses |
| `bcc` | string[] | No | — | BCC recipient email addresses |
| `subject` | string | Yes | — | Email subject line |
| `body` | string | Yes | — | Email body content |
| `content_type` | string | No | `text/plain` | `text/plain` or `text/html` |
| `reply_to` | string | No | — | Reply-To email address |

**Response:**

```json
{
  "status": "sent",
  "from": "user@proton.me",
  "recipients": ["alice@example.com"],
  "subject": "Hello"
}
```

**Security:** All email addresses are validated using `net/mail.ParseAddress`. Header values are sanitized to prevent header injection attacks (CR/LF characters are stripped). STARTTLS is used when available.

---

### `protonmail.read_inbox`

Fetches recent emails from a mailbox folder via IMAP.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder` | string | No | `INBOX` | Mailbox folder to read from |
| `limit` | integer | No | `10` | Maximum emails to fetch (1–50) |
| `unread_only` | boolean | No | `false` | Only fetch unread emails |

**Response:**

```json
{
  "emails": [
    {
      "seq_num": 42,
      "subject": "Meeting tomorrow",
      "from": ["Alice <alice@example.com>"],
      "to": ["user@proton.me"],
      "date": "2026-03-07T10:00:00Z",
      "flags": ["\\Seen"]
    }
  ],
  "total": 1
}
```

---

### `protonmail.search_emails`

Searches emails by subject, sender, or date range via IMAP SEARCH.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `folder` | string | No | `INBOX` | Mailbox folder to search in |
| `subject` | string | No | — | Search by subject (substring match) |
| `from` | string | No | — | Search by sender email address |
| `since` | string | No | — | Emails on or after this date (YYYY-MM-DD) |
| `before` | string | No | — | Emails before this date (YYYY-MM-DD) |
| `limit` | integer | No | `10` | Maximum results (1–50) |

At least one search criterion (`subject`, `from`, `since`, or `before`) is required.

**Response:** Same format as `read_inbox`.

---

### `protonmail.read_email`

Fetches a specific email by sequence number with full body content.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `message_id` | integer | Yes | — | Sequence number from `read_inbox` or `search_emails` |
| `folder` | string | No | `INBOX` | Mailbox folder containing the email |

**Response:**

```json
{
  "seq_num": 42,
  "subject": "Meeting tomorrow",
  "from": ["Alice <alice@example.com>"],
  "to": ["user@proton.me"],
  "cc": [],
  "reply_to": [],
  "date": "2026-03-07T10:00:00Z",
  "message_id_header": "abc123@proton.me",
  "flags": ["\\Seen"],
  "content_type": "text/plain",
  "body": "Hi, let's meet tomorrow at 10am.",
  "attachments": [
    {
      "filename": "agenda.pdf",
      "content_type": "application/pdf",
      "size": 12345
    }
  ]
}
```

**Note:** Uses `BODY[].PEEK` so reading an email does not mark it as read. Body content is truncated at 1 MB.

## Error Handling

| Scenario | Connector Error | Likely Cause |
|----------|-----------------|--------------|
| IMAP login failed | `AuthError` | Wrong Bridge password or username |
| SMTP auth failed | `AuthError` | Wrong Bridge password or username |
| Connection refused/timeout | `ExternalError` / `TimeoutError` | Bridge not running or wrong host/port |
| Mailbox SELECT failed | `ExternalError` | Folder doesn't exist |
| Invalid email address | `ValidationError` | Malformed email in parameters |
| Missing required parameter | `ValidationError` | Parameter not provided |

## Limitations

- **Bridge must be running** — If Bridge crashes or is stopped, all actions will fail with connection errors.
- **Localhost-only by default** — Bridge binds to `127.0.0.1`. For containerized deployments, configure Bridge networking or use custom host/port credentials.
- **No push notifications** — IMAP requires polling; there's no real-time inbox notification.
- **Single user per Bridge** — Each Bridge instance serves one Proton account.
- **Sequence numbers are volatile** — Sequence numbers from `read_inbox`/`search_emails` can change if messages are deleted. Use them promptly.

## Architecture

The connector follows the standard Permission Slip connector pattern:

- **`ProtonMailConnector`** implements `connectors.Connector` and `connectors.ManifestProvider`
- Each action (send, read inbox, search, read email) is a separate struct implementing `connectors.Action`
- IMAP actions share helpers in `imap_helpers.go` for connection management, envelope fetching, limit validation, and error mapping
- The connector is gated behind the `ENABLE_PROTONMAIL_CONNECTOR` environment variable since it requires a local Bridge daemon
- Credentials use the `custom` auth type — there's no OAuth flow; users provide their Bridge-generated password directly

## File Structure

```
connectors/protonmail/
├── protonmail.go          # ProtonMailConnector struct, New(), Manifest(), Actions(), ValidateCredentials()
├── imap_helpers.go        # IMAP session, shared helpers (fetchEnvelopes, validateLimit), error mapping
├── send_email.go          # protonmail.send_email action (SMTP)
├── read_inbox.go          # protonmail.read_inbox action (IMAP)
├── search_emails.go       # protonmail.search_emails action (IMAP)
├── read_email.go          # protonmail.read_email action (IMAP)
├── *_test.go              # Tests for each action + connector
├── helpers_test.go        # Shared test helpers (validCreds)
└── README.md              # This file
```

## Testing

Tests use a mock `sendFunc` for SMTP actions. IMAP actions test parameter validation and helpers — integration tests require a running Bridge instance.

```bash
go test ./connectors/protonmail/... -v
```

## Docker / Self-Hosted Setup

For containerized deployments, you can use the [unofficial Bridge Docker image](https://github.com/shenxn/protonmail-bridge-docker):

```yaml
services:
  protonmail-bridge:
    image: shenxn/protonmail-bridge
    restart: unless-stopped
    volumes:
      - bridge-data:/root
    ports:
      - "1025:25"
      - "1143:143"
```

Then configure the connector credentials with the appropriate `smtp_host`/`imap_host` pointing to the Bridge container.
