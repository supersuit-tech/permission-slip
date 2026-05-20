# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded. You'll run it on your own machine and expose it to the internet through a free Cloudflare Tunnel — no port forwarding, no manual TLS, your own HTTPS hostname.

> **Recommended hardware: Raspberry Pi 5 (4GB+).** Cheap, silent, always-on. The steps below use a Pi as the example but work on any Linux machine, VM, or VPS. You need **Go 1.24+** and **Node.js 20+** to build from source.

```
 ┌──────────────┐
 │  You, the    │
 │  mobile app  │
 └──────┬───────┘
        │ https://permissions.yourdomain.com
        ▼
 ┌──────────────────┐
 │  Cloudflare      │  TLS termination, DDoS protection
 │  Tunnel          │
 └──────┬───────────┘
        │ encrypted tunnel
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

You need a **Cloudflare account** and a **domain managed by Cloudflare** (free if you already own one — just point its nameservers at Cloudflare; ~$10/year if you register a new one through them). All other steps below are scriptable.

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

## Step 2: Set Up Cloudflare Tunnel

This is the only step where you make a choice: **pick the hostname you'll use for Permission Slip** (e.g. `permissions.yourdomain.com`). Set it as a shell variable so the rest of the guide is copy-paste:

```bash
export PS_HOSTNAME=permissions.yourdomain.com   # ← change this
```

> These shell variables are **only needed during setup** — they get written into config files and aren't referenced again. No need to add them to `.bashrc`.

Install `cloudflared` (use `cloudflared-linux-amd64` on x86 servers):

```bash
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64 -o cloudflared
chmod +x cloudflared && sudo mv cloudflared /usr/local/bin/
```

Authenticate and create the tunnel:

```bash
# Opens a browser — pick the domain that contains $PS_HOSTNAME
cloudflared tunnel login

# Create the tunnel and capture its ID
cloudflared tunnel create permission-slip
export TUNNEL_ID=$(cloudflared tunnel list | grep permission-slip | awk '{print $1}')
```

Write the tunnel config and install it as a system service:

```bash
sudo mkdir -p /etc/cloudflared
sudo cp ~/.cloudflared/$TUNNEL_ID.json /etc/cloudflared/
sudo tee /etc/cloudflared/config.yml > /dev/null <<EOF
tunnel: $TUNNEL_ID
credentials-file: /etc/cloudflared/$TUNNEL_ID.json

ingress:
  - hostname: $PS_HOSTNAME
    service: http://localhost:8080
  - service: http_status:404
EOF

# Route DNS for $PS_HOSTNAME to this tunnel
cloudflared tunnel route dns permission-slip $PS_HOSTNAME

# Install and start the cloudflared service
sudo cloudflared service install
sudo systemctl enable --now cloudflared

# Remove the broad account credentials — the tunnel only needs its own JSON credentials file
rm ~/.cloudflared/cert.pem
```

> **Why delete `cert.pem`?** `cloudflared tunnel login` creates `~/.cloudflared/cert.pem`, scoped to the domain you selected. It grants management-level access to that zone — enough to create/delete DNS records and manage tunnels. Once the tunnel is created and its credentials file is copied to `/etc/cloudflared/`, the running service only needs that tunnel-scoped JSON file, which can do nothing except maintain this specific tunnel's connection. Deleting `cert.pem` removes the unnecessary zone management access.

Your tunnel is now running. Once Permission Slip is up (next steps), it'll be reachable at `https://$PS_HOSTNAME`.

---

## Step 3: Configure Permission Slip

```bash
mkdir -p ~/permission-slip/data
cat > ~/permission-slip/.env <<EOF
DATABASE_PATH=$HOME/permission-slip/data/app.db
BASE_URL=https://$PS_HOSTNAME

# Allow Cloudflare's web analytics beacon through the Content Security Policy.
# Cloudflare injects this script automatically when serving via a tunnel; without
# it you'll see CSP errors in the browser console.
CLOUDFLARE_INSIGHTS=true

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

That's it — `BASE_URL` is your public HTTPS hostname, and `ALLOWED_ORIGINS` doesn't need to be set (the server allows the origin the browser used).

---

## Step 4: Run on Boot (systemd)

```bash
sudo tee /etc/systemd/system/permission-slip.service > /dev/null <<EOF
[Unit]
Description=Permission Slip
After=network.target cloudflared.service

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
curl https://$PS_HOSTNAME/api/health            # through the tunnel
```

---

## Step 5: Connect Google

Permission Slip's Google connector handles Gmail and Calendar actions. To enable it, register an OAuth client in Google Cloud:

1. In the [Google Cloud Console](https://console.cloud.google.com/), create or pick a project.
2. Under **APIs & Services > Library**, enable the **Gmail API** and **Google Calendar API**.
3. Under **APIs & Services > OAuth consent screen**, choose **External** (or **Internal** for Google Workspace). Fill in:
   - App name: `Permission Slip`
   - User support email: your email
   - Authorized domain: your `$PS_HOSTNAME` domain

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
