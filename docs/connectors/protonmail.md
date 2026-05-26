# Proton Mail (built-in connector)

Permission Slip includes a built-in **Proton Mail** connector that talks to your mailbox through a local IMAP/SMTP proxy. Two proxies are supported and the connector works the same way with either:

- **[Proton Mail Bridge](https://proton.me/mail/bridge)** — official Proton product, recommended on x86_64 Linux, macOS, and Windows.
- **[hydroxide](https://github.com/emersion/hydroxide)** — community-built open-source Go proxy, recommended on Raspberry Pi / ARM (Proton does not publish an ARM build of Bridge).

Both expose IMAP on `127.0.0.1:1143` and SMTP on `127.0.0.1:1025` by default, so the connector's defaults work for either.

## Why a local proxy is required

Proton Mail does not offer a public Gmail-style API for third-party apps on free or paid mail plans. A local proxy runs on your machine, decrypts mail locally, and forwards IMAP/SMTP to Permission Slip. Every self-hoster who uses Proton must run one of these proxies on the same host (or one reachable on the LAN).

## Choosing between Bridge and hydroxide

| | Proton Mail Bridge | hydroxide |
|---|---|---|
| Maintained by | Proton (official) | Community ([emersion](https://github.com/emersion)) |
| Architectures | x86_64 only (no ARM build published) | Anything Go cross-compiles for (incl. arm64, armhf) |
| Linux setup | `.deb` / `.rpm` + `pass` keychain | `go build`, no keychain |
| IMAP maturity | Production | Upstream tags IMAP as "work-in-progress" |
| Recommended for | x86_64 Linux / macOS / Windows hosts | Raspberry Pi and other ARM boards |

## What the connector supports

| Action | Risk | Description |
|--------|------|-------------|
| `protonmail.send_email` | high | Send mail via SMTP through the proxy |
| `protonmail.read_inbox` | low | List recent messages in a folder |
| `protonmail.search_emails` | low | Search by subject, sender, or date |
| `protonmail.read_email` | low | Read one message by sequence number |
| `protonmail.archive_email` | medium | Move messages to Archive (IMAP MOVE) |

Calendar, Drive, Contacts, VPN, and Pass are **not** available through either proxy (no CalDAV/WebDAV for those products).

## Option A — Proton Mail Bridge (x86_64)

> **Heads up:** Proton publishes Bridge only for x86_64 Linux, macOS, and Windows. If you're self-hosting on a Raspberry Pi or other ARM board, skip to [Option B (hydroxide)](#option-b--hydroxide-raspberry-pi--arm).

These steps assume a dedicated Linux user (example: `proton`) on the same host as Permission Slip.

### 1. Install Bridge

On Debian/Ubuntu (x86_64):

```bash
# Download the official package from https://proton.me/mail/bridge
sudo apt install ./protonmail-bridge_*.deb
```

### 2. Create a service user

```bash
sudo useradd -m -s /bin/bash proton
sudo -u proton bash
```

### 3. Initialize a `pass` password store (Bridge keychain)

Bridge stores its encryption key in `pass`:

```bash
sudo apt install pass gnupg
gpg --full-generate-key   # as the proton user
pass init "$(gpg --list-secret-keys --keyid-format LONG | awk '/^sec/{print $2}' | head -1)"
```

### 4. Log in to Proton (one-time, interactive)

Still as `proton`:

```bash
protonmail-bridge --cli
```

In the CLI: log in with your Proton account, complete 2FA, and let Bridge finish syncing. Then note the **Bridge-generated password** and IMAP/SMTP ports (defaults are usually IMAP `127.0.0.1:1143`, SMTP `127.0.0.1:1025`).

### 5. Run Bridge under systemd (user unit)

```bash
mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/protonmail-bridge.service << 'EOF'
[Unit]
Description=Proton Mail Bridge
After=network.target

[Service]
ExecStart=/usr/bin/protonmail-bridge --no-window
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
EOF

loginctl enable-linger proton
systemctl --user daemon-reload
systemctl --user enable --now protonmail-bridge.service
systemctl --user status protonmail-bridge.service
```

`enable-linger` keeps the user's systemd instance running after logout so Bridge stays up for Permission Slip.

Skip ahead to [Configure credentials in Permission Slip](#configure-credentials-in-permission-slip).

## Option B — hydroxide (Raspberry Pi / ARM)

These steps assume a dedicated Linux user (example: `proton`) and a working Go toolchain. On Raspberry Pi OS / Debian:

```bash
sudo apt install golang git
```

### 1. Build and install hydroxide

As the service user:

```bash
sudo useradd -m -s /bin/bash proton
sudo -u proton bash

# Builds for whatever architecture you're on, including arm64 and armhf.
git clone https://github.com/emersion/hydroxide.git
cd hydroxide
go build ./cmd/hydroxide
sudo install -m 0755 hydroxide /usr/local/bin/
```

### 2. Generate a bridge password

```bash
hydroxide auth your-username@proton.me
```

Hydroxide prints a **32-byte bridge password** at the end. **Copy it now** — it's not stored anywhere recoverable. This password is what you'll give Permission Slip; your real Proton password is not.

If you have 2FA enabled, pass the code with `-tfa-code <code>`.

### 3. Run hydroxide under systemd (user unit)

```bash
mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/hydroxide.service << 'EOF'
[Unit]
Description=Hydroxide Proton Mail IMAP/SMTP proxy
After=network.target

[Service]
ExecStart=/usr/local/bin/hydroxide serve
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
EOF

loginctl enable-linger proton
systemctl --user daemon-reload
systemctl --user enable --now hydroxide.service
systemctl --user status hydroxide.service
```

`hydroxide serve` starts SMTP on `1025`, IMAP on `1143`, and CardDAV on `8080` (the last is unused by Permission Slip).

## Configure credentials in Permission Slip

Once your proxy (Bridge or hydroxide) is running, add **Proton Mail** credentials (`custom` auth) in the UI:

| Field | Value |
|-------|--------|
| `username` | Your Proton address (as shown in Bridge or `hydroxide auth`) |
| `password` | The bridge password — **not** your Proton account password |
| `imap_host` / `imap_port` | Optional; default `127.0.0.1` / `1143` |
| `smtp_host` / `smtp_port` | Optional; default `127.0.0.1` / `1025` |

Saving credentials runs a real **IMAP LOGIN** against your proxy. The proxy must be running at save time.

## Compatibility notes (hydroxide)

Permission Slip's connector was originally written against Bridge. The protocol surface used by each action matches what hydroxide implements:

| Action | Hydroxide status |
|---|---|
| `send_email` | Hydroxide's SMTP server on `127.0.0.1:1025` accepts PLAIN auth — matches our client (`connectors/protonmail/send_email.go`). |
| `read_inbox`, `read_email` | `SELECT INBOX` + `FETCH` are supported by hydroxide's IMAP backend. |
| `search_emails` | Hydroxide implements `SearchMessages` covering subject / from / date criteria. |
| `archive_email` | Hydroxide exposes Proton's `LabelArchive` as the IMAP mailbox `"Archive"` — matches the literal in our archive action. IMAP MOVE is supported via `MoveMessages`. |

Upstream still flags hydroxide's IMAP as "work-in-progress". If you hit an action that misbehaves with hydroxide but works with Bridge, please [open an issue](https://github.com/supersuit-tech/permission-slip/issues).

## Migrating from `permission-slip-proton`

If you previously used the external [permission-slip-proton](https://github.com/supersuit-tech/permission-slip-proton) subprocess connector, remove it from `custom-connectors.json`. Create new built-in Proton Mail credentials with the same field names — no schema migration is required.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| Credential validation fails immediately | Proxy running? `systemctl --user status protonmail-bridge` (Bridge) or `systemctl --user status hydroxide` |
| Auth / LOGIN errors | Username must match the proxy; password must be the bridge password, not your Proton account password |
| Connection refused on 1143/1025 | Another process using the port, or the proxy bound to a different interface |
| Login loop in Bridge CLI | Clock skew, 2FA, or `pass` store not initialized |
| `hydroxide auth` fails | Network issue, 2FA required (use `hydroxide auth -tfa-code <code>`), or clock skew |
| Archive action fails | Proton's Archive folder must exist; both Bridge and hydroxide expose it as `"Archive"` |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
