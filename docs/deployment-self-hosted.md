# Self-Hosted Deployment Guide

This guide covers deploying Permission Slip on your own infrastructure. Permission Slip ships as a single Go binary with the React frontend embedded — no separate web server or static file hosting needed.

## Prerequisites

Before deploying, you'll need:

| Service | Purpose | Notes |
|---|---|---|
| **PostgreSQL 16+** | Application database | Any managed provider works (Supabase, Neon, AWS RDS, self-hosted) |
| **Supabase project** | User authentication (JWT-based) | Free tier is sufficient. Provides login, MFA, and JWT verification |
| **Docker** (optional) | Container-based deployment | Only needed if deploying via Docker |

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
- React SPA for all other routes
- Database migrations run automatically on startup

## Step 1: Set Up Supabase Auth

Permission Slip uses [Supabase Auth](https://supabase.com/auth) for user authentication. You need a Supabase project — the free tier works fine.

1. Create a project at [supabase.com](https://supabase.com)
2. From your project dashboard, note these values:
   - **Project URL** (e.g., `https://abcdefgh.supabase.co`) — used as `SUPABASE_URL`
   - **Anon key** (public) — used as `VITE_SUPABASE_ANON_KEY` at build time
3. Under **Authentication > URL Configuration**, add your deployment URL to the redirect allow list

> **Note:** You can use Supabase's hosted database as your `DATABASE_URL` too, or use a separate PostgreSQL instance. Either works.

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
- A `VAULT_SECRET_KEY` configured on the **database side** (not as an app env var)

If you're using a Supabase-hosted database, vault is already available. If using a non-Supabase database, you'll need to install the `pgsodium` and `supabase_vault` extensions manually, or implement an alternative vault backend.

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
| `ALLOWED_ORIGINS` | Comma-separated CORS origins | `https://permissions.mycompany.com` |
| `INVITE_HMAC_KEY` | HMAC key for invite code signing | Generate: `openssl rand -hex 32` |

### Build-Time Variables (Frontend)

These are inlined into the JavaScript bundle by Vite at build time. They must be passed as `--build-arg` when building Docker images, not as runtime env vars.

| Variable | Description | Example |
|---|---|---|
| `VITE_SUPABASE_URL` | Supabase project URL (frontend auth) | `https://abcdefgh.supabase.co` |
| `VITE_SUPABASE_ANON_KEY` | Supabase anon (public) key | From Supabase dashboard |
| `VITE_SENTRY_DSN` | Frontend error tracking (optional) | `https://key@o0.ingest.sentry.io/0` |

> The anon key is safe to include in the build — it's a public key visible in client-side JavaScript by design.

### Web Push Notifications (VAPID)

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

### SMS Notifications (Twilio)

| Variable | Description |
|---|---|
| `TWILIO_ACCOUNT_SID` | Twilio Account SID (`ACxxxx`) |
| `TWILIO_AUTH_TOKEN` | Twilio Auth Token |
| `TWILIO_FROM_NUMBER` | Twilio phone number (`+15551234567`) |

All three are required to enable SMS. If partially configured, SMS is disabled with a warning.

### Error Tracking (Sentry)

| Variable | Description |
|---|---|
| `SENTRY_DSN` | Backend Sentry DSN (runtime env var) |
| `SENTRY_CSP_ENDPOINT` | CSP violation report-uri (runtime env var) |
| `VITE_SENTRY_DSN` | Frontend Sentry DSN (**build-time** arg) |
| `SENTRY_AUTH_TOKEN` | Source map upload token (**build-time** arg) |
| `SENTRY_ORG` | Sentry org slug (**build-time** arg) |
| `SENTRY_PROJECT` | Sentry project slug (**build-time** arg) |

### Other Optional Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `MODE` | (production) | Set to `development` to disable rate limiting and relax config validation |
| `TRUSTED_PROXY_HEADER` | `Fly-Client-IP` | HTTP header for real client IP (behind reverse proxy) |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout for in-flight requests |
| `CONNECTORS_DIR` | `~/.permission_slip/connectors/` | Directory for external connector plugins |
| `CUSTOM_CONNECTORS_JSON` | (empty) | Inline JSON connector config (alternative to file on disk) |
| `SUPABASE_JWT_SECRET` | (empty) | Legacy HS256 JWT secret (only for Supabase CLI v1 — not needed with modern Supabase) |
| `SUPABASE_JWKS_URL` | (auto-derived) | Override JWKS endpoint URL (auto-derived from `SUPABASE_URL` if not set) |

## Deployment Options

### Option A: Docker (Recommended)

The included multi-stage Dockerfile produces a minimal (~30MB) Alpine image.

**Build the image:**

```bash
docker build \
  --build-arg VITE_SUPABASE_URL=https://abcdefgh.supabase.co \
  --build-arg VITE_SUPABASE_ANON_KEY=your-anon-key \
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
        VITE_SUPABASE_ANON_KEY: your-anon-key
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
  --build-arg VITE_SUPABASE_ANON_KEY=your-anon-key
```

### Option C: Build from Source

If you prefer running the binary directly (e.g., on a VM or bare metal):

**Prerequisites:** Go 1.24+, Node.js 20+

```bash
# Clone and build
git clone https://github.com/supersuit-tech/permission-slip-web.git
cd permission-slip-web
make install

# Set build-time vars for the frontend
export VITE_SUPABASE_URL=https://abcdefgh.supabase.co
export VITE_SUPABASE_ANON_KEY=your-anon-key
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
3. For Docker-based platforms, configure build args for `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY`
4. Ensure the health check is configured for `GET /api/health` on port 8080

## TLS / Reverse Proxy

The server listens on plain HTTP. In production, terminate TLS in front of it:

- **Fly.io** — handles TLS automatically
- **nginx / Caddy** — reverse proxy to `localhost:8080`
- **AWS ALB / GCP Load Balancer** — route to the container on port 8080
- **Cloudflare Tunnel** — expose the local service via a tunnel

If using a reverse proxy other than Fly.io, set `TRUSTED_PROXY_HEADER` to the header your proxy uses for the real client IP (e.g., `X-Forwarded-For` or `X-Real-IP`). The default is `Fly-Client-IP`.

## Custom Connectors

Permission Slip ships with built-in GitHub and Slack connectors. To add custom connectors:

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

## Troubleshooting

**Server won't start — "required configuration value(s) missing":**
In production mode, `DATABASE_URL` and one of `SUPABASE_URL`/`SUPABASE_JWT_SECRET`/`SUPABASE_JWKS_URL` are required. Check your env vars.

**Frontend shows "Missing VITE_SUPABASE_URL" error:**
The Supabase build args were not passed at build time. Rebuild the Docker image with `--build-arg VITE_SUPABASE_URL=...`.

**Health check fails after deploy:**
Check logs. Common causes: missing `DATABASE_URL`, incorrect Supabase credentials, or the database being unreachable from the deployment network.

**Connection refused to database:**
If using Supabase, ensure the connection string uses the pooler endpoint (port 6543) with `?sslmode=require`. Direct connections (port 5432) may be blocked.

**VAPID error on startup:**
If any VAPID variable is set, all three (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`) must be set. Either set all three or remove them all.

**Migrations fail:**
Migrations run automatically on startup. If they fail, check database connectivity and permissions. The database user needs `CREATE TABLE`, `ALTER TABLE`, and schema modification privileges.

## Complete Environment Variable Reference

| Variable | Required | Build/Runtime | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | Runtime | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Runtime | Supabase project URL (JWT + auth) |
| `VITE_SUPABASE_URL` | Yes | Build | Supabase URL for frontend auth |
| `VITE_SUPABASE_ANON_KEY` | Yes | Build | Supabase public anon key |
| `BASE_URL` | Recommended | Runtime | Public deployment URL |
| `ALLOWED_ORIGINS` | Recommended | Runtime | CORS allowed origins (comma-separated) |
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
| `TWILIO_ACCOUNT_SID` | For SMS | Runtime | Twilio Account SID |
| `TWILIO_AUTH_TOKEN` | For SMS | Runtime | Twilio Auth Token |
| `TWILIO_FROM_NUMBER` | For SMS | Runtime | Twilio sender phone number |
| `SENTRY_DSN` | No | Runtime | Backend error tracking |
| `SENTRY_CSP_ENDPOINT` | No | Runtime | CSP violation reporting |
| `VITE_SENTRY_DSN` | No | Build | Frontend error tracking |
| `SENTRY_AUTH_TOKEN` | No | Build | Sentry source map upload |
| `SENTRY_ORG` | No | Build | Sentry org slug |
| `SENTRY_PROJECT` | No | Build | Sentry project slug |
| `CONNECTORS_DIR` | No | Runtime | Custom connector directory |
| `CUSTOM_CONNECTORS_JSON` | No | Runtime | Inline connector JSON config |
