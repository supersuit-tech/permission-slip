# Self-Hosted Deployment Guide — Linux

> **Looking for the default recommendation?** Most self-hosters should use the **[Mac Mini / macOS guide](deployment-self-hosted.md)** instead — it's simpler to keep on 24/7 and runs every connector (including Proton Mail) natively regardless of chip. Use this Linux guide if you'd rather run on a mini PC, Raspberry Pi, VM, or VPS.

Permission Slip ships as a **single Go binary** with the React frontend embedded. You'll run it on your own machine and reach it privately over Tailscale — no port forwarding, no manual TLS, your own HTTPS hostname. Your instance is **only reachable from devices on your tailnet**, never exposed to the public internet.

> **Recommended Linux hardware:**
> - **x86_64 mini PC (amd64)** — e.g. an Intel N100/NUC-class box. Silent, always-on, and **the only Linux option if you want the Proton Mail connector**, since Proton Bridge's Linux build is x86_64 only (no ARM). Pick this if in doubt.
> - **Raspberry Pi 5 (4GB+)** — cheaper and lower-power, but ARM-only, so it **cannot run the Proton Mail connector** on Linux (see [Email: Proton Mail](#email-proton-mail-bridge)).
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

### Rate limiting (optional)

In production (any `MODE` other than `development`), the server enforces per-IP and per-agent rate limits on API traffic. Rate limiting is disabled entirely when `MODE=development`.

When running behind a reverse proxy, set `TRUSTED_PROXY_HEADER` (default: `X-Forwarded-For`) so the per-IP limiter keys on the real client IP rather than the proxy's.

To tune limits without a code change, set these optional env vars in `.env` (invalid values log a warning and fall back to the default):

| Env var | Default | Description |
|---|---|---|
| `RATE_LIMIT_IP_RATE` | `50` | Per-IP sustained requests/second |
| `RATE_LIMIT_IP_BURST` | `100` | Per-IP burst capacity |
| `RATE_LIMIT_IP_GLOBAL_RATE` | `200` | Global sustained requests/second (all IPs) |
| `RATE_LIMIT_IP_GLOBAL_BURST` | `400` | Global burst capacity |
| `RATE_LIMIT_AGENT_RATE` | `20` | Per-agent sustained requests/second (post-auth) |
| `RATE_LIMIT_AGENT_BURST` | `40` | Per-agent burst capacity |

Values are read once at startup; restart the service after changing them.

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
server, restarts the systemd service, publishes an over-the-air mobile update
via EAS when configured, and prints a deploy summary with web, mobile, and CLI
release versions so you can confirm the update took effect. New connector
fields, manifest changes, and migrations are picked up on the restart — there's
nothing else to remember.

If EAS isn't set up on the host (no `EXPO_TOKEN` or `eas login`, etc.),
`make redeploy` prints a note and skips the mobile step — the server redeploy
still completes normally.

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

Bridge's Linux build is x86_64 only — Proton does not publish an ARM build for Linux, so Raspberry Pi and other ARM boards can't run the Proton Mail connector under this Linux guide. If you're on ARM and want Proton Mail, either switch to an x86_64 mini PC here, or use the **[Mac Mini / macOS guide](deployment-self-hosted.md)** instead, since Proton ships a native Apple Silicon build of Bridge for macOS.

1. Install and run Bridge headless (systemd user unit). Full steps: **[Proton Mail connector guide](connectors/protonmail.md)**.
2. In the Permission Slip UI, add **Proton Mail** credentials with your Proton address and the bridge password printed by Bridge.
3. Grant agent permissions using the Proton templates (send, read inbox, search, read message, archive).

The proxy must be running when you save credentials; validation performs a real IMAP LOGIN.

---

## iMessage gateway (Mac)

Permission Slip on Linux cannot run `imsg` natively — iMessage reads and sends require macOS and Messages.app. You can point the built-in **iMessage** connector at a separate Mac that acts as a gateway over SSH.

### Mac gateway prerequisites

On the Mac (not the Linux host):

- **Same Apple ID** signed into Messages.app on the Mac and on your iPhone.
- macOS 14+ with Messages.app signed in.
- [imsg](https://github.com/openclaw/imsg): `brew install steipete/tap/imsg`
- **Full Disk Access** for reads — grant this to the process that runs `imsg` over SSH (typically `sshd` / `/usr/sbin/sshd` when Permission Slip connects remotely). See the [macOS deployment guide](deployment-self-hosted.md#persistent-full-disk-access-optional-recommended-for-imessage) for signing and persistence notes if the Mac also runs Permission Slip.
- **Automation** permission for sends (Messages.app) — see the verification step below.
- **Text Message Forwarding** enabled on your iPhone, pointed at this Mac.
- Tailscale (or another private route) so the Linux host can reach the Mac over SSH.

### SSH access from the Linux host

1. On the Mac, enable **Remote Login** (**System Settings → General → Sharing → Remote Login**).
2. On the Linux host, add an SSH config alias (e.g. `~/.ssh/config`):
   ```
   Host messages-mac
     HostName 100.x.x.x
     User your-mac-username
     IdentityFile ~/.ssh/id_ed25519
   ```
   Use the Mac's tailnet IP (or LAN address). Confirm key-based login works:
   ```bash
   ssh -T messages-mac imsg --version
   ```
3. In Permission Slip, add **iMessage** credentials with **Remote SSH host** set to `messages-mac` (just the alias — Permission Slip runs `ssh -T messages-mac imsg …` internally).

Saving credentials runs a read-only probe (`chats.list`) over SSH. That exercises Full Disk Access but does **not** trigger the Automation prompt.

### Verify Automation at the Mac's desktop (required for sends)

The Automation TCC prompt appears only on the **Mac's GUI session**, not in your SSH terminal on the Linux box. While physically at the Mac (or screen-sharing into it), run:

```bash
imsg send --to <your-own-number-or-email> --text "setup test"
```

Click **Allow** when macOS prompts for Automation (Messages.app) and confirm the message arrives.

When Permission Slip on Linux triggers sends over SSH, the responsible process on the Mac is the SSH session context — verify Automation is granted for the process that actually runs `imsg` (check **System Settings → Privacy & Security → Automation** after the test send).

**If you skip this step**, sends from the Linux host hang or time out with no visible error on either machine. The TCC prompt blocks in the Mac's GUI only; if nobody is logged in or watching that screen, the send looks like a network failure. If you previously denied the prompt, macOS will not ask again — re-enable under **System Settings → Privacy & Security → Automation**, or run `tccutil reset AppleEvents` on the Mac to force fresh prompts.

For action details, send policy, and permission granularity, see the [iMessage connector README](../connectors/imessage/README.md).

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

## OpenClaw push wakes

For reliable non-blocking approvals, register the OpenClaw gateway **hooks** endpoint so Permission Slip can push a wake when a human resolves an approval.

### 1. Enable hooks on the OpenClaw gateway

```json5
{
  hooks: {
    enabled: true,
    token: "<your-hooks-token>",
    path: "/hooks",
    allowRequestSessionKey: true,
    allowedSessionKeyPrefixes: ["hook:", "agent:main:"],
  },
}
```

Include **`"hook:"`** in `allowedSessionKeyPrefixes`, not just your agent session prefix — OpenClaw's default session key (commonly `"hook:ingress"`) must pass the prefix check too. **No custom `hooks.mappings` needed** — Permission Slip POSTs directly to built-in `/hooks/agent` and `/hooks/wake`.

Restart the gateway after changing config.

### 2. Verify reachability from the Permission Slip server

```bash
curl -sS -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer <your-hooks-token>" \
  -H "Content-Type: application/json" \
  -d '{"text":"ping","mode":"now"}' \
  http://100.x.x.x:18789/hooks/wake
```

Expect `200` from the server host over tailnet.

### 3. Register and test

From the agent settings page (**Push Wake Webhook**) or via CLI:

```bash
permission-slip webhook set --url http://100.x.x.x:18789/hooks --token <your-hooks-token>
permission-slip webhook status --test
```

The dashboard **Test wake** button runs the same check as `webhook status --test`.

Grok Bot uses `--provider grokbot` and a Cursor automation webhook URL instead of a private OpenClaw hooks URL. See [Grok Bot push wakes](../integrations/openclaw.md#grok-bot).

### 4. Heartbeat backstop

Enable OpenClaw heartbeat and run `permission-slip pending` each beat. See [OpenClaw integration](../integrations/openclaw.md).

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
