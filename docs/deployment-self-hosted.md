# Self-Hosted Deployment Guide

Permission Slip ships as a **single Go binary** with the React frontend embedded — no separate web server, database server, or external auth service needed.

> **Recommended hardware: Raspberry Pi 5 (4GB+).** Cheap, silent, always-on, and keeps your approval server on your own network. The steps below use a Pi as the example but work on any Linux machine, VM, or VPS. You need **Go 1.24+** and **Node.js 20+** to build from source.

```
┌─────────────────────────────────────────────┐
│           Permission Slip Server            │
│  ┌──────────────┐  ┌────────────────────┐   │
│  │  Go API      │  │  Embedded React    │   │
│  │  (port 8080) │  │  Frontend          │   │
│  └──────┬───────┘  └────────────────────┘   │
└─────────┼────────────────────────────────────┘
          │
    ┌─────▼─────┐
    │  SQLite   │
    └───────────┘
```

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

## Step 2: Configure

Find your Pi's IP address — you'll use it for `BASE_URL` and to access the web UI:

```bash
hostname -I | awk '{print $1}'
```

> **Tip:** Assign a static DHCP lease in your router so the IP doesn't change after reboots (look for "DHCP reservation" in your router's admin UI). On macOS and Linux, `raspberrypi.local` also works as an alternative to the IP address.

Create a `.env` file:

```bash
mkdir -p ~/permission-slip/data
cat > ~/permission-slip/.env <<EOF
# SQLite database path (created automatically on first run)
DATABASE_PATH=$HOME/permission-slip/data/app.db

# Secrets — generate each with: openssl rand -base64 32
SECRET_ENCRYPTION_KEY=replace-me
JWT_SIGNING_SECRET=replace-me
INVITE_HMAC_KEY=replace-me

# Public URL — required for OAuth connector callbacks; use your IP or hostname
# BASE_URL=http://192.168.1.100:8080
BASE_URL=
EOF

# Fill in the secrets
sed -i "s|SECRET_ENCRYPTION_KEY=replace-me|SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|JWT_SIGNING_SECRET=replace-me|JWT_SIGNING_SECRET=$(openssl rand -base64 32)|" ~/permission-slip/.env
sed -i "s|INVITE_HMAC_KEY=replace-me|INVITE_HMAC_KEY=$(openssl rand -hex 32)|" ~/permission-slip/.env
```

> **`ALLOWED_ORIGINS`** does not need to be set. The server automatically allows requests from whatever address the browser used to reach it.

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

Verify it's running:

```bash
curl http://localhost:8080/api/health
```

---

## Step 4: Create Your Account

```bash
DATABASE_PATH=~/permission-slip/data/app.db \
  go run ./cmd/create-user you@example.com 'your-password'
```

Then open `http://raspberrypi.local:8080` (or your IP) and log in.

> If `raspberrypi.local` doesn't resolve, use the IP address from Step 2. Not all networks support mDNS.

---

## Notifications (Optional)

The base setup works without notifications — you can poll the web UI or use the iPhone app. When you're ready:

### Web Push (no account needed)

```bash
cd ~/permission-slip
go run ./cmd/generate-vapid-keys
```

Add the output (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) to `.env` and restart.

### Email (SMTP)

```bash
NOTIFICATION_EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=you@gmail.com
SMTP_PASSWORD=your-app-password
NOTIFICATION_EMAIL_FROM=you@gmail.com
```

### SMS (Amazon SNS)

1. Create an AWS account and an IAM user with `sns:Publish` permission:
   ```json
   {"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sns:Publish","Resource":"*"}]}
   ```
2. Exit the [SMS Sandbox](https://docs.aws.amazon.com/sns/latest/dg/sns-sms-sandbox.html) in the SNS console.
3. Set the environment variables:

| Variable | Description |
|---|---|
| `AWS_REGION` | AWS region (e.g. `us-east-1`) — **required** |
| `AWS_ACCESS_KEY_ID` | AWS access key (optional with IAM roles) |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key (optional with IAM roles) |
| `SNS_SMS_SENDER_ID` | Optional alphanumeric sender ID |
| `SNS_SMS_ORIGINATION_NUMBER` | Origination number in E.164 format (required for US) |

---

## Internet Access (Optional)

To approve requests when away from home:

### Cloudflare Tunnel (recommended — free, no port forwarding)

```bash
# Install cloudflared
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update && sudo apt install -y cloudflared

# Create and run a tunnel
cloudflared tunnel login
cloudflared tunnel create permission-slip
cloudflared tunnel route dns permission-slip permissions.yourdomain.com
cloudflared tunnel run --url http://localhost:8080 permission-slip
```

Update `BASE_URL` in `.env` to your public URL and restart. `ALLOWED_ORIGINS` does not need to be set.

### Tailscale (good for personal use)

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Access via your Tailscale IP or MagicDNS hostname. Set `BASE_URL` to your MagicDNS hostname if you use OAuth connectors over Tailscale.

### Gateway Secret

When your server is reachable from the internet, set `GATEWAY_SECRET` to reject requests without a matching header — so a leaked hostname alone isn't enough to reach the app:

```bash
echo "GATEWAY_SECRET=$(openssl rand -hex 32)" >> ~/permission-slip/.env
sudo systemctl restart permission-slip
```

Configure the **mobile app**: Settings → Server → enable Custom Server → paste the secret into the gateway secret field.

> **Note:** With `GATEWAY_SECRET` set, the web UI won't work from a browser (browsers can't inject custom headers). Use the mobile app, or Tailscale instead if you need the web UI.

The comparison is constant-time over SHA-256 digests. Genuine CORS preflights are exempt.

---

## Updating

```bash
cd ~/permission-slip
git pull origin main
make build
sudo systemctl restart permission-slip
```

Database migrations run automatically on startup.

---

## Other Deployment Options

### Fly.io

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

See the dedicated [Fly.io deployment guide](deployment.md) for full details.

### Railway / Render / Other PaaS

1. Connect your repo
2. Set the required environment variables in the platform dashboard
3. Point the health check at `GET /api/health` on port 8080

---

## TLS / Reverse Proxy

The server listens on plain HTTP. Terminate TLS in front of it:

- **Fly.io** — handles TLS automatically
- **nginx / Caddy** — reverse proxy to `localhost:8080`
- **Cloudflare Tunnel** — TLS handled by Cloudflare

If using a reverse proxy, set `TRUSTED_PROXY_HEADER` to the header your proxy uses for the real client IP (e.g., `X-Forwarded-For` or `X-Real-IP`).

---

## Scaling

Permission Slip is stateless (all state is in the database):

- **SQLite (default):** Single instance only.
- **PostgreSQL:** Run as many copies as needed behind a load balancer. Set `DATABASE_URL` instead of `DATABASE_PATH`.
- **VAPID keys:** When running multiple instances, set keys explicitly so all instances share the same pair.

---

## Secret Rotation

| Secret | Effect of rotation |
|--------|-------------------|
| `JWT_SIGNING_SECRET` | Invalidates all active sessions — users must log in again |
| `SECRET_ENCRYPTION_KEY` | Requires re-encrypting all stored credentials — do not rotate without a migration plan |
| `INVITE_HMAC_KEY` | Invalidates pending invite links (accepted invites unaffected) |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` | Invalidates all push subscriptions — users must re-subscribe |
| `GATEWAY_SECRET` | Every client must be updated simultaneously — no overlap window |

Recommended cadence: every 90 days for `JWT_SIGNING_SECRET` and `INVITE_HMAC_KEY`.

---

## Troubleshooting

**"required configuration value(s) missing" on startup**
Check that `DATABASE_PATH`, `JWT_SIGNING_SECRET`, and `SECRET_ENCRYPTION_KEY` are all set.

**Health check fails after deploy**
Check logs. Common causes: missing env vars, bad `DATABASE_PATH`, or the `data/` directory isn't writable.

**Can't reach `raspberrypi.local`**
Not all networks support mDNS. Use the IP address (`hostname -I`) instead. Consider a static DHCP lease in your router.

**Out of memory during build**
On a 4GB Pi, add swap:
```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup && sudo dphys-swapfile swapon
```

**VAPID error on startup**
If any VAPID variable is set, all three (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) must be present.

**Migrations fail**
The `data/` directory must be writable by the user running the server.

**CLI fails with "No route to host"**
The Pi and the machine running the CLI must be on the same subnet without network isolation (e.g., AP client isolation). On macOS, grant Node.js local network access: **System Settings → Privacy & Security → Local Network**, find **node** and toggle it on.

---

## Environment Variable Reference

| Variable | Required | Description |
|---|---|---|
| `DATABASE_PATH` | Yes | SQLite file path (e.g. `/var/lib/permission-slip/app.db`) |
| `SECRET_ENCRYPTION_KEY` | Yes | AES-256-GCM master key — 32 bytes, base64-encoded. `openssl rand -base64 32` |
| `JWT_SIGNING_SECRET` | Yes | HS256 signing key for session tokens — 32+ bytes. `openssl rand -base64 32` |
| `INVITE_HMAC_KEY` | Recommended | HMAC key for invite code signing. `openssl rand -hex 32` |
| `BASE_URL` | Recommended | Public deployment URL (for OAuth callbacks and invite links) |
| `ALLOWED_ORIGINS` | — | Comma-separated CORS origins. Leave unset for same-origin mode. |
| `PORT` | No | Listen port (default: `8080`) |
| `MODE` | No | Set to `development` to relax config validation |
| `TRUSTED_PROXY_HEADER` | No | Client IP header from reverse proxy (default: `X-Forwarded-For`) |
| `SHUTDOWN_TIMEOUT` | No | Graceful shutdown timeout (default: `30s`) |
| `OAUTH_STATE_SECRET` | No | HMAC key for OAuth CSRF state tokens (falls back to `JWT_SIGNING_SECRET`) |
| `GATEWAY_SECRET` | No | Shared secret for private deployments. Rejects requests without `X-Gateway-Secret` header. |
| `VAPID_PUBLIC_KEY` | For Web Push | VAPID public key |
| `VAPID_PRIVATE_KEY` | For Web Push | VAPID private key |
| `VAPID_SUBJECT` | For Web Push | VAPID contact (`mailto:` or `https://`) |
| `NOTIFICATION_EMAIL_PROVIDER` | For email | `twilio-sendgrid` or `smtp` |
| `SENDGRID_API_KEY` | For SendGrid | SendGrid API key |
| `NOTIFICATION_EMAIL_FROM` | For email | Sender email address |
| `SMTP_HOST` | For SMTP | SMTP server hostname |
| `SMTP_PORT` | For SMTP | SMTP port (default: `587`) |
| `SMTP_USERNAME` | For SMTP | SMTP username |
| `SMTP_PASSWORD` | For SMTP | SMTP password |
| `AWS_REGION` | For SMS | AWS region for SNS (e.g. `us-east-1`) |
| `AWS_ACCESS_KEY_ID` | For SMS | AWS access key (optional with IAM roles) |
| `AWS_SECRET_ACCESS_KEY` | For SMS | AWS secret key (optional with IAM roles) |
| `SNS_SMS_SENDER_ID` | No | Alphanumeric SMS sender ID |
| `SNS_SMS_ORIGINATION_NUMBER` | No | Origination phone number (E.164 format) |
| `SMS_NOTIFICATIONS_HIDDEN` | No | Set to `true` to hide SMS from notification preferences UI |
| `SENTRY_DSN` | No | Backend error tracking (Sentry) |
| `SENTRY_CSP_ENDPOINT` | No | CSP violation reporting endpoint |
| `VITE_SENTRY_DSN` | No (build) | Frontend error tracking (Sentry) |
| `SENTRY_AUTH_TOKEN` | No (build) | Sentry source map upload token |
| `SENTRY_ORG` | No (build) | Sentry org slug |
| `SENTRY_PROJECT` | No (build) | Sentry project slug |
| `VITE_POSTHOG_KEY` | No (build) | PostHog project API key |
| `VITE_POSTHOG_HOST` | No (build) | PostHog API host (default: `us.i.posthog.com`) |
| `BILLING_ENABLED` | No | Enable billing (`true`/`false`, default: `false`) |
| `STRIPE_SECRET_KEY` | For billing | Stripe API secret key |
| `STRIPE_PUBLISHABLE_KEY` | For billing | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | For billing | Stripe webhook signing secret |
| `STRIPE_PRICE_ID_REQUEST` | For billing | Metered Stripe Price ID |
| `VITE_STRIPE_PUBLISHABLE_KEY` | For billing (build) | Stripe publishable key (frontend) |
| `CONNECTORS_DIR` | No | Custom connector directory |
| `CUSTOM_CONNECTORS_JSON` | No | Inline connector JSON config |
