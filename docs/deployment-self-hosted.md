# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded — no separate web server, database server, or external auth service needed.

> **Recommended hardware: Raspberry Pi 5 (4GB+).** It's cheap, silent, always-on, and keeps your approval server on your own network with zero cloud accounts required. All instructions below use a Pi as the example, but the same steps work on any Linux machine, VM, or VPS.

## What You'll Need

### Hardware (Raspberry Pi)

- **Raspberry Pi 5 (4GB or 8GB)** — a Raspberry Pi 4 also works
- A microSD card (32GB+) or USB SSD (preferred for longevity)
- Power supply (USB-C, 27W for Pi 5)
- Ethernet cable or Wi-Fi

### Software prerequisites

To build from source you need **Go 1.24+** and **Node.js 20+** on the machine running the build.

---

## Architecture

```
┌─────────────────────────────────────────────┐
│           Permission Slip Server            │
│  ┌──────────────┐  ┌────────────────────┐   │
│  │  Go API      │  │  Embedded React    │   │
│  │  (port 8080) │  │  Frontend          │   │
│  └──────┬───────┘  └────────────────────┘   │
│         │                                    │
└─────────┼────────────────────────────────────┘
          │
    ┌─────▼─────┐
    │  SQLite   │
    │ (on-disk) │
    └───────────┘
```

Everything runs in one process on one port. Database migrations apply automatically on startup.

---

## Step 1: Get the Binary

**On the Pi itself:**

```bash
# Install Go (arm64)
wget https://go.dev/dl/go1.24.1.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.24.1.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 22 via nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source ~/.bashrc
nvm install 22
nvm use 22

# Clone and build
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
make install
make build
```

> **Faster build on arm64:** Cross-compile on your development machine and copy the binary over:
> ```bash
> # On your Mac or Linux desktop
> GOOS=linux GOARCH=arm64 CGO_ENABLED=0 make build
> scp bin/server pi@raspberrypi.local:~/permission-slip/bin/server
> ```

---

## Step 2: Configure Environment Variables

### Find your Pi's address first

You need to know how other devices on your network will reach the Pi. **Use the local IP address — it's the most reliable option.**

**Option 1 — Local IP address (recommended, e.g. `192.168.1.100`)**
Always works regardless of OS or mDNS support on client devices. The risk is that DHCP can reassign the address after a reboot. Fix that with either approach below — pick one.

```bash
hostname -I | awk '{print $1}'   # prints the Pi's current IP
```

**Option 1a — Static DHCP lease (set it at the router)**
Most routers let you bind a fixed IP to the Pi's MAC address — look for "DHCP reservation" or "address binding" in your router's admin UI. Nothing to configure on the Pi itself.

**Option 1b — Static IP on the Pi**
Set it directly on the Pi so it doesn't rely on the router. The method depends on your OS:

```bash
cat /etc/os-release | grep VERSION_CODENAME   # bookworm/trixie → NetworkManager; bullseye/older → dhcpcd
```

*Bookworm (Raspberry Pi OS 12, 2023+) or Trixie (Raspberry Pi OS 13) — NetworkManager:*
```bash
nmcli connection show   # find your WiFi connection name (often "preconfigured")
sudo nmcli connection modify "preconfigured" \
  ipv4.method manual \
  ipv4.addresses 192.168.1.100/24 \
  ipv4.gateway 192.168.1.1 \
  ipv4.dns "8.8.8.8"
sudo nmcli connection up "preconfigured"
```

*Bullseye (Raspberry Pi OS 11) and earlier — dhcpcd:*
```bash
echo "
interface wlan0
static ip_address=192.168.1.100/24
static routers=192.168.1.1
static domain_name_servers=8.8.8.8" | sudo tee -a /etc/dhcpcd.conf
sudo systemctl restart dhcpcd
```

Replace `192.168.1.100` with your chosen address and `192.168.1.1` with your router's IP (usually ends in `.1`). Verify with `hostname -I` after restarting.

**Option 2 — mDNS hostname (e.g. `raspberrypi.local`)**
Works out of the box on macOS and most Linux desktops, but **may not be reachable from other devices on your network** — Android and some Windows configurations don't support mDNS without extra software ([Bonjour](https://support.apple.com/kb/DL999) or iTunes). Prefer the IP address unless you know all clients on your network support mDNS.

```bash
hostname   # prints e.g. "raspberrypi" → reachable as "raspberrypi.local"
```

Use whichever address you choose in `BASE_URL` below. `ALLOWED_ORIGINS` does not need to be set — the server automatically allows requests from whatever address the browser used to reach it, so it works on all your networks without configuration.

---

Create a `.env` file with your configuration. All values except `BASE_URL` are required in production.

```bash
mkdir -p ~/permission-slip
cat > ~/permission-slip/.env <<EOF
# SQLite database path (created automatically on first run)
DATABASE_PATH=$HOME/permission-slip/data/app.db

# Secrets — generate each with: openssl rand -base64 32
SECRET_ENCRYPTION_KEY=replace-me
JWT_SIGNING_SECRET=replace-me
INVITE_HMAC_KEY=replace-me

# Public URL of this server — used for OAuth connector callback URLs and
# as a fallback base for invite links.
#
# If you use OAuth connectors (Google, Slack, etc.), set this to the address
# you'll register as the redirect URI with each provider. If you access the
# server from multiple networks (e.g. LAN + Tailscale), register all of them
# as redirect URIs in your OAuth app and pick any one here — or use a stable
# hostname (e.g. Tailscale MagicDNS) that works across networks.
#
# If you don't use OAuth connectors, you can leave BASE_URL unset — invite
# links will automatically use the address the request came in on.
#
# Examples:
#   BASE_URL=http://192.168.1.100:8080    # LAN IP
#   BASE_URL=http://raspberrypi.local:8080 # mDNS (macOS/Linux clients only)
#   BASE_URL=http://mypi.tailnet.ts.net:8080 # Tailscale MagicDNS
BASE_URL=

# ALLOWED_ORIGINS — leave unset. The server automatically allows requests from
# whatever address the browser used to reach it (same-origin mode), so it works
# on all your networks without listing specific IPs here.
ALLOWED_ORIGINS=
EOF

# Replace the placeholders with real secrets
sed -i "s|SECRET_ENCRYPTION_KEY=replace-me|SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|JWT_SIGNING_SECRET=replace-me|JWT_SIGNING_SECRET=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|INVITE_HMAC_KEY=replace-me|INVITE_HMAC_KEY=$(openssl rand -hex 32)|" ~/permission-slip/.env
```

```bash
mkdir -p ~/permission-slip/data
```

---

## Step 3: Run on Boot (systemd)

```bash
sudo tee /etc/systemd/system/permission-slip.service > /dev/null <<EOF
[Unit]
Description=Permission Slip
After=network.target

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

Check that it's running:

```bash
curl http://localhost:8080/api/health
```

---

## Step 4: Create Your Account

Create the first user with the bundled CLI tool:

```bash
DATABASE_PATH=~/permission-slip/data/app.db \
  go run ./cmd/create-user you@example.com 'your-password'
```

Then open your browser:

```
http://raspberrypi.local:8080
```

Log in with the email and password you just created.

> **Can't reach the address?** If you used `raspberrypi.local` and it doesn't resolve, your network may not support mDNS — switch to the Pi's IP address. See [Step 2](#step-2-configure-environment-variables) for how to find it and update `BASE_URL`.

---

## Optional: Add Notifications

The base setup works without any notification service — you can poll the web UI or approve from the iPhone app. When you're ready to add push notifications:

### Web Push (no account needed)

```bash
cd ~/permission-slip
go run ./cmd/generate-vapid-keys
```

Add the output (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) to your `.env` file and restart Permission Slip.

### Email (SMTP)

```bash
# Add to .env
NOTIFICATION_EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=you@gmail.com
SMTP_PASSWORD=your-app-password
NOTIFICATION_EMAIL_FROM=you@gmail.com
```

### SMS (requires AWS account)

See the [SMS Notifications](#sms-notifications-amazon-sns) section below for setup.

---

## Optional: Expose to the Internet

To approve requests when you're away from home:

### Cloudflare Tunnel (recommended — free, no port forwarding)

```bash
# Install cloudflared
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update && sudo apt install -y cloudflared

# Authenticate and create a tunnel
cloudflared tunnel login
cloudflared tunnel create permission-slip
cloudflared tunnel route dns permission-slip permissions.yourdomain.com

# Run the tunnel
cloudflared tunnel run --url http://localhost:8080 permission-slip
```

After setting up a tunnel, update `BASE_URL` in your `.env` to your public URL and restart Permission Slip. `ALLOWED_ORIGINS` does not need to be set.

> **Strongly recommended:** Once your Pi is reachable from the internet, set a gateway secret so a leaked hostname alone isn't enough to reach the app. See [Lock Down with a Gateway Secret](#lock-down-the-tunnel-with-a-gateway-secret).

### Tailscale (good for personal use)

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Access via your Tailscale IP or MagicDNS hostname. No CORS config changes needed — the server allows requests from any address automatically. If you use OAuth connectors and want them to work over Tailscale, add your Tailscale address as an additional redirect URI in each OAuth app. Set `BASE_URL` to your Tailscale MagicDNS hostname (e.g. `http://mypi.tailnet.ts.net:8080`) if you primarily access via Tailscale.

### Lock Down the Tunnel with a Gateway Secret

When Permission Slip is exposed through a public URL, set `GATEWAY_SECRET` so the app rejects requests without a matching header:

```bash
echo "GATEWAY_SECRET=$(openssl rand -hex 32)" >> ~/permission-slip/.env
sudo systemctl restart permission-slip
```

Configure the **mobile app**: Settings → Server → enable Custom Server → paste the secret into the gateway secret field.

> **Note:** With `GATEWAY_SECRET` set, the web UI won't work from a browser (browsers can't inject custom headers). Use the mobile app, or leave this unset and rely on Tailscale for access control if you need the web UI.

See [Private Deployments: Gateway Secret](#private-deployments-gateway-secret) for the full reference.

---

## Updating Permission Slip

```bash
cd ~/permission-slip

# Pull latest code
git pull origin main

# Rebuild and restart
make build
sudo systemctl restart permission-slip
```

Database migrations run automatically on startup — no manual migration step needed.

---

## Deployment Options Beyond the Pi

Permission Slip works on any platform that supports Go binaries:

### Fly.io

See the dedicated [Fly.io deployment guide](deployment.md) for full instructions including `fly.toml`, secrets, DNS, and scaling.

Quick version:

```bash
fly launch
fly secrets set \
  DATABASE_PATH=/data/app.db \
  SECRET_ENCRYPTION_KEY="$(openssl rand -base64 32)" \
  JWT_SIGNING_SECRET="$(openssl rand -base64 32)" \
  INVITE_HMAC_KEY="$(openssl rand -hex 32)" \
  BASE_URL="https://your-app.fly.dev" \
  ALLOWED_ORIGINS="https://your-app.fly.dev"
fly deploy
```

### Railway / Render / Other PaaS

1. Connect your repo
2. Set the required environment variables in the platform dashboard
3. Ensure the health check hits `GET /api/health` on port 8080

---

## TLS / Reverse Proxy

The server listens on plain HTTP. In production, terminate TLS in front of it:

- **Fly.io** — handles TLS automatically
- **nginx / Caddy** — reverse proxy to `localhost:8080`
- **Cloudflare Tunnel** — exposes the local service via a tunnel, TLS handled by Cloudflare

If using a reverse proxy other than Fly.io, set `TRUSTED_PROXY_HEADER` to the header your proxy uses for the real client IP (e.g., `X-Forwarded-For` or `X-Real-IP`). The default is `X-Forwarded-For`.

---

## Private Deployments: Gateway Secret

For private deployments (home servers, Raspberry Pis, internal-only instances) you can require a shared secret on every request as an outer access gate. This protects the login page from being probed by anyone who discovers your hostname.

Set `GATEWAY_SECRET` to a long random string. When set, the server rejects any non-preflight request without a matching `X-Gateway-Secret` header and returns `403 Forbidden` before routing, CORS, or auth runs. When unset, the middleware is a no-op.

```bash
export GATEWAY_SECRET="$(openssl rand -hex 32)"
```

**Client configuration:**

- **Mobile app:** In Settings → Server, enable **Custom Server**, enter your deployment URL, and paste the gateway secret. The app stores it in the platform keystore and injects the header on every request.
- **curl / scripts:** Send `X-Gateway-Secret: <your secret>` with every request.
- **Web browser:** Because the header can't be injected by the browser, the web UI is not usable with `GATEWAY_SECRET` enabled. Use the mobile app, or put the web UI behind Tailscale / VPN instead.

**Scope:** Genuine CORS preflights (`OPTIONS` with `Access-Control-Request-Method`) are exempt. The comparison is constant-time over SHA-256 digests to avoid leaking the secret length.

---

## Custom Connectors

Permission Slip ships with built-in GitHub, HubSpot, Slack, and PostgreSQL connectors. To add custom connectors:

**Option A — Inline JSON:**

```bash
export CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.0.0"}]}'
```

**Option B — File on disk:**

Create a `custom-connectors.json` in the project root and run `make install-connectors`. Set `CONNECTORS_DIR` to a persistent path if your filesystem is ephemeral.

See [Custom Connectors](custom-connectors.md) for details on building your own.

---

## Health Check

`GET /api/health` returns:
- `200 OK` when the server is running
- `503 Service Unavailable` if the database is unreachable

Use this endpoint for uptime monitoring.

---

## Scaling

Permission Slip is stateless (all state is in the database), so horizontal scaling works — though SQLite is single-writer by design:

- **SQLite (default):** Single instance only. For multi-instance deployments, switch to PostgreSQL via `DATABASE_URL`.
- **PostgreSQL:** Run as many copies as needed behind a load balancer.
- **VAPID keys:** When running multiple instances, set keys explicitly so all instances use the same pair — don't rely on auto-generation.

---

## Secret Rotation

Rotate secrets on a regular cadence (every 90 days recommended):

- **`JWT_SIGNING_SECRET`** — changing this invalidates all active sessions. Users will need to log in again.
- **`SECRET_ENCRYPTION_KEY`** — requires re-encrypting all stored credentials. Do not rotate without a migration plan.
- **`INVITE_HMAC_KEY`** — regenerate with `openssl rand -hex 32`. **Invalidates pending invite links** (accepted invites are unaffected).
- **`VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY`** — only rotate if compromised. **Invalidates all push subscriptions** (users must re-subscribe).
- **`GATEWAY_SECRET`** — regenerate with `openssl rand -hex 32`. **Every client must be updated simultaneously** — there is no overlap window.

---

## SMS Notifications (Amazon SNS)

SMS is a solid notification channel for self-hosted deployments when configured.

1. **Create an AWS account** at [aws.amazon.com](https://aws.amazon.com)
2. **Create an IAM user** with the `sns:Publish` permission:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{"Effect": "Allow", "Action": "sns:Publish", "Resource": "*"}]
   }
   ```
3. **Request production SMS access** — new accounts are in the [SMS Sandbox](https://docs.aws.amazon.com/sns/latest/dg/sns-sms-sandbox.html) by default. Go to the SNS console → Text messaging → Exit sandbox.
4. **Set the environment variables:**

| Variable | Description |
|---|---|
| `AWS_REGION` | AWS region for SNS (e.g. `us-east-1`) — **required** |
| `AWS_ACCESS_KEY_ID` | AWS access key (optional with IAM roles) |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key (optional with IAM roles) |
| `SNS_SMS_SENDER_ID` | Optional alphanumeric sender ID |
| `SNS_SMS_ORIGINATION_NUMBER` | Optional origination phone number in E.164 format |

> **Tip:** For US destinations, you'll likely need a toll-free number or 10DLC registration. Set `SNS_SMS_ORIGINATION_NUMBER` to your registered number (e.g., `+15551234567`).

---

## Troubleshooting

**Server won't start — "required configuration value(s) missing":**
Check that `DATABASE_PATH`, `JWT_SIGNING_SECRET`, and `SECRET_ENCRYPTION_KEY` are all set.

**Health check fails after deploy:**
Check logs. Common causes: missing env vars, bad file path for `DATABASE_PATH`, or permissions issue on the data directory.

**Can't reach `raspberrypi.local`:**
Not all networks support mDNS. Find your Pi's IP:
```bash
hostname -I
```
Use the IP directly (e.g., `http://192.168.1.100:8080`). Consider a static IP in your router's DHCP settings.

**Out of memory during build:**
On a 4GB Pi, add swap:
```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup && sudo dphys-swapfile swapon
```

**CORS errors in browser (403 on API calls):**
Ensure `ALLOWED_ORIGINS` includes your deployment's exact origin — no trailing slash. When `ALLOWED_ORIGINS` is unset, the server runs in same-origin only mode (which works for the standard embedded-SPA deployment but rejects requests from a different origin).

**VAPID error on startup:**
If any VAPID variable is set, all three (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) must be set. Either set all three or remove them all.

**Migrations fail:**
Check database path permissions. The `data/` directory must be writable by the user running the server.

**CLI fails with "No route to host" or "fetch failed":**
Make sure the Raspberry Pi and the machine running the CLI are on the same network without any network isolation between them. For example, if your Pi is on Ethernet and your laptop is on Wi-Fi, they need to be on the same subnet — a direct Ethernet connection or a router that bridges both interfaces. Network isolation features (such as AP client isolation on some routers) will also block this. The simplest setup is both devices connected to the same router.

> ### ⚠️ macOS: Grant Node.js Local Network Access
>
> On macOS, Node.js must be explicitly granted permission to access devices on
> your local network — without this, the CLI will fail to reach your Pi even if
> the network is set up correctly.
>
> **System Settings → Privacy & Security → Local Network**
>
> Find **"node"** in the list and toggle it **on**.
>
> If Node.js doesn't appear in the list yet, run the CLI command once — macOS
> will prompt you for permission. If the prompt never appeared, check here and
> add it manually.

---

## Complete Environment Variable Reference

| Variable | Required | Build/Runtime | Description |
|---|---|---|---|
| `DATABASE_PATH` | Yes | Runtime | SQLite file path (e.g. `/var/lib/permission-slip/app.db`) |
| `SECRET_ENCRYPTION_KEY` | Yes (with SQLite) | Runtime | AES-256-GCM master key — 32 bytes, base64-encoded. Generate: `openssl rand -base64 32` |
| `JWT_SIGNING_SECRET` | Yes | Runtime | HS256 signing key for session tokens — 32+ bytes. Generate: `openssl rand -base64 32` |
| `INVITE_HMAC_KEY` | Recommended | Runtime | HMAC key for invite code signing. Generate: `openssl rand -hex 32` |
| `BASE_URL` | Recommended | Runtime | Public deployment URL (for OAuth callbacks and invite links) |
| `ALLOWED_ORIGINS` | Recommended | Runtime | Comma-separated CORS origins (exact match, no trailing slash). Same-origin only when unset. |
| `PORT` | No | Runtime | Listen port (default: `8080`) |
| `MODE` | No | Runtime | Set to `development` to relax config validation |
| `TRUSTED_PROXY_HEADER` | No | Runtime | Client IP header when behind a reverse proxy (default: `X-Forwarded-For`) |
| `SHUTDOWN_TIMEOUT` | No | Runtime | Graceful shutdown timeout (default: `30s`) |
| `OAUTH_STATE_SECRET` | No | Runtime | HMAC key for OAuth CSRF state tokens (falls back to `JWT_SIGNING_SECRET`) |
| `GATEWAY_SECRET` | No | Runtime | Shared secret for private deployments. Rejects any non-preflight request without a matching `X-Gateway-Secret` header. No-op when unset. |
| `VAPID_PUBLIC_KEY` | For Web Push | Runtime | VAPID public key |
| `VAPID_PRIVATE_KEY` | For Web Push | Runtime | VAPID private key |
| `VAPID_SUBJECT` | For Web Push | Runtime | VAPID contact (`mailto:` or `https://`) |
| `NOTIFICATION_EMAIL_PROVIDER` | For email | Runtime | `twilio-sendgrid` or `smtp` |
| `SENDGRID_API_KEY` | For SendGrid | Runtime | SendGrid API key |
| `NOTIFICATION_EMAIL_FROM` | For email | Runtime | Sender email address |
| `SMTP_HOST` | For SMTP | Runtime | SMTP server hostname |
| `SMTP_PORT` | For SMTP | Runtime | SMTP port (default: `587`) |
| `SMTP_USERNAME` | For SMTP | Runtime | SMTP username |
| `SMTP_PASSWORD` | For SMTP | Runtime | SMTP password |
| `AWS_REGION` | For SMS | Runtime | AWS region for SNS (e.g. `us-east-1`) |
| `AWS_ACCESS_KEY_ID` | For SMS | Runtime | AWS access key (optional with IAM roles) |
| `AWS_SECRET_ACCESS_KEY` | For SMS | Runtime | AWS secret key (optional with IAM roles) |
| `SNS_SMS_SENDER_ID` | No | Runtime | Alphanumeric SMS sender ID |
| `SNS_SMS_ORIGINATION_NUMBER` | No | Runtime | Origination phone number (E.164 format) |
| `SMS_NOTIFICATIONS_HIDDEN` | No | Runtime | Set to `true` to hide SMS from notification preferences UI |
| `SENTRY_DSN` | No | Runtime | Backend error tracking (Sentry) |
| `SENTRY_CSP_ENDPOINT` | No | Runtime | CSP violation reporting endpoint |
| `VITE_SENTRY_DSN` | No | Build | Frontend error tracking (Sentry) |
| `SENTRY_AUTH_TOKEN` | No | Build | Sentry source map upload token |
| `SENTRY_ORG` | No | Build | Sentry org slug |
| `SENTRY_PROJECT` | No | Build | Sentry project slug |
| `VITE_POSTHOG_KEY` | No | Build | PostHog project API key |
| `VITE_POSTHOG_HOST` | No | Build | PostHog API host (default: `us.i.posthog.com`) |
| `BILLING_ENABLED` | No | Runtime | Enable billing (`true`/`false`, default: `false`) |
| `STRIPE_SECRET_KEY` | For billing | Runtime | Stripe API secret key |
| `STRIPE_PUBLISHABLE_KEY` | For billing | Runtime | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | For billing | Runtime | Stripe webhook signing secret |
| `STRIPE_PRICE_ID_REQUEST` | For billing | Runtime | Metered Stripe Price ID |
| `VITE_STRIPE_PUBLISHABLE_KEY` | For billing | Build | Stripe publishable key (frontend) |
| `CONNECTORS_DIR` | No | Runtime | Custom connector directory |
| `CUSTOM_CONNECTORS_JSON` | No | Runtime | Inline connector JSON config |
