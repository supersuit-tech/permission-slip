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
| `protonmail.read_email` | low | Read one message by stable IMAP UID |
| `protonmail.download_attachment` | low | Download one attachment by MIME part path (`part_id` from `read_email`) |
| `protonmail.archive_email` | medium | Move messages to Archive (IMAP UID MOVE) |

### Archiving whole conversations

By default, `protonmail.archive_email` archives the **entire conversation**, not just the UID you pass — matching Proton Mail's Archive button and Gmail's thread-level archive. The connector searches the source folder by normalized subject and `In-Reply-To` headers to find older replies that may fall outside a `read_inbox` / `search_emails` listing window.

To archive only the exact UID(s) listed, pass `"include_thread": false`.

The approval card lists every message that will be archived when thread expansion applies. The expanded set is not capped by the 50-message input limit (that cap applies only to the requested `message_id` / `message_ids`).

### Message identifiers (breaking change)

`read_inbox` and `search_emails` return each message's **stable IMAP UID** (with its `folder`) instead of volatile sequence numbers. `read_email` and `archive_email` accept that same UID as `message_id` / `message_ids`, scoped by `folder`.

`read_email` lists attachment metadata including a `part_id` (MIME part path, e.g. `"2.1"`). Pass that `part_id` to `protonmail.download_attachment` along with the same `message_id` and `folder` to fetch the attachment bytes (base64-encoded in the JSON result). Attachments larger than 1 MB are rejected with a clear error.

Sequence numbers shift whenever another message is deleted or moved; UIDs do not. Permission Slip records **UIDVALIDITY** server-side per folder so pending approvals still target the same mailbox generation — if the folder is recreated, execution fails safe instead of acting on the wrong message.

RFC `Message-ID` headers are included in list/read responses as read-only `message_id_header` metadata. They are **not** the operational key for IMAP actions.

Agents or automations that still pass raw IMAP sequence numbers must be updated to use `{folder, uid}` from list/search results.

Calendar, Drive, Contacts, VPN, and Pass are **not** available through Bridge (no CalDAV/WebDAV for those products).

## Install and run Bridge

These steps assume a dedicated Linux user (example: `proton`) on the same host as Permission Slip.

> **Which user runs each step — read this first.** This is the #1 source of setup
> problems. Running a step as the wrong user creates a *second, separate* Bridge
> instance with its own account and keychain. Both then try to serve the same IMAP
> port, and you get baffling `no such user` errors because the instance answering
> on `127.0.0.1:1143` isn't the one you logged into. Get the user right at every step.
>
> | Step | Run as | Why |
> |------|--------|-----|
> | 1. Install Bridge + dependencies | **root / admin** (`sudo`) | System-wide package install |
> | 2. Create the `proton` user | **root / admin** (`sudo`) | Creates the unprivileged service account |
> | 3. Initialize the `pass` keychain | **`proton`** | The keychain lives in proton's home directory |
> | 4. Log in to Proton (one-time) | **`proton`** | The account must live in proton's Bridge config |
> | 5. Run Bridge under systemd | **`proton`** | Headless `--user` service owned by proton |
>
> **Only steps 1–2 use `sudo`/root.** From step 3 onward you work *as the `proton`
> user* and never run `sudo` — the account has no password and no sudo rights (that's
> intentional; it's an unprivileged service account), so a `sudo` there just fails.
> Every command block below is prefixed with a `# AS root` or `# AS proton` comment.
> If you're ever unsure who you are, run `whoami`.

### 1. Install Bridge and its dependencies

Run these as your **admin user** (the one with `sudo`). Installing the system packages up front means you never need `sudo` once you switch to the `proton` user.

On Debian/Ubuntu (x86_64):

```bash
# AS root (admin)
# Bridge's keychain backend
sudo apt install pass gnupg

# Download the official package from https://proton.me/mail/bridge
sudo apt install ./protonmail-bridge_*.deb
```

### 2. Create the service user

Still as your **admin user**:

```bash
# AS root (admin)
sudo useradd -m -s /bin/bash proton
```

This account has no password and cannot use `sudo` — that's expected. You operate it with `sudo -u proton ...` from your admin account, never by logging in as `proton`.

### 3. Initialize a `pass` password store (Bridge keychain) — *as `proton`*

Switch into the proton user's **login** shell. **Everything from here through step 5 runs as `proton` and needs no `sudo`:**

```bash
# AS root (admin) — open a LOGIN shell as the proton user:
sudo -iu proton
# You are now `proton`. Confirm with:  whoami    →    proton
# Do NOT run sudo from here.
```

Use `-iu` (a login shell), not `sudo -u proton bash`. The login shell sets `HOME=/home/proton`
*and* the systemd user-session environment (`XDG_RUNTIME_DIR`) that `systemctl --user` needs in
step 5 — without it you'll hit `Failed to connect to bus`.

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

### 4. Log in to Proton (one-time, interactive) — *as `proton`*

Still in the `proton` login shell from step 3 (no `sudo`). Confirm with `whoami` → `proton`
**before** running this — logging in as the wrong user is what creates a second, conflicting
Bridge instance:

```bash
# AS proton
protonmail-bridge --cli
```

In the CLI: log in with your Proton account, complete 2FA, and let Bridge finish syncing. Then note the **Bridge-generated password** and IMAP/SMTP ports (defaults are usually IMAP `127.0.0.1:1143`, SMTP `127.0.0.1:1025`).

**Can't find the server address (host/port)?** Type `info` at the Bridge CLI prompt. Bridge prints the exact connection details it's serving — the IMAP and SMTP **host** and **port**, your username, and the bridge password — so you don't have to guess. That `info` output is the source of truth for the server address you enter in Permission Slip below. If you ever lose the bridge password, `info` shows it again (there's no separate "reveal" step).

This is a **one-time, interactive login** — you do **not** keep this terminal open. Bridge saves your account into its local store (under the `proton` user's home), so once you've logged in, synced, and copied down the bridge password, type `exit` to quit the CLI. Quit it **before** moving to step 5: the systemd service starts Bridge headless, and two Bridge instances can't both bind ports `1143`/`1025` at the same time. From step 5 onward, systemd runs Bridge for you — you never need to start `--cli` again unless you have to re-authenticate.

> **If a Proton Mail Bridge GUI window opens, you can ignore it.** This guide drives Bridge entirely through the CLI (step 4) and then headless under systemd (step 5) — it never uses the desktop GUI. Some desktop Linux installs add an autostart entry that launches the Bridge window on login, or running `protonmail-bridge` with no flags opens it. Just close it. The only thing to avoid is running the GUI **and** the systemd service at the same time, since both would try to bind the same ports.
>
> **Especially: don't run `protonmail-bridge` as root or your admin user.** Doing so
> starts a *completely separate* Bridge instance with its own account and keychain under
> that user's home. If it (or its GUI) grabs port `1143` first, Permission Slip talks to
> *that* instance — which has no account — and every login fails with `no such user`,
> even though the `proton` instance is configured correctly. Bridge must only ever run
> as the `proton` user.

### 5. Run Bridge under systemd (user unit) — *as `proton`*

Use `--noninteractive`, **not** `--no-window`. On a headless server `--no-window` still
starts Bridge's Qt **GUI** (it only hides the window), so it crashes immediately with
`qt.qpa.xcb: could not connect to display` and the service crash-loops, never binding the
IMAP port. `--noninteractive` runs the bridge *core* with no GUI and no display.

```bash
# AS proton
mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/protonmail-bridge.service << 'EOF'
[Unit]
Description=Proton Mail Bridge
After=network.target

[Service]
ExecStart=/usr/bin/protonmail-bridge --noninteractive
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

**Where do the host/port values come from?** Bridge and Permission Slip run on the same machine, so leave the host/port fields blank — the `127.0.0.1` defaults are correct and match what Bridge serves. If the defaults don't connect, re-read `info` **from the `proton` instance that owns the port** — reading it from any other user shows a *different* instance's details and is the most common cause of `no such user`. Because the running service already holds the port, stop it first, read `info`, then start it again:

```bash
# AS proton (login shell: sudo -iu proton)
systemctl --user stop protonmail-bridge
protonmail-bridge --cli      # type: info   (note username + bridge password), then: exit
systemctl --user start protonmail-bridge
```

Saving credentials runs a real **IMAP LOGIN** against Bridge. Bridge must be running at save time.

## Migrating from `permission-slip-proton`

If you previously used the external [permission-slip-proton](https://github.com/supersuit-tech/permission-slip-proton) subprocess connector, remove it from `custom-connectors.json`. Create new built-in Proton Mail credentials with the same field names — no schema migration is required.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| Credential validation fails immediately | Bridge running? `systemctl --user status protonmail-bridge` |
| Auth / LOGIN errors | Username must match the address in Bridge; password must be the bridge password, not your Proton account password |
| Connection refused on 1143/1025 | Another process using the port, or Bridge bound to a different interface. Run `protonmail-bridge --cli` and type `info` to confirm the host/port Bridge is actually serving |
| Login loop in Bridge CLI | Clock skew, 2FA, or `pass` store not initialized |
| `sudo` asks for a password inside the proton shell | You ran `sudo` after `sudo -u proton bash`. The `proton` account has no password and no sudo rights. `exit` back to your admin user and run system/`apt` commands there (steps 1–2); only run the non-`sudo` commands as `proton` |
| Archive action fails | Proton's Archive folder must exist; Bridge exposes it as `"Archive"` |
| IMAP says `no such user` (but `info` shows that user) | A **second Bridge instance under a different user** is serving the port — usually one accidentally started as root or your admin user. Find the owner: `sudo ss -tlnp \| grep 1143`, then `ps -o user= -p <pid>`. If it isn't `proton`, stop that instance (and remove any root autostart / systemd unit), then ensure the `proton` systemd service owns the port |
| `systemctl --user`: `Failed to connect to bus` | You entered the proton shell without a user session. Use `sudo -iu proton` (login shell), or `export XDG_RUNTIME_DIR=/run/user/$(id -u)` before running `systemctl --user` |
| Not sure which user a running Bridge belongs to | `sudo ss -tlnp \| grep 1143` shows the pid; `ps -o user=,cmd= -p <pid>` shows the owner. Only the `proton` instance should ever be listening |
| Service crash-loops; logs show `qt.qpa.xcb: could not connect to display` / `bridge-gui` / `core dumped` | The unit is launching the **GUI**. On a headless server the `ExecStart` must use `--noninteractive`, not `--no-window` (the latter still loads Qt and needs a display). Fix the `ExecStart`, `systemctl --user daemon-reload`, then restart |

For general self-hosted setup (Tailscale, Google, Slack), see [Self-hosted deployment](../deployment-self-hosted.md).
