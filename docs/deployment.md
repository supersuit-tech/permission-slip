# Deploying to Fly.io

Permission Slip ships as a single Go binary with the React frontend embedded. The included `Dockerfile` and `fly.toml` make it straightforward to deploy on [Fly.io](https://fly.io).

## Prerequisites

- [Fly CLI (`flyctl`)](https://fly.io/docs/flyctl/install/)
- A Fly.io account
- A hosted PostgreSQL database (e.g., [Supabase](https://supabase.com), [Neon](https://neon.tech), or Fly Postgres)
- A Supabase project for authentication

## Quick Start

### 1. Launch the app

From the project root:

```bash
fly launch
```

This detects the `Dockerfile` and `fly.toml` automatically. Review the settings and confirm. If you already have the app created, skip this step and go straight to deploying.

### 2. Set secrets

All secrets are set via `fly secrets set`. At minimum you need a database and auth:

```bash
# Required
fly secrets set \
  DATABASE_URL="postgres://user:pass@host:PORT/dbname?sslmode=require" \
  SUPABASE_URL="https://your-project.supabase.co"

# Recommended
fly secrets set \
  BASE_URL="https://app.permissionslip.dev" \
  ALLOWED_ORIGINS="https://app.permissionslip.dev" \
  INVITE_HMAC_KEY="$(openssl rand -hex 32)"
```

The server auto-derives a JWKS URL from `SUPABASE_URL` for ES256 JWT verification. If your Supabase project uses legacy HS256, also set `SUPABASE_JWT_SECRET`.

See [Secrets Reference](#secrets-reference) below for the full list.

### 3. Configure frontend build args

The React frontend uses Vite, which inlines `VITE_*` environment variables into the JavaScript bundle at build time. You **must** pass your Supabase URL and anon key as build args — they cannot be set as runtime secrets.

The anon key is safe to include in the build: it's a public key that's always visible in client-side JavaScript.

**Option A** — Pass as `fly deploy` flags:

```bash
fly deploy \
  --build-arg VITE_SUPABASE_URL=https://your-project.supabase.co \
  --build-arg VITE_SUPABASE_ANON_KEY=your-anon-key
```

**Option B** — Hardcode in `fly.toml` (simpler for CI):

```toml
[build.args]
  VITE_SUPABASE_URL = "https://your-project.supabase.co"
  VITE_SUPABASE_ANON_KEY = "your-anon-key"
```

**Option C** — Use the Makefile shortcut (reads from your shell environment):

```bash
# Export the variables first, or inline them:
export VITE_SUPABASE_URL=https://your-project.supabase.co
export VITE_SUPABASE_ANON_KEY=your-anon-key
make deploy

# Or inline on the same command:
VITE_SUPABASE_URL=https://your-project.supabase.co VITE_SUPABASE_ANON_KEY=your-anon-key make deploy
```

### 4. Deploy

Run the deploy using whichever option you chose in step 3. If you configured `[build.args]` in `fly.toml` (Option B), just run `fly deploy`. The multi-stage Docker build runs on Fly's remote builders — no local Docker required.

### 5. Verify

```bash
fly status
fly logs
curl https://your-app.fly.dev/api/health
```

## Secrets Reference

| Secret | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Supabase project URL (JWT verification via JWKS + auth) |
| `SUPABASE_SERVICE_ROLE_KEY` | Recommended | Supabase service_role key — required for account deletion (removes auth user). Get from Supabase dashboard |
| `BASE_URL` | Recommended | Public URL (e.g., `https://app.permissionslip.dev`) — required for invite link generation |
| `ALLOWED_ORIGINS` | Recommended | CORS origins (e.g., `https://app.permissionslip.dev`). Defaults to same-origin only if unset |
| `INVITE_HMAC_KEY` | Recommended | HMAC key for invite codes — `openssl rand -hex 32` |
| `SUPABASE_JWT_SECRET` | Legacy only | JWT secret for HS256 verification. Not needed when using ES256/JWKS (derived automatically from `SUPABASE_URL`) |
| `SENTRY_DSN` | Optional | Sentry DSN for backend error tracking |
| `SENTRY_CSP_ENDPOINT` | Optional | Sentry CSP report-uri endpoint for CSP violation tracking |
| `VAPID_PUBLIC_KEY` | Optional | VAPID public key for Web Push (auto-generated if unset) |
| `VAPID_PRIVATE_KEY` | Optional | VAPID private key for Web Push (auto-generated if unset) |
| `VAPID_SUBJECT` | Optional | `mailto:` URL for VAPID (e.g., `mailto:admin@mycompany.com`) |

**Note:** `VAULT_SECRET_KEY` is configured on the Supabase side (in `supabase/config.toml`), not as a Fly secret. Your hosted Supabase project manages its own vault encryption key.

### Error tracking (Sentry)

Frontend error tracking requires `VITE_SENTRY_DSN` as a **build arg** (it's inlined at build time like other `VITE_*` variables):

```bash
fly deploy --build-arg VITE_SENTRY_DSN="https://examplePublicKey@o0.ingest.sentry.io/0"
```

To upload source maps during builds, also set these as build args or CI secrets:

```bash
fly deploy \
  --build-arg VITE_SENTRY_DSN="https://..." \
  --build-arg SENTRY_AUTH_TOKEN="sntrys_..." \
  --build-arg SENTRY_ORG="your-org" \
  --build-arg SENTRY_PROJECT="your-project"
```

### Optional notification secrets

```bash
# Email via SendGrid
fly secrets set \
  NOTIFICATION_EMAIL_PROVIDER="twilio-sendgrid" \
  SENDGRID_API_KEY="SG.xxxx" \
  NOTIFICATION_EMAIL_FROM="approvals@mycompany.com"

# Email via SMTP
fly secrets set \
  NOTIFICATION_EMAIL_PROVIDER="smtp" \
  SMTP_HOST="smtp.gmail.com" \
  SMTP_PORT="587" \
  SMTP_USERNAME="you@gmail.com" \
  SMTP_PASSWORD="app-password" \
  NOTIFICATION_EMAIL_FROM="you@gmail.com"

# SMS via Twilio
fly secrets set \
  TWILIO_ACCOUNT_SID="ACxxxx" \
  TWILIO_AUTH_TOKEN="xxxx" \
  TWILIO_FROM_NUMBER="+15551234567"
```

## DNS Configuration

To use a custom domain (e.g., `app.permissionslip.dev`):

```bash
fly certs add app.permissionslip.dev
```

Then add a CNAME record pointing `app.permissionslip.dev` to your Fly app's hostname (shown in `fly certs show app.permissionslip.dev`). Fly handles TLS automatically via Let's Encrypt.

## Architecture Notes

- **Single binary** — the Go server embeds the React frontend via `go:embed`, so there's no separate static file server
- **Health check** — `GET /api/health` returns 200 when the server is reachable and reports database status when configured (503 if a configured database is unreachable)
- **Graceful shutdown** — the app handles `SIGTERM` (sent by Fly during deploys) and drains in-flight requests for up to 30 seconds
- **Auto-migrations** — database migrations run automatically on startup, so deploys are zero-touch
- **Auto-stop/start** — `fly.toml` is configured to stop machines during low traffic and start them on incoming requests, saving costs

## Scaling

The default `fly.toml` starts with 256MB RAM and a shared CPU. To scale:

```bash
# Vertical — increase memory/CPU
fly scale memory 512
fly scale vm shared-cpu-2x

# Horizontal — add more instances
fly scale count 2
```

When running multiple instances, ensure `VAPID_PUBLIC_KEY` and `VAPID_PRIVATE_KEY` are set as secrets (rather than relying on auto-generation) so all instances use the same Web Push keys.

## Custom Connectors

Fly.io machines have ephemeral filesystems — files written at runtime are lost on restart. If you use [custom connectors](custom-connectors.md), there are two options:

**Option A — `CUSTOM_CONNECTORS_JSON` env var (recommended):**
Provide the connector list as inline JSON via a Fly secret. The server reads this instead of a file on disk.

```bash
fly secrets set CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.0.0"}]}'
```

**Option B — Persistent volume:**
Attach a Fly volume for the connectors directory and set `CONNECTORS_DIR` to the mount path.

```bash
fly volumes create connectors_data --size 1 --region iad
```

Then add to `fly.toml`:

```toml
[mounts]
  source = "connectors_data"
  destination = "/app/connectors"

[env]
  CONNECTORS_DIR = "/app/connectors"
```

## Local Docker Testing

Build and run the Docker image locally to verify before deploying:

```bash
# Build (pass your Supabase build args)
make docker-build

# Or directly:
docker build \
  --build-arg VITE_SUPABASE_URL=https://your-project.supabase.co \
  --build-arg VITE_SUPABASE_ANON_KEY=your-anon-key \
  -t permission-slip-web .

# Run
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  -e SUPABASE_URL="https://your-project.supabase.co" \
  permission-slip-web
```

## Troubleshooting

**Build fails on frontend stage:**
Ensure `spec/openapi/openapi.bundle.yaml` is committed — the `npm ci` postinstall hook generates TypeScript types from it.

**Health check fails after deploy:**
Check logs with `fly logs`. Common causes: missing `DATABASE_URL`, incorrect Supabase credentials, or the database being unreachable from Fly's network.

**Frontend shows "Missing VITE_SUPABASE_URL" error:**
The Supabase build args were not passed during `fly deploy`. Re-deploy with `--build-arg VITE_SUPABASE_URL=... --build-arg VITE_SUPABASE_ANON_KEY=...` or add `[build.args]` to `fly.toml`. See [Configure frontend build args](#3-configure-frontend-build-args).

**Connection refused to database:**
If using Supabase, ensure the connection string uses the pooler endpoint (port 6543) with `?sslmode=require`. Direct connections (port 5432) may be blocked by firewall rules.
