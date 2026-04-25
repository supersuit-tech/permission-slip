# Developer guide

This page is for people who want to **run, build, or change** Permission Slip locally. For workflow, code standards, and pull requests, see [CONTRIBUTING.md](../CONTRIBUTING.md) in the repository root.

---

## Getting started (local dev)

**Prerequisites:** Go 1.24+, Node.js 20+, Supabase CLI, Docker

```bash
# 1. Clone and install
git clone https://github.com/supersuit-tech/permission-slip.git
cd permission-slip
make install

# 2. Configure environment
cp .env.example .env
# Edit .env — see .env.example for all options

# 3. Start Supabase (auth + local Postgres)
supabase start
# Copy the publishable key from `supabase status` into VITE_SUPABASE_PUBLISHABLE_KEY

# 4. Run migrations and generate types
make migrate-up
make generate

# 5. Start the dev servers
make dev-backend   # Go API server → http://localhost:8080
make dev-frontend  # Vite + HMR   → http://localhost:5173
```

For the full walkthrough including PostgreSQL setup and test database configuration, see the [self-hosted deployment guide](deployment-self-hosted.md).

---

## Mobile app

The approval app lives in `mobile/` (React Native / Expo). Approve or deny Openclaw's requests from your phone with push notifications, biometric lock, and deep linking.

```bash
make mobile-install  # install dependencies
make mobile-start    # start Expo dev server (scan QR with Expo Go)
make mobile-test     # run mobile tests
```

For builds, code signing, OTA updates, and App Store submission, see [Mobile builds](mobile-builds.md).

---

## Production build

```bash
make build   # single Go binary with embedded React frontend
./bin/server
```

The server serves both the API and frontend on a single port (default 8080). The most critical environment variables:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SUPABASE_URL` | Yes | Supabase project URL for JWT verification |
| `BASE_URL` | Yes | Public URL (e.g. `https://app.permissionslip.dev`) |
| `INVITE_HMAC_KEY` | Recommended | HMAC key for invite codes — `openssl rand -hex 32` |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` | For Web Push | Generate with `make generate-vapid-keys` |

For the full environment variable reference, Dockerfile, Fly.io setup, and hardening checklist, see [Self-hosted deployment](deployment-self-hosted.md).

---

## Observability

Set `SENTRY_DSN` (backend) and `VITE_SENTRY_DSN` (frontend) to enable Sentry error tracking. Set `VITE_POSTHOG_KEY` to enable PostHog analytics — fully consent-gated, no data collected until the user accepts cookies.

---

## Testing

```bash
make test            # all tests (backend + frontend + mobile)
make test-backend    # Go tests (requires Postgres)
make test-frontend   # frontend tests (no database needed)
make mobile-test     # mobile tests (no database needed)
```

See [CONTRIBUTING.md](../CONTRIBUTING.md) for the full testing strategy and development workflow.

---

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go, PostgreSQL (pgx), JWT (ES256/HS256), goose migrations |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS v4, shadcn/ui |
| Mobile | React Native (Expo 55), TypeScript |
| API Client | openapi-fetch with generated TypeScript types |
| Auth | Supabase Auth (JWT-based, MFA support) |
| Credential Vault | Supabase Vault (AES-256-GCM encryption at rest) |
| State | React Query (TanStack Query) |
| Testing | Go test + real Postgres, Vitest + RTL, Jest (mobile) |
