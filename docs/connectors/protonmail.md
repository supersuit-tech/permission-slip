# Proton Mail (built-in connector)

Permission Slip includes a built-in **Proton Mail** connector that talks to your mailbox through [Proton Mail Bridge](https://proton.me/mail/bridge) — Proton's official local IMAP/SMTP proxy. Bridge exposes IMAP on `127.0.0.1:1143` and SMTP on `127.0.0.1:1025` by default, so the connector's defaults work out of the box.

## Why a local proxy is required

Proton Mail does not offer a public Gmail-style API for third-party apps on free or paid mail plans. Bridge runs on your machine, decrypts mail locally, and forwards IMAP/SMTP to Permission Slip. Every self-hoster who uses Proton must run Bridge on the same host (or one reachable on the LAN).

## Supported hosts

Bridge is officially distributed for **x86_64 Linux, macOS, and Windows**. Proton does not publish an ARM build, so Raspberry Pi and other ARM boards are not supported hosts for the Proton Mail connector.

If you're currently self-hosting on ARM hardware, we recommend running Permission Slip (and Bridge) on a small **x86_64 mini PC** with the specs you need. Any model that fits your budget and footprint will work; the connector only requires that Bridge itself is supported on the host OS.

## What the connector supports

| Action | Risk | Description |
|--------|------|-------------|
| `protonmail.send_email` | high | Send mail via SMTP through Bridge |
| `protonmail.read_inbox` | low | List recent messages in a folder |
| `protonmail.search_emails` | low | Search by subject, sender, or date |
| `protonmail.read_email` | low | Read one message by sequence number |
| `protonmail.archive_email` | medium | Move messages to Archive (IMAP MOVE) |

Calendar, Drive, Contacts, VPN, and Pass are **not** available through Bridge (no CalDAV/WebDAV for those products).

## Install and run Bridge

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

## Configure credentials in Permission Slip

Once Bridge is running, add **Proton Mail** credentials (`custom` auth) in the UI:

| Field | Value |
|-------|--------|
| `username` | Your Proton address (as shown in Bridge) |
| `password` | The bridge password — **not** your Proton account password |
| `imap_host` / `imap_port` | Optional; default `127.0.0.1` / `1143` |
| `smtp_host` / `smtp_port` | Optional; default `127.0.0.1` / `1025` |

Saving credentials runs a real **IMAP LOGIN** against Bridge. Bridge must be running at save time.

## Migrating from `permission-slip-proton`

If you previously used the external [permission-slip-proton](https://github.com/supersuit-tech/permission-slip-proton) subprocess connector, remove it from `custom-connectors.json`. Create new built-in Proton Mail credentials with the same field names — no schema migration is required.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| Credential validation fails immediately | Bridge running? `systemctl --user status protonmail-bridge` |
| Auth / LOGIN errors | Username must match the address in Bridge; password must be the bridge password, not your Proton account password |
| Connection refused on 1143/1025 | Another process using the port, or Bridge bound to a different interface |
| Login loop in Bridge CLI | Clock skew, 2FA, or `pass` store not initialized |
| Archive action fails | Proton's Archive folder must exist; Bridge exposes it as `"Archive"` |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
