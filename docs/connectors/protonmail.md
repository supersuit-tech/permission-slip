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

> **About `sudo` and the `proton` user.** Steps 1–3 are system-level and must run as your normal, sudo-capable admin account. Step 2 creates the `proton` user with **no password** — `useradd` leaves the account locked, and it is *not* in the `sudo`/`sudoers` group. That's intentional: it's an unprivileged service account.
>
> Do all `sudo apt install` (and any other `sudo`) work **before** you switch into the proton shell in step 4. Once you run `sudo -u proton bash`, every command runs *as* `proton`, so running `sudo` there will prompt for a password the account doesn't have (and would fail authorization anyway). Everything from step 4 onward is designed to run **without `sudo`**.

### 1. Install Bridge and its dependencies

Run these as your **admin user** (the one with `sudo`). Installing the system packages up front means you never need `sudo` once you switch to the `proton` user.

On Debian/Ubuntu (x86_64):

```bash
# Bridge's keychain backend
sudo apt install pass gnupg

# Download the official package from https://proton.me/mail/bridge
sudo apt install ./protonmail-bridge_*.deb
```

### 2. Create the service user

Still as your **admin user**:

```bash
sudo useradd -m -s /bin/bash proton
```

This account has no password and cannot use `sudo` — that's expected. You operate it with `sudo -u proton ...` from your admin account, never by logging in as `proton`.

### 3. Initialize a `pass` password store (Bridge keychain)

Switch into the proton shell — **everything from here on runs as `proton` and needs no `sudo`:**

```bash
sudo -u proton bash   # you are now the proton user; do NOT run sudo from here
```

Bridge stores its encryption key in `pass`, which in turn is unlocked by a GPG key. You need to generate that GPG key, then point `pass` at it.

Run:

```bash
gpg --full-generate-key
```

This is interactive. Answer the prompts as follows:

| Prompt | What to enter | Why |
|--------|---------------|-----|
| *Please select what kind of key you want* | Press **Enter** to accept the default (`RSA and RSA`) | The default is fine for this purpose. |
| *What keysize do you want?* | Press **Enter** to accept the default (`3072`) | Default is secure. |
| *Key is valid for?* | Type `0`, press **Enter**, then `y` to confirm | `0` means the key never expires — important, because an expired key would lock Bridge out of its own store. |
| *Real name* | Type anything memorable, e.g. `Proton Bridge` | This is just a label; it isn't checked against your Proton account. |
| *Email address* | Leave blank and press **Enter** (or use `proton@localhost`) | Only used to label the key. |
| *Comment* | Leave blank and press **Enter** | Optional. |
| *Change (N)ame, (C)omment, (E)mail or (O)kay/(Q)uit?* | Type `O` and press **Enter** | Confirms the details above. |
| *Passphrase* | **Leave it empty** — press **Enter**, then confirm the empty passphrase | See the note below. |

> **Why no passphrase?** Bridge runs unattended under systemd (step 5). If the GPG key has a passphrase, `gpg-agent` will block startup waiting for someone to type it interactively — which never happens for a background service, so Bridge fails to start. An empty passphrase is acceptable here because the key lives in a dedicated, **unprivileged** `proton` account with no password and no `sudo`, and the `pass` store only holds Bridge's own local keychain secret. If your threat model requires a passphrase, you'll need to configure `gpg-agent` caching or a systemd credential to supply it on boot — that's beyond this guide.
>
> On a headless server, the empty-passphrase confirmation may appear as a text-based `pinentry-curses` dialog rather than inline prompts. Use the arrow keys / **Tab** to select **OK**/**Yes** and confirm you want no passphrase. If `gpg` can't open a pinentry program at all, install one with `sudo apt install pinentry-curses` (run this as your admin user before switching to `proton`).

Once the key is generated, initialize the `pass` store with its ID:

```bash
pass init "$(gpg --list-secret-keys --keyid-format LONG | awk '/^sec/{print $2}' | head -1)"
```

The `awk` command pulls the new key's ID out of `gpg --list-secret-keys` automatically, so you don't have to copy it by hand. If you have more than one secret key in this account, `head -1` takes the first one — in a fresh `proton` account there's only the key you just created.

### 4. Log in to Proton (one-time, interactive)

Still in the `proton` shell (no `sudo`):

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

> The `systemctl --user` commands run as `proton` (no `sudo`). `loginctl enable-linger proton` enables linger for the current user and usually succeeds without `sudo` via polkit; if your system prompts for admin authentication, run it once from your **admin account** instead: `sudo loginctl enable-linger proton`.

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
| `sudo` asks for a password inside the proton shell | You ran `sudo` after `sudo -u proton bash`. The `proton` account has no password and no sudo rights. `exit` back to your admin user and run system/`apt` commands there (steps 1–2); only run the non-`sudo` commands as `proton` |
| Archive action fails | Proton's Archive folder must exist; Bridge exposes it as `"Archive"` |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
