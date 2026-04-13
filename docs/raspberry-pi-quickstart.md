# Raspberry Pi Quickstart

Get Permission Slip running on a Raspberry Pi with **zero external accounts**. Everything runs locally — Supabase handles auth and the database in Docker containers, and Permission Slip runs alongside it.

## What You'll Need

### Hardware

- **[Raspberry Pi 5 (8GB)](https://a.co/d/0cQXzRi1)** — recommended; the Supabase stack uses ~1.5GB RAM
- A microSD card (32GB+) or USB SSD (preferred for better performance and longevity)
- Power supply (USB-C, 27W for Pi 5)
- Ethernet cable or Wi-Fi connection

> **4GB Pi 5?** It works, but you'll want to [disable unused Supabase services](#slim-down-supabase-4gb-pi) to free up RAM.

### What Gets Installed

| Component | Purpose | How it runs |
|---|---|---|
| Docker | Container runtime for Supabase | System package |
| Supabase CLI | Manages local auth, database, and vault | arm64 binary |
| PostgreSQL 17 | Application database | Inside Supabase's Docker stack |
| GoTrue | Auth (login, MFA, JWTs) | Inside Supabase's Docker stack |
| Inbucket | Captures auth emails locally | Inside Supabase's Docker stack |
| Permission Slip | The app itself | Docker container or native binary |

**No cloud accounts.** No Supabase subscription, no AWS, no third-party services. Everything runs on the Pi.

## Step 1: Set Up Your Raspberry Pi

If your Pi is already running Raspberry Pi OS (64-bit), skip to Step 2.

1. Download and install [Raspberry Pi Imager](https://www.raspberrypi.com/software/)
2. Flash **Raspberry Pi OS (64-bit)** to your SD card or SSD
3. In the imager settings, enable SSH and configure Wi-Fi (if not using Ethernet)
4. Boot the Pi and SSH in:

```bash
ssh pi@raspberrypi.local
```

## Step 2: Install Docker and Supabase CLI

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Log out and back in for the docker group to take effect
exit
```

SSH back in, then install the Supabase CLI:

```bash
ssh pi@raspberrypi.local

# Install Supabase CLI (arm64)
curl -sSL https://github.com/supabase/cli/releases/latest/download/supabase_linux_arm64.deb -o /tmp/supabase.deb
sudo dpkg -i /tmp/supabase.deb

# Verify
docker --version
supabase --version
```

## Step 3: Start the Local Supabase Stack

Clone the repository — it contains the Supabase configuration:

```bash
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
```

Generate a vault encryption key (Supabase Vault uses this to encrypt stored credentials at rest). The append-if-missing pattern protects any existing `.env`:

```bash
grep -q "^VAULT_SECRET_KEY=" .env 2>/dev/null || echo "VAULT_SECRET_KEY=$(openssl rand -hex 32)" >> .env
```

Start the local Supabase stack. The first run pulls Docker images — expect this to take 5-10 minutes on a Pi:

```bash
supabase start
```

Once it's running, verify and grab your credentials:

```bash
supabase status
```

You'll see output like:

```
         API URL: http://127.0.0.1:54321
          DB URL: postgresql://postgres:postgres@127.0.0.1:54322/postgres
      Studio URL: http://127.0.0.1:54323
    Inbucket URL: http://127.0.0.1:54324
        anon key: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
service_role key: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

The **anon key** is the publishable key used by the frontend. The **DB URL** is the PostgreSQL connection string.

## Step 4: Deploy Permission Slip

### Option A: Docker (recommended)

Build the Docker image. The only build-time argument needed is the Supabase publishable key:

```bash
# Extract the anon key automatically
ANON_KEY=$(supabase status -o env 2>/dev/null | grep '^ANON_KEY=' | cut -d= -f2-)

# Build (takes 10-20 minutes on a Pi 5 — Go + Node.js compilation)
docker build \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY="$ANON_KEY" \
  -t permission-slip .
```

> **Note:** `VITE_SUPABASE_URL` is intentionally omitted. The Go server includes a built-in reverse proxy that routes `/supabase/*` to the local Supabase stack. The frontend uses this automatically when no explicit Supabase URL is baked in, so it works regardless of your Pi's hostname or IP address.

Run the container:

```bash
docker run -d --name permission-slip \
  --network host \
  -e DATABASE_URL="postgresql://postgres:postgres@127.0.0.1:54322/postgres" \
  -e SUPABASE_URL="http://127.0.0.1:54321" \
  -e BASE_URL="http://raspberrypi.local:8080" \
  -e ALLOWED_ORIGINS="http://raspberrypi.local:8080" \
  -e INVITE_HMAC_KEY="$(openssl rand -hex 32)" \
  --restart unless-stopped \
  permission-slip
```

> **`--network host`** lets the container reach Supabase on localhost without Docker networking complexity.

Check that it's healthy:

```bash
docker ps
# Wait ~30 seconds, then:
curl http://localhost:8080/api/health
```

### Option B: Build from Source

If you prefer to run the binary directly:

```bash
# Install Go (arm64)
wget https://go.dev/dl/go1.24.1.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.24.1.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 22
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs

# Install dependencies and build
make install

ANON_KEY=$(supabase status -o env 2>/dev/null | grep '^ANON_KEY=' | cut -d= -f2-)
export VITE_SUPABASE_PUBLISHABLE_KEY="$ANON_KEY"
make build

# Run
export DATABASE_URL="postgresql://postgres:postgres@127.0.0.1:54322/postgres"
export SUPABASE_URL="http://127.0.0.1:54321"
export BASE_URL="http://raspberrypi.local:8080"
export ALLOWED_ORIGINS="http://raspberrypi.local:8080"
export INVITE_HMAC_KEY="$(openssl rand -hex 32)"
./bin/server
```

> **Tip:** To keep the server running after you close the terminal, use a systemd service. See [Running on boot](#running-on-boot) below.

## Step 5: Create Your Account

1. Open your browser and go to:

   ```
   http://raspberrypi.local:8080
   ```

2. Sign up with your email address.

3. Since Supabase runs locally, the confirmation email is captured by **Inbucket** (the built-in email testing server). Open it in another tab:

   ```
   http://raspberrypi.local:54324
   ```

4. Find the email, copy the 6-digit OTP code, and enter it back in the app.

You're in! After the initial signup, your session stays active via JWT refresh — you won't need Inbucket again unless you log out or sign up another user.

> **Can't reach `raspberrypi.local`?** Not all networks support mDNS. Find your Pi's IP with `hostname -I` and use that instead (e.g., `http://192.168.1.100:8080`).

## Running on Boot

### Supabase

Create a systemd service so Supabase starts automatically on boot:

```bash
sudo tee /etc/systemd/system/supabase.service > /dev/null <<'EOF'
[Unit]
Description=Supabase Local Stack
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
User=pi
WorkingDirectory=/home/pi/permission-slip
ExecStart=/usr/bin/supabase start
ExecStop=/usr/bin/supabase stop
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable supabase
```

### Permission Slip (Docker)

The `--restart unless-stopped` flag on the Docker container already handles restarts. Docker starts automatically on boot, and the container restarts with it.

To make Permission Slip wait for Supabase, add a dependency:

```bash
sudo tee /etc/systemd/system/permission-slip-wait.service > /dev/null <<'EOF'
[Unit]
Description=Wait for Supabase before Permission Slip
After=supabase.service
Requires=supabase.service

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'until curl -sf http://localhost:54321/auth/v1/health > /dev/null 2>&1; do sleep 2; done && docker restart permission-slip'
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable permission-slip-wait
```

### Permission Slip (build from source)

```bash
sudo tee /etc/systemd/system/permission-slip.service > /dev/null <<'EOF'
[Unit]
Description=Permission Slip
After=supabase.service docker.service
Requires=supabase.service

[Service]
Type=simple
User=pi
WorkingDirectory=/home/pi/permission-slip
ExecStartPre=/bin/bash -c 'until curl -sf http://localhost:54321/auth/v1/health > /dev/null 2>&1; do sleep 2; done'
ExecStart=/home/pi/permission-slip/bin/server
EnvironmentFile=/home/pi/permission-slip/.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable permission-slip
```

Create `/home/pi/permission-slip/.env` with your runtime environment variables (one per line, `KEY=value` format):

```bash
cat > /home/pi/permission-slip/.env <<'EOF'
DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres
SUPABASE_URL=http://127.0.0.1:54321
BASE_URL=http://raspberrypi.local:8080
ALLOWED_ORIGINS=http://raspberrypi.local:8080
INVITE_HMAC_KEY=your-generated-key-here
EOF
```

## Optional: Add Notifications

The base setup works without any notification service. Approvers can check the web UI manually. When you're ready to add notifications, here are your options — none require a cloud account unless noted:

### Web Push (no account needed)

Browser push notifications work entirely with self-generated VAPID keys:

```bash
# Requires Go (install per Option B above if not already installed)
cd /home/pi/permission-slip
go run ./cmd/generate-vapid-keys
```

Add the output (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) to your `.env` file and restart Permission Slip.

### Email (SMTP)

Use any SMTP server — Gmail with an app password, self-hosted Postfix, Mailgun, etc.:

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

If you want SMS notifications, set up Amazon SNS. See the [Self-Hosted Deployment Guide — SMS Notifications](deployment-self-hosted.md#sms-notifications-amazon-sns--recommended-for-self-hosted) for instructions.

## Optional: Expose to the Internet

To access your Pi from outside your local network:

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

After setting up a tunnel, update `BASE_URL` and `ALLOWED_ORIGINS` in your `.env` to your public URL (e.g., `https://permissions.yourdomain.com`) and restart Permission Slip.

**Tailscale (good for personal use):**

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Access via your Tailscale IP or MagicDNS hostname. No config changes needed since it's a private network.

## Troubleshooting

**`supabase start` fails or hangs:**

Ensure Docker is running and your user is in the `docker` group:

```bash
docker info > /dev/null 2>&1 || echo "Docker is not running"
groups | grep -q docker || echo "User not in docker group — run: sudo usermod -aG docker $USER && exit"
```

If it's pulling images slowly, this is normal on first run. Subsequent starts are fast (~10 seconds).

**Docker container can't connect to Supabase (connection refused):**

If you're using `--network host`, the container shares the host's network and can reach Supabase at `127.0.0.1:54321`. Verify Supabase is running:

```bash
curl http://localhost:54321/auth/v1/health
# Should return {"status":"ready"}
```

If not, run `supabase start` again from the `permission-slip` directory.

**Can't access Inbucket from your laptop:**

Inbucket listens on port 54324. If `http://raspberrypi.local:54324` doesn't load, try the Pi's IP address directly. Docker binds to all interfaces by default, so it should be accessible on the LAN.

**Slow Docker build on the Pi:**

Building the Docker image involves compiling Go and Node.js — expect 10-20 minutes on a Pi 5. This is normal. For faster iteration, use the [build from source](#option-b-build-from-source) approach (incremental builds are much faster after the first one). Alternatively, cross-compile on a faster machine:

```bash
# On your development machine
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 make build
# Copy bin/server to the Pi via scp
```

**Out of memory during build:**

The 8GB Pi 5 should be fine. On a 4GB model, add swap:

```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

**Can't reach `raspberrypi.local`:**

Not all networks support mDNS. Find your Pi's IP address:

```bash
# On the Pi
hostname -I
```

Then use the IP directly (e.g., `http://192.168.1.100:8080`). Consider setting a static IP in your router's DHCP settings for a stable address.

### Slim Down Supabase (4GB Pi)

If you're on a 4GB Pi, you can disable unused Supabase services before running `supabase start` to save ~500MB of RAM. Edit `supabase/config.toml` in your cloned repo:

```bash
cd ~/permission-slip

# Disable Studio (database admin UI)
sed -i '/^\[studio\]$/,/^\[/{s/^enabled = true$/enabled = false/}' supabase/config.toml

# Disable Edge Runtime (serverless functions — not used)
sed -i '/^\[edge_runtime\]$/,/^\[/{s/^enabled = true$/enabled = false/}' supabase/config.toml

# Disable Analytics (log analytics — not needed for personal use)
sed -i '/^\[analytics\]$/,/^\[/{s/^enabled = true$/enabled = false/}' supabase/config.toml
```

Then run `supabase start` as normal. To undo, run `git checkout supabase/config.toml`.

## Updating Permission Slip

To update to a newer version:

```bash
cd ~/permission-slip

# Stop Permission Slip
docker stop permission-slip && docker rm permission-slip
# Or: sudo systemctl stop permission-slip (if using systemd)

# Pull latest code
git pull origin main

# Rebuild
ANON_KEY=$(supabase status -o env 2>/dev/null | grep '^ANON_KEY=' | cut -d= -f2-)
docker build \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY="$ANON_KEY" \
  -t permission-slip .

# Start again (same docker run command from Step 4)
```

Database migrations run automatically on startup — no manual migration step needed.

## What's Next?

- **Add agents:** Follow the [Agent Integration Guide](agents.md) to connect your first AI agent
- **Custom connectors:** Add integrations beyond the built-in ones — see [Custom Connectors](custom-connectors.md)
- **More configuration:** For billing, error tracking, analytics, and more — read the full [Self-Hosted Deployment Guide](deployment-self-hosted.md)
