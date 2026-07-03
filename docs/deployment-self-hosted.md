# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded. You'll run it on your own machine and reach it privately over Tailscale — no port forwarding, no manual TLS, your own HTTPS hostname. Your instance is **only reachable from devices on your tailnet**, never exposed to the public internet.

> **Recommended hardware: Mac Mini.** A Mac Mini (Apple Silicon or Intel) is the recommended host — it's small, silent, sips power, and stays on 24/7 without complaint. Unlike a Linux ARM board, it runs **every connector natively, including Proton Mail** (Proton ships a native Apple Silicon build of Bridge — no x86_64-only restriction). This guide walks through a native macOS install.
>
> **Prefer Linux instead?** — e.g. a mini PC, Raspberry Pi, VM, or VPS you already have — see the **[Linux deployment guide](deployment-self-hosted-linux.md)**. Both guides produce an equivalent, fully-featured install; pick whichever OS you're more comfortable operating. You need **Go 1.24+** and **Node.js 20+** to build from source either way.

```
 ┌──────────────┐
 │  You, the    │   on the same tailnet (laptop, phone, etc.)
 │  mobile app  │
 └──────┬───────┘
        │ https://permissions.your-tailnet.ts.net
        ▼
 ┌──────────────────┐
 │  Tailscale       │  Private WireGuard mesh + free Let's Encrypt TLS
 │  (tailnet)       │  Not reachable from the public internet
 └──────┬───────────┘
        │ tailscale serve → localhost:8080
        ▼
┌──────────────────────────────────────────┐
│   Permission Slip (single Go binary)     │
│   API + embedded React UI → port 8080    │
│              │                            │
│              ▼                            │
│         ┌────────┐                        │
│         │ SQLite │                        │
│         └────────┘                        │
└──────────────────────────────────────────┘
```

### Before you start

You need a **[Tailscale account](https://login.tailscale.com/start)** — the free personal plan covers up to 3 users and 100 devices, which is plenty for a personal Permission Slip instance. No domain registration, DNS configuration, or port forwarding required.

You also need **[Homebrew](https://brew.sh)** installed, and the Xcode Command Line Tools (Homebrew's installer prompts for these automatically if they're missing).

---

## Step 1: Get the Binary

```bash
# Install Go via Homebrew
brew install go

# Install Node.js 22 via nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source ~/.zshrc   # macOS's default shell is zsh; use ~/.bash_profile instead if you use bash
nvm install 22 && nvm use 22

# Clone and build
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
make install
make build
```

This builds a native binary for whichever chip you're on (Apple Silicon or Intel) — no cross-architecture flags needed, since you're building directly on the Mac Mini that will run it.

> **Building on a different Mac?** Cross-compile and copy the binary over:
> ```bash
> # Apple Silicon (M-series) target:
> GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 make build
> scp bin/server user@mac-mini.local:~/permission-slip/bin/server
>
> # Intel target:
> GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 make build
> scp bin/server user@mac-mini.local:~/permission-slip/bin/server
> ```

---

## Step 2: Set Up Tailscale Serve

Install Tailscale:

```bash
brew install --cask tailscale-app
```

Open the Tailscale app once from `/Applications` — macOS will prompt you to approve its system extension under **System Settings → Privacy & Security**; click **Allow**, then reopen Tailscale. This one-time approval is required before `tailscale up` will work.

Sign in and join your tailnet (this prints a browser URL — open it and authenticate). No `sudo` needed — the standalone app's CLI talks to the already-privileged background daemon over a local socket instead of touching the network stack directly:

```bash
tailscale up
```

> **Headless Mac Mini (no display)?** If you're setting this up over SSH with no monitor attached, generate a one-time auth key at [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys) — leave **Reusable** off (single-use), leave **Ephemeral** off (so the server stays registered when offline), then run:
>
> ```bash
> tailscale up --authkey=tskey-auth-xxxxx --hostname=permissions
> ```
>
> You still need to open the Tailscale app locally at least once (or via screen sharing) to approve the system extension before this works. Once registered, open [Machines](https://login.tailscale.com/admin/machines) → click the machine → **Disable key expiry** so you don't have to re-authenticate every ~180 days. The auth key itself is only needed for this one-time bootstrap — don't bake it into a config file or launchd plist.

**Enable HTTPS for your tailnet.** In the [Tailscale admin console](https://login.tailscale.com/admin/dns), under **DNS**, click **Enable HTTPS**. This unlocks free Let's Encrypt certificates on your `<tailnet-name>.ts.net` domain.

> (Optional) Give your server a memorable name in the [Machines page](https://login.tailscale.com/admin/machines) — click the machine, then **Edit machine name**. The final hostname will be `<machine-name>.<tailnet-name>.ts.net`.

Capture the hostname so the rest of the guide is copy-paste:

```bash
export PS_HOSTNAME=$(tailscale status --json | jq -r '.Self.DNSName' | sed 's/\.$//')
echo "Permission Slip will be reachable at: https://$PS_HOSTNAME"
```

> No `jq`? `brew install jq` first. These shell variables are **only needed during setup** — they get written into config files and aren't referenced again. No need to add them to `.zshrc`.

Expose Permission Slip on that hostname over HTTPS:

```bash
tailscale serve --bg --https=443 8080
```

This proxies `https://$PS_HOSTNAME` to local port 8080 (where Permission Slip will run in the next step). `--bg` runs it in the background and persists the configuration across reboots. Verify:

```bash
tailscale serve status
```

Finally, install Tailscale on the devices you want to use Permission Slip from:

- **iOS** — [Tailscale on the App Store](https://apps.apple.com/app/tailscale/id1470499037)
- **Android** — [Tailscale on Google Play](https://play.google.com/store/apps/details?id=com.tailscale.ipn)
- **macOS / Windows / Linux** — [tailscale.com/download](https://tailscale.com/download/)

Sign each device into the same tailnet. Any device on the tailnet (and only those devices) can now reach `https://$PS_HOSTNAME`.

---

## Step 3: Configure Permission Slip

```bash
mkdir -p ~/permission-slip/data
cat > ~/permission-slip/.env <<EOF
DATABASE_PATH=$HOME/permission-slip/data/app.db
BASE_URL=https://$PS_HOSTNAME

# Generated below — leave as placeholders for now
SECRET_ENCRYPTION_KEY=replace-me
JWT_SIGNING_SECRET=replace-me
INVITE_HMAC_KEY=replace-me
EOF

# Fill in the secrets (macOS/BSD sed needs the empty '' arg after -i)
sed -i '' "s|SECRET_ENCRYPTION_KEY=replace-me|SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i '' "s|JWT_SIGNING_SECRET=replace-me|JWT_SIGNING_SECRET=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i '' "s|INVITE_HMAC_KEY=replace-me|INVITE_HMAC_KEY=$(openssl rand -hex 32)|" ~/permission-slip/.env
```

That's it — `BASE_URL` is your tailnet HTTPS hostname, and `ALLOWED_ORIGINS` doesn't need to be set (the server allows the origin the browser used).

---

## Step 4: Run on Boot (launchd)

On macOS, `launchd` is the equivalent of systemd. Create a per-user **LaunchAgent** so Permission Slip starts automatically at login and restarts if it crashes:

```bash
mkdir -p ~/Library/LaunchAgents
cat > ~/Library/LaunchAgents/com.permissionslip.server.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.permissionslip.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>$HOME/permission-slip/bin/server</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$HOME/permission-slip</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$HOME/permission-slip/server.log</string>
    <key>StandardErrorPath</key>
    <string>$HOME/permission-slip/server.log</string>
</dict>
</plist>
EOF

launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.permissionslip.server.plist
```

> No explicit env-file directive is needed: the server loads `.env` from its current directory on startup, and `WorkingDirectory` above points there.

> **For this to survive a reboot with nobody logged in**, turn on **automatic login** for this user under **System Settings → Users & Groups → Login Options**, and disable sleep (**System Settings → Energy** → set "Prevent automatic sleeping" / turn off display sleep on Mac Mini desktops, which have no battery to worry about).

Verify both services are healthy:

```bash
curl http://localhost:8080/api/health           # local check
curl https://$PS_HOSTNAME/api/health            # through Tailscale (run from any tailnet device)
```

To restart the service manually after config changes:

```bash
launchctl kickstart -k gui/$(id -u)/com.permissionslip.server
```

---

## Updating an existing install

To pull the latest release and apply it, run one command from the repo:

```bash
cd ~/permission-slip
make redeploy
```

This pulls `origin/main`, reinstalls dependencies, rebuilds the frontend and
server, restarts the `launchd` service set up in Step 4, publishes an
over-the-air mobile update via EAS when configured, and prints the build it's
now running (the same short SHA shown in the app footer, so you can confirm
the update took effect). New connector fields, manifest changes, and
migrations are picked up on the restart — there's nothing else to remember.

If EAS isn't set up on the host (no `EXPO_TOKEN` or `eas login`, etc.),
`make redeploy` prints a note and skips the mobile step — the server redeploy
still completes normally.

> **Named the LaunchAgent something other than `com.permissionslip.server`?** Pass it through:
> ```bash
> PS_LAUNCHD_LABEL=my.custom.label make redeploy
> ```

> The running server is only ever replaced by a **successful** build. If the
> build fails, `make redeploy` stops before restarting and your service keeps
> serving the previous working binary. A transient `git pull` failure is
> non-fatal — it rebuilds the current checkout instead.

> **Cross-compiling on a beefier box?** `make redeploy` builds and restarts on
> the machine it runs on. If you build elsewhere and `scp bin/server` over (see
> the build note in Step 1), pull + build there, copy the binary, then restart
> on the host with `launchctl kickstart -k gui/$(id -u)/com.permissionslip.server`.

---

## Step 5: Connect Google

Permission Slip's Google connector handles Gmail and Calendar actions. To enable it, register an OAuth client in Google Cloud:

1. In the [Google Cloud Console](https://console.cloud.google.com/), create or pick a project.
2. Under **APIs & Services > Library**, enable the **Gmail API** and **Google Calendar API**.
3. Under **APIs & Services > OAuth consent screen**, choose **External** (or **Internal** for Google Workspace). Fill in:
   - App name: `Permission Slip`
   - User support email: your email
   - Authorized domain: leave blank for now (see note below — `ts.net` isn't a domain you own, so Google won't accept it as an authorized domain)

   > **About `ts.net` and Google's verification:** Google requires authorized domains to be ones you own and have verified, which rules out `ts.net`. Two ways to live with this:
   > 1. **Keep the app in Testing mode** (the default). Add yourself and anyone else who needs access under **Test users**. Testing-mode apps work without domain verification.
   > 2. **Use Workspace Internal mode** if you have a Google Workspace account — Internal apps skip verification since they're scoped to your organization.
   >
   > If you later want a verified, published app, swap to a real domain you own (point it at the same Tailscale machine via [Funnel](https://tailscale.com/kb/1223/funnel) or a reverse proxy, then update `BASE_URL` and the OAuth redirect URI).

   Add these scopes:
   - `openid`
   - `https://www.googleapis.com/auth/userinfo.email`
   - `https://www.googleapis.com/auth/gmail.send`
   - `https://www.googleapis.com/auth/gmail.readonly`
   - `https://www.googleapis.com/auth/calendar.events`
4. Under **APIs & Services > Credentials**, click **Create Credentials > OAuth 2.0 Client ID**. Application type: **Web application**. Add the authorized redirect URI:
   ```
   https://$PS_HOSTNAME/api/v1/oauth/google/callback
   ```
5. Copy the Client ID and Client Secret into `~/permission-slip/.env`:
   ```bash
   GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-client-secret
   ```
6. Restart Permission Slip:
   ```bash
   launchctl kickstart -k gui/$(id -u)/com.permissionslip.server
   ```

---

## Step 6: Connect Slack

Slack uses [OAuth 2.0 with the V2 flow](https://api.slack.com/authentication/oauth-v2): bot scopes produce a bot token (`xoxb-`), and user scopes produce a user token (`xoxp-`) used for APIs that only accept user tokens (e.g. `search.messages`).

1. At [api.slack.com/apps](https://api.slack.com/apps), click **Create New App > From scratch**. Name it `Permission Slip` and pick a development workspace.
2. Open **OAuth & Permissions**. Under **Redirect URLs**, add:
   ```
   https://$PS_HOSTNAME/api/v1/oauth/slack/callback
   ```
3. Still under **OAuth & Permissions**, scroll to **Scopes** and add:

   **Bot Token Scopes**
   - `channels:history`, `channels:join`, `channels:manage`, `channels:read`
   - `chat:write`
   - `files:write`
   - `groups:history`, `groups:read`
   - `im:history`, `im:read`, `im:write`
   - `mpim:history`, `mpim:read`, `mpim:write`
   - `reactions:write`
   - `users:read`, `users:read.email`

   **User Token Scopes**
   - `search:read` — required for `search.messages` (the granular `search:read.*` scopes only work with the newer `assistant.search.context` endpoint).
4. Under **Basic Information > App Credentials**, copy the **Client ID** and **Client Secret** into `~/permission-slip/.env`:
   ```bash
   SLACK_CLIENT_ID=your-slack-client-id
   SLACK_CLIENT_SECRET=your-slack-client-secret
   ```
5. Restart Permission Slip:
   ```bash
   launchctl kickstart -k gui/$(id -u)/com.permissionslip.server
   ```

When a user connects Slack from Permission Slip, they'll complete Slack's OAuth consent and install the app to their workspace.

---

## Email: Proton Mail (Bridge)

Gmail is covered by the [Google connector](#step-5-connect-google) (OAuth). **Proton Mail** is a built-in connector that uses [Proton Mail Bridge](https://proton.me/mail/bridge) — Proton's official local IMAP/SMTP proxy — on the same machine as Permission Slip. There is no cloud OAuth flow.

Proton ships a native macOS build of Bridge for both Apple Silicon and Intel, so a Mac Mini runs the Proton Mail connector with no architecture caveats (unlike Linux, where Bridge is x86_64-only).

1. Install and run Bridge. Full steps, including a headless `launchd` option: **[Proton Mail connector guide](connectors/protonmail.md)**.
2. In the Permission Slip UI, add **Proton Mail** credentials with your Proton address and the bridge password printed by Bridge.
3. Grant agent permissions using the Proton templates (send, read inbox, search, read message, archive).

The proxy must be running when you save credentials; validation performs a real IMAP LOGIN.

---

## Setup iMessage connector

Permission Slip includes a built-in **iMessage** connector that talks to Messages.app via [openclaw/imsg](https://github.com/openclaw/imsg) — no cloud OAuth, no Apple sign-in flow, just local automation against the same Mac. Because this guide already targets a Mac Mini, the connector runs natively with no extra hardware.

### Prerequisites

- **Same Apple ID** signed into Messages.app on this Mac and on your iPhone — a separate "agent" Apple ID won't see your conversations.
- macOS 14+ with Messages.app signed in.
- [imsg](https://github.com/openclaw/imsg): `brew install steipete/tap/imsg`
- **Full Disk Access** (for reading `chat.db`) and **Automation** permission (for sending via Messages.app) — macOS prompts for both under **System Settings → Privacy & Security** the first time `imsg` runs.
- **Text Message Forwarding** enabled on your iPhone (**Settings → Messages → Text Message Forwarding**), pointed at this Mac — required to see and send green-bubble SMS/MMS threads.
- (Recommended) **Messages in iCloud** turned on for full history sync across devices.

SMS relay routes through Apple's push servers (APNs), not your Wi-Fi/LAN — Tailscale is neither required nor sufficient for SMS forwarding.

### Configure credentials in Permission Slip

Once `imsg` is installed and both permissions are granted, add **iMessage** credentials in the UI. The defaults work when `imsg` runs on this same Mac. If you're running Permission Slip on Linux instead and want a Mac to act as the iMessage gateway, set `remote_host` to an SSH alias that runs `imsg` on that Mac (e.g. `ssh -T messages-mac imsg`) — see the [Linux deployment guide](deployment-self-hosted-linux.md).

Saving credentials runs a real read probe (`chats.list`); `imsg` and Messages.app must be reachable at save time.

For the full action list, send policy (iMessage vs. SMS fallback), and permission granularity, see the [iMessage connector README](../connectors/imessage/README.md).

---

## Other Connectors

Permission Slip ships with 15+ more OAuth providers — Atlassian (Jira), Datadog, Dropbox, Figma, GitHub, HubSpot, Linear, Meta (Facebook/Instagram), Microsoft, Notion, PagerDuty, Square, PayPal, Stripe, and X (Twitter). See the [OAuth setup guide](oauth-setup.md) for per-provider instructions.

To build your own connector for a service Permission Slip doesn't yet support, see [custom connectors](custom-connectors.md).

---

## Set Up Tailscale on Your Openclaw Machines

Because your Permission Slip instance is private to your tailnet, **any machine that runs Openclaw (or any other tool that calls the Permission Slip API) also needs Tailscale installed** — otherwise it can't reach `https://$PS_HOSTNAME`.

- **macOS machines:**
  ```bash
  brew install --cask tailscale-app
  # then open Tailscale.app once to approve the system extension, as in Step 2
  tailscale up
  ```
- **Linux machines (laptop, dev box):**
  ```bash
  curl -fsSL https://tailscale.com/install.sh | sh
  sudo tailscale up
  ```
- **Headless machines (cloud VMs, CI runners, etc.):** generate a single-use, non-ephemeral auth key at [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys), then run:
  ```bash
  sudo tailscale up --authkey=tskey-auth-xxxxx --hostname=my-openclaw-box
  ```
  Then [disable key expiry](https://login.tailscale.com/admin/machines) on the machine so it stays online indefinitely (same approach as Step 2).
- **Docker / containerized agents:** install Tailscale on the host (and use host networking) or run it as a sidecar — see [Tailscale's Docker guide](https://tailscale.com/kb/1282/docker).

Once Tailscale is up on the agent host, point Openclaw at `https://$PS_HOSTNAME` — the same URL you set as `BASE_URL`. It'll connect like any normal HTTPS endpoint, no proxy config or special headers required.

---

## Mobile Push Notifications

The mobile app delivers approval request notifications via [Expo's push service](https://docs.expo.dev/push-notifications/overview/), which routes through APNs (iOS) or FCM (Android). Two things are required on the server side:

### 1. `BASE_URL` must be set

Push notification dispatch is skipped entirely if `BASE_URL` is empty. Confirm it's in your `.env` and matches the URL the mobile app connects to:

```bash
BASE_URL=https://your-server-hostname
```

If you followed Step 3, this is already set to your Tailscale hostname. If you're using Cloudflare Tunnel or another reverse proxy, set it to that public URL instead.

### 2. Outbound internet access on port 443

The server sends push requests to `exp.host` (Expo's push API). Your server needs outbound HTTPS access to reach it — this is typically available by default on any machine with internet access.

### 3. `EXPO_ACCESS_TOKEN` (only for your own build)

What you need here depends on **which app you installed**. For most self-hosters — using the official App Store build — the answer is **nothing**: leave `EXPO_ACCESS_TOKEN` unset and push just works.

| Mobile app | What you need |
|------------|----------------|
| **App Store / official build** | **Nothing.** Leave `EXPO_ACCESS_TOKEN` unset. Push is delivered through the official **Permission Slip** Expo project (`@supersuit-tech/permission-slip`), which the App Store binary is permanently tied to, and that project accepts unauthenticated sends. Your server sends to Expo without a token; Expo routes to your device. Set `BASE_URL` (§1) and you're done. |
| **Your own EAS build** | An access token from **your own** Expo account — see below. |

**Why the App Store build needs no token.** Push credentials (the iOS APNs key) are bound to the app's bundle identifier, which is baked into the binary at build time. The App Store build's bundle ID belongs to the Permission Slip Expo project, so every push for that app — no matter whose server sends it — must route through that one project. It's configured to accept unauthenticated sends, so each self-hosted server delivers to it with no shared secret. Your server holds only your own users' push tokens; there's no central server pooling them.

On startup you'll see `Mobile Push: no EXPO_ACCESS_TOKEN set, using unauthenticated mode (lower rate limits)` — that's expected and correct for the App Store build. The rate limits are far above anything an approval workflow will reach.

**Your own EAS build.** If you build and ship your own copy of the app (your own bundle ID, your own Apple/Expo accounts), push runs entirely on your infrastructure and you set your own token:

1. Create an [Expo account](https://expo.dev/signup) and link a project (`EXPO_PROJECT_ID`, `EXPO_OWNER`, `APP_BUNDLE_ID` in `mobile/.env` — see [mobile/README.md](../mobile/README.md)).
2. Configure push credentials in EAS (`eas credentials` for iOS APNs; Android FCM as needed).
3. Build and install the app on your devices ([mobile-builds.md](mobile-builds.md)).
4. Generate an access token at [expo.dev/settings/access-tokens](https://expo.dev/settings/access-tokens) with permission to **send push notifications**, and set it in `.env`:

   ```bash
   EXPO_ACCESS_TOKEN=your_expo_access_token
   ```

   Restart and confirm startup logs say `Mobile Push: Expo access token configured (authenticated mode)`.
5. Point the app at your self-hosted URL, sign in, and confirm `POST /api/v1/push-subscriptions` returns `201`.

This path gives you full independence (no dependency on the Permission Slip Expo project) and lets you keep Expo's push-security enforcement on with your own token.

### Troubleshooting

**"Failed to update notification preference" error in the app:**
This was a bug (fixed in build `23c1cb5`) where self-hosted POST/PUT requests arrived with empty bodies due to how React Native's `Request` constructor works. Update to the latest mobile build.

**Notifications aren't arriving:**
1. Check server logs for `skipping notification` — this means `BASE_URL` was empty or not loaded.
2. Check server logs for `[mobilepush]` lines showing push dispatch errors.
3. Confirm the push subscription was registered: look for `POST /api/v1/push-subscriptions` returning `201` in your logs. If you don't see it, sign out and back in to force re-registration.
4. On the phone, go to **Settings → Permission Slip → Notifications** and confirm notifications are allowed. The app's Settings screen will show a warning and an "Open Settings" link if device-level permission is blocked.

**Expo `403` / "Insufficient permissions to send push notifications":**
Your `EXPO_ACCESS_TOKEN` is set but doesn't match the Expo project for the app on the device. On the **App Store build**, leave `EXPO_ACCESS_TOKEN` unset — a token from your personal Expo account is wrong for the official project and triggers this error. On your **own EAS build**, use a token from the same Expo account that built the app (see §3 above).

**Using Cloudflare Tunnel instead of Tailscale:**
Set `BASE_URL` to your Cloudflare Tunnel public URL (e.g. `https://your-subdomain.trycloudflare.com`). Everything else works the same — the server reaches Expo's push API via its own outbound connection, not through a callback to your URL.
