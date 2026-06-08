# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded. You'll run it on your own machine and reach it privately over Tailscale — no port forwarding, no manual TLS, your own HTTPS hostname. Your instance is **only reachable from devices on your tailnet**, never exposed to the public internet.

> **Recommended hardware:**
> - **x86_64 mini PC (amd64)** — e.g. an Intel N100/NUC-class box. Silent, always-on, and **the only option if you want the Proton Mail connector**, since Proton Bridge ships x86_64 builds only (no ARM). Pick this if in doubt.
> - **Raspberry Pi 5 (4GB+)** — cheaper and lower-power, but ARM-only, so it **cannot run the Proton Mail connector** (see [Email: Proton Mail](#email-proton-mail-bridge)).
>
> The steps below work on any Linux machine, VM, or VPS. Where the Go install differs by CPU architecture, both **amd64** (mini PC / most desktops & VMs) and **arm64** (Raspberry Pi) commands are shown — run the one that matches your hardware (`uname -m` prints `x86_64` for amd64, `aarch64` for arm64). You need **Go 1.24+** and **Node.js 20+** to build from source.

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

---

## Step 1: Get the Binary

```bash
# Install Go — pick the line that matches your CPU (run `uname -m` to check)

# amd64 / x86_64 (mini PC, most desktops & VMs):
wget https://go.dev/dl/go1.24.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz

# arm64 / aarch64 (Raspberry Pi):
# wget https://go.dev/dl/go1.24.1.linux-arm64.tar.gz
# sudo tar -C /usr/local -xzf go1.24.1.linux-arm64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 22 via nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source ~/.bashrc
nvm install 22 && nvm use 22

# Clone and build
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
make install
make build
```

> **Faster build:** Cross-compile on a beefier development machine and copy the binary over. Set `GOARCH` to match the target host:
> ```bash
> # Target an amd64 / x86_64 mini PC:
> GOOS=linux GOARCH=amd64 CGO_ENABLED=0 make build
> scp bin/server user@minipc.local:~/permission-slip/bin/server
>
> # Target an arm64 Raspberry Pi:
> GOOS=linux GOARCH=arm64 CGO_ENABLED=0 make build
> scp bin/server pi@raspberrypi.local:~/permission-slip/bin/server
> ```

---

## Step 2: Set Up Tailscale Serve

Install Tailscale on the server:

```bash
curl -fsSL https://tailscale.com/install.sh | sh
```

Sign in and join your tailnet (this prints a browser URL — open it and authenticate):

```bash
sudo tailscale up
```

> **Headless server? Use an auth key instead.** If the machine has no browser (or you're scripting the setup), generate a one-time auth key at [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys) — leave **Reusable** off (single-use), leave **Ephemeral** off (so the server stays registered when offline), then run:
>
> ```bash
> sudo tailscale up --authkey=tskey-auth-xxxxx --hostname=permissions
> ```
>
> Once registered, open [Machines](https://login.tailscale.com/admin/machines) → click the machine → **Disable key expiry** so you don't have to re-authenticate every ~180 days. The auth key itself is only needed for this one-time bootstrap — don't bake it into a config file or systemd unit.

**Enable HTTPS for your tailnet.** In the [Tailscale admin console](https://login.tailscale.com/admin/dns), under **DNS**, click **Enable HTTPS**. This unlocks free Let's Encrypt certificates on your `<tailnet-name>.ts.net` domain.

> (Optional) Give your server a memorable name in the [Machines page](https://login.tailscale.com/admin/machines) — click the machine, then **Edit machine name**. The final hostname will be `<machine-name>.<tailnet-name>.ts.net`.

Capture the hostname so the rest of the guide is copy-paste:

```bash
export PS_HOSTNAME=$(tailscale status --json | jq -r '.Self.DNSName' | sed 's/\.$//')
echo "Permission Slip will be reachable at: https://$PS_HOSTNAME"
```

> These shell variables are **only needed during setup** — they get written into config files and aren't referenced again. No need to add them to `.bashrc`.

Expose Permission Slip on that hostname over HTTPS:

```bash
sudo tailscale serve --bg --https=443 8080
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

# Fill in the secrets
sed -i "s|SECRET_ENCRYPTION_KEY=replace-me|SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|JWT_SIGNING_SECRET=replace-me|JWT_SIGNING_SECRET=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|INVITE_HMAC_KEY=replace-me|INVITE_HMAC_KEY=$(openssl rand -hex 32)|" ~/permission-slip/.env
```

That's it — `BASE_URL` is your tailnet HTTPS hostname, and `ALLOWED_ORIGINS` doesn't need to be set (the server allows the origin the browser used).

---

## Step 4: Run on Boot (systemd)

```bash
sudo tee /etc/systemd/system/permission-slip.service > /dev/null <<EOF
[Unit]
Description=Permission Slip
After=network.target tailscaled.service

[Service]
Type=simple
User=$(whoami)
WorkingDirectory=$HOME/permission-slip
EnvironmentFile=$HOME/permission-slip/.env
ExecStart=$HOME/permission-slip/bin/server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now permission-slip
```

Verify both services are healthy:

```bash
curl http://localhost:8080/api/health           # local check
curl https://$PS_HOSTNAME/api/health            # through Tailscale (run from any tailnet device)
```

---

## Updating an existing install

To pull the latest release and apply it, run one command from the repo:

```bash
cd ~/permission-slip
make redeploy
```

This pulls `origin/main`, reinstalls dependencies, rebuilds the frontend and
server, restarts the systemd service, and prints the build it's now running
(the same short SHA shown in the app footer, so you can confirm the update took
effect). New connector fields, manifest changes, and migrations are picked up on
the restart — there's nothing else to remember.

> The running server is only ever replaced by a **successful** build. If the
> build fails, `make redeploy` stops before restarting and your service keeps
> serving the previous working binary. A transient `git pull` failure is
> non-fatal — it rebuilds the current checkout instead.

> **Named the service something other than `permission-slip`?** Pass it through:
> ```bash
> PS_SERVICE=my-unit make redeploy
> ```

> **Cross-compiling on a beefier box?** `make redeploy` builds and restarts on
> the machine it runs on. If you build elsewhere and `scp bin/server` over (see
> the build note in Step 1), pull + build there, copy the binary, then restart
> on the host with `sudo systemctl restart permission-slip`.

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
   sudo systemctl restart permission-slip
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
   sudo systemctl restart permission-slip
   ```

When a user connects Slack from Permission Slip, they'll complete Slack's OAuth consent and install the app to their workspace.

---

## Email: Proton Mail (Bridge)

Gmail is covered by the [Google connector](#step-5-connect-google) (OAuth). **Proton Mail** is a built-in connector that uses [Proton Mail Bridge](https://proton.me/mail/bridge) — Proton's official local IMAP/SMTP proxy — on the same machine as Permission Slip. There is no cloud OAuth flow.

Bridge runs on x86_64 Linux, macOS, and Windows. Proton does not publish an ARM build, so Raspberry Pi and other ARM boards are not supported hosts for the Proton Mail connector. If you're on ARM, we recommend running Permission Slip on a small x86_64 mini PC with the specs you need.

1. Install and run Bridge headless (systemd user unit). Full steps: **[Proton Mail connector guide](connectors/protonmail.md)**.
2. In the Permission Slip UI, add **Proton Mail** credentials with your Proton address and the bridge password printed by Bridge.
3. Grant agent permissions using the Proton templates (send, read inbox, search, read message, archive).

The proxy must be running when you save credentials; validation performs a real IMAP LOGIN.

---

## Other Connectors

Permission Slip ships with 15+ more OAuth providers — Atlassian (Jira), Datadog, Dropbox, Figma, GitHub, HubSpot, Linear, Meta (Facebook/Instagram), Microsoft, Notion, PagerDuty, Square, PayPal, Stripe, and X (Twitter). See the [OAuth setup guide](oauth-setup.md) for per-provider instructions.

To build your own connector for a service Permission Slip doesn't yet support, see [custom connectors](custom-connectors.md).

---

## Set Up Tailscale on Your Openclaw Machines

Because your Permission Slip instance is private to your tailnet, **any machine that runs Openclaw (or any other tool that calls the Permission Slip API) also needs Tailscale installed** — otherwise it can't reach `https://$PS_HOSTNAME`.

- **Interactive machines (laptop, dev box):**
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

### 3. `EXPO_ACCESS_TOKEN` (Expo project must match the mobile app)

Expo push tokens are tied to the **Expo project that built the app**, not to your Permission Slip server URL. When you send a push, Expo checks that your server is allowed to deliver to that project.

Set in `.env`:

```bash
EXPO_ACCESS_TOKEN=your_expo_access_token
```

Generate the token at [expo.dev/settings/access-tokens](https://expo.dev/settings/access-tokens). When creating it, enable permission to **send push notifications** (wording may vary; see Expo's [additional push security](https://docs.expo.dev/push-notifications/sending-notifications/#additional-security) docs).

Restart the server and confirm startup logs say `Mobile Push: Expo access token configured (authenticated mode)`.

**Which Expo account?** The token must belong to the **same Expo project** as the app on your phone:

| Mobile app | What you need |
|------------|----------------|
| **App Store / official build** | An access token for the **Permission Slip** Expo project (`@supersuit-tech/permission-slip`). A token from your personal Expo account will **not** work — Expo returns `403` / "Insufficient permissions to send push notifications". Ask the project maintainers for a self-host token, or use a custom build (below). |
| **Your own EAS build** | Your own Expo account: run `npx eas-cli init` in `mobile/`, configure iOS APNs (and Android FCM) in EAS, build and install that app, then set `EXPO_ACCESS_TOKEN` from **that** account. See [Mobile builds](mobile-builds.md). |

**Self-hosted push with your own Expo (recommended if you cannot use the official token):**

1. Create an [Expo account](https://expo.dev/signup) and link a project (`EXPO_PROJECT_ID`, `EXPO_OWNER` in `mobile/.env` — see [mobile/README.md](../mobile/README.md)).
2. Configure push credentials in EAS (`eas credentials` for iOS APNs; Android FCM as needed).
3. Build and install the app on your devices ([mobile-builds.md](mobile-builds.md)) — do **not** rely on the App Store build for this path.
4. Create an access token with push permission and set `EXPO_ACCESS_TOKEN` on your server.
5. Point the app at your self-hosted URL, sign in, and confirm `POST /api/v1/push-subscriptions` returns `201`.

Without a matching token, registration can still succeed (`201`) while delivery fails when an approval is created.

### Troubleshooting

**"Failed to update notification preference" error in the app:**
This was a bug (fixed in build `23c1cb5`) where self-hosted POST/PUT requests arrived with empty bodies due to how React Native's `Request` constructor works. Update to the latest mobile build.

**Notifications aren't arriving:**
1. Check server logs for `skipping notification` — this means `BASE_URL` was empty or not loaded.
2. Check server logs for `[mobilepush]` lines showing push dispatch errors.
3. Confirm the push subscription was registered: look for `POST /api/v1/push-subscriptions` returning `201` in your logs. If you don't see it, sign out and back in to force re-registration.
4. On the phone, go to **Settings → Permission Slip → Notifications** and confirm notifications are allowed. The app's Settings screen will show a warning and an "Open Settings" link if device-level permission is blocked.

**Expo `403` / "Insufficient permissions to send push notifications":**
Your `EXPO_ACCESS_TOKEN` does not match the Expo project for the app on the device (common with the App Store build plus a personal Expo token). Use a token from the app's project, or switch to your own EAS build and your own token (see §3 above).

**Using Cloudflare Tunnel instead of Tailscale:**
Set `BASE_URL` to your Cloudflare Tunnel public URL (e.g. `https://your-subdomain.trycloudflare.com`). Everything else works the same — the server reaches Expo's push API via its own outbound connection, not through a callback to your URL.
