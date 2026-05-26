# Proton Mail (built-in connector)

Permission Slip includes a built-in **Proton Mail** connector that talks to your mailbox through [Proton Mail Bridge](https://proton.me/mail/bridge). Bridge exposes standard IMAP and SMTP on localhost so agents can send and read mail without a separate external connector binary.

## Why Bridge is required

Proton Mail does not offer a public Gmail-style API for third-party apps on free or paid mail plans. Bridge runs on your machine (or your self-hosted server), decrypts mail locally, and forwards IMAP/SMTP to Permission Slip. Every self-hoster who uses Proton must run Bridge anyway; host and ports are usually `127.0.0.1` with Bridge’s default ports.

## What the connector supports

| Action | Risk | Description |
|--------|------|-------------|
| `protonmail.send_email` | high | Send mail via SMTP through Bridge |
| `protonmail.read_inbox` | low | List recent messages in a folder |
| `protonmail.search_emails` | low | Search by subject, sender, or date |
| `protonmail.read_email` | low | Read one message by sequence number |
| `protonmail.archive_email` | medium | Move messages to Archive (IMAP MOVE) |

Calendar, Drive, Contacts, VPN, and Pass are **not** available through Bridge (no CalDAV/WebDAV for those products).

## Headless Bridge on a Raspberry Pi or Linux server

These steps assume a dedicated Linux user (example: `proton`) on the same host as Permission Slip.

### 1. Install Bridge

On Debian/Ubuntu (including Raspberry Pi OS):

```bash
# Get Official package — see https://proton.me/mail/bridge for other distros
sudo apt install ./protonmail-bridge_*.deb
```

Use the ARM build on a Pi when applicable.

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

In the CLI: log in with your Proton account, complete 2FA, and let Bridge finish syncing. Then note the **Bridge-generated password** and IMAP/SMTP ports (defaults are often IMAP `127.0.0.1:1143`, SMTP `127.0.0.1:1025`).

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

`enable-linger` keeps the user’s systemd instance running after logout so Bridge stays up for Permission Slip.

### 6. Configure credentials in Permission Slip

In the UI, add **Proton Mail** credentials (`custom` auth):

| Field | Value |
|-------|--------|
| `username` | Your Proton address (as shown in Bridge) |
| `password` | Bridge-generated password (not your Proton account password) |
| `imap_host` / `imap_port` | Optional; default `127.0.0.1` / `1143` |
| `smtp_host` / `smtp_port` | Optional; default `127.0.0.1` / `1025` |

Saving credentials runs a real **IMAP LOGIN** against Bridge. Bridge must be running at save time.

## Migrating from `permission-slip-proton`

If you previously used the external [permission-slip-proton](https://github.com/supersuit-tech/permission-slip-proton) subprocess connector, remove it from `custom-connectors.json`. Create new built-in Proton Mail credentials with the same field names — no schema migration is required.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| Credential validation fails immediately | Bridge service running? `systemctl --user status protonmail-bridge` as the `proton` user |
| Auth / LOGIN errors | Username must match Bridge; password must be the Bridge-generated password |
| Connection refused on 1143/1025 | Another process using the port, or Bridge bound to a different interface |
| Login loop in Bridge CLI | Clock skew, 2FA, or `pass` store not initialized |
| Archive action fails | Proton’s Archive folder must exist; Bridge must support IMAP MOVE |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
