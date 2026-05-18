# Permission Slip

[![CI](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml)
**Approve what [Openclaw](https://openclaw.org) does before it does it.**

Permission Slip is an open-source approval layer for [Openclaw](https://openclaw.org). Every action Openclaw wants to take — sending emails, merging PRs, booking flights — goes through you first. Nothing happens without your say-so.

```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐
│ Openclaw │ ──────→ │ Permission Slip │ ──────→ │   Gmail,     │
│          │ ←────── │   (approval     │ ←────── │   GitHub,    │
└──────────┘         │    layer)       │         │   Slack…     │
                           │                     └──────────────┘
                           │ push notification
                           ▼
                     ┌───────────┐
                     │  You      │
                     │  (approve │
                     │  / deny)  │
                     └───────────┘
```

## Run it yourself

Permission Slip is **self-hosted**: you run the API and web UI on your own machine or network.

Start with the **[Self-hosted deployment guide](docs/deployment-self-hosted.md)** — covers bare metal, Docker, Fly.io, and more. A Raspberry Pi is the recommended home-server environment. You will set `DATABASE_PATH`, `JWT_SIGNING_SECRET`, and `SECRET_ENCRYPTION_KEY` as documented there.

Get the **[iPhone app](https://apps.apple.com/us/app/permission-slip/id6761718603)** to approve requests on the go — the app asks for your server URL on first launch.

The **[CLI](cli/README.md)** talks to your server; pass `--server` or set `PS_SERVER` (there is no built-in default host).

---

## ✨ Why Permission Slip?

You want Openclaw to book flights, send emails, and merge PRs. But you can't trust it with full access to your accounts. Your options today:

- 😬 **Give Openclaw your passwords** — it can do anything, anytime, with no oversight
- 😩 **Do everything manually** — defeats the purpose of having Openclaw
- 🤞 **Hope it asks nicely** — it could hallucinate, misunderstand, or get compromised

Permission Slip solves this with a **secure proxy + human-in-the-loop approval** model. Openclaw submits structured, schema-validated actions — never arbitrary API calls. Nothing executes without your explicit sign-off.

---

## Beta status & how we build

The project is **in beta**: behavior, APIs, and connectors will keep evolving, and you should expect rough edges.

The **architecture and product direction are designed by humans**; **the codebase is largely written with AI-assisted development** (with human review and iteration layered on top). Treat the implementation as fast-moving software, not a formally verified system while it is in beta. If this ever gets more enough use, I'll update some processes, get a formal security review done, etc. Still just exploring at the moment.

**Want to run, build, or change the code?** Start with the **[Developer guide](docs/development.md)** — local setup, production builds, testing, and tech stack — then [CONTRIBUTING.md](CONTRIBUTING.md) for workflow and standards.

Self-hosted deployments need a **`JWT_SIGNING_SECRET`** (at least 32 bytes; use `openssl rand -base64 32`) so the API can sign HS256 access tokens. To create the first account on a fresh database after running migrations, use **`go run ./cmd/create-user user@example.com 'your-password'`** (optional **`-username`**). See [docs/development.md](docs/development.md) for full environment variable reference.

---

## 🔑 Key Features

- 🛡️ **Action-based security** — Openclaw submits structured actions, not raw API calls
- 🔔 **Per-request approval** — push notifications with human-readable summaries
- ✅ **Standing approvals** — pre-authorize trusted, repetitive actions with constraints
- 🔐 **Cryptographic identity** — Ed25519 key pairs for tamper-proof request signing
- 🙈 **Zero credential exposure** — Openclaw never sees your API keys or passwords
- 📋 **Full audit trail** — every request, approval, and execution logged
- 🔌 **OAuth 2.0 connections** — Google, Microsoft, Dropbox, and custom providers; PKCE where required; tokens encrypted at rest with automatic refresh
- 📱 **iPhone app** — approve on the go from your phone
- 🏠 **Self-hostable** — your data, your infrastructure
- 📦 **Single binary deployment** — Go server with embedded React frontend

---

## 🔌 Connector Health

Connectors are being tested incrementally during the beta; maturity varies by integration.

| Connector | Status |
|-----------|--------|
| GitHub | 🟡 Early Preview |
| Google | 🟡 Early Preview |
| Microsoft | 🟡 Early Preview |
| Slack | 🟡 Early Preview |

<details>
<summary>🔴 Untested connectors (click to expand)</summary>

These connectors are wired up but have not yet been end-to-end verified. If you try one, we'd love a report — see the "connector report" issue template.

| Connector | Status |
|-----------|--------|
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

</details>

Have you tested a connector? [Open an issue](https://github.com/supersuit-tech/permission-slip/issues/new?template=connector_report.md) to let us know!

---

## 📚 Documentation

### 👩‍💻 For developers & contributors

**[Developer guide](docs/development.md)** — clone the repo, local dev servers, production `make build`, observability env vars, testing commands, and tech stack overview.

**[CONTRIBUTING.md](CONTRIBUTING.md)** — issue workflow, code standards, migrations, and pull request expectations.

### 📖 Getting Started
- [Self-Hosted Deployment](docs/deployment-self-hosted.md) — Raspberry Pi, bare metal, Docker, and PaaS
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
- [Mobile Builds](docs/mobile-builds.md) — EAS builds, OTA updates, App Store submission

### 🧪 Contributing & Testing
- [Developer guide](docs/development.md) — local dev, builds, tests, stack
- [CONTRIBUTING.md](CONTRIBUTING.md) — development workflow and code standards
- [Integration Testing](docs/integration-testing.md) — end-to-end test strategy
- [Manual Testing: Agent Registration](docs/manual-testing-agent-registration.md) — invite/registration flow walkthrough

---

## 🤝 Contributing

Contributions are welcome! For setup and commands, see the **[Developer guide](docs/development.md)**; for process and standards, see [CONTRIBUTING.md](CONTRIBUTING.md). Browse [open issues](https://github.com/supersuit-tech/permission-slip/issues) to find something to work on.

---

## 📜 License

Permission Slip is licensed under the [Apache License 2.0](LICENSE).
Built by [SuperSuit](https://supersuit.tech) — questions or feedback welcome at [supersuit.tech](https://supersuit.tech).

---

## 👥 Contributors

Thanks to everyone who has contributed!

<a href="https://github.com/supersuit-tech/permission-slip/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=supersuit-tech/permission-slip" alt="Contributors" />
</a>

<sub>Made with [contrib.rocks](https://contrib.rocks).</sub>
