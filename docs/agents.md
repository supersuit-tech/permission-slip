# Permission Slip Agent Guide

How to integrate your autonomous agent with Permission Slip.

**Base URL**: `https://app.permissionslip.dev/api/v1`

> **Note**: The invite endpoint (`POST /invite/{code}`) is served at the host root (`https://app.permissionslip.dev/invite/{code}`), not under the API base. All other endpoints use the API base above.

---

## Overview

Permission Slip sits between your agent and external services (Gmail, Stripe, GitHub, etc.). Your agent submits structured **actions** for human approval, and Permission Slip executes them using the user's stored credentials. Your agent never sees credentials and can never make arbitrary API calls.

```
Agent ──→ Permission Slip ──→ External Service
               │
               ▼
          User (approve/deny)
```

## Quick Start (CLI — recommended)

The `@permission-slip/cli` npm package handles key generation, signing, config
storage, and API calls automatically. Requires Node.js 18+.

```bash
# Step 1: Register (user shares an invite code with you)
npx @permission-slip/cli register --invite-code <code>

# Step 2: Verify (user shares the confirmation code from their dashboard)
npx @permission-slip/cli verify --code <confirmation_code>

# Step 3: Discover what you can do
npx @permission-slip/cli capabilities

# Step 4: Request approval and execute
npx @permission-slip/cli request --action email.send --params '{"to":"user@example.com","subject":"Hi"}'
npx @permission-slip/cli execute --token <token> --action email.send --params '{"to":"user@example.com","subject":"Hi"}'
```

Run `npx @permission-slip/cli --help` for all available commands.

## Manual Quick Start (raw HTTP)

If you can't use Node.js, you can interact with the API directly:

1. [Generate an Ed25519 key pair](#1-generate-your-key-pair)
2. [Register with an invite URL](#2-register-with-permission-slip)
3. [Discover available actions](#3-discover-available-actions)
4. [Request approval and execute](#4-request-approval-and-execute)

---

## 1. Generate Your Key Pair

Permission Slip uses Ed25519 keys for agent identity. Generate one:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/permission_slip_agent -N "" -C ""
```

This creates:
- `~/.ssh/permission_slip_agent` — **private key** (keep secret, never share)
- `~/.ssh/permission_slip_agent.pub` — **public key** (sent to Permission Slip during registration)

Read the public key:

```bash
cat ~/.ssh/permission_slip_agent.pub
# ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
```

Only `ssh-ed25519` keys are supported. RSA, ECDSA, and other key types are rejected.

---

## 2. Register with Permission Slip

### How Registration Works

1. A user creates an **invite URL** from the Permission Slip dashboard and shares it with you
2. You POST your public key to the invite URL
3. The user sees a **confirmation code** on their dashboard and sends it to you
4. You submit the confirmation code to complete registration
5. You're registered and can start requesting actions

### Step 2a: POST to the Invite URL

The user gives you a URL like `https://app.permissionslip.dev/invite/PS-ABCD-1234`.

```http
POST /invite/PS-ABCD-1234
Content-Type: application/json
X-Permission-Slip-Signature: <see "Signing Requests" below>

{
  "request_id": "6f1a7c30-9b2e-4d91-8c3f-2a4b6c7d8e9f",
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...",
  "metadata": {
    "name": "My Agent",
    "version": "1.0.0"
  }
}
```

**Response** (200):

```json
{
  "agent_id": 42,
  "expires_at": "2026-02-23T12:25:00Z",
  "verification_required": true
}
```

Save `agent_id` — you'll use it in all future requests.

**Notes**:
- `request_id` must be a non-empty string (e.g., a UUID) — see [Request ID](#request-id) for details
- `metadata` is optional but recommended — it helps the user identify your agent
- Invites expire (default 15 minutes) and are single-use
- For the registration request specifically, use `agent_id` of `9223372036854775807` (max int64) in the signature header since you don't have an ID yet

### Step 2b: Verify with Confirmation Code

The user will send you a 6-character confirmation code (e.g., `XK7-M9P`) from their dashboard.

```http
POST /agents/42/verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "confirmation_code": "XK7-M9P"
}
```

**Response** (200):

```json
{
  "status": "registered",
  "registered_at": "2026-02-23T12:20:15Z"
}
```

You're now registered. The confirmation code is case-insensitive and the hyphen is optional (`XK7M9P` works too).

**Timing**: You have ~5 minutes from registration to submit the code before it expires.

### Check Your Registration Status

After registering, you can check your own agent record at any time:

```http
GET /agents/me
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...
```

**Response** (200):

```json
{
  "agent_id": 42,
  "status": "registered",
  "metadata": { "name": "My Agent", "version": "1.0.0" },
  "registered_at": "2026-02-23T12:20:15Z",
  "last_active_at": "2026-02-23T12:25:00Z",
  "created_at": "2026-02-23T12:15:00Z"
}
```

This is a lightweight endpoint for health-checking your registration status. Only registered agents can call it — pending or deactivated agents receive a `404`.

---

## 3. Discover Available Actions

### Global Connector Catalog (Public)

See what services and actions Permission Slip supports — no authentication required:

```http
GET /connectors
```

```json
{
  "data": [
    {
      "id": "gmail",
      "name": "Gmail",
      "description": "Send and manage emails via Gmail API",
      "actions": ["email.send", "email.read"],
      "required_credentials": ["gmail"]
    },
    {
      "id": "stripe",
      "name": "Stripe",
      "description": "Payment processing via Stripe API",
      "actions": ["payment.charge"],
      "required_credentials": ["stripe"]
    }
  ]
}
```

### Action Parameter Schema (Public)

Get full parameter schemas for a specific connector's actions:

```http
GET /connectors/gmail
```

```json
{
  "id": "gmail",
  "name": "Gmail",
  "actions": [
    {
      "action_type": "email.send",
      "name": "Send Email",
      "description": "Send an email via Gmail",
      "risk_level": "low",
      "parameters_schema": {
        "type": "object",
        "required": ["to", "subject", "body"],
        "properties": {
          "to": {
            "type": "array",
            "items": { "type": "string", "format": "email" },
            "description": "Recipient email addresses"
          },
          "subject": {
            "type": "string",
            "maxLength": 998,
            "description": "Email subject line"
          },
          "body": {
            "type": "string",
            "description": "Email body (plain text or HTML)"
          }
        }
      }
    }
  ]
}
```

Use `parameters_schema` (JSON Schema) to validate your parameters before submitting.

### Your Agent's Capabilities (Authenticated)

See what **you specifically** can do — which connectors are enabled for you, what action configurations are available, and what standing approvals you have:

```http
GET /agents/42/capabilities
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...
```

```json
{
  "agent_id": 42,
  "connectors": [
    {
      "id": "gmail",
      "name": "Gmail",
      "credentials_ready": true,
      "actions": [
        {
          "action_type": "email.send",
          "name": "Send Email",
          "risk_level": "low",
          "parameters_schema": {
            "type": "object",
            "required": ["to", "subject", "body"],
            "properties": {
              "to": { "type": "array", "items": { "type": "string", "format": "email" } },
              "subject": { "type": "string", "maxLength": 998 },
              "body": { "type": "string" }
            }
          },
          "action_configurations": [
            {
              "configuration_id": "ac_1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d",
              "name": "Send weekly report to Alice",
              "parameters": {
                "to": "alice@example.com",
                "subject": "Weekly Report",
                "body": "*"
              },
              "credential_ready": true
            },
            {
              "configuration_id": "ac_9f8e7d6c5b4a39281706f5e4d3c2b1a0",
              "name": "Email anyone at the company",
              "parameters": {
                "to": "*",
                "subject": "*",
                "body": "*"
              },
              "credential_ready": true
            }
          ],
          "standing_approvals": [
            {
              "standing_approval_id": "sa_def456",
              "constraints": { "recipient_pattern": "*@mycompany.com" },
              "max_executions": 100,
              "executions_remaining": 88,
              "expires_at": "2026-05-15T00:00:00Z"
            }
          ]
        }
      ]
    },
    {
      "id": "stripe",
      "name": "Stripe",
      "credentials_ready": false,
      "credentials_setup_url": "https://app.permissionslip.dev/connect/stripe",
      "actions": [ ... ]
    }
  ]
}
```

#### Action Configurations

**Action configurations are the core unit of permission.** A user creates configurations that define exactly how an agent is allowed to use an action — which parameters are locked to specific values, which are free (wildcard `"*"`), and which credential to use. The agent selects from these pre-configured options rather than composing requests from scratch.

Each configuration in the response includes:

| Field | Description |
|---|---|
| `configuration_id` | Unique ID to reference when requesting approval or executing |
| `name` | User-facing label describing what this configuration permits |
| `parameters` | The configured values — fixed values are locked, `"*"` means the agent chooses |
| `credential_ready` | Whether a credential is bound and available for execution |

**Wildcard examples:**

- `"to": "alice@example.com"` — locked: must be exactly this value
- `"to": "*"` — wildcard: agent can use any value that passes schema validation
- `"to": "*@mycompany.com"` — pattern wildcard (future): restricted to a pattern

The `parameters_schema` is still included on the action for context so the agent understands what each parameter means and what values are valid.

#### Decision Tree

```
Has action_configurations for this action?
├─ YES → Pick the configuration that fits your task
│        └─ credential_ready?
│           ├─ NO  → Tell user: "Configuration X needs a credential"
│           └─ YES → Has standing_approvals with matching constraints?
│                    ├─ YES → Execute directly (reference configuration_id)
│                    └─ NO  → Request one-off approval (reference configuration_id)
└─ NO  → Tell user: "I need <action> configured — can you set that up?"
```

If no configuration matches what you need to do, you **cannot proceed** — requests for unconfigured actions are rejected. Ask the user to create an appropriate configuration from their dashboard.

---

## 4. Request Approval and Execute

### Option A: One-Off Approval (Default)

For actions without a matching standing approval, you need per-request approval:

#### 4a-1. Request Approval

Reference the `configuration_id` from your capabilities response. For wildcard parameters, provide the actual values you want to use:

```http
POST /approvals/request
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "agent_id": 42,
  "request_id": "unique-uuid-here",
  "configuration_id": "ac_1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d",
  "parameters": {
    "body": "Here are this week's highlights..."
  },
  "context": {
    "description": "Sending weekly report email to Alice",
    "risk_level": "low"
  }
}
```

**Response** (200):

```json
{
  "approval_id": "appr_xyz789",
  "approval_url": "https://app.permissionslip.dev/.../approve/appr_xyz789",
  "status": "pending",
  "expires_at": "2026-02-23T12:30:00Z",
  "verification_required": true
}
```

The user receives a notification and reviews the request on their dashboard.

#### 4a-2. Wait for Approval

The user reviews and approves (or denies) the request. After approving, they see a confirmation code and share it with you.

The Permission Slip CLI returns immediately by default with an `approval_id` and a `next_step` hint. Use `permission-slip status <approval_id>` to check the result once the user has approved. Pass `--wait` to `request` if you prefer blocking behavior.

```bash
# Default: returns immediately with approval_id
permission-slip request --action email.send --params '{"to":["alice@example.com"]}'

# Check result (single snapshot):
permission-slip status <approval_id>

# Fire-and-forget, then block later when ready:
APPROVAL_ID=$(permission-slip request --action email.send --params '{}' | jq -r '.approval_id')
# ... do other work ...
permission-slip status --wait "$APPROVAL_ID"

# Block from the start (up to 120s):
permission-slip request --action email.send --params '{}' --wait

# Block with custom timeout:
permission-slip request --action email.send --params '{}' --wait --timeout 30

# Agent-friendly: poll at a fixed interval (default 5s), exit code 2 on timeout (default 600s):
permission-slip request --action email.send --params '{}' --poll

# Custom poll interval and timeout:
permission-slip request --action email.send --params '{}' \
  --poll --poll-interval 10 --poll-timeout 120
```

Alternatively, the user can share the approval confirmation code out-of-band (paste it in chat, set it in your config, etc.).

#### 4a-3. Verify and Get Token

Submit the confirmation code to get a single-use execution token:

```http
POST /approvals/appr_xyz789/verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "request_id": "another-unique-uuid",
  "confirmation_code": "RK3-P7M"
}
```

**Response** (200):

```json
{
  "status": "approved",
  "token": {
    "access_token": "eyJhbGciOi...",
    "expires_at": "2026-02-23T12:35:00Z",
    "scope": "email.send",
    "scope_version": "1"
  }
}
```

**If the user denied the request**, you get a `403` with `"code": "approval_denied"`. Do not retry without user intervention.

#### 4a-4. Execute with Token

```http
POST /actions/execute
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "token": "eyJhbGciOi...",
  "action_id": "email.send",
  "parameters": {
    "to": ["recipient@example.com"],
    "subject": "Hello from your agent",
    "body": "This email was sent on your behalf."
  }
}
```

**Important**: The `parameters` must be **exactly** the same as what you submitted in the approval request. Permission Slip hashes them and compares against the approved hash.

**Response** (200):

```json
{
  "status": "success",
  "action_id": "email.send",
  "executed_at": "2026-02-23T12:26:00Z",
  "result": {
    "message_id": "msg_abc123",
    "delivery_status": "queued"
  }
}
```

The token is consumed on use (even if the external service errors). To retry, request a new approval.

### Option B: Standing Approval (Pre-Authorized)

If the capabilities endpoint shows a standing approval that matches your action and parameters, you can execute directly without per-request approval. Reference the `configuration_id` to specify which configuration you're operating under:

```http
POST /actions/execute
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "request_id": "unique-uuid-here",
  "configuration_id": "ac_9f8e7d6c5b4a39281706f5e4d3c2b1a0",
  "parameters": {
    "to": ["coworker@mycompany.com"],
    "subject": "Meeting notes",
    "body": "Here are the notes from today's meeting."
  }
}
```

No `token` field — Permission Slip matches the request against active standing approvals automatically. The `configuration_id` identifies which action configuration the agent is operating under; the `parameters` supply values for any wildcard fields.

**Response** (200):

```json
{
  "result": { "message_id": "msg_def456" },
  "standing_approval_id": "sa_def456",
  "executions_remaining": 87
}
```

If no standing approval matches, you get a `404` with a hint to use the one-off approval flow.

---

## Signing Requests

Every request to Permission Slip (except the connector discovery endpoints `GET /connectors` and `GET /connectors/{id}`) must include an `X-Permission-Slip-Signature` header.

### Header Format

```
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", timestamp="1708617600", signature="<base64url>"
```

### How to Compute the Signature

1. **Build the canonical request string** (5 lines, joined by `\n`):

   ```
   POST
   /approvals/request

   1708617600
   a3b8c9d0e1f2...
   ```

   The 5 components are:
   - **HTTP method** (uppercase)
   - **URL path** (the router path as listed in this doc — e.g., `/agents/42/verify`, not `/api/v1/agents/42/verify`. The `/invite/{code}` path is used as-is since it's served at the host root.)
   - **Query string** (sorted params, RFC 3986 encoding; empty string if no params)
   - **Timestamp** (Unix seconds, from the signature header)
   - **Body hash** (lowercase hex SHA-256 of the raw JSON body; use `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` for empty/GET requests)

2. **Sign** the canonical string bytes with your Ed25519 private key (RFC 8032)

3. **Encode** the 64-byte signature as base64url with no padding

4. **Build the header** with your agent ID, algorithm, timestamp, and signature

### Timestamp Window

Signatures are valid for **300 seconds (5 minutes)**. Keep your system clock synchronized.

### Registration Exception

During registration (`POST /invite/{code}`), you don't have an agent ID yet. Use `9223372036854775807` (max int64) as a placeholder in the signature header. The server ignores this value for registration requests and verifies the signature using the public key from the request body.

### Request ID

All POST requests must include a non-empty `request_id` (for example, a UUID v4) in the JSON body.

The current implementation validates only that `request_id` is present and non-empty; it does not yet enforce uniqueness, idempotency, or replay protection based on this field.

---

## Cancelling Requests — *Planned*

If you need to cancel a pending approval (user changed their mind, action no longer relevant):

```http
POST /approvals/appr_xyz789/cancel
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", ...

{
  "request_id": "unique-uuid-here"
}
```

Only the agent that created the approval can cancel it. You cannot cancel already-approved, denied, or expired approvals.

---

## Error Handling

All errors follow a consistent format:

```json
{
  "error": {
    "code": "error_code_here",
    "message": "Human-readable description",
    "retryable": false,
    "details": { },
    "trace_id": "trace_abc123"
  }
}
```

### Common Error Codes

| Code | HTTP | Meaning |
|---|---|---|
| `invalid_signature` | 401 | Signature verification failed — check signing logic |
| `timestamp_expired` | 401 | Signature timestamp outside 5-minute window — sync clock |
| `invalid_public_key` | 400 | Public key format invalid — must be `ssh-ed25519 ...` |
| `agent_id_mismatch` | 400 | Agent ID in URL path doesn't match signature header |
| `agent_not_authorized` | 403 | Agent not registered with this approver |
| `invite_expired` | 410 | Invite TTL elapsed — user must create a new one |
| `invite_locked` | 423 | 5 failed registration attempts — user must create new invite |
| `registration_expired` | 410 | Didn't verify within 5 minutes — re-register |
| `invalid_code` | 401 | Wrong confirmation code — check `attempts_remaining` in details |
| `approval_denied` | 403 | User explicitly denied — do not retry without user intervention |
| `approval_expired` | 410 | Approval TTL elapsed — re-request |
| `invalid_parameters` | 403 | Parameters don't match what was approved — must match exactly |
| `token_already_used` | 403 | Token consumed — request new approval |
| `insufficient_scope` | 403 | Token scope doesn't match action — check `action_id` |
| `credentials_not_found` | 404 | User hasn't set up credentials for this service |
| `duplicate_request_id` | 409 | `request_id` already used — generate a new UUID (not yet enforced; reserved for future use) |
| `no_matching_standing_approval` | 404 | No standing approval matches — use one-off flow |
| `constraint_violation` | 403 | Parameters violate standing approval constraints |
| `upstream_error` | 502 | External service returned an error |

### Retry Guidance

- **`retryable: true`** — safe to retry with exponential backoff
- **`retryable: false`** — do not retry without changing the request or getting user intervention
- **429 Too Many Requests** — back off and retry after the `Retry-After` header value
- **502 Bad Gateway** — external service issue; for one-off approvals the token is consumed so you need a new approval to retry

---

## Action Type Format

Action types follow these conventions:

**Standard types** (2 segments): `<category>.<operation>`
- Examples: `email.send`, `payment.charge`, `calendar.create`
- Max length: 64 characters

**Custom types** (4+ segments): `<reverse-domain>.<category>.<operation>`
- Examples: `com.example.deploy.production`, `io.myservice.report.generate`
- Max length: 128 characters

Use `GET /connectors` to see which action types are available.

---

## Complete Lifecycle Summary

```
1. GENERATE KEY     ssh-keygen -t ed25519
2. RECEIVE INVITE   User shares invite URL
3. REGISTER         POST /invite/{code}              → get agent_id
4. VERIFY           POST /agents/{id}/verify         → status: registered
5. DISCOVER         GET  /agents/{id}/capabilities   → action configurations + standing approvals
6. PICK CONFIG      Select a configuration_id that fits the task
7. REQUEST          POST /approvals/request          (planned) → reference configuration_id
8. GET CODE         User approves, shares confirmation code
9. VERIFY APPROVAL  POST /approvals/{id}/verify      (planned) → get token
10. EXECUTE         POST /actions/execute            (planned) → action result
```

Steps 6-10 repeat for each action. With standing approvals, steps 7-9 are skipped. If no configuration matches, ask the user to create one.

---

## Reference

| Endpoint | Auth | Status | Description |
|---|---|---|---|
| `GET /connectors` | None | Implemented | List all available connectors and action types |
| `GET /connectors/{id}` | None | Implemented | Get connector details with parameter schemas |
| `POST /invite/{code}` | Signature | Implemented | Register with an invite URL (served at host root, not under API base) |
| `POST /agents/{id}/verify` | Signature | Implemented | Verify registration with confirmation code |
| `GET /agents/me` | Signature | Implemented | Get your own agent record (status, metadata, timestamps) |
| `GET /agents/{id}/capabilities` | Signature | Implemented | Discover action configurations, connectors, actions, and standing approvals |
| `POST /approvals/request` | Signature | Implemented | Request one-off approval for an action |
| `GET /approvals/{id}/status` | Signature | Implemented | Poll approval status (pending → approved/denied/cancelled/expired) |
| `POST /approvals/{id}/cancel` | Signature | Implemented | Cancel a pending approval request |
| `POST /approvals/{id}/verify` | Signature | Planned | Submit approval confirmation code, get execution token |
| `POST /actions/execute` | Signature | Planned | Execute an action (with token or standing approval) |

Full machine-readable spec: [`spec/openapi/`](../spec/openapi/)
