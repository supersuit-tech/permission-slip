# Raspberry Pi Quickstart

Get Permission Slip running on a Raspberry Pi in under 30 minutes. This is an opinionated guide that gets you to a working self-hosted instance with the minimum number of services and accounts.

## What You'll Need

### Hardware

- **[Raspberry Pi 5 (8GB)](https://a.co/d/0cQXzRi1)** — recommended for running Permission Slip, PostgreSQL, and Docker comfortably
- A microSD card (32GB+) or USB SSD (preferred for better performance and longevity)
- Power supply (USB-C, 27W for Pi 5)
- Ethernet cable or Wi-Fi connection

### Accounts to Sign Up For

You only need **one** external account:

| Service | Why | Cost |
|---|---|---|
| [Supabase](https://supabase.com) | User authentication (login, MFA, JWTs) | Free tier is sufficient |

That's it. PostgreSQL runs locally on the Pi — no managed database needed.

## Step 1: Set Up Your Raspberry Pi

If your Pi is already running Raspberry Pi OS, skip to Step 2.

1. Download and install [Raspberry Pi Imager](https://www.raspberrypi.com/software/)
2. Flash **Raspberry Pi OS (64-bit)** to your SD card or SSD
3. In the imager settings, enable SSH and configure Wi-Fi (if not using Ethernet)
4. Boot the Pi and SSH in:

```bash
ssh pi@raspberrypi.local
```

## Step 2: Install Dependencies

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Log out and back in for group changes to take effect
exit
# SSH back in
ssh pi@raspberrypi.local

# Install PostgreSQL
sudo apt install -y postgresql postgresql-client

# Start PostgreSQL and enable on boot
sudo systemctl enable --now postgresql
```

Verify both are working:

```bash
docker --version
sudo -u postgres psql -c "SELECT version();"
```

## Step 3: Create the Database

```bash
# Create a database and user for Permission Slip
sudo -u postgres psql <<'SQL'
CREATE USER permissionslip WITH PASSWORD 'changeme-use-a-real-password';
CREATE DATABASE permissionslip OWNER permissionslip;
-- Install required extensions for credential vault
\c permissionslip
CREATE EXTENSION IF NOT EXISTS pgsodium;
CREATE EXTENSION IF NOT EXISTS supabase_vault;
SQL
```

> **Note:** The `pgsodium` and `supabase_vault` extensions are needed for credential encryption. If they're not available in your Postgres packages, Permission Slip will still work — you just won't be able to store encrypted credentials in the vault. You can skip those two `CREATE EXTENSION` lines if they fail.

## Step 4: Set Up Supabase Auth

1. Go to [supabase.com](https://supabase.com) and create a free account
2. Create a new project (any region)
3. From your project dashboard, grab:
   - **Project URL** (e.g., `https://abcdefgh.supabase.co`)
   - **Publishable key** (under Project Settings > API)
4. Configure auth settings:
   - **Authentication > URL Configuration > Site URL:** Set to your Pi's URL (e.g., `http://raspberrypi.local:8080` for LAN access, or your domain/tunnel URL)
   - **Authentication > URL Configuration > Redirect URLs:** Add the same URL
   - **Authentication > Email:** Ensure email sign-in is enabled

## Step 5: Deploy with Docker

Create a directory for your deployment:

```bash
mkdir ~/permission-slip && cd ~/permission-slip
```

Create a `docker-compose.yml`:

```yaml
services:
  permission-slip:
    image: ghcr.io/supersuit-tech/permission-slip:latest
    # Or build from source (uncomment below, comment out image above):
    # build:
    #   context: .
    #   args:
    #     VITE_SUPABASE_URL: https://abcdefgh.supabase.co
    #     VITE_SUPABASE_PUBLISHABLE_KEY: your-publishable-key
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://permissionslip:changeme-use-a-real-password@host.docker.internal:5432/permissionslip?sslmode=disable
      SUPABASE_URL: https://abcdefgh.supabase.co
      BASE_URL: http://raspberrypi.local:8080
      ALLOWED_ORIGINS: http://raspberrypi.local:8080
      INVITE_HMAC_KEY: ${INVITE_HMAC_KEY}
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/health"]
      interval: 15s
      timeout: 3s
      retries: 3
```

> **Note:** `host.docker.internal` lets the container reach PostgreSQL running on the host. The `extra_hosts` line makes this work on Linux.

Create a `.env` file with your secrets:

```bash
# Generate an HMAC key for invite codes
echo "INVITE_HMAC_KEY=$(openssl rand -hex 32)" > .env
```

Edit the `docker-compose.yml` to fill in your actual Supabase URL and publishable key, then start it:

```bash
docker compose up -d
```

Check that it's healthy:

```bash
docker compose ps
# Should show "healthy" after ~30 seconds

curl http://localhost:8080/api/health
# Should return 200 OK
```

## Step 6: Build from Source (Alternative)

If you prefer to build and run the binary directly instead of Docker:

```bash
# Install Go
wget https://go.dev/dl/go1.24.1.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.24.1.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Clone and build
git clone https://github.com/supersuit-tech/permission-slip-web.git
cd permission-slip-web
make install

export VITE_SUPABASE_URL=https://abcdefgh.supabase.co
export VITE_SUPABASE_PUBLISHABLE_KEY=your-publishable-key
make build

# Run
export DATABASE_URL="postgres://permissionslip:changeme-use-a-real-password@localhost:5432/permissionslip?sslmode=disable"
export SUPABASE_URL="https://abcdefgh.supabase.co"
export BASE_URL="http://raspberrypi.local:8080"
export ALLOWED_ORIGINS="http://raspberrypi.local:8080"
export INVITE_HMAC_KEY="$(openssl rand -hex 32)"
./bin/server
```

> **Tip:** To keep the server running after you close the terminal, use a systemd service. See [Running as a systemd service](#running-as-a-systemd-service) below.

## Step 7: Access Permission Slip

Open your browser and go to:

```
http://raspberrypi.local:8080
```

Sign up with your email. You'll receive a confirmation email via Supabase Auth.

## Optional: Running as a systemd Service

To start Permission Slip automatically on boot (for the build-from-source approach):

```bash
sudo tee /etc/systemd/system/permission-slip.service > /dev/null <<'EOF'
[Unit]
Description=Permission Slip
After=network.target postgresql.service

[Service]
Type=simple
User=pi
WorkingDirectory=/home/pi/permission-slip-web
ExecStart=/home/pi/permission-slip-web/bin/server
EnvironmentFile=/home/pi/permission-slip-web/.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now permission-slip
```

Create `/home/pi/permission-slip-web/.env` with your environment variables (one per line, `KEY=value` format).

## Optional: Expose to the Internet

To access your Pi from outside your local network, you have a few options:

**Cloudflare Tunnel (recommended — free, no port forwarding needed):**

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

After setting up a tunnel, update your environment:
- `BASE_URL` and `ALLOWED_ORIGINS` → your public URL (e.g., `https://permissions.yourdomain.com`)
- Supabase **Site URL** and **Redirect URLs** → same public URL

**Tailscale (good for personal use):**

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Access via your Tailscale IP or MagicDNS hostname. No config changes needed since it's a private network.

## Optional: Enable Notifications

Push notifications are optional but recommended for the full approval flow experience.

**Web Push (no signup needed):**

```bash
# If using build-from-source, generate VAPID keys:
cd ~/permission-slip-web
make generate-vapid-keys
# Add the output to your .env file
```

**Email notifications (pick one):**

- **Gmail SMTP** (free, easiest for personal use): Use an [App Password](https://myaccount.google.com/apppasswords) and set `NOTIFICATION_EMAIL_PROVIDER=smtp`, `SMTP_HOST=smtp.gmail.com`, `SMTP_PORT=587`, `SMTP_USERNAME=you@gmail.com`, `SMTP_PASSWORD=your-app-password`
- **SendGrid** (free tier: 100 emails/day): Sign up at [sendgrid.com](https://sendgrid.com), set `NOTIFICATION_EMAIL_PROVIDER=twilio-sendgrid` and `SENDGRID_API_KEY`

## Troubleshooting

**Docker container can't connect to PostgreSQL:**

PostgreSQL defaults to rejecting non-local connections. Edit `pg_hba.conf` to allow Docker's network:

```bash
# Find the config file
sudo -u postgres psql -c "SHOW hba_file;"

# Add this line (adjust the subnet to match your Docker network)
# host all all 172.17.0.0/16 md5
sudo nano /etc/postgresql/*/main/pg_hba.conf

# Also ensure PostgreSQL listens on all interfaces
sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/*/main/postgresql.conf

sudo systemctl restart postgresql
```

**Slow builds on the Pi:**

Building from source (especially the frontend) can take several minutes on a Pi. This is normal. If it's too slow, consider cross-compiling on a faster machine:

```bash
# On your development machine (Linux/macOS with Go installed)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 make build
# Then copy bin/server to your Pi
```

**Out of memory during build:**

The 8GB Pi 5 should be fine, but if you're on a 4GB model, add swap:

```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

**Can't reach `raspberrypi.local`:**

Not all networks support mDNS. Find your Pi's IP address instead:

```bash
# On the Pi
hostname -I
```

Then use the IP directly (e.g., `http://192.168.1.100:8080`).

## What's Next?

- **Add agents:** Follow the [Agent Integration Guide](agents.md) to connect your first AI agent
- **Custom connectors:** Add integrations beyond the built-in GitHub and Slack connectors — see [Custom Connectors](custom-connectors.md)
- **Mobile app:** Install the Permission Slip mobile app for push notification approvals from your phone
- **Full configuration:** See the [Self-Hosted Deployment Guide](deployment-self-hosted.md) for all available options (SMS, error tracking, analytics, billing, etc.)
