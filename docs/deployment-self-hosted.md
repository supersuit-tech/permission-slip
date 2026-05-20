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

## Step 5: Create Your Account

```bash
DATABASE_PATH=~/permission-slip/data/app.db \
  go run ./cmd/create-user you@example.com 'your-password'
```

Open `https://$PS_HOSTNAME` in your browser and log in.

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

## Hardening (Optional)

Your tunnel is public — anyone with the hostname can reach the login page. Two ways to add a gate in front:

### Cloudflare Access (recommended)

Put a Cloudflare Access policy in front of your tunnel hostname. Users authenticate with email, Google, GitHub, etc. before reaching Permission Slip. Works with both the web UI and the mobile app. Configure it in the [Cloudflare Zero Trust dashboard](https://one.dash.cloudflare.com/).

### Gateway Secret

For mobile-only deployments, set `GATEWAY_SECRET` so the server rejects any request without a matching `X-Gateway-Secret` header:

```bash
echo "GATEWAY_SECRET=$(openssl rand -hex 32)" >> ~/permission-slip/.env
sudo systemctl restart permission-slip
```

Configure the **mobile app**: Settings → Server → enable Custom Server → paste the secret into the gateway secret field.

> **Note:** Browsers can't inject custom headers, so the web UI won't work with `GATEWAY_SECRET` set. Use Cloudflare Access if you need browser access.

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

## Alternative Deployments

### Fly.io

Fly handles TLS itself, so you can skip Cloudflare Tunnel:

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

### Tailscale (personal/VPN-only)

If you only need to reach Permission Slip from your own devices, skip Cloudflare Tunnel and use Tailscale instead:

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Set `BASE_URL` to your MagicDNS hostname (e.g. `http://mypi.tailnet.ts.net:8080`). No public hostname, no TLS — the Tailscale network is your security boundary.

### Railway / Render / Other PaaS

1. Connect your repo
2. Set the required environment variables in the platform dashboard
3. Point the health check at `GET /api/health` on port 8080

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

**`https://$PS_HOSTNAME` returns 502 or 1033**
The tunnel is up but Permission Slip isn't. Check `sudo systemctl status permission-slip` and `journalctl -u permission-slip -n 50`.

**`https://$PS_HOSTNAME` returns 1016 ("Origin DNS error")**
The DNS record hasn't propagated yet, or `cloudflared tunnel route dns` wasn't run. Re-run it and wait a minute.

**`cloudflared` won't start**
Check `sudo systemctl status cloudflared`. Common cause: credentials file path in `/etc/cloudflared/config.yml` doesn't match the file you copied. Run `ls /etc/cloudflared/` to verify.

**Health check fails on localhost**
Common causes: missing env vars, bad `DATABASE_PATH`, or the `data/` directory isn't writable.

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
