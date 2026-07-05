# ADR-002: Standing Approvals (Pre-approved, Time-bound, Multi-use Actions)

**Status:** Accepted
**Date:** 2026-02-14
**Authors:** SuperSuit team
**Supersedes:** Partially amends ADR-001 Decision 6 (single-use tokens)

## Context

ADR-001 established a "break glass" model: every action requires an approval prompt, a confirmation code, and produces a single-use token. This is the right default for high-stakes or unfamiliar operations.

But in practice, many agent workflows are repetitive and predictable:

- "Check my inbox for new emails every 10 minutes"
- "Send a daily summary email to my team"
- "Read emails from @notifications.github.com throughout the day"

Requiring human approval for each of these is unsustainable. Users will either:
1. Stop using the agent (too many prompts)
2. Approve reflexively without reading (defeats the purpose)
3. Give the agent direct credentials and bypass Permission Slip entirely (worst outcome)

We need a way for users to **approve in advance** — grant an agent permission to perform a specific action, within specific constraints, for a defined period of time, as many times as needed.

---

## Decision: Introduce Standing Approvals

### Summary

A **standing approval** is a time-bound, constraint-scoped pre-authorization that lets an agent execute a specific action type **without per-request human approval**. The agent can execute the action as many times as it wants within the approval window.

### How it works

#### User creates a standing approval

From the web interface, the user configures:

| Field | Description | Example |
|---|---|---|
| **Agent** | Which agent this applies to | "My AI Assistant" |
| **Action type** | Which action is pre-approved | `email.read` |
| **Constraints** | Boundaries the agent must stay within | sender: `*@github.com`, max results: 10 |
| **Duration** | How long the standing approval is valid | Until revoked (default), or optional user-set expiry |

#### Agent discovers its standing approvals

Standing approvals are agent-specific and require authentication, so they are **not** returned by the unauthenticated connector discovery endpoint (`GET /v1/connectors`). Instead, a dedicated authenticated endpoint (e.g., `GET /v1/agents/{agent_id}/standing-approvals`) will return the agent's active standing approvals. The exact endpoint design is deferred to implementation, but the response includes active standing approvals alongside the agent's capabilities, so the agent knows which actions it can execute immediately vs. which require one-off approval.

#### Agent executes without approval prompt

```
Agent                          Permission Slip                    Gmail
  │                                  │                              │
  │  POST /v1/approvals/request      │                              │
  │  {request_id: "...",             │                              │
  │   action: {type: "email.read",   │                              │
  │    parameters: {sender: "*@github.com"}}} │                     │
  │─────────────────────────────────>│                              │
  │                                  │ 1. Verify agent signature    │
  │                                  │ 2. Match standing approval   │
  │                                  │ 3. Validate constraints      │
  │                                  │ 4. Check TTL (when set)    │
  │                                  │ 5. Execute with credentials  │
  │                                  │─────────────────────────────>│
  │                                  │<─────────────────────────────│
  │<─────────────────────────────────│                              │
  │  {result: {emails: [...]}}       │                              │
```

No push notification. No confirmation code. No single-use token. The standing approval **is** the authorization.

#### What changes vs. one-off flow

| Aspect | One-off (ADR-001) | Standing approval |
|---|---|---|
| Approval | Per-request, push notification + confirmation code | Pre-approved for a duration |
| Authorization | `Authorization: Bearer <jwt>` header | Agent signature only (no `Authorization` header) |
| Request body | Same structure (`request_id` + `action`) | Same structure (`request_id` + `action`) |
| Executions | Exactly once per approval | Unlimited within the rule window |
| Constraints enforced | At request validation time | At execution time (every call) |
| Audit trail | One entry per approval + execution | One entry per execution (many per standing approval) |
| User involvement | Every time | Once at creation, then only to review/revoke |

---

### Standing Approval Lifecycle

```
Created ──> Active ──> Expired
              │
              └──> Revoked (user manually revokes)
```

1. **Created:** User configures the standing approval from the web UI.
2. **Active:** Agent can execute the action freely within constraints.
3. **Expired:** Duration elapsed. Agent must request a new standing approval or fall back to one-off.
4. **Revoked:** User manually cancels at any time. Takes effect immediately.

---

### Constraint enforcement

Standing approvals **do not relax constraints** — they relax the approval prompt. Every execution is still validated:

1. **Agent signature verified** (Ed25519, same as always)
2. **Standing approval matched** (agent + action type + active status)
3. **Parameters validated against constraints** (same rules as ADR-001 Decision 4)
4. **TTL checked** (standing approval not expired, when `expires_at` is set)
5. **Action executed** via connector with stored credentials
6. **Audit log entry created** (every execution, not just the standing approval creation)

If parameters violate constraints, the request is **rejected immediately** — same as the one-off flow. The standing approval doesn't make constraint violations pass.

---

### Coexistence with one-off approvals

Standing approvals and one-off approvals coexist:

- If an agent's request **matches** an active standing approval (action type + constraints), it executes immediately.
- If an agent's request **doesn't match** any standing approval (different action type, or parameters outside constraint bounds), it falls through to the one-off approval flow.
- An agent can have standing approvals for some actions and use one-off for others.

**Example:** Agent has a standing approval for `email.read` scoped to `sender: *@github.com`. If the agent tries to read emails from `sender: *@competitor.com`, that doesn't match the standing approval constraints. Permission Slip falls through to the one-off flow and sends the user a push notification.

---

### Duration

Standing approvals default to **until revoked** (`expires_at` is null). Users creating rules from the web UI may optionally set an expiry date; there is no maximum duration when a date is set.

Agent-proposed rules do not include expiry or usage limits — approved proposals always become until-revoked rules. The human reviewer controls risk via approval and instant revocation.

**Revocation is always instant.** Regardless of duration, the user can revoke any standing approval at any time from the web UI.

---

### Security considerations

**Standing approvals shift risk.** Instead of reviewing each action in real-time, the user is making a forward-looking trust decision. This is acceptable when:

1. **Constraints are tight.** A standing approval for `email.send` to `*@mycompany.com` with a 50-recipient cap is bounded. One for `email.send` to `*` with no cap is dangerous — the UI should warn about wide-open constraints.
2. **Revocation is instant.** User can kill a standing approval at any time from the web interface.
3. **Every execution is audited.** The user can review what the agent actually did during the standing approval window.
4. **Constraints are enforced every time.** A standing approval is not a blank check — it's a scoped delegation with enforcement on every call.

**UI warnings for broad standing approvals:**

The web interface SHOULD warn users when creating standing approvals with:
- No recipient constraints on `email.send`
- Multiple broad standing approvals for the same agent

---

### Data model additions

```
STANDING_APPROVAL {
    string  standing_approval_id  PK
    string  agent_id              FK
    string  user_id               FK
    string  action_type           "email.read | email.send"
    string  action_version        "1"
    json    constraints           "same schema as ACTION_CONFIG"
    string  status                "active | expired | revoked"
    timestamp starts_at
    timestamp expires_at          "nullable; null = until revoked"
    timestamp created_at
    timestamp revoked_at          "null if not revoked"
}
```

Every execution under a standing approval also writes to `AUDIT_LOG` with `event_type: "standing_execution"` and a reference to `standing_approval_id`.

---

## Rationale

- **Prevents credential bypass.** If we don't offer pre-approval, users who want automation will give agents direct credentials — defeating Permission Slip entirely.
- **Reduces approval fatigue.** Repetitive, predictable actions shouldn't require a prompt every time.
- **Constraints are still enforced.** This relaxes the *approval prompt*, not the *security boundary*.
- **Instant revocation limits open-ended delegation.** Users approve rules until revoked and can cancel at any time from the web UI.
- **User controls the risk window.** Optional expiry from the web UI and instant revocation are user-defined.
- **Progressive trust.** Users start with one-off approvals, then graduate to standing approvals for actions they trust — the system grows with the user's confidence.

## Alternatives considered

- **Longer TTL on single-use tokens.** Doesn't solve the problem — you'd need 288 tokens per day for a 5-minute check. And single-use means one execution per token.
- **Batch approvals (approve N actions at once).** Awkward UX — user has to approve a specific count upfront. Standing approvals handle recurring automation via duration and constraints instead.
- **Auto-approve rules as a separate concept.** Standing approvals already cover this — an until-revoked standing approval is functionally an auto-approve rule, managed through the same UI with revocation as the off switch.
- **Refresh tokens.** Adds complexity (refresh flow, token rotation) for something that standing approvals handle more cleanly at the authorization layer rather than the token layer.

## Consequences

- The `POST /v1/approvals/request` endpoint now handles both flows: it checks for matching standing approvals before creating a pending approval. If a standing approval matches, the action is auto-approved and executed immediately (returning `status: "approved"` with the result inline). If no standing approval matches, a pending approval is created as before (`status: "pending"`).
- The web interface needs a standing approval management UI (create, list, revoke).
- A new authenticated endpoint is needed for agents to discover their active standing approvals (separate from the unauthenticated `GET /v1/connectors` discovery endpoint).
- Audit queries need to distinguish one-off executions from standing executions.
- `expires_at` is nullable; null means until revoked. The former 90-day database CHECK constraint was removed in migration `20260317124702`.

---

## Summary

| Aspect | Decision |
|---|---|
| What | Standing approvals: time-bound, constraint-scoped pre-authorization |
| When | User creates proactively from web UI |
| Duration | Until revoked (default); optional user-set expiry from web UI |
| Executions | Unlimited within the rule window |
| Constraints | Same per-action constraints from ADR-001, enforced every execution |
| Coexistence | Falls through to one-off approval if no standing approval matches |
| Revocation | Instant, from web UI |
| Audit | Every execution logged, even under standing approvals |
