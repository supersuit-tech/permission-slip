# Permission Slip Protocol — Terminology

This document defines the key terms and concepts used throughout the Permission Slip protocol specification.

---

## Core Actors

### Agent
Any automated system requesting approval to perform actions on behalf of a user. This includes:
- AI agents (chatbots, autonomous assistants)
- Scripts and automation tools
- Bots and workflow engines

**Key characteristics:**
- Holds a private key for signing requests
- Registers with Permission Slip by providing a public key
- Makes approval requests for one-off actions, or executes pre-approved actions via standing approvals
- Uses single-use tokens (one-off) or standing approvals (pre-approved) to execute actions
- **Never has access to user credentials** — only communicates with Permission Slip

### Permission Slip (Service)
The centralized middle-man application that mediates between agents and external services. Permission Slip is the only system that holds user credentials and makes API calls to external services on the user's behalf.

**Key characteristics:**
- Stores user credentials for external services (Gmail, Stripe, etc.) in an encrypted vault
- Provides the Permission Slip protocol API endpoints
- Verifies agent signatures on every request
- Sends push notifications to users for one-off approval
- Issues single-use tokens after one-off approval
- Enforces standing approvals for pre-authorized actions (no per-request notification needed)
- Executes approved actions by calling external service APIs using stored credentials
- Returns results to the agent

**Important:** In this architecture, Permission Slip is the service. External services (Gmail, Stripe, Expedia, etc.) are not aware of the Permission Slip protocol — they receive normal API calls from Permission Slip using the user's credentials.

### User (Approver)
The human who owns the accounts and has the authority to approve or deny agent actions. The user registers agents, stores credentials, and reviews all approval requests.

**Key characteristics:**
- Creates a Permission Slip account
- Adds API credentials for external services they use
- Registers agents and manages their access
- Reviews approval requests in real-time via push notifications
- Approves or denies specific one-off actions
- Creates standing approvals to pre-authorize repetitive actions for a defined period (or indefinitely)
- Can revoke agent access or standing approvals at any time

### External Service
Any third-party API that a user has an account with (Gmail, Stripe, Expedia, Slack, etc.). External services are unaware of Permission Slip — they simply receive API calls made by Permission Slip using the user's credentials.

**Key characteristics:**
- Provides APIs that Permission Slip calls on the user's behalf
- Does not need to implement any Permission Slip protocol
- Does not need to install any SDK
- Sees normal API calls — indistinguishable from the user calling the API directly

---

## Core Concepts

### Action
A pre-defined, structured request that an agent submits for approval. Actions are the fundamental primitive of the Permission Slip protocol — they define *what* an agent wants to do, and constrain it to a well-known set of operations.

**Key characteristics:**
- Every action has a **type** (e.g., `email.send`, `payment.charge`, `flight.book`) that identifies the operation
- Every action has **parameters** with validation rules defined by the action type schema
- Actions have a **display format** that determines how the approval UI renders the request to the user
- Actions have **execution logic** — when approved, Permission Slip knows exactly which external service API to call and how
- Agents can **only request pre-defined actions**, never arbitrary API calls — this is a security boundary, not just a UX convenience

**Why actions matter:**
- **Security boundary:** Agents cannot construct arbitrary API calls. They can only request actions that Permission Slip knows about, with parameters that match the action's schema.
- **Human-readable approvals:** Because actions are structured, the approval UI can render them clearly (e.g., "Send email to bob@example.com" instead of showing raw JSON).
- **Extensibility:** New actions can be added to expand what agents can do, without changing the core protocol.
- **Validation:** Permission Slip validates action parameters before showing the approval request to the user, catching malformed requests early.

**Action type naming:**
- **Standard types** (reserved for future standardization): `<category>.<operation>` (e.g., `email.send`, `payment.charge`)
- **Custom types** (service-specific): `<reverse-domain>.<category>.<operation>` (e.g., `com.example.deploy.production`)

See the [API Reference](api.md) for detailed action type specifications, naming conventions, and versioning rules.

### Registration
The one-time setup process where an agent proves its identity to Permission Slip by providing its public key. Registration links the agent to a specific user account.

**Key characteristics:**
- **User-initiated:** The user generates a registration invite from the Permission Slip dashboard and communicates the invite code to the agent (see **Registration Invite** below and [ADR-005](../adr/005-user-initiated-registration.md))
- Agent registers using the invite code and its public key
- A confirmation code handshake verifies that the agent holds the private key matching its registered public key
- Establishes agent identity within Permission Slip
- Links agent to the user who created the invite
- Required before making approval requests

### Registration Invite
A single-use, time-limited code generated by a user from the Permission Slip dashboard to authorize an agent to register. The invite code is embedded in an invite URL (e.g., `https://app.permissionslip.dev/invite/PS-R7K3-X9M4`). The user shares this URL with the agent, and the agent POSTs directly to it to register.

**Key characteristics:**
- Generated by the user (not the agent) — prevents unsolicited registration spam
- Format: `PS-XXXX-XXXX` (8 alphanumeric characters, uppercase)
- Single-use: consumed when an agent registers with it
- Time-limited: default 15 minutes, configurable up to 24 hours
- Stored as SHA-256 hash (same security pattern as confirmation codes)
- Brute-force protection: 5 failed attempts lock out the invite

See [ADR-005](../adr/005-user-initiated-registration.md) for the full rationale.

### Credential Vault
The secure storage within Permission Slip where user API keys, OAuth tokens, and other credentials for external services are kept.

**Key characteristics:**
- Encrypted at rest
- Only Permission Slip can access credentials (agents cannot)
- Credentials are used only when executing approved actions
- Users can add, remove, or rotate credentials at any time

### Connector
A Permission Slip module that knows how to execute actions against a specific external service's API. Each connector defines the available actions for its service, validates action parameters, and translates approved actions into the correct API calls.

Internally, connectors use a two-level interface: the **Connector** owns shared concerns (HTTP client, authentication, error mapping) while each **Action** handles a single operation (parameter parsing, API call, response mapping). This separation means connectors scale cleanly to hundreds of actions without becoming monolithic.

**Examples:**
- GitHub connector: defines and executes `github.create_issue`, `github.merge_pr` actions via the GitHub API
- Slack connector: defines and executes `slack.send_message`, `slack.create_channel`, `slack.schedule_message`, `slack.upload_file`, and 6 more actions via the Slack API

**Relationship to actions:** Connectors are the bridge between the abstract action definition and the concrete API call. A connector registers the action types it supports, provides parameter schemas for validation, and contains the execution logic that runs when an action is approved.

**Security boundary:** Connectors are operator-defined Go implementations compiled into the Permission Slip binary. Agents cannot register custom actions at runtime — all available actions go through code review. See [ADR-009](../adr/009-connector-execution-architecture.md) for the execution architecture.

### Capability Discovery
The process by which an agent learns what it can do on behalf of a specific user. The `GET /v1/agents/{agent_id}/capabilities` endpoint returns a complete view: enabled connectors, available actions with parameter schemas, active standing approvals, and credential readiness — giving the agent everything it needs to know in a single call.

### Approval Request
An asynchronous request for permission to perform a specific one-off action. The request includes detailed information about the action, parameters, and context to help the user make an informed decision. This is the per-request approval flow — see also **Standing Approval** for pre-authorized actions.

**Key characteristics:**
- Describes a specific action with concrete parameters
- Includes risk level and context
- Has a configurable TTL (default: 5 minutes)
- Does not block the agent while waiting for approval
- Can expire if not approved within the TTL

**Example:** *"Send email to bob@example.com with subject 'Meeting tomorrow' and body 'Let's meet at 3pm.'"*

### Standing Approval
A time-bound (or indefinite), constraint-scoped pre-authorization that lets an agent execute a specific action type **without per-request human approval**. The agent can execute the action as many times as it wants (or up to a configured cap) within the approval window.

**Key characteristics:**
- Created proactively by the user from the web interface
- Scoped to a specific agent, action type, and set of constraints
- Duration is user-defined: from 1 hour to a maximum of 90 days
- Executions can be unlimited (default) or capped at a user-defined number
- Constraints are enforced on every execution (same rules as one-off approvals)
- Revocable instantly from the web UI at any time
- Every execution is audited, even without per-request approval
- Falls through to one-off approval if the agent's request doesn't match the standing approval's constraints

**Example:** *"Agent 'My Assistant' can read emails from @github.com for the next 7 days, unlimited times."*

See [ADR-002](../adr/002-standing-approvals.md) for full design details.

### Token
A single-use credential issued to an agent after a **one-off** approval request is approved. The token authorizes the agent to trigger execution of the specific approved action exactly once. Tokens are part of the one-off approval flow; standing approvals do not use tokens.

**Key characteristics:**
- Single-use: consumed after one API call
- Time-bound: expires at the approval request's TTL if not used
- Action-specific: tied to the exact parameters in the approval request
- Scope cannot exceed what was approved in the original request
- Not reusable for similar or related actions

**Important:** When the agent presents the token, Permission Slip executes the action using stored credentials. The token does **not** give the agent direct access to external service APIs.

**Note:** Standing approvals bypass the token mechanism entirely. The standing approval itself is the authorization — no token is issued or consumed per execution. See **Standing Approval** above.

### Time-to-Live (TTL)
The maximum duration an approval request remains valid before expiring. The TTL also determines token expiration if the token is not used.

**Default:** 5 minutes
**Customization:** Configurable per approval request

The TTL balances two concerns:
- Short enough to minimize risk if an approval is granted but not immediately used
- Long enough to accommodate real-world approval delays (user checking their phone, reviewing details)

---

## Protocol Model

Permission Slip supports two approval models: a **"break glass" model** for one-off actions, and a **standing approval model** for pre-authorized, repetitive actions.

#### One-off Approval (Break Glass)

1. **Agent submits action:** `submitAction('email.send', {to: 'bob@example.com', ...})` (signed with agent's private key)
2. **Permission Slip validates:** Checks action type exists, validates parameters against schema, verifies cryptographic signature
3. **User reviews:** Gets push notification showing the action in human-readable format (e.g., "Send email to bob@example.com"), approves or denies
4. **Permission Slip issues token:** Single-use token tied to the approved action and its exact parameters
5. **Agent presents token:** Sends token back to Permission Slip to trigger execution
6. **Permission Slip executes:** Calls the external service API using user's stored credentials
7. **Token consumed:** Cannot be reused, even for identical future actions

#### Standing Approval (Pre-authorized)

For repetitive, predictable actions, users can create standing approvals that let agents execute without per-request prompts:

1. **User creates standing approval:** Specifies agent, action type, constraints, duration (or no expiration), and optional execution cap
2. **Agent submits action:** Same `submitAction()` call — the agent doesn't need to know whether a standing approval exists
3. **Permission Slip matches:** Finds an active standing approval matching the agent, action type, and constraints
4. **Permission Slip executes:** Calls the external service API immediately — no notification, no confirmation code, no token
5. **Result returned:** Same as one-off, but instant
6. **Fallthrough:** If no standing approval matches, falls through to the one-off flow above

**This is NOT:**
- A way to give agents direct access to APIs
- A way for agents to make arbitrary API calls (only pre-defined actions)
- A token the agent can use to call external services directly

---

## Security Concepts

### Public/Private Key Pair
Agents use asymmetric cryptography to prove their identity and sign requests.

**Recommendations:**
- Use the same key pair across all interactions with Permission Slip
- Agents may generate new keys if the old ones are compromised

**Supported algorithms:**
- Ed25519 (preferred)
- ECDSA P-256

### Signature Verification
Permission Slip verifies that all requests are signed by the agent's private key (matching the registered public key). This prevents impersonation and ensures request integrity.

### Credential Isolation
The fundamental security property of the middle-man architecture: agents never see user credentials, and credentials never leave Permission Slip except when making API calls to external services on the user's behalf.

---

## Multi-Service Operations

When an agent needs to perform actions across multiple external services in a single user interaction, Permission Slip can aggregate multiple independent approval requests into one notification for the user.

**Key characteristics:**
- Each action generates its own approval request independently
- Permission Slip may present multiple approvals in one UI screen for convenience
- Each approval results in a separate single-use token
- Each action is executed independently against its respective external service

**Example:** An agent booking a trip might request:
- "Book flight LAX→NYC on 3/15" (executed via Expedia connector)
- "Send confirmation email to user@example.com" (executed via Gmail connector)

The user sees both on one screen, but they are two separate protocol transactions with two separate tokens and two separate API calls.

---

## Glossary

| Term | Definition |
|------|------------|
| **Action** | Pre-defined, structured request that an agent submits for approval — the core primitive of the protocol |
| **Agent** | Automated system requesting approval to perform actions via Permission Slip |
| **Permission Slip** | The centralized middle-man service that mediates between agents and external APIs |
| **User / Approver** | Human who owns the accounts and approves/denies actions |
| **External Service** | Third-party API (Gmail, Stripe, etc.) — unaware of Permission Slip |
| **Capability Discovery** | How an agent learns what it can do for a specific user (`GET /v1/agents/{id}/capabilities`) |
| **Connector** | Module that defines available actions and executes them against external service APIs |
| **Credential Vault** | Encrypted storage for user's external service credentials |
| **Registration** | One-time, user-initiated setup linking an agent to a user account in Permission Slip |
| **Registration Invite** | Single-use, time-limited code generated by a user to authorize an agent to register |
| **Approval Request** | Async request for one-off action approval |
| **Standing Approval** | Pre-authorized, constraint-scoped grant allowing an agent to execute an action repeatedly without per-request approval |
| **Token** | Single-use credential issued after one-off approval to trigger action execution |
| **TTL** | Time-to-live for approval requests and tokens (default: 5 minutes) |
| **Break Glass** | Model where approval is for immediate, one-off actions only |
| **Public Key** | Agent's identity credential, shared with Permission Slip during registration |
| **Private Key** | Agent's secret key, used to sign requests |

---

## Next Steps

See other specification documents for details on:
- **[Authentication](authentication.md)** — Key generation, registration, and signature verification
- **[API](api.md)** — Complete endpoint documentation and request/response formats
- **[Notifications](notifications.md)** — How users are notified of pending approvals
