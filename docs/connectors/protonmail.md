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

Generate the key non-interactively with a batch parameter file:

```bash
cat > /tmp/proton-gpg-keygen <<'EOF'
%no-protection
Key-Type: RSA
Key-Length: 3072
Subkey-Type: RSA
Subkey-Length: 3072
Name-Real: Proton Bridge
Name-Email: proton@localhost
Expire-Date: 0
%commit
EOF

gpg --batch --generate-key /tmp/proton-gpg-keygen
rm /tmp/proton-gpg-keygen
```

What each line does:

| Line | Meaning |
|------|---------|
| `%no-protection` | Creates the key **without a passphrase**. This is the important one — see the note below. |
| `Key-Type` / `Key-Length` | A 3072-bit RSA primary key (GnuPG's secure default). |
| `Subkey-Type` / `Subkey-Length` | A matching RSA encryption subkey — `pass` needs an encryption-capable key. |
| `Name-Real` / `Name-Email` | Just a label for the key; not checked against your Proton account. Use anything. |
| `Expire-Date: 0` | The key never expires. Important — an expired key would lock Bridge out of its own store. |

> **Why no passphrase?** Bridge runs unattended under systemd (step 5). If the GPG key has a passphrase, `gpg-agent` will block startup waiting for someone to type it interactively — which never happens for a background service, so Bridge fails to start. An empty passphrase is acceptable here because the key lives in a dedicated, **unprivileged** `proton` account with no password and no `sudo`, and the `pass` store only holds Bridge's own local keychain secret. If your threat model requires a passphrase, you'll need to configure `gpg-agent` caching or a systemd credential to supply it on boot — that's beyond this guide.
>
> Avoid the interactive `gpg --full-generate-key` for this: modern GnuPG routes the passphrase prompt through `pinentry`, which on a headless server often loops back instead of accepting an empty passphrase cleanly. The `--batch` file above sidesteps `pinentry` entirely, so there's no screen to fight with.

Once the key is generated, move into the `proton` user's home directory and initialize the `pass` store with the key's fingerprint:

```bash
cd ~                  # the proton user's home: /home/proton
pass init "$(gpg --list-secret-keys --with-colons | awk -F: '/^fpr:/{print $10; exit}')"
```

`pass` keeps its data in `$HOME/.password-store`, so it's tied to the **`proton` user's home directory**, not to wherever you happen to be standing in the filesystem. `cd ~` makes sure you're operating in `/home/proton` before initializing, so the store is created in the right place — the same home Bridge will read from in the steps below. If `cd ~` doesn't land you in `/home/proton` (check with `echo $HOME`), your shell didn't inherit the `proton` user's home; start it as a login shell instead with `sudo -iu proton`.

The `pass init` command pulls the new key's fingerprint out of `gpg`'s machine-readable output automatically, so you don't have to copy it by hand. We use the full fingerprint (not the short key ID) because it's the unambiguous form `gpg` always accepts.

### 4. Log in to Proton (one-time, interactive)

Still in the `proton` shell (no `sudo`):

```bash
protonmail-bridge --cli
```

In the CLI: log in with your Proton account, complete 2FA, and let Bridge finish syncing. Then note the **Bridge-generated password** and IMAP/SMTP ports (defaults are usually IMAP `127.0.0.1:1143`, SMTP `127.0.0.1:1025`).

**Can't find the server address (host/port)?** Type `info` at the Bridge CLI prompt. Bridge prints the exact connection details it's serving — the IMAP and SMTP **host** and **port**, your username, and the bridge password — so you don't have to guess. That `info` output is the source of truth for the server address you enter in Permission Slip below. If you ever lose the bridge password, `info` shows it again (there's no separate "reveal" step).

This is a **one-time, interactive login** — you do **not** keep this terminal open. Bridge saves your account into its local store (under the `proton` user's home), so once you've logged in, synced, and copied down the bridge password, type `exit` to quit the CLI. Quit it **before** moving to step 5: the systemd service starts Bridge headless, and two Bridge instances can't both bind ports `1143`/`1025` at the same time. From step 5 onward, systemd runs Bridge for you — you never need to start `--cli` again unless you have to re-authenticate.

> **If a Proton Mail Bridge GUI window opens, you can ignore it.** This guide drives Bridge entirely through the CLI (step 4) and then headless under systemd (step 5) — it never uses the desktop GUI. Some desktop Linux installs add an autostart entry that launches the Bridge window on login, or running `protonmail-bridge` with no flags opens it. Just close it. The only thing to avoid is running the GUI **and** the systemd service at the same time, since both would try to bind the same ports.

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

**Where do the host/port values come from?** If Permission Slip and Bridge run on the **same machine**, leave the host/port fields blank — the `127.0.0.1` defaults are correct and match what Bridge serves. If you're unsure or the defaults don't connect, run `protonmail-bridge --cli` and type `info` to see the exact host and ports Bridge is listening on.

If Permission Slip runs on a **different machine** on your network, **do not** change these host fields to the Bridge machine's LAN IP — that won't work (see [Running Permission Slip on a separate machine](#running-permission-slip-on-a-separate-machine) for why, and what to do instead). You'll set up a tunnel and keep these fields on the `127.0.0.1` defaults.

Saving credentials runs a real **IMAP LOGIN** against Bridge. Bridge must be running at save time.

## Running Permission Slip on a separate machine

A common setup is Bridge on one box (say a mini PC that stays on) and the Permission Slip CLI on another machine on the same internal network. This works, but **not** by pointing the credential host fields at the Bridge machine's LAN IP. Two things make that fail:

- **Bridge only listens on loopback.** By [deliberate design](https://protonmail.uservoice.com/forums/284483-proton-mail-calendar/suggestions/50891396-allow-bridge-to-bind-to-ip), Bridge binds IMAP/SMTP to `127.0.0.1` and has no setting to bind to a LAN interface. Other devices can't reach it directly.
- **The connector changes transport based on the host.** When `imap_host` is loopback, the connector speaks the plain IMAP that Bridge serves on `1143`. For any non-loopback host it switches to implicit TLS — which Bridge doesn't offer on that port — and SMTP's STARTTLS would fail the certificate check, because Bridge's certificate is issued for `127.0.0.1`, not your LAN IP. So a LAN IP breaks both reading and sending.

The clean fix is an **SSH tunnel from the Permission Slip machine to the Bridge machine**. The tunnel re-exposes Bridge's ports on the Permission Slip machine's *own* `127.0.0.1`, so the connector sees exactly what it expects (plain loopback IMAP, a `127.0.0.1` cert for STARTTLS), while the hop across your network is encrypted by SSH. Bridge stays loopback-only — nothing is exposed on the LAN in the clear.

### 1. Open the tunnel from the Permission Slip machine

Run this **on the machine where Permission Slip runs**, pointing at the Bridge host (use SSH key auth so it can run unattended):

```bash
ssh -N \
  -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes \
  -L 127.0.0.1:1143:127.0.0.1:1143 \
  -L 127.0.0.1:1025:127.0.0.1:1025 \
  youruser@bridge-host.lan
```

`-L 127.0.0.1:1143:127.0.0.1:1143` means "listen on this machine's loopback port 1143 and forward to the Bridge host's loopback port 1143." You can SSH as any user that has shell access on the Bridge box — the forward targets that box's *own* `127.0.0.1`, which is where Bridge is listening, so it doesn't have to be the `proton` user.

### 2. Keep it up with systemd (Permission Slip machine)

So the tunnel survives reboots and reconnects after a network blip, run it as a user systemd unit on the Permission Slip machine:

```bash
mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/proton-bridge-tunnel.service << 'EOF'
[Unit]
Description=SSH tunnel to Proton Mail Bridge
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/bin/ssh -N \
  -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes \
  -L 127.0.0.1:1143:127.0.0.1:1143 \
  -L 127.0.0.1:1025:127.0.0.1:1025 \
  youruser@bridge-host.lan
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
EOF

loginctl enable-linger "$USER"
systemctl --user daemon-reload
systemctl --user enable --now proton-bridge-tunnel.service
systemctl --user status proton-bridge-tunnel.service
```

> Use SSH key authentication (`ssh-copy-id youruser@bridge-host.lan`) so the unit never blocks on a password prompt. If you prefer automatic reconnection with a single process instead of relying on `Restart=always`, `autossh` is a drop-in alternative for the `ExecStart` line.

### 3. Configure credentials normally

With the tunnel up, Bridge is reachable on the Permission Slip machine at `127.0.0.1:1143` / `127.0.0.1:1025`, so add credentials exactly as in [Configure credentials](#configure-credentials-in-permission-slip) and **leave the host/port fields blank** (the `127.0.0.1` defaults). The username and bridge password are still the ones from `info` on the Bridge host.

## Migrating from `permission-slip-proton`

If you previously used the external [permission-slip-proton](https://github.com/supersuit-tech/permission-slip-proton) subprocess connector, remove it from `custom-connectors.json`. Create new built-in Proton Mail credentials with the same field names — no schema migration is required.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| Credential validation fails immediately | Bridge running? `systemctl --user status protonmail-bridge` |
| Auth / LOGIN errors | Username must match the address in Bridge; password must be the bridge password, not your Proton account password |
| Connection refused on 1143/1025 | Another process using the port, or Bridge bound to a different interface. Run `protonmail-bridge --cli` and type `info` to confirm the host/port Bridge is actually serving |
| Login loop in Bridge CLI | Clock skew, 2FA, or `pass` store not initialized |
| TLS / certificate errors from a remote Permission Slip host | You pointed `imap_host`/`smtp_host` at the Bridge machine's LAN IP. Don't — use an SSH tunnel and keep the `127.0.0.1` defaults (see [Running Permission Slip on a separate machine](#running-permission-slip-on-a-separate-machine)) |
| `sudo` asks for a password inside the proton shell | You ran `sudo` after `sudo -u proton bash`. The `proton` account has no password and no sudo rights. `exit` back to your admin user and run system/`apt` commands there (steps 1–2); only run the non-`sudo` commands as `proton` |
| Archive action fails | Proton's Archive folder must exist; Bridge exposes it as `"Archive"` |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
