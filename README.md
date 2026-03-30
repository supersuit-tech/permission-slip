# Permission Slip

[![CI](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml)
[![Deploy](https://github.com/supersuit-tech/permission-slip/actions/workflows/deploy.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/deploy.yml)

**Approve what [Openclaw](https://openclaw.org) does before it does it.**

Permission Slip is an open-source approval layer for [Openclaw](https://openclaw.org). Every action Openclaw wants to take — sending emails, merging PRs, booking flights — goes through you first. Nothing happens without your say-so.

```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐
│ Openclaw │ ──────→ │ Permission Slip │ ──────→ │   Gmail,     │
│          │ ←────── │   (approval     │ ←────── │   Stripe,    │
└──────────┘         │    layer)       │         │   GitHub,    │
                     └─────────────────┘         │   Slack…     │
                           │                     └──────────────┘
                           │ push notification
                           ▼
                     ┌───────────┐
                     │  You      │
                     │  (approve │
                     │  / deny)  │
                     └───────────┘
```

## 🚀 Try it now

**[permissionslip.dev](https://www.permissionslip.dev)** — hosted, no setup required.

Or **[self-host it](docs/deployment-self-hosted.md)** on Docker, Fly.io, or bare metal. Even runs on a [Raspberry Pi 5](docs/raspberry-pi-quickstart.md) in under 30 minutes.

---

## ✨ Why Permission Slip?

You want Openclaw to book flights, send emails, and merge PRs. But you can't trust it with full access to your accounts. Your options today:

- 😬 **Give Openclaw your passwords** — it can do anything, anytime, with no oversight
- 😩 **Do everything manually** — defeats the purpose of having Openclaw
- 🤞 **Hope it asks nicely** — it could hallucinate, misunderstand, or get compromised

Permission Slip solves this with a **secure proxy + human-in-the-loop approval** model. Openclaw submits structured, schema-validated actions — never arbitrary API calls. Nothing executes without your explicit sign-off.

---

## 🔑 Key Features

- 🛡️ **Action-based security** — Openclaw submits structured actions, not raw API calls
- 🔔 **Per-request approval** — push notifications with human-readable summaries
- ✅ **Standing approvals** — pre-authorize trusted, repetitive actions with constraints
- 🔐 **Cryptographic identity** — Ed25519 key pairs for tamper-proof request signing
- 🙈 **Zero credential exposure** — Openclaw never sees your API keys or passwords
- 📋 **Full audit trail** — every request, approval, and execution logged
- 🔌 **OAuth 2.0 connections** — Google, Microsoft, Dropbox, and custom providers; PKCE where required; tokens encrypted at rest with automatic refresh
- 🏠 **Self-hostable** — your data, your infrastructure
- 📦 **Single binary deployment** — Go server with embedded React frontend

---

## 🔌 Connector Health

> **⚠️ Permission Slip is in active development and currently in beta.** Not all features are complete, and connectors are being tested incrementally. Expect rough edges.

| Connector | Status |
|-----------|--------|
| GitHub | 🟢 Tested · 🟡 Early Preview |
| Google | 🟡 Early Preview |
| Microsoft | 🟡 Early Preview |
| Slack | 🟡 Early Preview |
| Airtable | 🔴 Untested |
| Amadeus | 🔴 Untested |
| Asana | 🔴 Untested |
| AWS | 🔴 Untested |
| Calendly | 🔴 Untested |
| Confluence | 🔴 Untested |
| Datadog | 🔴 Untested |
| Discord | 🔴 Untested |
| DocuSign | 🔴 Untested |
| DoorDash | 🔴 Untested |
| Dropbox | 🔴 Untested |
| Expedia | 🔴 Untested |
| Figma | 🔴 Untested |
| HubSpot | 🔴 Untested |
| Intercom | 🔴 Untested |
| Jira | 🔴 Untested |
| Kroger | 🔴 Untested |
| Linear | 🔴 Untested |
| LinkedIn | 🔴 Untested |
| Make | 🔴 Untested |
| Meta | 🔴 Untested |
| Monday | 🔴 Untested |
| MongoDB | 🔴 Untested |
| MySQL | 🔴 Untested |
| Netlify | 🔴 Untested |
| Notion | 🔴 Untested |
| PagerDuty | 🔴 Untested |
| Plaid | 🔴 Untested |
| Postgres | 🔴 Untested |
| QuickBooks | 🔴 Untested |
| Redis | 🔴 Untested |
| Salesforce | 🔴 Untested |
| SendGrid | 🔴 Untested |
| Shopify | 🔴 Untested |
| Square | 🔴 Untested |
| Stripe | 🔴 Untested |
| Supabase | 🔴 Untested |
| Trello | 🔴 Untested |
| Twilio | 🔴 Untested |
| Vercel | 🔴 Untested |
| Walmart | 🔴 Untested |
| X | 🔴 Untested |
| Zapier | 🔴 Untested |
| Zendesk | 🔴 Untested |
| Zoom | 🔴 Untested |

Have you tested a connector? [Open an issue](https://github.com/supersuit-tech/permission-slip/issues) to let us know!

---

## 🤖 Built for Openclaw

Permission Slip is purpose-built for [Openclaw](https://openclaw.org). Openclaw has full local shell access and unrestricted networking — everything it needs to register, sign requests, and submit actions through Permission Slip.

### Claude Code skills

This repo includes [Claude Code](https://docs.anthropic.com/en/docs/claude-code/overview) skills under [`.claude/skills/`](.claude/skills/). For example, **`/fix-ci`** (see [`fix-ci/SKILL.md`](.claude/skills/fix-ci/SKILL.md)) triggers **CI + audit** on the current branch, pulls failure logs, and loops on fix → push → re-run until green (with a round cap).

---

## 🛠️ Getting Started (local dev)

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

For the full walkthrough including PostgreSQL setup and test database configuration, see the [self-hosted deployment guide](docs/deployment-self-hosted.md).

---

## 📱 Mobile App

The approval app lives in `mobile/` (React Native / Expo). Approve or deny Openclaw's requests from your phone with push notifications, biometric lock, and deep linking.

```bash
make mobile-install  # install dependencies
make mobile-start    # start Expo dev server (scan QR with Expo Go)
make mobile-test     # run mobile tests
```

For builds, code signing, OTA updates, and App Store submission, see [docs/mobile-builds.md](docs/mobile-builds.md).

---

## 🏗️ Production Build

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

For the full environment variable reference, Dockerfile, Fly.io setup, and hardening checklist, see [docs/deployment-self-hosted.md](docs/deployment-self-hosted.md).

---

## 🔭 Observability

Set `SENTRY_DSN` (backend) and `VITE_SENTRY_DSN` (frontend) to enable Sentry error tracking. Set `VITE_POSTHOG_KEY` to enable PostHog analytics — fully consent-gated, no data collected until the user accepts cookies.

---

## 🧪 Testing

```bash
make test            # all tests (backend + frontend + mobile)
make test-backend    # Go tests (requires Postgres)
make test-frontend   # frontend tests (no database needed)
make mobile-test     # mobile tests (no database needed)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full testing strategy and development workflow.

---

## ⚙️ Tech Stack

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

---

## 📚 Documentation

### 📖 Getting Started
- [Self-Hosted Deployment](docs/deployment-self-hosted.md) — Docker, Fly.io, bare metal
- [Raspberry Pi Quickstart](docs/raspberry-pi-quickstart.md) — up and running in 30 minutes
- [Architecture](docs/architecture.md) — system diagrams and component overview
- [SPEC.md](SPEC.md) — protocol design, security model, and full spec

### 🔌 Integrations & Connectors
- [Openclaw Integration Guide](docs/agents.md) — how Openclaw connects to Permission Slip
- [Creating Connectors](docs/creating-connectors.md) — build new built-in connectors
- [Custom Connectors](docs/custom-connectors.md) — add connectors from external Git repos
- [Community Connectors](docs/community-connectors.md) — third-party connector directory

### 🔒 Protocol Reference
- [Terminology](docs/spec/terminology.md) — core concepts and definitions
- [Authentication](docs/spec/authentication.md) — agent identity and request signing
- [API Reference](docs/spec/api.md) — complete endpoint documentation
- [Notifications](docs/spec/notifications.md) — push notification and webhook delivery
- [OpenAPI Spec](spec/openapi/) — machine-readable API definition

### 🚀 Deployment
- [Fly.io Deployment](docs/deployment.md) — Dockerfile, fly.toml, DNS setup
- [Production Deployment (internal)](docs/deployment-production.md) — infrastructure for app.permissionslip.dev
- [Mobile Builds](docs/mobile-builds.md) — EAS builds, OTA updates, App Store submission

### 🧪 Contributing & Testing
- [CONTRIBUTING.md](CONTRIBUTING.md) — development workflow and code standards
- [Integration Testing](docs/integration-testing.md) — end-to-end test strategy
- [Manual Testing: Agent Registration](docs/manual-testing-agent-registration.md) — invite/registration flow walkthrough

---

## 🤝 Contributing

Contributions are welcome! Check [CONTRIBUTING.md](CONTRIBUTING.md) to get started, or browse [open issues](https://github.com/supersuit-tech/permission-slip/issues).

---

## 📜 License

Permission Slip is licensed under the [Apache License 2.0](LICENSE).
Built by [SuperSuit](https://supersuit.tech) — questions or feedback welcome at [supersuit.tech](https://supersuit.tech).

---

## 👥 Contributors

<a href="https://github.com/chiedo"><img src="https://github.com/chiedo.png" width="50" height="50" alt="chiedo" style="border-radius:50%"></a>
<a href="https://github.com/chiedobot"><img src="https://github.com/chiedobot.png" width="50" height="50" alt="chiedobot" style="border-radius:50%"></a>
<a href="https://github.com/chiedoclaude"><img src="https://github.com/chiedoclaude.png" width="50" height="50" alt="chiedoclaude" style="border-radius:50%"></a>
