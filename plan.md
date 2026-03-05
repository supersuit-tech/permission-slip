# Plan: Agent Registration Clarity

## Problem

When an agent registers with Permission Slip, it has no idea:
1. **Who** it registered to (no user identity in any response)
2. **What to do next** (no instructions or guidance in registration responses)
3. **What it can do** through that user (capabilities endpoint exists but agent doesn't know to call it, and it doesn't surface the user context)

For agents registered to multiple users (each registration = a separate `agent_id`), the agent has no way to correlate IDs to users.

## Changes

### 1. Add `approver` context to registration responses

Add an `approver` object to both registration step responses so the agent immediately knows who it's working with.

**`POST /invite/{code}` response** — add:
```json
{
  "agent_id": 42,
  "expires_at": "...",
  "verification_required": true,
  "approver": {
    "username": "alice"
  }
}
```

**`POST /v1/agents/{agent_id}/verify` response** — add:
```json
{
  "status": "registered",
  "registered_at": "...",
  "approver": {
    "username": "alice"
  }
}
```

We expose `username` only (not email/phone) — it's the user's public-facing identifier and is not PII.

**Files to change:**
- `api/registration.go` — add `approver` field to both response structs; look up profile in both handlers
- `spec/openapi/components/schemas/registration.yaml` — add `approver` to both response schemas
- `spec/openapi/components/paths/registration.yaml` — update examples

### 2. Add `next_steps` instructions to registration responses

Add a `next_steps` field with machine-readable instructions telling the agent what to do at each stage.

**`POST /invite/{code}` response** — add:
```json
{
  "next_steps": {
    "action": "verify_registration",
    "instructions": "A confirmation code will be displayed on the user's Permission Slip dashboard. Ask the user for this code, then submit it to complete registration.",
    "verify_url": "/v1/agents/42/verify"
  }
}
```

**`POST /v1/agents/{agent_id}/verify` response** — add:
```json
{
  "next_steps": {
    "action": "discover_capabilities",
    "instructions": "Registration complete. Call the capabilities endpoint to discover what actions you can perform on behalf of this user.",
    "capabilities_url": "/v1/agents/42/capabilities"
  }
}
```

**Files to change:**
- `api/registration.go` — add `next_steps` to response structs and populate with URLs/instructions
- `spec/openapi/components/schemas/registration.yaml` — add `next_steps` schemas

### 3. Add `approver` context to capabilities response

The capabilities endpoint already scopes correctly to the user on the backend, but the agent can't see *whose* capabilities these are. Add the approver context.

**`GET /v1/agents/{agent_id}/capabilities` response** — add:
```json
{
  "agent_id": 42,
  "approver": {
    "username": "alice"
  },
  "connectors": [...]
}
```

**Files to change:**
- `api/capabilities.go` — add `approver` field to `capabilitiesResponse`; look up profile
- `db/capabilities.go` — either join profiles in existing query or fetch separately
- `spec/openapi/components/schemas/capabilities.yaml` — add `approver` to `AgentCapabilitiesResponse`
- `spec/openapi/components/paths/capabilities.yaml` — update description and examples

### 4. Add `approver` context to `GET /v1/agents/me`

The agent self-status endpoint currently excludes approver info. Add it so the agent always knows who it belongs to.

**`GET /v1/agents/me` response** — add:
```json
{
  "agent_id": 42,
  "status": "registered",
  "approver": {
    "username": "alice"
  },
  ...
}
```

**Files to change:**
- `api/agents.go` — add `approver` field to `agentSelfResponse`; look up profile in handler
- `spec/openapi/components/schemas/agents.yaml` — add `approver` to `AgentSelfResponse`
- `spec/openapi/components/paths/agents.yaml` — update `agentMe` description and example

### 5. Shared `approver` type

To avoid duplication, create a small shared struct/schema:

**Go struct** (in `api/registration.go` or a new `api/types.go`):
```go
type approverInfo struct {
    Username string `json:"username"`
}
```

**OpenAPI schema** (new file or inline in `schemas/common.yaml`):
```yaml
ApproverInfo:
  type: object
  required:
    - username
  properties:
    username:
      type: string
      description: The approver's username on Permission Slip
```

This keeps it DRY and makes it easy to add fields later (e.g. display_name).

### 6. Update OpenAPI descriptions with agent guidance

Update the endpoint descriptions to be more instructive for agents:

- **`POST /invite/{code}`** description: add a "What happens next" section explaining the confirmation code flow
- **`POST /v1/agents/{agent_id}/verify`** description: add a "You're registered — now what?" section pointing to capabilities discovery
- **`GET /agents/{agent_id}/capabilities`** description: already good, but add note that `approver.username` identifies whose capabilities these are

### 7. Tests

- **Registration handler tests**: verify `approver` and `next_steps` fields in both responses
- **Capabilities handler tests**: verify `approver` field in response
- **Agent self endpoint tests**: verify `approver` field in response
- **OpenAPI spec validation**: ensure spec still validates after schema changes

## Out of scope (for now)

- **`GET /v1/agents/{agent_id}/registrations` (list all registrations)**: This would let an agent enumerate all its user relationships. Useful but requires a new endpoint and auth model (authenticate by public key, not agent_id). Defer to a follow-up.

## Order of implementation

1. Shared `approverInfo` type (Go struct + OpenAPI schema)
2. Registration responses (invite + verify) — approver + next_steps
3. Capabilities response — approver
4. Agent self response — approver
5. Tests for all changes
6. OpenAPI description updates
