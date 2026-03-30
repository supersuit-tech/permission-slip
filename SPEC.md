# Permission Slip — Protocol Overview

**The approval layer for [Openclaw](https://openclaw.org). Every action Openclaw takes goes through human approval first. Actions are the core primitive.**

---

## The Problem

Openclaw needs to perform actions on behalf of users — making purchases, accessing accounts, sending emails — but current options are broken:

- **Give Openclaw full access** to your credentials → security nightmare
- **Do everything manually** → defeats the purpose of having Openclaw
- **No standard approval flow** → no structured way to review what Openclaw wants to do before it does it
- **No structured request format** → arbitrary API calls or free-text intentions with no validation

## The Solution

Permission Slip is an approval layer that sits between Openclaw and external APIs. Openclaw submits **structured, pre-defined actions** — it never touches user credentials or makes arbitrary API calls. Actions require explicit human approval before execution — either **per-request** (one-off approval with push notification and confirmation code) or **pre-approved** via a standing approval that the user creates in advance for repetitive, trusted actions.

Think of it as a **secure proxy with human-in-the-loop approval, where "actions" are the core primitive**.

### What Are Actions?

Actions are pre-defined templates that define what an agent can request. Each action has:

- **A unique type** (e.g., `email.send`, `flight.book`, `payment.charge`)
- **Required parameters** with validation schemas
- **A display format** for the approval UI — so users see "Send email to bob@example.com" instead of raw JSON
- **Execution logic** — Permission Slip knows exactly which API to call and how

Agents can *only* request pre-defined actions, never arbitrary API calls. This is a security boundary, not just a UX convenience.

## Architecture

```
Openclaw ──→ Permission Slip ──→ External Service (Gmail, Stripe, etc.)
                    │
                    ▼
               User (approve/deny)
```

**Key insight:** Permission Slip is the only party that holds user credentials. Openclaw never sees them. The external service never knows Openclaw is involved — it just sees a normal API call from Permission Slip on the user's behalf.

### Why a Middle-Man (Not a Protocol for Services)

Previous versions of this spec envisioned Permission Slip as an open protocol that each service would implement. That approach required:

- Every company to adopt the protocol
- Client SDKs for agents
- Server SDKs for services
- Years of adoption effort before it becomes useful

The middle-man architecture eliminates all of that:

- **Works today** with any service that has an API
- **No adoption required** — services don't need to change anything
- **Single integration point** for agents — submit actions once, access everything
- **Consistent UX** — same approval flow for every action, regardless of the underlying service
- **Growing action library** — new actions can be added to expand agent capabilities

## How It Works

### Setup (One-Time)

1. User creates a Permission Slip account
2. User stores API credentials for services they use (stored in encrypted vault via the dashboard)
3. User generates a **registration invite** from the Permission Slip dashboard
4. User shares the invite code with the agent (or the agent's operator)
5. Agent generates a cryptographic key pair and registers using the invite code
6. User and agent complete a confirmation code handshake to verify identity
7. Permission Slip stores the agent's public key and links it to the user
8. User **enables connectors** for the agent (e.g., Gmail, Stripe) — controlling which action types the agent can request

### Credential & Connector Model

- **Credentials are user-scoped** — the user stores their Gmail API key once; any of their agents that has Gmail enabled can trigger actions that use it
- **Connector enablement is per-agent** — Agent A might have Gmail enabled but not Stripe; Agent B might have both
- **Users manage credentials** through the dashboard (Supabase session JWT auth), never through agents
- **Agents never see credentials** — credentials are only decrypted at the moment of action execution by Permission Slip's server

> **Why user-initiated?** If agents could register by simply naming an approver username, any agent could spam any user with registration notifications. The invite code ensures the user explicitly chose to add this agent. See [ADR-005](docs/adr/005-user-initiated-registration.md).

### Runtime (Every Action)

**One-off approval (default):**

1. Agent submits a **structured action** to Permission Slip: `submitAction('email.send', {to: 'bob@example.com', subject: 'Meeting tomorrow'})`
2. Permission Slip **validates the action** against its schema and **verifies the agent's signature**
3. Permission Slip sends the user a **push notification** showing the action in a human-readable format
4. User reviews and **approves or denies** the action
5. If approved: Permission Slip **executes the action** by calling the external service's API using the **user's stored credentials**
6. Result is returned to the agent

**Standing approval (pre-authorized):**

1. Agent submits the same structured action
2. Permission Slip **matches an active standing approval** for this agent, action type, and constraints
3. Permission Slip **executes immediately** — no notification, no confirmation code, no token
4. Result is returned to the agent
5. If no standing approval matches, falls through to the one-off flow above

### What the Agent Never Sees

- User's API keys or passwords
- OAuth tokens for external services
- Direct access to any external API

### What the User Always Sees

- Exactly what action the agent wants to perform (action type, parameters, context)
- Risk level assessment
- Human-readable rendering of the action (not raw JSON)
- Ability to approve, deny, or revoke agent access entirely

## Design Principles

| Principle | Description |
|---|---|
| **Actions as the core primitive** | Openclaw submits structured, pre-defined actions — never arbitrary API calls |
| **Middle-man, not a protocol** | Permission Slip is the service — no adoption required from external APIs |
| **Cryptographic trust** | Agents prove identity with public/private keys |
| **Built for Openclaw** | Purpose-built approval layer for Openclaw |
| **Zero credentials exposure** | Openclaw never sees user credentials |
| **User always in control** | Every action requires approval — per-request or pre-approved via standing approvals |
| **Self-hostable** | Run your own instance for full control |

## Openclaw Integration

The SDK provides a simple interface for submitting actions:

```javascript
import { PermissionSlip } from '@permissionslip/sdk';

const ps = new PermissionSlip({ agentKeyPath: './agent_key' });

// Submit a pre-defined action — agents can never make arbitrary API calls
const result = await ps.submitAction('email.send', {
  to: ['bob@example.com'],
  subject: 'Meeting tomorrow',
  body: 'Let\'s meet at 3pm.'
});
// → Permission Slip validates against email.send schema
// → User sees: "Approve action: Send email to bob@example.com"
// → If approved: Permission Slip calls Gmail API with user's credentials
```

Actions are the only way agents interact with external services. The `submitAction()` method:
1. Validates parameters against the action's schema (client-side)
2. Signs the request with the agent's private key
3. Sends the action to Permission Slip for approval
4. Returns the result after the user approves (or rejects)

### Available Actions

Permission Slip ships with a standard action library and supports custom actions:

| Action Type | Description | Example Parameters |
|---|---|---|
| `email.send` | Send an email | `to`, `subject`, `body` |
| `flight.book` | Book a flight | `from`, `to`, `date`, `class` |
| `payment.charge` | Charge a payment | `amount`, `currency`, `customer_id` |
| `calendar.create` | Create calendar event | `title`, `start`, `end`, `attendees` |
| `data.delete` | Delete data | `resource_type`, `resource_id` |

Custom actions use reverse-DNS naming: `com.example.deploy.production`

### Adding Custom Connectors

Permission Slip's connector system is built on a Go interface (`Connector` + `Action`). New connectors are added by implementing the interface and registering with the connector registry at startup — they compile into the single binary alongside the built-in connectors.

Each connector is a Go package that defines its available actions, parameter schemas, and execution logic. Actions are individually defined (one per file) and share common connector code (HTTP client, auth helpers, error mapping).

```
connectors/
├── connector.go              # Connector and Action interfaces
├── github/
│   ├── github.go             # Shared client, auth helpers
│   ├── create_issue.go       # github.create_issue action
│   └── merge_pr.go           # github.merge_pr action
├── slack/
│   ├── slack.go              # Shared client
│   ├── send_message.go       # slack.send_message action
│   └── create_channel.go     # slack.create_channel action
└── registry.go               # Startup registration
```

**Key design property:** Connectors are a security boundary. Only operator-defined, compiled-in connectors can execute actions — agents cannot register custom actions at runtime. This is intentional: every action type goes through code review before it's available.

See [ADR-009](docs/adr/009-connector-execution-architecture.md) for the full architecture.

## Protocol Specification

The detailed protocol specification covers:

- **[Terminology](docs/spec/terminology.md)** — actors, concepts, and definitions
- **[Authentication](docs/spec/authentication.md)** — agent identity, request signing, and verification
- **[API Reference](docs/spec/api.md)** — complete endpoint documentation
- **[Notifications](docs/spec/notifications.md)** — how users are notified of pending approvals
- **[OpenAPI Spec](spec/openapi/)** — machine-readable API definition

## Security Model

Permission Slip's security is layered:

1. **Action validation** — agents can only submit pre-defined action types with schema-validated parameters
2. **Transport layer (TLS)** — all communication is encrypted in transit
3. **Agent authentication (Ed25519 signatures)** — every agent-facing request is cryptographically signed
4. **User authentication (Supabase session JWTs)** — all user-facing endpoints (credentials, agent config, standing approvals) use session tokens
5. **Connector gating** — agents can only request actions from connectors explicitly enabled by the user (per-agent `agent_connectors` allowlist)
6. **Human approval** — no action executes without user consent (per-request approval or pre-authorized standing approval)
7. **Single-use tokens** (one-off) — approved one-off actions get a one-time-use token with a short TTL
8. **Standing approvals** (pre-authorized) — user-defined, constraint-scoped grants that allow repeated execution without per-request approval; revocable instantly
9. **Credential isolation** — agent never sees credentials; credentials are user-scoped and only decrypted at the moment of action execution by Permission Slip's server
10. **Audit trail** — every action request, approval, and execution is logged (including all standing approval executions)
