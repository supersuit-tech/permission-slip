# Permission Slip

[![CI](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml/badge.svg)](https://github.com/supersuit-tech/permission-slip/actions/workflows/ci.yml)

**Approve what [Openclaw](https://openclaw.org) does before it does it.**

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

Permission Slip is an open-source approval layer for [Openclaw](https://openclaw.org). Openclaw submits structured, schema-validated actions — sending emails, merging PRs, booking flights — and nothing executes until you sign off. Your credentials never leave your server.

---

## Get Started

Permission Slip is **self-hosted**: you run the server on your own machine or network.

- **[Self-hosted deployment guide](docs/deployment-self-hosted.md)** — Raspberry Pi (recommended) or any Linux machine, with Google + Slack OAuth setup
- **[iPhone app](https://apps.apple.com/us/app/permission-slip/id6761718603)** — approve requests on the go; enter your server URL on first launch
- **[CLI](cli/README.md)** — talks to your server; set `PS_SERVER` or pass `--server`

## Features

- **Action-based security** — Openclaw submits structured actions, not raw API calls
- **Per-request approval** — push notifications with human-readable summaries
- **Standing approvals** — pre-authorize trusted, repetitive actions with constraints
- **Cryptographic identity** — Ed25519 key pairs for tamper-proof request signing
- **Zero credential exposure** — Openclaw never sees your API keys or passwords
- **Full audit trail** — every request, approval, and execution logged
- **OAuth 2.0 connections** — Google, Microsoft, Slack, and more; tokens encrypted at rest
- **Single binary deployment** — Go server with embedded React frontend, SQLite database

## Connectors

Connectors are tested incrementally during the beta; maturity varies by integration.

| Connector | Status |
|-----------|--------|
| GitHub | 🟡 Early Preview |
| Google | 🟡 Early Preview |
| Microsoft | 🟡 Early Preview |
| Slack | 🟡 Early Preview |
| Proton Mail | 🟡 Early Preview |

<details>
<summary>Untested connectors (click to expand)</summary>

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

Tested a connector? [Open an issue](https://github.com/supersuit-tech/permission-slip/issues/new?template=connector_report.md) to let us know.

## Documentation

**Setup & deployment**
- [Self-Hosted Deployment](docs/deployment-self-hosted.md) — Raspberry Pi or any Linux machine, with Google + Slack OAuth setup
- [Developer guide](docs/development.md) — local dev servers, builds, testing, tech stack

**Architecture & protocol**
- [Architecture](docs/architecture.md) — system diagrams and component overview
- [SPEC.md](SPEC.md) — protocol design and security model
- [Terminology](docs/spec/terminology.md) · [Authentication](docs/spec/authentication.md) · [API Reference](docs/spec/api.md) · [Notifications](docs/spec/notifications.md)

**Integrations & connectors**
- [Openclaw Integration Guide](docs/agents.md) — how Openclaw connects to Permission Slip
- [Creating Connectors](docs/creating-connectors.md) — build new built-in connectors
- [Custom Connectors](docs/custom-connectors.md) — add connectors from external Git repos
- [Proton Mail](docs/connectors/protonmail.md) — built-in Proton Mail via Proton Mail Bridge on self-hosted instances

**Contributing**
- [CONTRIBUTING.md](CONTRIBUTING.md) — workflow, code standards, PR process
- [Integration Testing](docs/integration-testing.md) · [OpenAPI Spec](spec/openapi/)

## Beta

The project is in beta: APIs and connectors will keep evolving. The architecture is designed by humans; the codebase is largely written with AI-assisted development and human review. Expect rough edges.

Contributions are welcome. Browse [open issues](https://github.com/supersuit-tech/permission-slip/issues) to find something to work on.

---

Permission Slip is licensed under the [Apache License 2.0](LICENSE). Built by [SuperSuit](https://supersuit.tech).

## Contributors

<a href="https://github.com/supersuit-tech/permission-slip/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=supersuit-tech/permission-slip" alt="Contributors" />
</a>

<sub>Made with [contrib.rocks](https://contrib.rocks).</sub>
