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
        ┌──────────────┬───────────┼───────────┐
        │              │           │           │
 ┌──────▼───┐  ┌──────▼────┐  ┌──▼────┐  ┌───▼─────┐
 │ Supabase  │  │  Sentry   │  │Twilio │  │ PostHog │
 │ - Auth    │  │  - Errors │  │-SGrid │  │-Analytics│
 │ - Postgres│  │  - Perf   │  │- SMS  │  │-Replays │
 │ - Vault   │  │  - CSP    │  │       │  │         │
 └───────────┘  └───────────┘  └───────┘  └─────────┘
                                                ┌───────────────┐
                                                │  UptimeRobot  │
                                                │  - Uptime     │
                                                │  - Status page│
                                                └───────────────┘
```

## Service Directory

Quick reference of every service in the production stack. See detailed sections below for setup instructions.

| # | Service | What it does | Status |
|---|---|---|---|
| 1 | [**Fly.io**](#flyio-compute) | Compute — hosts the Go+React app | Live |
| 2 | [**Supabase**](#supabase-auth--database--vault) | Auth (email OTP + MFA), PostgreSQL, credential vault | Live |
| 3 | [**Cloudflare**](#cloudflare-dns--proxy--web-analytics) | DNS proxy, TLS termination, Web Analytics | Live |
| 4 | [**Sentry**](#sentry-error-tracking) | Error tracking (backend + frontend) + CSP violation reports | Live |
| 5 | [**Twilio SendGrid**](#notification-services) | Email delivery for approval notifications | Live |
| 6 | [**Twilio**](#notification-services) | SMS delivery for approval notifications | Live |
| 7 | [**VAPID / Web Push**](#notification-services) | Browser push notifications (FCM / Mozilla Push) | Live |
| 8 | [**Expo Push Service**](#notification-services) | Mobile push notifications (iOS + Android via Expo) | Live |
| 9 | [**GitHub Actions**](#cicd-pipeline) | CI (tests, build, audit) + planned CD | Live |
| 10 | [**PostHog**](#posthog-product-analytics) | Product analytics + session replay | Planned |
| 11 | [**UptimeRobot**](#uptimerobot-uptime-monitoring) | Uptime monitoring + public status page | Planned |
| 12 | [**Stripe**](#stripe-billing--payments) | Billing, subscriptions, usage-based payments | Live |

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
| Publishable key (public) | Settings > API | `VITE_SUPABASE_PUBLISHABLE_KEY` (build) |
| Database connection string | Settings > Database > Connection string | `DATABASE_URL` (runtime) |

**Database connection notes:**
- Use the **pooler endpoint** (port 6543) with `?sslmode=require` — direct connections (port 5432) may be blocked by Supabase firewall rules
- Connection string format: `postgres://postgres.[project-ref]:[password]@[host]:6543/postgres?sslmode=require`

**Auth configuration** (in Supabase dashboard):

> Permission Slip uses passwordless email OTP login. The Supabase dashboard settings below must match your production environment — the local `supabase/config.toml` only applies to `supabase start` (local dev).

- [ ] **Authentication > URL Configuration > Site URL:** Set to `https://app.permissionslip.dev`. This is the base URL Supabase uses in email templates (magic links, OTP emails, password reset). If this is wrong, email links will point to localhost or the wrong domain.
- [ ] **Authentication > URL Configuration > Redirect URLs:** Add `https://app.permissionslip.dev` to the allow list. Without this, post-login OAuth/magic-link redirects will fail.
- [ ] **Authentication > Email Templates:** Review all templates (Confirm signup, Magic Link, Change Email, Reset Password). Ensure they reference the production domain, not localhost. Customize branding as needed.
- [ ] **Authentication > Email > Enable email OTP:** Ensure email sign-in is enabled (this is the primary auth method).
- [ ] **Authentication > Rate Limits:** Review and tighten for production traffic. The local config sets all limits to 1000 for dev convenience — production should use Supabase's defaults or stricter values.
- [ ] **Authentication > Multi-Factor Authentication:** Verify TOTP (App Authenticator) enroll and verify are both enabled. The app supports MFA enrollment and challenge flows.
- [ ] **Authentication > Sessions:** Review session timeouts. The local config uses 24h timebox / 8h inactivity — adjust for your production security requirements.

**Vault (credential encryption):**

- `VAULT_SECRET_KEY` is **not** a Fly.io runtime secret. Hosted Supabase manages the pgsodium encryption key automatically at the infrastructure level. The Go app never reads this env var — it only exists in `supabase/config.toml` for local development (`supabase start`). For local dev, generate with `openssl rand -hex 32`.

**Row-Level Security (RLS):**

RLS is enabled on all application tables via a database migration that runs automatically on startup. This locks down the Supabase PostgREST data API — with RLS enabled and no permissive policies, the `anon` and `authenticated` roles cannot read or write any table through PostgREST, even with a valid JWT.

The Go backend connects as the `app_backend` role (a least-privilege role with only `SELECT`, `INSERT`, `UPDATE`, `DELETE` on application tables). RLS is bypassed because the `app_backend` role is not subject to RLS policies (only `anon` and `authenticated` roles used by PostgREST are). Authorization is enforced in Go application code. The `postgres` superuser is reserved for running migrations.

To verify RLS is enabled after deployment:

```sql
SELECT tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public' AND tablename != 'goose_db_version'
ORDER BY tablename;
```

All rows should show `rowsecurity = true`. A database test (`TestRLSEnabledOnAllTables`) enforces this in CI so new tables can't be added without RLS.

**Fly.io setup:**

```bash
# Runtime secrets (superuser — for migrations only)
fly secrets set \
  DATABASE_URL="postgres://postgres.[project-ref]:[password]@[host]:6543/postgres?sslmode=require" \
  SUPABASE_URL="https://[project-ref].supabase.co" \
  SUPABASE_SERVICE_ROLE_KEY="<service_role key from Supabase dashboard>"

# App backend role (least-privilege — used at runtime)
# After the migration runs, set the production password in Supabase SQL Editor:
#   ALTER ROLE app_backend PASSWORD '<strong-password>';
# Then add the secret:
fly secrets set \
  DATABASE_URL_APP="postgresql://app_backend:<password>@[pooler-host]:6543/postgres?sslmode=require"

# Build-time args (inlined into frontend JS bundle)
fly deploy \
  --build-arg VITE_SUPABASE_URL="https://[project-ref].supabase.co" \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY="<publishable key from Supabase dashboard>"
```

### Sentry (Error Tracking)

Two Sentry projects (or one with separate DSNs):
- **Backend:** Captures Go panics, 5xx errors, and uncaught exceptions
- **Frontend:** Captures React errors, failed API calls, CSP violations

Source maps are uploaded during Docker builds for readable stack traces.

**Fly.io setup:**

```bash
# Runtime secrets
fly secrets set \
  SENTRY_DSN="https://[key]@[org].ingest.sentry.io/[project]" \
  SENTRY_CSP_ENDPOINT="https://[org].ingest.sentry.io/api/[project]/security/?sentry_key=[key]"

# Build-time args (frontend error tracking + source map upload)
fly deploy \
  --build-arg VITE_SENTRY_DSN="https://[key]@[org].ingest.sentry.io/[project]" \
  --build-arg SENTRY_AUTH_TOKEN="sntrys_xxxx" \
  --build-arg SENTRY_ORG="supersuit" \
  --build-arg SENTRY_PROJECT="permission-slip"
```

### Notification Services

**Email — Twilio SendGrid:**
- Used for approval request notifications
- Requires a verified sender domain and API key

**SMS — Twilio:**
- Used for SMS approval notifications
- Requires a Twilio account with a phone number

**Web Push — VAPID:**
- Browser push notifications via FCM / Mozilla Push Service
- No external account or signup needed — VAPID keys are a self-generated key pair
- VAPID key pair must be consistent across all instances

**Mobile Push — Expo Push Service:**
- Push notifications to the Permission Slip mobile app (iOS + Android)
- Uses Expo's push notification infrastructure — no direct APNs/FCM setup needed
- Works without an access token (lower rate limits); set `EXPO_ACCESS_TOKEN` for production throughput
- The server discovers registered Expo push tokens for each user automatically

**Fly.io setup:**

```bash
# Email (SendGrid)
fly secrets set \
  NOTIFICATION_EMAIL_PROVIDER="twilio-sendgrid" \
  SENDGRID_API_KEY="SG.xxxx" \
  NOTIFICATION_EMAIL_FROM="approvals@permissionslip.dev"

# SMS (Twilio)
fly secrets set \
  TWILIO_ACCOUNT_SID="ACxxxx" \
  TWILIO_AUTH_TOKEN="xxxx" \
  TWILIO_FROM_NUMBER="+15551234567"

# Web Push (VAPID) — generate keys first: go run ./cmd/generate-vapid-keys --format=fly
fly secrets set \
  VAPID_PUBLIC_KEY="<base64url public key>" \
  VAPID_PRIVATE_KEY="<base64url private key>" \
  VAPID_SUBJECT="mailto:admin@supersuit.tech"

# Mobile Push (Expo) — optional but recommended for production rate limits
fly secrets set \
  EXPO_ACCESS_TOKEN="<access token from expo.dev>"
```

### PostHog (Product Analytics)

> **Status:** Planned — see [#352](https://github.com/supersuit-tech/permission-slip-web/issues/352)

Product analytics for understanding user behavior, feature adoption, and funnel drop-off.

- **Service:** [PostHog Cloud](https://posthog.com) (US or EU hosting)
- **Free tier:** 1M events/month, 5K session recordings/month
- **What it tracks:** Agent registration, approval flows, standing approvals, action execution, notification config
- **Privacy:** Respects Do Not Track / cookie consent; no PII in event properties

**Setup steps:**
1. Sign up at [posthog.com](https://posthog.com) and create a new project
2. During project creation, choose **US Cloud** (`us.i.posthog.com`) or **EU Cloud** (`eu.i.posthog.com`) based on your data residency requirements
3. Copy the **Project API Key** from **Project Settings > Project Variables** — this becomes `VITE_POSTHOG_KEY`
4. Set `VITE_POSTHOG_KEY` as a build arg (see Fly.io setup below)
5. If you chose EU Cloud, also set `VITE_POSTHOG_HOST` to `https://eu.i.posthog.com` (US Cloud is the default)
6. The backend automatically adds the PostHog host to the CSP `connect-src` directive when `POSTHOG_HOST` is set
7. If the key is not set, PostHog is a no-op (safe for dev/staging)

**Env vars (build-time):**

| Variable | Description |
|---|---|
| `VITE_POSTHOG_KEY` | PostHog project API key |
| `VITE_POSTHOG_HOST` | PostHog API host (default: `https://us.i.posthog.com`) |

**Env vars (runtime):**

| Variable | Description |
|---|---|
| `POSTHOG_HOST` | PostHog API host for backend CSP (must match `VITE_POSTHOG_HOST`) |

**Fly.io setup:**

```bash
# Runtime secret (backend CSP — allows browser to send events to PostHog)
fly secrets set POSTHOG_HOST="https://us.i.posthog.com"

# Build-time args (inlined into frontend JS bundle)
fly deploy \
  --build-arg VITE_POSTHOG_KEY="phc_xxxx" \
  --build-arg VITE_POSTHOG_HOST="https://us.i.posthog.com"
```

### Cloudflare (DNS / Proxy / Web Analytics)

The production domain routes through Cloudflare as a DNS proxy. Cloudflare Web Analytics auto-injects a `beacon.min.js` script, which requires CSP exceptions.

**Fly.io setup:**

```bash
# Set as a regular env var (not a secret — it's just a boolean toggle)
# Already configured in fly.toml [env] section
```

In `fly.toml`:

```toml
[env]
  CLOUDFLARE_INSIGHTS = "true"
```

This adds:
- `https://static.cloudflareinsights.com` to CSP `script-src` (loads the beacon script)
- `https://cloudflareinsights.com` to CSP `connect-src` (beacon sends analytics data)

If Cloudflare Web Analytics is disabled in the Cloudflare dashboard, this setting is harmless — it just allows the domains in CSP without effect.

### UptimeRobot (Uptime Monitoring)

> **Status:** Planned — see [#332](https://github.com/supersuit-tech/permission-slip-web/issues/332)

External uptime monitoring — catches issues that Fly's internal health checks miss (DNS, TLS cert expiry, CDN/proxy problems).

- **Service:** [UptimeRobot](https://uptimerobot.com)
- **Free tier:** 50 monitors, 5-minute check intervals

**Setup steps:**
1. Create an UptimeRobot account
2. Add HTTP monitor:
   - **URL:** `https://app.permissionslip.dev/api/health`
   - **Interval:** 5 minutes
   - **Expected status:** 200
   - **Keyword check:** `"status":"ok"` (validates response body, not just HTTP 200)
3. Configure alert contacts (email, Slack, or webhook)
4. Optional: set up a public status page at `status.permissionslip.dev`

**No app env vars needed** — UptimeRobot is external.

### Stripe (Billing & Payments)

Payment processing for the paid tier: payment method collection, subscription management, and usage-based billing.

- **Service:** [Stripe](https://stripe.com)
- **Gated by:** `BILLING_ENABLED` env var (default `false`). When disabled, all users get unlimited plan and Stripe is skipped entirely
- **Full setup guide:** [Stripe Setup Guide](stripe-setup.md) — covers production setup, local development testing, webhook configuration, and verification steps

**Env vars (runtime secrets):**

| Variable | Description |
|---|---|
| `BILLING_ENABLED` | `true` to enable billing. Default `false` — all users get unlimited plan |
| `STRIPE_SECRET_KEY` | Stripe API secret key (`sk_live_xxxx`) |
| `STRIPE_PUBLISHABLE_KEY` | Stripe publishable key for frontend Checkout (`pk_live_xxxx`) |
| `STRIPE_WEBHOOK_SECRET` | Webhook signature verification (`whsec_xxxx`) |
| `STRIPE_PRICE_ID_REQUEST` | Metered Stripe Price ID for per-request billing |

**Env vars (build-time):**

| Variable | Description |
|---|---|
| `VITE_STRIPE_PUBLISHABLE_KEY` | Stripe publishable key for frontend (build-time) |

**Fly.io setup:**

```bash
# Runtime secrets
fly secrets set \
  BILLING_ENABLED="true" \
  STRIPE_SECRET_KEY="sk_live_xxxx" \
  STRIPE_PUBLISHABLE_KEY="pk_live_xxxx" \
  STRIPE_WEBHOOK_SECRET="whsec_xxxx" \
  STRIPE_PRICE_ID_REQUEST="price_xxxx"
```

`VITE_STRIPE_PUBLISHABLE_KEY` is configured in `fly.toml` under `[build.args]` — no manual `--build-arg` flag needed on deploy.

**When `BILLING_ENABLED=false`:**
- New users get `pay_as_you_go` plan (unlimited)
- Stripe integration is skipped
- Request metering is skipped
- Billing API endpoints return 404

## Secrets & Environment Variables

### Fly.io Runtime Secrets

Set via `fly secrets set`. These are encrypted at rest by Fly and injected as environment variables at runtime.

> **Note:** The `MODE` variable controls dev vs. production behavior. When `MODE` is **not set** (the default), the server runs in production mode — config validation is strict, rate limiting is enabled, and Sentry reports its environment as `production`. **Do not** set `MODE=development` in production. There is no need to explicitly set `MODE` for production deploys.

```bash
# ── Required ──────────────────────────────────────────────────────────────

fly secrets set \
  DATABASE_URL="postgres://postgres.[ref]:[pass]@[host]:6543/postgres?sslmode=require" \
  DATABASE_URL_APP="postgresql://app_backend:[password]@[pooler-host]:6543/postgres?sslmode=require" \
  SUPABASE_URL="https://[project-ref].supabase.co" \
  BASE_URL="https://app.permissionslip.dev" \
  ALLOWED_ORIGINS="https://app.permissionslip.dev" \
  INVITE_HMAC_KEY="<output of: openssl rand -hex 32>"

# ── Recommended ───────────────────────────────────────────────────────────

fly secrets set \
  SUPABASE_SERVICE_ROLE_KEY="<service_role key from Supabase dashboard>"

# ── Web Push (VAPID) ─────────────────────────────────────────────────────

# Generate keys: go run ./cmd/generate-vapid-keys --format=fly
fly secrets set \
  VAPID_PUBLIC_KEY="<base64url public key>" \
  VAPID_PRIVATE_KEY="<base64url private key>" \
  VAPID_SUBJECT="mailto:admin@supersuit.tech"

# ── Mobile Push (Expo) — optional ────────────────────────────────────────

fly secrets set \
  EXPO_ACCESS_TOKEN="<access token from expo.dev>"

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

# ── Product Analytics (PostHog) — when enabled ───────────────────────────
# Not needed until PostHog is enabled. See #352.

fly secrets set \
  POSTHOG_HOST="https://us.i.posthog.com"

# ── Billing (Stripe) — when BILLING_ENABLED=true ─────────────────────────
# Not needed until billing is enabled. See #341, #364.

fly secrets set \
  BILLING_ENABLED="true" \
  STRIPE_SECRET_KEY="sk_live_xxxx" \
  STRIPE_PUBLISHABLE_KEY="pk_live_xxxx" \
  STRIPE_WEBHOOK_SECRET="whsec_xxxx" \
  STRIPE_PRICE_ID_REQUEST="price_xxxx"
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
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY="<publishable key from Supabase dashboard>" \
  --build-arg VITE_SENTRY_DSN="https://[key]@[org].ingest.sentry.io/[project]" \
  --build-arg SENTRY_AUTH_TOKEN="sntrys_xxxx" \
  --build-arg SENTRY_ORG="supersuit" \
  --build-arg SENTRY_PROJECT="permission-slip" \
  --build-arg VITE_POSTHOG_KEY="phc_xxxx" \
  --build-arg VITE_STRIPE_PUBLISHABLE_KEY="pk_live_xxxx"
```

Or hardcode in `fly.toml` for simpler deploys:

```toml
[build.args]
  VITE_SUPABASE_URL = "https://[project-ref].supabase.co"
  VITE_SUPABASE_PUBLISHABLE_KEY = "<publishable key>"
  VITE_POSTHOG_KEY = "phc_xxxx"
  # VITE_SENTRY_DSN, SENTRY_AUTH_TOKEN, etc. can go here too
```

### Complete Variable Checklist

| Variable | Type | Status | Description |
|---|---|---|---|
| `DATABASE_URL` | Runtime secret | **Required** | Supabase Postgres pooler connection string (superuser — used for migrations) |
| `DATABASE_URL_APP` | Runtime secret | **Required** | Least-privilege `app_backend` role connection string (used at runtime) |
| `SUPABASE_URL` | Runtime secret | **Required** | Supabase project URL (JWT verification) |
| `BASE_URL` | Runtime secret | **Required** | `https://app.permissionslip.dev` |
| `ALLOWED_ORIGINS` | Runtime secret | **Required** | `https://app.permissionslip.dev` |
| `INVITE_HMAC_KEY` | Runtime secret | **Required** | HMAC key for invite codes |
| `VAPID_PUBLIC_KEY` | Runtime secret | **Required** | Web Push public key |
| `VAPID_PRIVATE_KEY` | Runtime secret | **Required** | Web Push private key |
| `VAPID_SUBJECT` | Runtime secret | **Required** | `mailto:admin@supersuit.tech` |
| `EXPO_ACCESS_TOKEN` | Runtime secret | **Optional** | Expo push notification access token (higher rate limits) |
| `NOTIFICATION_EMAIL_PROVIDER` | Runtime secret | **Set** | `twilio-sendgrid` |
| `SENDGRID_API_KEY` | Runtime secret | **Set** | SendGrid API key |
| `NOTIFICATION_EMAIL_FROM` | Runtime secret | **Set** | Sender address |
| `TWILIO_ACCOUNT_SID` | Runtime secret | **Set if SMS** | Twilio SID |
| `TWILIO_AUTH_TOKEN` | Runtime secret | **Set if SMS** | Twilio auth token |
| `TWILIO_FROM_NUMBER` | Runtime secret | **Set if SMS** | Twilio phone number |
| `SENTRY_DSN` | Runtime secret | **Set** | Backend error tracking DSN |
| `SENTRY_CSP_ENDPOINT` | Runtime secret | **Set** | CSP violation reporting |
| `CLOUDFLARE_INSIGHTS` | Runtime env | **Set** | `true` — allows Cloudflare Web Analytics beacon in CSP |
| `SUPABASE_SERVICE_ROLE_KEY` | Runtime secret | **Recommended** | Admin API operations (e.g. account deletion) |
| `VITE_SUPABASE_URL` | Build arg | **Required** | Frontend auth URL |
| `VITE_SUPABASE_PUBLISHABLE_KEY` | Build arg | **Required** | Frontend auth publishable key |
| `VITE_SENTRY_DSN` | Build arg | **Set** | Frontend error tracking DSN |
| `SENTRY_AUTH_TOKEN` | Build arg | **Set** | Source map upload token |
| `SENTRY_ORG` | Build arg | **Set** | `supersuit` |
| `SENTRY_PROJECT` | Build arg | **Set** | `permission-slip` |
| `VITE_POSTHOG_KEY` | Build arg | **Planned** ([#352]) | PostHog project API key |
| `VITE_POSTHOG_HOST` | Build arg | **Planned** ([#352]) | PostHog API host (default: `us.i.posthog.com`) |
| `POSTHOG_HOST` | Runtime secret | **Planned** ([#352]) | PostHog API host for backend CSP |
| `BILLING_ENABLED` | Runtime secret | **Set if billing** | `true` to enable billing (default: `false`) |
| `STRIPE_SECRET_KEY` | Runtime secret | **Set if billing** | Stripe API secret key |
| `STRIPE_PUBLISHABLE_KEY` | Runtime secret | **Set if billing** | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | Runtime secret | **Set if billing** | Stripe webhook signing secret |
| `STRIPE_PRICE_ID_REQUEST` | Runtime secret | **Set if billing** | Metered Stripe Price ID |
| `VITE_STRIPE_PUBLISHABLE_KEY` | Build arg | **Set if billing** | Stripe publishable key (frontend) |

[#352]: https://github.com/supersuit-tech/permission-slip-web/issues/352
[#364]: https://github.com/supersuit-tech/permission-slip-web/issues/364
[#341]: https://github.com/supersuit-tech/permission-slip-web/issues/341

## Deployment Process

### Standard Deploy

```bash
# From the repo root — reads VITE_* from environment or fly.toml [build.args]
fly deploy \
  --build-arg VITE_SUPABASE_URL=https://[ref].supabase.co \
  --build-arg VITE_SUPABASE_PUBLISHABLE_KEY=<key>
```

Or use the Makefile shortcut:

```bash
VITE_SUPABASE_URL=https://[ref].supabase.co \
VITE_SUPABASE_PUBLISHABLE_KEY=<key> \
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

### CI (Testing)

GitHub Actions runs on every push to `main` and on pull requests:

| Job | What it does |
|---|---|
| **Backend Tests** | Go tests against Postgres 16 service container |
| **Frontend Tests** | Vitest + React Testing Library |
| **Build** | Full production build (Go binary + React) to catch compilation errors |

### Audit Workflow

A separate `audit.yml` workflow runs dependency vulnerability scans:
- `govulncheck` for Go modules
- `npm audit` for frontend packages

### CD (Deployment)

> **Status:** Planned — see [#328](https://github.com/supersuit-tech/permission-slip-web/issues/328)

Currently, deploys are **manual** via `fly deploy`. The planned CD workflow:

- **Trigger:** Push to `main` (after CI passes)
- **Action:** Deploy to Fly.io using `superfly/flyctl-actions`
- **Secret:** `FLY_API_TOKEN` GitHub Actions secret (Fly.io deploy token)

**Planned workflow** (`.github/workflows/deploy.yml`):

```yaml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    needs: [backend, frontend, build]  # require CI to pass
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

**GitHub Actions secrets needed:**

| Secret | Description |
|---|---|
| `FLY_API_TOKEN` | Fly.io deploy token (generate via `fly tokens create deploy`) |

**Optional:** A staging workflow deploying to `staging.app.permissionslip.dev` on a separate Fly.io app.

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
- Environment tag: derived from `MODE` (defaults to `production` when `MODE` is not set)
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

Setting any secret via `fly secrets set` triggers an automatic redeploy.

#### Rotation Schedule

| Secret | Cadence | Impact of Rotation |
|---|---|---|
| `DATABASE_URL` (password) | Every 90 days | None — new connections use the new password immediately |
| `DATABASE_URL_APP` (password) | Every 90 days | None — rotate via Supabase SQL Editor (`ALTER ROLE app_backend PASSWORD '...'`), then update Fly secret |
| `INVITE_HMAC_KEY` | Every 90 days | **Invalidates pending (unused) invite links.** Accepted invites are unaffected |
| `SENDGRID_API_KEY` | Every 90 days | None — revoke the old key in SendGrid after deploying the new one |
| `TWILIO_AUTH_TOKEN` | Every 90 days | None — regenerate in Twilio console, then update Fly |
| `STRIPE_SECRET_KEY` | Every 90 days | None — roll the key in Stripe dashboard, update Fly, then revoke the old key |
| `STRIPE_WEBHOOK_SECRET` | Every 90 days | None — create a new webhook endpoint in Stripe, update Fly, then delete the old endpoint |
| `SENTRY_DSN` | Rarely (only if compromised) | None — DSNs are project-scoped and low-risk |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` | Rarely (only if compromised) | **Invalidates all push subscriptions.** Users must re-subscribe to browser notifications |
| `SENTRY_AUTH_TOKEN` (build-time) | Every 90 days | None — only used during Docker builds for source map uploads |
| `FLY_API_TOKEN` (CI/CD) | Every 90 days | None — regenerate via `fly tokens create deploy`, update in GitHub Actions secrets |

#### How to Rotate Each Secret

**Database password:**

1. Change the password in your database provider (e.g., Supabase dashboard > Settings > Database)
2. Update the connection string:
   ```bash
   fly secrets set DATABASE_URL="postgres://postgres.[ref]:[new-pass]@[host]:6543/postgres?sslmode=require"
   ```

**App backend database password (`DATABASE_URL_APP`):**

The `app_backend` role is created by a migration with a dev password. In production, set a strong password via the Supabase SQL Editor:

1. Generate a new password:
   ```bash
   openssl rand -hex 32
   ```
2. In **Supabase dashboard > SQL Editor**, run:
   ```sql
   ALTER ROLE app_backend PASSWORD 'paste-new-password-here';
   ```
3. Update the Fly secret (same pooler host as `DATABASE_URL`, just swap username and password):
   ```bash
   fly secrets set DATABASE_URL_APP="postgresql://app_backend:new-password@[pooler-host]:6543/postgres?sslmode=require"
   ```

**HMAC key:**

Invalidates any pending invite links that haven't been accepted yet. Already-accepted invites are unaffected.

```bash
fly secrets set INVITE_HMAC_KEY="$(openssl rand -hex 32)"
```

**SendGrid API key:**

1. Create a new API key in the [SendGrid console](https://app.sendgrid.com/settings/api_keys) with the same permissions
2. Deploy the new key:
   ```bash
   fly secrets set SENDGRID_API_KEY="SG.new-key"
   ```
3. After verifying email still works, revoke the old key in SendGrid

**Twilio auth token:**

1. In the [Twilio console](https://console.twilio.com), go to Account > API keys and tokens and regenerate the auth token
2. Deploy the new token:
   ```bash
   fly secrets set TWILIO_AUTH_TOKEN="new-token"
   ```

**Stripe keys:**

1. In the [Stripe dashboard](https://dashboard.stripe.com/apikeys), roll the secret key (Stripe supports rolling keys with an overlap period)
2. Deploy the new key:
   ```bash
   fly secrets set STRIPE_SECRET_KEY="sk_live_new"
   ```
3. For webhook secrets: create a new webhook endpoint, update `STRIPE_WEBHOOK_SECRET`, then delete the old endpoint

**VAPID keys (avoid unless compromised):**

This invalidates **all** existing push subscriptions. Users will see push notifications stop working until they re-visit the app and re-subscribe.

```bash
go run ./cmd/generate-vapid-keys --format=fly
# Copy the output and run the fly secrets set command it prints
```

**Sentry auth token (build-time):**

1. Create a new auth token in [Sentry](https://sentry.io/settings/auth-tokens/)
2. Update the build arg in `fly.toml` or your CI pipeline
3. Revoke the old token in Sentry

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

## Service Status Overview

Quick reference for what's live vs. planned.

| Service | Purpose | Status | Issue |
|---|---|---|---|
| **Fly.io** | Compute / hosting | Live | — |
| **Cloudflare** | DNS proxy + Web Analytics | Live | — |
| **Supabase** | Auth + Postgres + Vault | Live | — |
| **Sentry** | Error tracking (backend + frontend) | Live | [#329](https://github.com/supersuit-tech/permission-slip-web/issues/329), [#330](https://github.com/supersuit-tech/permission-slip-web/issues/330) |
| **SendGrid** | Email notifications | Live | — |
| **Twilio** | SMS notifications | Live | — |
| **VAPID / Web Push** | Browser push notifications | Live | — |
| **Expo Push Service** | Mobile push notifications (iOS + Android) | Live | — |
| **PostHog** | Product analytics + session replay | Planned | [#352](https://github.com/supersuit-tech/permission-slip-web/issues/352) |
| **UptimeRobot** | Uptime monitoring + status page | Planned | [#332](https://github.com/supersuit-tech/permission-slip-web/issues/332) |
| **Stripe** | Billing + payments | Live | [#168](https://github.com/supersuit-tech/permission-slip/issues/168) |
| **GitHub Actions CD** | Auto-deploy on push to main | Planned | [#328](https://github.com/supersuit-tech/permission-slip-web/issues/328) |

### Future Hardening (Phase 3)

These are tracked under [#321](https://github.com/supersuit-tech/permission-slip-web/issues/321) Phase 3 and should be addressed when the app has real users:

- Prometheus metrics or Grafana Cloud for infrastructure metrics
- Define SLOs and alerting thresholds
- Database slow query logging
- Connection pooling verification under load
- Horizontal scaling load testing
- Database migration linting in CI
- Automated backup restore tests
- Penetration test / security audit
- Row-Level Security policies — add per-user RLS policies and connect `SET LOCAL` in the Go database layer for defense-in-depth on backend queries (RLS is already enabled on all tables; this adds actual policies)
- Auth endpoint rate limiting (brute force protection)
- Dependency update automation (Dependabot or Renovate)

## Troubleshooting

**Frontend shows "Missing VITE_SUPABASE_URL" error:**
- The Vite build args were not inlined during the Docker build. Re-deploy with `fly deploy --no-cache` to bypass Fly's remote builder cache
- Verify `[build.args]` in `fly.toml` contains `VITE_SUPABASE_URL` and `VITE_SUPABASE_PUBLISHABLE_KEY`

**CSP blocks Cloudflare Web Analytics (beacon.min.js):**
- Ensure `CLOUDFLARE_INSIGHTS = "true"` is in `fly.toml` `[env]` section, then redeploy

**Deploy fails on frontend build stage:**
- Ensure `spec/openapi/openapi.bundle.yaml` is committed — the `npm ci` postinstall hook needs it
- Check that build args (`VITE_SUPABASE_URL`, `VITE_SUPABASE_PUBLISHABLE_KEY`) are being passed

**Health check fails after deploy:**
- Check `fly logs` for startup errors
- Common: bad `DATABASE_URL`, expired database password, Supabase project paused (free tier)

**"VAPID_SUBJECT is required in production":**
- All three VAPID vars must be set, or none. Check `fly secrets list`

**Users can't log in:**
- Verify `SUPABASE_URL` is correct and the Supabase project is active (free tier projects pause after inactivity)
- Check that the **Site URL** in Supabase dashboard (Authentication > URL Configuration) is set to `https://app.permissionslip.dev` — if it points to localhost, email links will be broken
- Check that `https://app.permissionslip.dev` is in Supabase's **Redirect URLs** allow list
- Verify email OTP is enabled in Supabase dashboard (Authentication > Email)
- Check the Supabase dashboard Auth logs for specific error messages

**MFA not working:**
- Verify TOTP enroll and verify are both enabled in Supabase dashboard (Authentication > Multi-Factor Authentication)
- Check that the user's device clock is synced (TOTP codes are time-based and drift-sensitive)

**OTP emails not arriving:**
- Check Supabase dashboard Auth logs for send failures
- Verify email rate limits haven't been hit (Authentication > Rate Limits)
- Check spam/junk folders — Supabase's built-in email sender may trigger spam filters. Consider configuring a custom SMTP provider in the Supabase dashboard for production email deliverability

**CORS errors in browser:**
- Ensure `ALLOWED_ORIGINS` includes `https://app.permissionslip.dev` (exact match, no trailing slash)

**Invite links don't work:**
- Check `BASE_URL` is set to `https://app.permissionslip.dev`
- Check `INVITE_HMAC_KEY` hasn't been rotated since the invite was generated (rotation invalidates pending invites)

**Database connection timeout:**
- Use the Supabase pooler endpoint (port 6543), not direct connection (port 5432)
- Check Supabase project status — free tier projects pause after inactivity
