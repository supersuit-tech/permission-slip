# Self-Hosted Deployment Guide

This guide covers deploying Permission Slip on your own infrastructure. Permission Slip ships as a single Go binary with the React frontend embedded — no separate web server or static file hosting needed.

> **Running on a Raspberry Pi?** Skip straight to the **[Raspberry Pi Quickstart](raspberry-pi-quickstart.md)** — an opinionated, step-by-step guide that gets you up and running in under 30 minutes with minimal services.

## Prerequisites

Before deploying, you'll need:

| Service | Purpose | Notes |
|---|---|---|
| **PostgreSQL 16+** | Application database | Any managed provider works (Supabase, Neon, AWS RDS, self-hosted) |
| **Supabase** | User authentication (JWT-based) | Cloud (free tier) **or** local via `supabase start` — no account needed |
| **Docker** (optional) | Container-based deployment | Required for local Supabase; optional otherwise |

## Architecture Overview

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
    ┌─────▼─────┐     ┌──────────────────┐
    │ PostgreSQL │     │  Supabase Auth   │
    │ (database) │     │  (JWT provider)  │
    └───────────┘     └──────────────────┘
```

The server handles everything on a single port:
- API endpoints under `/api/v1/`
- Health check at `/api/health`
- Supabase Auth reverse proxy at `/supabase/auth/v1/*` (forwards to `SUPABASE_URL`)
- React SPA for all other routes
- Database migrations run automatically on startup

## Step 1: Set Up Supabase Auth

Permission Slip uses [Supabase Auth](https://supabase.com/auth) for user authentication. You need a Supabase project — the free tier works fine.

1. Create a project at [supabase.com](https://supabase.com)
2. From your project dashboard, note these values:
   - **Project URL** (e.g., `https://abcdefgh.supabase.co`) — used as `SUPABASE_URL`
   - **Publishable key** (public) — used as `VITE_SUPABASE_PUBLISHABLE_KEY` at build time
3. Configure the following in the Supabase dashboard:

| Setting | Where | What to set |
|---|---|---|
| **Site URL** | Authentication > URL Configuration | Your deployment URL (e.g., `https://permissions.mycompany.com`). Used as the base URL in all auth emails. |
| **Redirect URLs** | Authentication > URL Configuration | Add your deployment URL to the allow list |
| **Email OTP** | Authentication > Email | Ensure email sign-in is enabled (the primary auth method) |
| **Email templates** | Authentication > Email Templates | Review templates reference your domain, not localhost |
| **MFA (TOTP)** | Authentication > Multi-Factor Authentication | Enable TOTP enroll and verify (if you want MFA support) |
| **Rate limits** | Authentication > Rate Limits | Review and adjust for your expected traffic |
| **Session timeouts** | Authentication > Sessions | Set timebox and inactivity timeout per your security policy |

> **Note:** You can use Supabase's hosted database as your `DATABASE_URL` too, or use a separate PostgreSQL instance. Either works.

### Alternative: Local Supabase (no cloud account)

If you prefer to run everything locally with zero external accounts, you can use the Supabase CLI to run the full auth stack in Docker:

```bash
# Clone the repo (contains supabase/config.toml)
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip

# Generate a vault encryption key (appends to .env if the key isn't already set)
grep -q "^VAULT_SECRET_KEY=" .env 2>/dev/null || echo "VAULT_SECRET_KEY=$(openssl rand -hex 32)" >> .env

# Start local Supabase (pulls Docker images on first run)
supabase start

# Get credentials
supabase status
```

This gives you PostgreSQL, Auth (GoTrue), Vault, and Inbucket (email capture) — all running locally. Use the credentials from `supabase status`:

| `supabase status` field | Environment variable | Example |
|---|---|---|
| DB URL | `DATABASE_URL` | `postgresql://postgres:postgres@127.0.0.1:54322/postgres` |
| API URL | `SUPABASE_URL` | `http://127.0.0.1:54321` |
| anon key | `VITE_SUPABASE_PUBLISHABLE_KEY` | `eyJhbGci...` |

**Built-in Supabase proxy:** The Go server includes a reverse proxy at `/supabase/*` that forwards requests to `SUPABASE_URL`. When `VITE_SUPABASE_URL` is omitted at build time, the frontend uses this proxy automatically. This means:
- No need to expose Supabase ports to the network
- No CORS configuration needed
- The frontend works regardless of hostname or IP changes
- Only port 8080 needs to be reachable

The proxy is deliberately narrow — it **only forwards `/auth/v1/*`** (the Supabase Auth surface, which is all the frontend uses). Requests for other Supabase surfaces (`/rest/v1/`, `/storage/v1/`, `/realtime/v1/`, `/functions/v1/`) return 404. This prevents the proxy from being usable as an open HTTP relay into arbitrary upstream paths.

**Auth emails** are captured by Inbucket at `http://localhost:54324` (no real emails are sent). Check Inbucket for OTP verification codes during signup and login.

> For a complete step-by-step guide with systemd services, notifications, and internet exposure, see the **[Raspberry Pi Quickstart](raspberry-pi-quickstart.md)** — it works on any Linux machine, not just Pis.

## Step 2: Set Up PostgreSQL

You need a PostgreSQL 16+ database. Options:

- **Supabase** (bundled with your auth project) — use the connection pooler URL (port 6543) with `?sslmode=require`
- **Neon** — serverless Postgres, generous free tier
- **AWS RDS / Google Cloud SQL / Azure Database** — managed Postgres
- **Self-hosted** — any PostgreSQL 16+ instance

The server runs migrations automatically on startup — no manual migration step needed.

### Supabase Vault (credential encryption)

Permission Slip uses [Supabase Vault](https://supabase.com/docs/guides/database/vault) for encrypting stored credentials (API keys, OAuth tokens) at rest using AES-256-GCM. This requires:

- The `pgsodium` and `vault` PostgreSQL extensions (included in all Supabase projects)
- A vault encryption key managed at the **database level** (not as an app env var)

> **Note:** `VAULT_SECRET_KEY` is **not** a runtime secret for the Go server. Hosted Supabase manages the pgsodium encryption key automatically. The `VAULT_SECRET_KEY` env var in `.env.example` and `supabase/config.toml` is only used by the local Supabase CLI (`supabase start`). For local dev, generate with `openssl rand -hex 32`.

If you're using a Supabase-hosted database, vault is already available. If using a non-Supabase database, you'll need to install the `pgsodium` and `supabase_vault` extensions manually, or implement an alternative vault backend.

### Row-Level Security (RLS)

RLS is enabled on all application tables via a migration that runs automatically on startup. If you're using a Supabase-hosted database, this locks down the PostgREST data API so the `anon` and `authenticated` roles cannot access tables directly — all access goes through the Go backend. If you're using a non-Supabase PostgreSQL instance, RLS is still enabled but has no practical effect since there's no PostgREST layer exposing the database.

The Go backend connects as a superuser and bypasses RLS entirely — no configuration is needed.

## Step 3: Configure Environment Variables

### Required Variables

| Variable | Description | Example |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/dbname?sslmode=require` |
| `SUPABASE_URL` | Supabase project URL (for JWT verification) | `https://abcdefgh.supabase.co` |

### Recommended Variables

| Variable | Description | Example |
|---|---|---|
| `BASE_URL` | Public URL of your deployment (used for invite links) | `https://permissions.mycompany.com` |
| `ALLOWED_ORIGINS` | Comma-separated CORS origins (exact match, no trailing slash). Defaults to same-origin only when unset — which works out of the box since the SPA is embedded in the binary. Set this if you serve the app behind a reverse proxy or CDN that changes the origin. | `https://permissions.mycompany.com` |
| `INVITE_HMAC_KEY` | HMAC key for invite code signing | Generate: `openssl rand -hex 32` |

### Build-Time Variables (Frontend)

These are inlined into the JavaScript bundle by Vite at build time. They must be passed as `--build-arg` when building Docker images, not as runtime env vars.

| Variable | Description | Example |
|---|---|---|
| `VITE_SUPABASE_URL` | Supabase project URL (frontend auth). **Optional** — when omitted, the frontend uses the built-in `/supabase` reverse proxy (see below). | `https://abcdefgh.supabase.co` |
| `VITE_SUPABASE_PUBLISHABLE_KEY` | Supabase publishable key | From Supabase dashboard or `supabase status` |
| `VITE_SENTRY_DSN` | Frontend error tracking (optional) | `https://key@o0.ingest.sentry.io/0` |

> The publishable key is safe to include in the build — it's a public key visible in client-side JavaScript by design. (Supabase previously called this the "anon key.")

**Supabase reverse proxy:** The Go server proxies `/supabase/auth/v1/*` to `SUPABASE_URL` at runtime. When `VITE_SUPABASE_URL` is omitted from the build, the frontend automatically uses this proxy. Only the Supabase Auth surface is forwarded — other Supabase surfaces (rest, storage, realtime, functions) return 404 to prevent the proxy from being used as an open HTTP relay. This is the recommended approach for self-hosted deployments because it eliminates hostname/IP dependencies in the frontend build and avoids CORS issues. Cloud Supabase deployments should still set `VITE_SUPABASE_URL` for direct browser-to-Supabase communication.

### Web Push Notifications (VAPID)

No external account or signup required — VAPID keys are a self-generated cryptographic key pair.

To enable browser push notifications, set all three:

| Variable | Description | Example |
|---|---|---|
| `VAPID_PUBLIC_KEY` | VAPID public key | Generate with `make generate-vapid-keys` |
| `VAPID_PRIVATE_KEY` | VAPID private key (keep secret) | Generate with `make generate-vapid-keys` |
| `VAPID_SUBJECT` | Contact URL for push services | `mailto:admin@mycompany.com` |

Generate a key pair:

```bash
# .env format
make generate-vapid-keys

# Fly.io format (ready-to-run fly secrets set command)
go run ./cmd/generate-vapid-keys --format=fly

# Heroku format
go run ./cmd/generate-vapid-keys --format=heroku
```

If none are set, Web Push is disabled (the app works fine without it). If partially configured, the server will refuse to start.

> **Warning:** Changing VAPID keys invalidates all existing push subscriptions. Users will need to re-subscribe.

### Email Notifications

Pick one provider:

**SendGrid (recommended):**

| Variable | Value |
|---|---|
| `NOTIFICATION_EMAIL_PROVIDER` | `twilio-sendgrid` |
| `SENDGRID_API_KEY` | Your SendGrid API key (`SG.xxxx`) |
| `NOTIFICATION_EMAIL_FROM` | Sender address (e.g., `approvals@mycompany.com`) |

**SMTP (Gmail, Mailgun, self-hosted, etc.):**

| Variable | Value |
|---|---|
| `NOTIFICATION_EMAIL_PROVIDER` | `smtp` |
| `SMTP_HOST` | SMTP server hostname (e.g., `smtp.gmail.com`) |
| `SMTP_PORT` | SMTP port (default: `587`) |
| `SMTP_USERNAME` | SMTP username |
| `SMTP_PASSWORD` | SMTP password or app password |
| `NOTIFICATION_EMAIL_FROM` | Sender address |

### SMS Notifications (Amazon SNS) — Recommended for Self-Hosted

SMS is the primary notification channel for self-hosted deployments. When configured, it appears automatically in each user's notification preferences — no plan or subscription required.

You'll need an AWS account with Amazon SNS access. Here's how to set it up:

1. **Create an AWS account** (if you don't have one) at [aws.amazon.com](https://aws.amazon.com)
2. **Create an IAM user** (or use an existing one) with the `sns:Publish` permission. The minimal IAM policy:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Action": "sns:Publish",
       "Resource": "*"
     }]
   }
   ```
3. **Request production SMS access** — new AWS accounts are in the [SMS Sandbox](https://docs.aws.amazon.com/sns/latest/dg/sns-sms-sandbox.html) by default. Go to the SNS console → Text messaging (SMS) → Exit sandbox. Until approved, SMS only reaches verified destination numbers.
4. **Set the environment variables:**

| Variable | Description |
|---|---|
| `AWS_REGION` | AWS region for SNS (e.g. `us-east-1`) — **required** |
| `AWS_ACCESS_KEY_ID` | AWS access key (optional — omit to use IAM roles) |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key (optional — omit to use IAM roles) |
| `SNS_SMS_SENDER_ID` | Optional alphanumeric sender ID (not supported in all countries) |
| `SNS_SMS_ORIGINATION_NUMBER` | Optional origination phone number in E.164 format |

`AWS_REGION` is required to enable SMS. Credentials are optional when running on AWS with an IAM role (EC2, ECS, Lambda).

> **Tip:** For US destinations, you'll likely need a toll-free number or 10DLC registration. Set `SNS_SMS_ORIGINATION_NUMBER` to your registered number in E.164 format (e.g., `+15551234567`).

To hide SMS from the UI (e.g., on a hosted platform), set `SMS_NOTIFICATIONS_HIDDEN=true`. Self-hosted deployments should leave this unset.

### Error Tracking (Sentry)

| Variable | Description |
|---|---|
| `SENTRY_DSN` | Backend Sentry DSN (runtime env var) |
| `SENTRY_CSP_ENDPOINT` | CSP violation report-uri (runtime env var) |
| `VITE_SENTRY_DSN` | Frontend Sentry DSN (**build-time** arg) |
| `SENTRY_AUTH_TOKEN` | Source map upload token (**build-time** arg) |
| `SENTRY_ORG` | Sentry org slug (**build-time** arg) |
| `SENTRY_PROJECT` | Sentry project slug (**build-time** arg) |

### Product Analytics (PostHog)

Optional. Tracks user behavior, feature adoption, and funnel analytics.

To get started: sign up at [posthog.com](https://posthog.com), create a project (choose US or EU cloud for data residency), and copy the **Project API Key** from **Project Settings > Project Variables**.

| Variable | Description |
|---|---|
| `VITE_POSTHOG_KEY` | PostHog project API key (**build-time**) |
| `VITE_POSTHOG_HOST` | PostHog API host (**build-time**, default: `https://us.i.posthog.com`) |

If `VITE_POSTHOG_KEY` is not set, analytics are disabled (no-op). PostHog Cloud has a free tier of 1M events/month, or you can [self-host PostHog](https://posthog.com/docs/self-host).

### Billing (Stripe)

Optional. Only needed if you want to enable paid tiers and usage-based billing. See the [Stripe Setup Guide](stripe-setup.md) for detailed instructions on creating Stripe products, configuring webhooks, and testing locally.

| Variable | Description |
|---|---|
| `BILLING_ENABLED` | Set to `true` to enable billing (default: `false`) |
| `STRIPE_SECRET_KEY` | Stripe API secret key |
| `STRIPE_PUBLISHABLE_KEY` | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | Webhook signature verification secret |
| `STRIPE_PRICE_ID_REQUEST` | Metered Stripe Price for per-request billing |
| `VITE_STRIPE_PUBLISHABLE_KEY` | Stripe publishable key for frontend (**build-time**) |

When `BILLING_ENABLED=false` (the default), all users get an unlimited plan, Stripe is skipped, request metering is skipped, and billing API endpoints are disabled. This is the recommended setting for self-hosted deployments unless you want to run your own billing.

### Other Optional Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `MODE` | (production) | Set to `development` to disable rate limiting and relax config validation |
| `TRUSTED_PROXY_HEADER` | `Fly-Client-IP` | HTTP header for real client IP (behind reverse proxy) |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout for in-flight requests |
| `CONNECTORS_DIR` | `~/.permission_slip/connectors/` | Directory for external connector plugins |
| `CUSTOM_CONNECTORS_JSON` | (empty) | Inline JSON connector config (alternative to file on disk) |
| `SMS_NOTIFICATIONS_HIDDEN` | `false` | Set to `true` to hide SMS from notification preferences UI even when SNS is configured |
| `SUPABASE_JWT_SECRET` | (empty) | Legacy HS256 JWT secret (only for Supabase CLI v1 — not needed with modern Supabase) |
| `SUPABASE_JWKS_URL` | (auto-derived) | Override JWKS endpoint URL (auto-derived from `SUPABASE_URL` if not set) |

## Deployment Options

### Option A: Docker (Recommended)

The included multi-stage Dockerfile produces a minimal (~30MB) Alpine image.

**Build the image:**

```bash
docker build \
  --build-arg VITE_SUPABASE_URL=https://abcdefgh.supabase.co \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY=your-publishable-key \
  -t permission-slip .
```

**Run the container:**

```bash
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=require" \
  -e SUPABASE_URL="https://abcdefgh.supabase.co" \
  -e BASE_URL="https://permissions.mycompany.com" \
  -e ALLOWED_ORIGINS="https://permissions.mycompany.com" \
  -e INVITE_HMAC_KEY="$(openssl rand -hex 32)" \
  permission-slip
```

**Docker Compose example:**

```yaml
version: "3.8"
services:
  permission-slip:
    build:
      context: .
      args:
        VITE_SUPABASE_URL: https://abcdefgh.supabase.co
        VITE_SUPABASE_PUBLISHABLE_KEY: your-publishable-key
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://user:pass@host:5432/dbname?sslmode=require
      SUPABASE_URL: https://abcdefgh.supabase.co
      BASE_URL: https://permissions.mycompany.com
      ALLOWED_ORIGINS: https://permissions.mycompany.com
      INVITE_HMAC_KEY: change-me-generate-with-openssl-rand-hex-32
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/health"]
      interval: 15s
      timeout: 3s
      retries: 3
```

### Option B: Fly.io

See the dedicated [Fly.io deployment guide](deployment.md) for full instructions including `fly.toml`, secrets management, DNS, scaling, and custom connectors.

Quick version:

```bash
fly launch
fly secrets set \
  DATABASE_URL="postgres://..." \
  SUPABASE_URL="https://abcdefgh.supabase.co" \
  BASE_URL="https://your-app.fly.dev" \
  ALLOWED_ORIGINS="https://your-app.fly.dev" \
  INVITE_HMAC_KEY="$(openssl rand -hex 32)"
fly deploy \
  --build-arg VITE_SUPABASE_URL=https://abcdefgh.supabase.co \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY=your-publishable-key
```

### Option C: Build from Source

If you prefer running the binary directly (e.g., on a VM or bare metal):

**Prerequisites:** Go 1.24+, Node.js 20+

```bash
# Clone and build
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
make install

# Set build-time vars for the frontend
export VITE_SUPABASE_URL=https://abcdefgh.supabase.co
export VITE_SUPABASE_PUBLISHABLE_KEY=your-publishable-key
make build

# Run
export DATABASE_URL="postgres://..."
export SUPABASE_URL="https://abcdefgh.supabase.co"
export BASE_URL="https://permissions.mycompany.com"
export ALLOWED_ORIGINS="https://permissions.mycompany.com"
export INVITE_HMAC_KEY="$(openssl rand -hex 32)"
./bin/server
```

The binary is fully static (`CGO_ENABLED=0`) and embeds the React frontend — just copy it to your server and run.

### Option D: Railway / Render / Other PaaS

Permission Slip works on any platform that supports Docker or Go builds:

1. Connect your repo (or push the Docker image)
2. Set the required environment variables in the platform's dashboard
3. For Docker-based platforms, configure build args for `VITE_SUPABASE_URL` and `VITE_SUPABASE_PUBLISHABLE_KEY`
4. Ensure the health check is configured for `GET /api/health` on port 8080

## TLS / Reverse Proxy

The server listens on plain HTTP. In production, terminate TLS in front of it:

- **Fly.io** — handles TLS automatically
- **nginx / Caddy** — reverse proxy to `localhost:8080`
- **AWS ALB / GCP Load Balancer** — route to the container on port 8080
- **Cloudflare Tunnel** — expose the local service via a tunnel

If using a reverse proxy other than Fly.io, set `TRUSTED_PROXY_HEADER` to the header your proxy uses for the real client IP (e.g., `X-Forwarded-For` or `X-Real-IP`). The default is `Fly-Client-IP`.

## Custom Connectors

Permission Slip ships with built-in GitHub, HubSpot, Slack, and PostgreSQL connectors. To add custom connectors:

**Option A — Inline JSON (recommended for containers):**

```bash
# Set as env var (no filesystem persistence needed)
export CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.0.0"}]}'
```

**Option B — File on disk:**

Create a `custom-connectors.json` in the project root and run `make install-connectors`. Set `CONNECTORS_DIR` to a persistent path if your filesystem is ephemeral.

See [Custom Connectors](custom-connectors.md) for details on building your own.

## Health Check

The server exposes `GET /api/health` which:
- Returns `200 OK` when the server is reachable
- Reports database connectivity status when `DATABASE_URL` is configured
- Returns `503 Service Unavailable` if the database is unreachable

Use this endpoint for load balancer health checks, container orchestration, and uptime monitoring.

## Scaling

Permission Slip is stateless (all state is in PostgreSQL), so horizontal scaling is straightforward:

- **Multiple instances:** Run as many copies as needed behind a load balancer
- **Database connections:** Each instance creates a connection pool. Monitor your database's max connections
- **VAPID keys:** When running multiple instances, set `VAPID_PUBLIC_KEY` and `VAPID_PRIVATE_KEY` explicitly so all instances use the same keys (don't rely on auto-generation)
- **Action token signing:** Keys are ephemeral (generated per-instance on startup). Tokens are short-lived (max 5 min), so this works without shared key storage

## Secret Rotation

Rotate secrets on a regular cadence (every 90 days recommended for API keys and passwords). Key things to know:

- **`DATABASE_URL`** — change the password in your database provider, then update the env var. No downtime.
- **`INVITE_HMAC_KEY`** — regenerate with `openssl rand -hex 32`. **Invalidates pending invite links** (accepted invites are unaffected).
- **`SENDGRID_API_KEY`** — create a new key in the SendGrid console, deploy it, then revoke the old key.
- **`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`** — rotate in AWS IAM console, deploy new credentials, then delete the old access key.
- **`VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY`** — only rotate if compromised. **Invalidates all push subscriptions** (users must re-subscribe).
- **`STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET`** — roll keys in Stripe dashboard (supports overlap periods), update env var, then revoke old key.

For detailed rotation instructions and a full schedule, see the [Production Deployment Guide — Rotating Secrets](deployment-production.md#rotating-secrets).

## Troubleshooting

**Server won't start — "required configuration value(s) missing":**
In production mode, `DATABASE_URL` and one of `SUPABASE_URL`/`SUPABASE_JWT_SECRET`/`SUPABASE_JWKS_URL` are required. Check your env vars.

**Frontend shows "Missing VITE_SUPABASE_URL" error:**
The Supabase build args were not passed at build time. Rebuild the Docker image with `--build-arg VITE_SUPABASE_URL=...`.

**Health check fails after deploy:**
Check logs. Common causes: missing `DATABASE_URL`, incorrect Supabase credentials, or the database being unreachable from the deployment network.

**Connection refused to database:**
If using Supabase, ensure the connection string uses the pooler endpoint (port 6543) with `?sslmode=require`. Direct connections (port 5432) may be blocked.

**CORS errors in browser (403 on API calls):**
Ensure `ALLOWED_ORIGINS` includes your deployment's exact origin (e.g., `https://permissions.mycompany.com`) — no trailing slash. If you're behind a reverse proxy or CDN, the origin seen by the browser may differ from `BASE_URL`. When `ALLOWED_ORIGINS` is unset, the server runs in same-origin only mode, which works for the standard embedded-SPA deployment but will reject requests from a different origin.

**VAPID error on startup:**
If any VAPID variable is set, all three (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) must be set. Either set all three or remove them all.

**Migrations fail:**
Migrations run automatically on startup. If they fail, check database connectivity and permissions. The database user needs `CREATE TABLE`, `ALTER TABLE`, and schema modification privileges.

## Complete Environment Variable Reference

| Variable | Required | Build/Runtime | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | Runtime | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Runtime | Supabase project URL (JWT + auth) |
| `VITE_SUPABASE_URL` | No | Build | Supabase URL for frontend auth (omit for self-hosted — uses built-in proxy) |
| `VITE_SUPABASE_PUBLISHABLE_KEY` | Yes | Build | Supabase publishable key |
| `BASE_URL` | Recommended | Runtime | Public deployment URL |
| `ALLOWED_ORIGINS` | Recommended | Runtime | CORS allowed origins (comma-separated, exact match, no trailing slash). Same-origin only when unset. |
| `INVITE_HMAC_KEY` | Recommended | Runtime | HMAC key for invite codes |
| `PORT` | No | Runtime | Listen port (default: `8080`) |
| `MODE` | No | Runtime | `development` to relax validation |
| `TRUSTED_PROXY_HEADER` | No | Runtime | Client IP header (default: `Fly-Client-IP`) |
| `SHUTDOWN_TIMEOUT` | No | Runtime | Graceful shutdown timeout (default: `30s`) |
| `SUPABASE_JWT_SECRET` | No | Runtime | Legacy HS256 JWT secret |
| `SUPABASE_JWKS_URL` | No | Runtime | JWKS endpoint override |
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
| `SNS_SMS_SENDER_ID` | No | Runtime | Alphanumeric SMS sender ID (not supported in all countries) |
| `SNS_SMS_ORIGINATION_NUMBER` | No | Runtime | Origination phone number in E.164 format |
| `SMS_NOTIFICATIONS_HIDDEN` | No | Runtime | Hide SMS from notification preferences UI (`true`/`false`, default: `false`) |
| `SENTRY_DSN` | No | Runtime | Backend error tracking |
| `SENTRY_CSP_ENDPOINT` | No | Runtime | CSP violation reporting |
| `VITE_SENTRY_DSN` | No | Build | Frontend error tracking |
| `SENTRY_AUTH_TOKEN` | No | Build | Sentry source map upload |
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
