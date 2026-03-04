# Permission Slip

[![CI](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml)
[![Deploy](https://github.com/supersuit-tech/permission-slip/actions/workflows/deploy.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/deploy.yml)

**The authorization layer between your AI and everything it touches.**

Open-source API key vault where agents execute scoped, time-limited actions on behalf of their humans — every call approved, logged, and revocable.

```
┌─────────┐         ┌─────────────────┐         ┌──────────────┐
│ Your AI │ ──────→ │ Permission Slip │ ──────→ │   Gmail,     │
│  Agent  │ ←────── │   (middle-man)  │ ←────── │   Stripe,    │
└─────────┘         └─────────────────┘         │   Expedia…   │
                           │                    └──────────────┘
                           │ push notification
                           ▼
                     ┌───────────┐
                     │  You      │
                     │  (approve │
                     │  / deny)  │
                     └───────────┘
```

## Get Started

**[Get started for free on permissionslip.dev](https://www.permissionslip.dev)** — no setup required.

Or **[self-host](docs/deployment-self-hosted.md)** on your own infrastructure for full control (Docker, Fly.io, or bare metal).

## Why Permission Slip?

You want your AI agent to book flights, send emails, order food — but you can't trust it with full access to your accounts.

Your options today:
- **Give the agent your passwords/API keys** — it can do anything, anytime, with no oversight
- **Do everything manually** — defeats the purpose of having an agent
- **Hope the agent asks permission** — it could lie, hallucinate, or get compromised

Permission Slip solves this by acting as a secure proxy with human-in-the-loop approval, where **actions** are the core primitive. Agents can only submit pre-defined action types with schema-validated parameters — never arbitrary API calls. You always see exactly what the agent wants to do, and nothing executes without your consent.

For the full protocol design, architecture, and security model, see [SPEC.md](SPEC.md).

## Key Features

- **Action-based security model** — agents submit structured actions, never arbitrary API calls
- **Per-request approval** — push notifications with human-readable action summaries
- **Standing approvals** — pre-authorize trusted, repetitive actions with constraints
- **Cryptographic agent identity** — Ed25519 key pairs for request signing
- **Zero credential exposure** — agents never see your API keys or passwords
- **Self-hostable** — run your own instance for full control
- **Single binary deployment** — Go server with embedded React frontend
- **Audit trail** — every request, approval, and execution is logged
- **User preferences** — per-channel notification settings, contact info, and credential vault management

## Documentation

- **[Docs Site](docs-site/)** — Mintlify-powered user-facing documentation (run `npx mintlify dev` from `docs-site/` to preview)
- **[SPEC.md](SPEC.md)** — protocol overview, architecture, and security model
- **[Terminology](docs/spec/terminology.md)** — core concepts and definitions
- **[Authentication](docs/spec/authentication.md)** — agent identity, request signing, and security
- **[API Reference](docs/spec/api.md)** — complete endpoint documentation
- **[Notifications](docs/spec/notifications.md)** — push notification and webhook delivery
- **[OpenAPI Spec](spec/openapi/)** — machine-readable API definition
- **[Architecture](docs/architecture.md)** — system diagrams and component overview
- **[Agent Integration Guide](docs/agents.md)** — how to integrate an autonomous agent with Permission Slip
- **[Custom Connectors](docs/custom-connectors.md)** — add connectors from external Git repos (subprocess-based plugin system)
- **[Community Connectors](docs/community-connectors.md)** — directory of third-party connectors built by the community
- **[Consent Banner](docs/consent-banner.md)** — cross-subdomain cookie consent banner (shared between www and app)
- **[Manual Testing: Agent Registration](docs/manual-testing-agent-registration.md)** — step-by-step guide to test the invite/registration flow
- **[Self-Hosted Deployment](docs/deployment-self-hosted.md)** — complete guide for deploying on your own infrastructure (Docker, Fly.io, bare metal)
- **[Production Deployment (internal)](docs/deployment-production.md)** — infrastructure, secrets, and operations for app.permissionslip.dev
- **[Fly.io Deployment](docs/deployment.md)** — Dockerfile, fly.toml, secrets, and DNS setup

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, PostgreSQL (pgx), JWT (ES256/HS256), goose migrations |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS v4 |
| Mobile | React Native (Expo 55), TypeScript |
| UI Components | shadcn/ui (Radix UI + Tailwind + Lucide icons) |
| API Client | openapi-fetch with generated TypeScript types |
| Auth | Supabase Auth (JWT-based, MFA support) |
| Credential Vault | Supabase Vault (AES-256-GCM encryption at rest) |
| State | React Query (TanStack Query) |
| Testing | Go test + real Postgres, Vitest + React Testing Library, Jest (mobile) |

## Getting Started

### Prerequisites

- **Go 1.24+** — [Install Go](https://go.dev/doc/install)
- **Node.js 20+** and **npm** — [Install Node.js](https://nodejs.org/)
- **Supabase CLI** and **Docker** — [Install Supabase CLI](https://supabase.com/docs/guides/local-development/cli/getting-started)
- **PostgreSQL 16** (for running tests only) — [Install Postgres](https://www.postgresql.org/download/)

### 1. Clone and install dependencies

```bash
git clone https://github.com/supersuit-tech/permission-slip-web.git
cd permission-slip-web
make install
```

### 2. Set up environment variables

```bash
cp .env.example .env
```

Edit `.env` as needed. See [.env.example](.env.example) for all available variables and their descriptions.

### 3. Start Supabase locally

Supabase provides both authentication and the development database (PostgreSQL).

```bash
supabase start                        # start local Supabase stack (requires Docker)
supabase status                       # get local URLs and keys for .env
```

Copy the `publishable key` from `supabase status` into your `.env` as `VITE_SUPABASE_PUBLISHABLE_KEY`. The default `DATABASE_URL` in `.env.example` already points to Supabase's local Postgres (`127.0.0.1:54322`).

See the [Supabase docs](https://supabase.com/docs/guides/local-development) for more details.

### 4. Run migrations

```bash
make migrate-up
```

The app starts without a database if `DATABASE_URL` is not set, so you can work on frontend-only features without Supabase running.

### 5. Generate the typed API client

TypeScript types are generated from the OpenAPI spec for both `frontend/` and `mobile/`. This happens automatically after `npm install` via a postinstall hook, but you can also run it manually:

```bash
make generate
```

### 6. Run in development

```bash
# Terminal 1 — Go API server (port 8080)
make dev-backend

# Terminal 2 — Vite dev server with HMR (port 5173)
make dev-frontend
```

Open **http://localhost:5173**. API requests to `/api/*` are automatically proxied to the Go server.

### Mobile App (Expo)

The mobile app lives in `mobile/` and shares the same OpenAPI spec for typed API access.

```bash
cd mobile
npm install                # also runs `generate:api` via postinstall
npm start                  # launches Expo dev server
```

Set `EXPO_PUBLIC_API_BASE_URL` in your `.env` (or Expo config) to point at your local backend (e.g. `http://<your-ip>:8080/api`). Without it, the app falls back to production and logs a warning in dev mode.

> **Accessing via ngrok or an external URL?** Set `ALLOWED_ORIGINS` to your public URL (e.g. `ALLOWED_ORIGINS=https://your-subdomain.ngrok-free.app make dev-backend`) so the Go backend allows cross-origin requests. Without it, API calls from a non-localhost origin will be blocked with 403.

### 7. Mobile app (optional)

The mobile app lives in `mobile/` and uses React Native (Expo). It shares the same Supabase project and OpenAPI spec as the web frontend.

```bash
cd mobile
npm install
cp .env.example .env   # set EXPO_PUBLIC_SUPABASE_URL and EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY
npm start              # Expo dev server — scan QR with Expo Go app
```

**Auth tokens** are stored in the device's secure keychain (iOS Keychain / Android EncryptedSharedPreferences) via `expo-secure-store`, not in plain-text storage.

See [`mobile/`](mobile/) for the full directory structure. Run `npm test` from the `mobile/` directory to run mobile unit tests.

## Production Build

Build a single Go binary with the React frontend embedded:

```bash
make build
./bin/server
```

The server serves both the API and the React app on a single port (default 8080). For containerized deployment, see **[Deploying to Fly.io](docs/deployment.md)** (includes Dockerfile, fly.toml, and step-by-step guide).

### Production Environment Variables

Beyond the variables in `.env.example`, these require attention for production:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Your Supabase project URL (for JWT verification) |
| `BASE_URL` | Yes | Public URL of your deployment (e.g. `https://app.permissionslip.dev`) |
| `INVITE_HMAC_KEY` | Recommended | HMAC key for invite codes — generate with `openssl rand -hex 32` |
| `SENTRY_DSN` | Optional | Sentry DSN for backend error tracking — panics and 5xx errors are captured automatically |
| `VITE_SENTRY_DSN` | Optional | Sentry DSN for frontend error tracking (build-time) — React errors, failed API calls, and performance data |
| `SENTRY_CSP_ENDPOINT` | Optional | Sentry CSP report-uri endpoint — captures Content-Security-Policy violations as Sentry events |
| `BILLING_ENABLED` | Optional | Set to `true` to enable billing (Stripe, metering, plan limits). Default: `false` (all users get unlimited access) |
| `VITE_POSTHOG_KEY` | Optional | PostHog project API key for product analytics (build-time) — consent-gated, no data sent until user accepts cookies |
| `VITE_POSTHOG_HOST` | Optional | PostHog API host (build-time, default: `https://us.i.posthog.com`) — use a custom host if self-hosting PostHog |
| `POSTHOG_HOST` | Optional | PostHog API host added to CSP `connect-src` — must match `VITE_POSTHOG_HOST` (runtime) |
| `SHUTDOWN_TIMEOUT` | Optional | Graceful shutdown timeout for draining in-flight requests (default: `30s`) |
| `AUDIT_PURGE_INTERVAL` | Optional | How often expired audit events are purged — Go duration format, minimum `1m` (default: `1h`) |
| `VAPID_PUBLIC_KEY` | For Web Push | VAPID public key for Web Push notifications |
| `VAPID_PRIVATE_KEY` | For Web Push | VAPID private key — keep secret, never commit to git |
| `VAPID_SUBJECT` | For Web Push | `mailto:` URL identifying the operator (e.g. `mailto:admin@mycompany.com`) |
| `EXPO_ACCESS_TOKEN` | Optional | Expo Push Service access token for higher rate limits — generate at [expo.dev](https://expo.dev/accounts/[account]/settings/access-tokens) |

**Billing (`BILLING_ENABLED`):** Controls whether billing features are active. When unset or `false` (the default), all users are automatically assigned the unlimited `pay_as_you_go` plan — no Stripe keys, metering, or plan restrictions needed. Set to `true` for managed deployments that require plan-based limits and Stripe integration. The server logs the billing mode at startup. The frontend can query `GET /v1/config` to adapt its UI based on whether billing is enabled.

**VAPID keys (Web Push):** Set all three to enable Web Push notifications. If none are set, Web Push is disabled. If partially configured, the server will refuse to start. In development mode (`MODE=development`), keys are auto-generated and stored in the database for convenience.

Generate keys and set them as env vars for your platform:

```bash
# Generate a VAPID key pair (.env format)
make generate-vapid-keys

# Fly.io — outputs a ready-to-run `fly secrets set` command
go run ./cmd/generate-vapid-keys --format=fly

# Heroku — outputs a ready-to-run `heroku config:set` command
go run ./cmd/generate-vapid-keys --format=heroku
```

> **Warning:** Changing VAPID keys invalidates all existing Web Push subscriptions. Users will need to re-subscribe to push notifications.

**Mobile Push (Expo):** Mobile push notifications are always enabled when a database is configured — no additional keys required. The sender uses the [Expo Push Service](https://docs.expo.dev/push-notifications/overview/) to deliver notifications to registered devices. Set `EXPO_ACCESS_TOKEN` for authenticated mode (higher rate limits); without it, unauthenticated mode is used.

## Mobile App

The mobile approval app lives in `mobile/` (React Native / Expo). It's a thin client for approving and viewing requests from your phone — similar to the Microsoft Authenticator approval flow.

**Current capabilities:** login, browse pending/approved/denied requests, view full request details (action parameters, risk level, agent info, expiry countdown), approve with confirmation code display (copyable `XXX-XXX` format), deny with confirmation, push notifications for new approval requests, notification tap → deep link to approval detail, and a Settings screen for managing mobile push notification preferences and signing out.

**Push notifications:** The app uses `expo-notifications` to request push permissions, retrieve the Expo push token, and register it with the backend on login. Token registration retries automatically with exponential backoff (up to 3 retries) on transient failures. On logout, the token is unregistered from the backend for clean session separation. Notifications are delivered via the [Expo Push Service](https://docs.expo.dev/push-notifications/overview/). Push notifications require a physical device (not simulators). On Android, a dedicated "Approval Requests" notification channel is created for user-configurable notification preferences. Enable `__DEV__` logging (automatic in development builds) to trace the full push token lifecycle in the console.

**Notification deep linking:** Tapping a push notification navigates directly to the relevant approval detail screen. This works in three scenarios: app in foreground, app in background, and cold start (app was killed). On cold start, the notification is queued until auth completes, then processed automatically. The approval ID is validated against the expected format (`appr_*`) before making any API calls. The approvals list cache is also refreshed on notification tap so the list is up to date when navigating back.

```bash
make mobile-install    # install mobile dependencies
make mobile-start      # start Expo development server
make mobile-test       # run mobile tests
```

See [issue #9](https://github.com/supersuit-tech/permission-slip/issues/9) for the full mobile roadmap.

## Testing

```bash
make test              # all tests (backend + frontend + mobile)
make test-backend      # Go tests (requires Postgres)
make test-frontend     # frontend tests (no database needed)
make mobile-test       # mobile tests (no database needed)
```

Backend tests run against a real Postgres database. Frontend and mobile tests use mocked clients. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full testing strategy.

## Observability

### Error Tracking (Sentry)

Sentry captures backend panics/5xx errors and frontend React errors. Set `SENTRY_DSN` (backend) and `VITE_SENTRY_DSN` (frontend) to enable.

### Product Analytics (PostHog)

PostHog provides privacy-focused product analytics. It is **fully consent-gated** — no data is collected until the user explicitly accepts cookies via the consent banner.

- Set `VITE_POSTHOG_KEY` and optionally `VITE_POSTHOG_HOST` to enable (build-time).
- Set `POSTHOG_HOST` to add the PostHog API host to the CSP `connect-src` directive (runtime).
- If `VITE_POSTHOG_KEY` is not set, PostHog is completely disabled — no SDK code executes.
- Events are defined in `frontend/src/lib/posthog-events.ts`. To add a new event, add a constant there and use `trackEvent()` from `@/lib/posthog`.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to get started, our development workflow, and code standards.

## License

Permission Slip is licensed under the [Apache License 2.0](LICENSE).

## Contact

Permission Slip is developed by [SuperSuit](https://supersuit.tech). For questions or feedback, [open an issue](https://github.com/supersuit-tech/permission-slip-web/issues) or reach out at [supersuit.tech](https://supersuit.tech).

## Contributors

<a href="https://github.com/chiedo"><img src="https://github.com/chiedo.png" width="50" height="50" alt="chiedo" style="border-radius:50%"></a>
<a href="https://github.com/chiedobot"><img src="https://github.com/chiedobot.png" width="50" height="50" alt="chiedobot" style="border-radius:50%"></a>
<a href="https://github.com/chiedoclaude"><img src="https://github.com/chiedoclaude.png" width="50" height="50" alt="chiedoclaude" style="border-radius:50%"></a>
