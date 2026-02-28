# Production Deployment Guide — app.permissionslip.dev

> **Internal reference** for deploying and operating the production instance at `app.permissionslip.dev`. For self-hosted deployments, see [Self-Hosted Deployment Guide](deployment-self-hosted.md).

## Infrastructure Overview

```
                    ┌─────────────────────────────┐
                    │      Cloudflare / DNS        │
                    │  app.permissionslip.dev      │
                    └──────────┬──────────────────┘
                               │ HTTPS
                    ┌──────────▼──────────────────┐
                    │         Fly.io               │
                    │  Region: iad (US East)       │
                    │  VM: 256MB, shared CPU       │
                    │  Auto-stop/start enabled     │
                    │  ┌────────────────────────┐  │
                    │  │  Permission Slip       │  │
                    │  │  (Go + React, :8080)   │  │
                    │  └───────────┬────────────┘  │
                    └──────────────┼───────────────┘
                                   │
                 ┌─────────────────┼──────────────────┐
                 │                 │                   │
      ┌──────────▼───┐  ┌────────▼────────┐  ┌──────▼──────┐
      │  Supabase     │  │  Sentry         │  │  Twilio     │
      │  - Auth (JWT) │  │  - Errors       │  │  - SendGrid │
      │  - Postgres   │  │  - Performance  │  │  - SMS      │
      │  - Vault      │  │  - CSP reports  │  │             │
      └───────────────┘  └─────────────────┘  └─────────────┘
```

## Services

### Fly.io (Compute)

- **App name:** Set via `fly.toml` (or `fly launch`)
- **Region:** `iad` (US East, primary)
- **Machine:** 256MB RAM, 1 shared CPU
- **Auto-scaling:** Min 0 machines, scales on request load (soft 200, hard 250 concurrent)
- **Auto-stop:** Machines stop during idle periods and restart on incoming requests
- **Health check:** `GET /api/health` — 15s interval, 3s timeout, 10s grace
- **Shutdown:** Graceful SIGTERM with 30s drain for in-flight requests
- **TLS:** Automatic via Let's Encrypt (Fly handles certificate provisioning)

**Key Fly commands:**

```bash
fly status                          # check app status
fly logs                            # tail logs
fly ssh console                     # SSH into running machine
fly scale show                      # current scale settings
fly scale memory 512                # increase memory
fly scale count 2                   # add instances
fly secrets list                    # list all secrets (values hidden)
```

### Supabase (Auth + Database + Vault)

Permission Slip uses a single Supabase project for:

1. **Authentication** — JWT-based user login (email OTP, MFA)
2. **PostgreSQL database** — all application data
3. **Vault** — AES-256-GCM encryption for stored credentials

**What you need from the Supabase dashboard:**

| Value | Where to find | Used as |
|---|---|---|
| Project URL | Settings > API | `SUPABASE_URL` (runtime) + `VITE_SUPABASE_URL` (build) |
| Anon key (public) | Settings > API | `VITE_SUPABASE_ANON_KEY` (build) |
| Database connection string | Settings > Database > Connection string | `DATABASE_URL` (runtime) |

**Database connection notes:**
- Use the **pooler endpoint** (port 6543) with `?sslmode=require` — direct connections (port 5432) may be blocked by Supabase firewall rules
- Connection string format: `postgres://postgres.[project-ref]:[password]@[host]:6543/postgres?sslmode=require`

**Auth configuration** (in Supabase dashboard):
- **Authentication > URL Configuration:** Add `https://app.permissionslip.dev` to the redirect allow list
- **Authentication > Email:** Configure email templates as needed
- **Authentication > Rate Limits:** Review and adjust for production traffic

### Sentry (Error Tracking)

Two Sentry projects (or one with separate DSNs):
- **Backend:** Captures Go panics, 5xx errors, and uncaught exceptions
- **Frontend:** Captures React errors, failed API calls, CSP violations

Source maps are uploaded during Docker builds for readable stack traces.

### Notification Services

**Email — Twilio SendGrid:**
- Used for approval request notifications
- Requires a verified sender domain and API key

**SMS — Twilio:**
- Used for SMS approval notifications
- Requires a Twilio account with a phone number

**Web Push — VAPID:**
- Browser push notifications via FCM / Mozilla Push Service
- VAPID key pair must be consistent across all instances

## Secrets & Environment Variables

### Fly.io Runtime Secrets

Set via `fly secrets set`. These are encrypted at rest by Fly and injected as environment variables at runtime.

```bash
# ── Required ──────────────────────────────────────────────────────────────

fly secrets set \
  DATABASE_URL="postgres://postgres.[ref]:[pass]@[host]:6543/postgres?sslmode=require" \
  SUPABASE_URL="https://[project-ref].supabase.co" \
  BASE_URL="https://app.permissionslip.dev" \
  ALLOWED_ORIGINS="https://app.permissionslip.dev" \
  INVITE_HMAC_KEY="<output of: openssl rand -hex 32>"

# ── Web Push (VAPID) ─────────────────────────────────────────────────────

# Generate keys: go run ./cmd/generate-vapid-keys --format=fly
fly secrets set \
  VAPID_PUBLIC_KEY="<base64url public key>" \
  VAPID_PRIVATE_KEY="<base64url private key>" \
  VAPID_SUBJECT="mailto:admin@supersuit.tech"

# ── Email (SendGrid) ─────────────────────────────────────────────────────

fly secrets set \
  NOTIFICATION_EMAIL_PROVIDER="twilio-sendgrid" \
  SENDGRID_API_KEY="SG.xxxx" \
  NOTIFICATION_EMAIL_FROM="approvals@permissionslip.dev"

# ── SMS (Twilio) ─────────────────────────────────────────────────────────

fly secrets set \
  TWILIO_ACCOUNT_SID="ACxxxx" \
  TWILIO_AUTH_TOKEN="xxxx" \
  TWILIO_FROM_NUMBER="+15551234567"

# ── Error Tracking (Sentry) ──────────────────────────────────────────────

fly secrets set \
  SENTRY_DSN="https://[key]@[org].ingest.sentry.io/[project]" \
  SENTRY_CSP_ENDPOINT="https://[org].ingest.sentry.io/api/[project]/security/?sentry_key=[key]"
```

**List current secrets** (values are hidden):

```bash
fly secrets list
```

### Build-Time Args

These are inlined into the JavaScript bundle by Vite. Pass via `fly deploy --build-arg` or configure in `fly.toml` under `[build.args]`.

```bash
fly deploy \
  --build-arg VITE_SUPABASE_URL="https://[project-ref].supabase.co" \
  --build-arg VITE_SUPABASE_ANON_KEY="<anon key from Supabase dashboard>" \
  --build-arg VITE_SENTRY_DSN="https://[key]@[org].ingest.sentry.io/[project]" \
  --build-arg SENTRY_AUTH_TOKEN="sntrys_xxxx" \
  --build-arg SENTRY_ORG="supersuit" \
  --build-arg SENTRY_PROJECT="permission-slip"
```

Or hardcode in `fly.toml` for simpler deploys:

```toml
[build.args]
  VITE_SUPABASE_URL = "https://[project-ref].supabase.co"
  VITE_SUPABASE_ANON_KEY = "<anon key>"
  # VITE_SENTRY_DSN, SENTRY_AUTH_TOKEN, etc. can go here too
```

### Complete Variable Checklist

| Variable | Type | Status | Description |
|---|---|---|---|
| `DATABASE_URL` | Runtime secret | **Required** | Supabase Postgres pooler connection string |
| `SUPABASE_URL` | Runtime secret | **Required** | Supabase project URL (JWT verification) |
| `BASE_URL` | Runtime secret | **Required** | `https://app.permissionslip.dev` |
| `ALLOWED_ORIGINS` | Runtime secret | **Required** | `https://app.permissionslip.dev` |
| `INVITE_HMAC_KEY` | Runtime secret | **Required** | HMAC key for invite codes |
| `VAPID_PUBLIC_KEY` | Runtime secret | **Required** | Web Push public key |
| `VAPID_PRIVATE_KEY` | Runtime secret | **Required** | Web Push private key |
| `VAPID_SUBJECT` | Runtime secret | **Required** | `mailto:admin@supersuit.tech` |
| `NOTIFICATION_EMAIL_PROVIDER` | Runtime secret | **Set** | `twilio-sendgrid` |
| `SENDGRID_API_KEY` | Runtime secret | **Set** | SendGrid API key |
| `NOTIFICATION_EMAIL_FROM` | Runtime secret | **Set** | Sender address |
| `TWILIO_ACCOUNT_SID` | Runtime secret | **Set if SMS** | Twilio SID |
| `TWILIO_AUTH_TOKEN` | Runtime secret | **Set if SMS** | Twilio auth token |
| `TWILIO_FROM_NUMBER` | Runtime secret | **Set if SMS** | Twilio phone number |
| `SENTRY_DSN` | Runtime secret | **Set** | Backend error tracking DSN |
| `SENTRY_CSP_ENDPOINT` | Runtime secret | **Set** | CSP violation reporting |
| `VITE_SUPABASE_URL` | Build arg | **Required** | Frontend auth URL |
| `VITE_SUPABASE_ANON_KEY` | Build arg | **Required** | Frontend auth public key |
| `VITE_SENTRY_DSN` | Build arg | **Set** | Frontend error tracking DSN |
| `SENTRY_AUTH_TOKEN` | Build arg | **Set** | Source map upload token |
| `SENTRY_ORG` | Build arg | **Set** | `supersuit` |
| `SENTRY_PROJECT` | Build arg | **Set** | `permission-slip` |

## Deployment Process

### Standard Deploy

```bash
# From the repo root — reads VITE_* from environment or fly.toml [build.args]
fly deploy \
  --build-arg VITE_SUPABASE_URL=https://[ref].supabase.co \
  --build-arg VITE_SUPABASE_ANON_KEY=<key>
```

Or use the Makefile shortcut:

```bash
VITE_SUPABASE_URL=https://[ref].supabase.co \
VITE_SUPABASE_ANON_KEY=<key> \
make deploy
```

The deploy process:
1. Fly's remote builders run the multi-stage Docker build
2. Frontend is compiled with Vite (build args inlined)
3. Go binary is compiled with the git SHA as the Sentry release
4. Minimal Alpine runtime image (~30MB) is created
5. Image is deployed, health check passes, old machine is stopped
6. Database migrations run automatically on first request

### Verify After Deploy

```bash
fly status                                          # machine running?
fly logs                                            # any errors?
curl -s https://app.permissionslip.dev/api/health   # health check
```

### Rollback

```bash
# List recent deployments
fly releases

# Rollback to a previous release
fly deploy --image <previous-image-ref>
```

## DNS Configuration

Domain: `app.permissionslip.dev`

```bash
# Add the certificate to Fly
fly certs add app.permissionslip.dev

# Check certificate status
fly certs show app.permissionslip.dev
```

Add a CNAME record in your DNS provider:
- **Type:** CNAME
- **Name:** `app`
- **Value:** `<fly-app-name>.fly.dev` (shown in `fly certs show`)

Fly handles TLS certificates automatically via Let's Encrypt.

## CI/CD Pipeline

GitHub Actions runs on every push to `main` and on pull requests:

| Job | What it does |
|---|---|
| **Backend Tests** | Go tests against Postgres 16 service container |
| **Frontend Tests** | Vitest + React Testing Library |
| **Build** | Full production build (Go binary + React) to catch compilation errors |

The CI pipeline does **not** auto-deploy. Deploys are manual via `fly deploy`.

### Audit Workflow

A separate `audit.yml` workflow runs dependency vulnerability scans:
- `govulncheck` for Go modules
- `npm audit` for frontend packages

## Monitoring & Observability

### Health Check

`GET /api/health` returns:
- `200 OK` — server and database are healthy
- `503 Service Unavailable` — database is unreachable

Use this for uptime monitoring (e.g., UptimeRobot, Fly's built-in checks).

### Sentry

**Backend errors:**
- All 5xx errors and panics are captured
- Sensitive headers (`Authorization`, `Cookie`, `X-Api-Key`) are scrubbed before sending
- Environment tag: `production`
- Release tag: git SHA (set via `-ldflags` at build time)

**Frontend errors:**
- React error boundaries capture component failures
- Failed API calls are tracked
- Source maps are uploaded for readable stack traces

**CSP violations:**
- Reported via `report-uri` directive in Content-Security-Policy header
- Shows up in Sentry as security events

### Logs

```bash
fly logs              # tail all logs
fly logs --app <name> # specific app
```

The server outputs structured JSON logs (`slog.JSONHandler`). Key log lines at startup:
- `server listening` — server is up
- `Connected to database` — DB pool established
- `Notifications: N channel(s) configured` — which notification channels are active
- `Connector registry: N connector(s) registered` — loaded connectors
- `JWT: using JWKS endpoint ...` — auth verification mode

## Operations Runbook

### Scaling

```bash
# Check current scale
fly scale show

# Vertical scaling
fly scale memory 512      # increase to 512MB
fly scale vm shared-cpu-2x # bigger CPU

# Horizontal scaling
fly scale count 2          # run 2 instances
```

When running multiple instances, VAPID keys and `INVITE_HMAC_KEY` must be set as Fly secrets (not auto-generated) so all instances share the same keys.

### Rotating Secrets

```bash
# Generate new HMAC key
fly secrets set INVITE_HMAC_KEY="$(openssl rand -hex 32)"

# Rotate VAPID keys (WARNING: invalidates all push subscriptions)
# Users will need to re-subscribe to push notifications
go run ./cmd/generate-vapid-keys --format=fly
# Copy the output and run the fly secrets set command

# Rotate SendGrid API key
fly secrets set SENDGRID_API_KEY="SG.new-key"
```

Setting a secret triggers an automatic redeploy.

### Database Migrations

Migrations run automatically on server startup. If you need to run them manually:

```bash
# SSH into the machine
fly ssh console

# Migrations are embedded in the binary — restart triggers them
# For manual control, use the migrate CLI locally:
DATABASE_URL="<prod-connection-string>" go run ./cmd/migrate up
DATABASE_URL="<prod-connection-string>" go run ./cmd/migrate down
```

> **Caution:** Running `migrate down` in production will drop tables. Always verify what the down migration does before running it.

### Custom Connectors

On Fly.io, the filesystem is ephemeral. Use the `CUSTOM_CONNECTORS_JSON` env var:

```bash
fly secrets set CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.0.0"}]}'
```

Alternatively, attach a persistent volume:

```bash
fly volumes create connectors_data --size 1 --region iad
```

Add to `fly.toml`:

```toml
[mounts]
  source = "connectors_data"
  destination = "/app/connectors"

[env]
  CONNECTORS_DIR = "/app/connectors"
```

### Checking Secret Values

Fly secrets are encrypted and can't be viewed via `fly secrets list` (it only shows names). If you need to verify a value:

```bash
# SSH into the machine and check the env var
fly ssh console -C "printenv DATABASE_URL"
```

## Troubleshooting

**Deploy fails on frontend build stage:**
- Ensure `spec/openapi/openapi.bundle.yaml` is committed — the `npm ci` postinstall hook needs it
- Check that build args (`VITE_SUPABASE_URL`, `VITE_SUPABASE_ANON_KEY`) are being passed

**Health check fails after deploy:**
- Check `fly logs` for startup errors
- Common: bad `DATABASE_URL`, expired database password, Supabase project paused (free tier)

**"VAPID_SUBJECT is required in production":**
- All three VAPID vars must be set, or none. Check `fly secrets list`

**Users can't log in:**
- Verify `SUPABASE_URL` is correct and the project is active
- Check that `https://app.permissionslip.dev` is in Supabase's redirect allow list
- Check Supabase dashboard for auth errors

**CORS errors in browser:**
- Ensure `ALLOWED_ORIGINS` includes `https://app.permissionslip.dev` (exact match, no trailing slash)

**Invite links don't work:**
- Check `BASE_URL` is set to `https://app.permissionslip.dev`
- Check `INVITE_HMAC_KEY` hasn't been rotated since the invite was generated (rotation invalidates pending invites)

**Database connection timeout:**
- Use the Supabase pooler endpoint (port 6543), not direct connection (port 5432)
- Check Supabase project status — free tier projects pause after inactivity
