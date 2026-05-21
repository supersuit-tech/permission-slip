# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded. You'll run it on your own machine and reach it privately over Tailscale — no port forwarding, no manual TLS, your own HTTPS hostname. Your instance is **only reachable from devices on your tailnet**, never exposed to the public internet.

> **Recommended hardware: Raspberry Pi 5 (4GB+).** Cheap, silent, always-on. The steps below use a Pi as the example but work on any Linux machine, VM, or VPS. You need **Go 1.24+** and **Node.js 20+** to build from source.

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
# Install Go (arm64)
wget https://go.dev/dl/go1.24.1.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.24.1.linux-arm64.tar.gz
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

> **Faster build on arm64:** Cross-compile on your development machine and copy the binary over:
> ```bash
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

## Other Connectors

Permission Slip ships with 15+ more OAuth providers — Atlassian (Jira), Datadog, Dropbox, Figma, GitHub, HubSpot, Linear, Meta (Facebook/Instagram), Microsoft, Notion, PagerDuty, Square, PayPal, Stripe, and X (Twitter). See the [OAuth setup guide](oauth-setup.md) for per-provider instructions.

To build your own connector for a service Permission Slip doesn't yet support, see [custom connectors](custom-connectors.md).
