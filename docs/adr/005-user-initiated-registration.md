# ADR-005: User-Initiated Agent Registration via Invite Codes

**Status:** Accepted
**Date:** 2026-02-17
**Authors:** SuperSuit team
**Amends:** Registration flow in API spec, Architecture diagrams, Notifications spec, Terminology

## Context

The original protocol design has **agents initiate registration** by calling `POST /v1/agents/register` with an `approver` username. Permission Slip then notifies the named user, who approves or denies the request. This is the flow described in the API spec, architecture diagrams, and notifications spec prior to this ADR.

### The problem: unsolicited registration is a spam vector

Agent-initiated registration means any party who knows (or guesses) a username can trigger notifications to that user. This creates several issues:

1. **Spam / harassment:** An attacker can flood a user with registration requests they never asked for. Even rate limiting doesn't fully solve this — a single unwanted notification is already a problem.
2. **Phishing risk:** A malicious agent could present itself with a convincing name ("Google Security Agent") to trick a user into approving it.
3. **Inverted trust model:** The *agent* decides who to contact, not the *user*. The user has no prior context about why an agent is trying to register, which makes informed approval harder.
4. **Notification fatigue:** Users who receive unsolicited registration requests may start ignoring notifications entirely, undermining the security model for legitimate approval requests.

### What other systems do

- **OAuth device authorization (RFC 8628):** The user initiates the flow by navigating to a verification URL and entering a code displayed by the device. The device does not contact the user — it polls the authorization server.
- **SSH authorized_keys:** The server owner adds public keys to `~/.ssh/authorized_keys`. The connecting client doesn't decide which servers trust it.
- **GitHub deploy keys / personal access tokens:** The repository owner creates the key or token and gives it to the system that needs access.

All of these share a common pattern: **the resource owner initiates trust, not the party requesting access.**

---

## User Stories

### User (Approver) Stories

**US-1: Generate a registration invite**
> As a user, I want to generate a registration invite code from my dashboard so that I can give it to an agent I want to add to my account.

Acceptance criteria:
- User clicks "Add Agent" on the Permission Slip dashboard
- System generates a single-use invite code in the format `PS-XXXX-XXXX`
- The code and a convenience invite URL are displayed on screen for the user to copy
- The invite URL bundles the public app base URL with the code, so the agent doesn't need to know the app URL separately (useful for autonomous agents without a human operator)
- The invite has a configurable TTL (default 15 minutes, max 24 hours)
- The invite is stored as an HMAC-SHA256 hash when `INVITE_HMAC_KEY` is configured (plain SHA-256 fallback). The plaintext code is never persisted.

**US-2: See pending registration after sharing invite**
> As a user, after I share an invite code with an agent, I want to see on my dashboard that the agent has used the invite and is awaiting confirmation, so I know the process is progressing.

Acceptance criteria:
- Dashboard shows invite status: active, consumed (pending confirmation), or expired
- When an agent registers with the invite, the dashboard updates to show the agent's name and metadata
- A confirmation code is displayed for the user to communicate to the agent
- No push notification is sent — the user is already on the dashboard because they initiated the flow

**US-3: Complete registration via confirmation code**
> As a user, I want to see a confirmation code on my dashboard after the agent uses my invite, so I can give the code to the agent to complete registration.

Acceptance criteria:
- Confirmation code (e.g. `XK7-M9P`) is displayed on the dashboard after the agent registers with the invite
- User shares the code with the agent out-of-band (copy-paste, chat, config file, etc.)
- Once the agent submits the code, the dashboard updates to show "Registration complete"
- The agent appears in the user's list of registered agents

**US-4: Cancel or let an invite expire**
> As a user, I want unused invites to expire automatically, and I want the option to cancel an invite I no longer need, so that stale invites cannot be used.

Acceptance criteria:
- Invites expire after their configured TTL with no action needed
- User can manually cancel an active invite from the dashboard
- Expired and cancelled invites cannot be used for registration
- Dashboard shows expired/cancelled status for audit purposes

**US-5: Receive informational notification on registration completion**
> As a user, I want to receive an optional notification when an agent I invited completes registration, so I have confirmation even if I've navigated away from the dashboard.

Acceptance criteria:
- Notification is informational ("My Assistant has been registered"), not an approval prompt
- Notification is optional — the primary feedback channel is the dashboard
- Notification does not require any action from the user

### Agent Stories

**AS-1: Register using an invite URL**
> As an agent, I want to register with Permission Slip by POSTing to the invite URL provided by the user, so that I can start making approval requests.

Acceptance criteria:
- Agent sends `POST /invite/{invite_code}` with `request_id` and `public_key` in the JSON body
- The invite code is in the URL path (e.g., `POST /invite/PS-R7K3-X9M4`) — the agent does not need to extract it separately
- Request is signed with the agent's private key via `X-Permission-Slip-Signature`
- On success, the response includes `agent_id`, `expires_at`, and `verification_required: true`
- On invalid/expired/consumed invite, the agent receives a clear error (`invalid_invite_code`, `invite_expired`, `invite_not_found`, `invite_locked`)
- The agent does not need to know the approver's username — the invite code already maps to the user who generated it
- The agent may be autonomous (no human operator) — the invite URL is the only input needed

**AS-2: Complete registration with confirmation code**
> As an agent, after registering with an invite code, I want to submit the confirmation code given to me by the user so that my registration is finalized.

Acceptance criteria:
- Agent sends `POST /v1/agents/{agent_id}/verify` with the confirmation code
- Request is signed with the agent's private key
- On success, the response includes `status: "approved"` and `registered_at`
- On failure (wrong code), the agent receives `invalid_code` and can retry up to 5 times before lockout
- After successful verification, the agent can submit approval requests

**AS-3: Handle invite code errors gracefully**
> As an agent, when my registration fails due to an invalid, expired, or locked invite code, I want a clear error response so I can report the problem to the user or handle it autonomously.

Acceptance criteria:
- `401 invalid_invite_code` — the code doesn't match
- `404 invite_not_found` — no invite exists with this code
- `410 invite_expired` — the invite's TTL has elapsed, a new invite must be generated
- `423 invite_locked` — too many failed attempts, a new invite must be generated
- Error responses include enough detail for the agent to surface a helpful message or retry autonomously

**AS-4: Registration without a valid invite URL is rejected**
> As an agent, if I attempt to POST to an invalid invite URL (malformed code, missing path segment), my request should be rejected immediately with a clear error.

Acceptance criteria:
- `POST /invite/{invalid_code}` returns `401 invalid_invite_code` or `404 invite_not_found`
- No notification is sent to any user
- No pending registration is created
- The error message indicates that a valid invite URL is required

---

## Decision: User-Initiated Registration via Invite Codes

### Summary

Replace agent-initiated registration with **user-initiated registration via invite URLs**. The user generates a single-use, time-limited registration invite from the Permission Slip dashboard. The user shares the invite URL with the agent. The agent may be autonomous — it does not necessarily have a human operator. The agent POSTs directly to the invite URL (`POST /invite/{invite_code}`) with its public key to register.

Registration requests without a valid invite code are rejected immediately — no notification, no pending request, nothing.

### Before (agent-initiated)

```
Agent                     Permission Slip              User
  │                            │                         │
  │── POST /register ─────────▶│                         │
  │   {approver: "alice"}      │                         │
  │◀── {registration_url} ─────│                         │
  │                            │── notification ────────▶│
  │                            │   "Agent X wants access"│
  │                            │                         │── approve/deny
  │                            │◀── approve ─────────────│
  │                            │── show code ───────────▶│
  │                            │                         │── gives code to agent
  │── POST /verify(code) ─────▶│                         │
  │◀── {status: "approved"} ───│                         │
```

**Problems:**
- Any agent can target any user by username
- User receives unsolicited notifications
- No prior context for why this agent is registering

### After (user-initiated with invite URL)

```
User                        Permission Slip              Agent
 │                               │                         │
 │── "Add Agent" ───────────────▶│                         │
 │◀── invite URL ────────────────│                         │
 │   .../invite/PS-R7K3-X9M4    │                         │
 │                               │                         │
 │── shares invite URL ────────────────────────────────▶ │
 │                               │                         │
 │                               │◀── POST /invite/PS-... ─│
 │                               │    {public_key, ...}    │
 │                               │                         │
 │                               │── validate invite ──────│
 │                               │── generate conf code ───│
 │                               │──▶ {agent_id, ...} ────▶│
 │                               │                         │
 │◀── show confirmation code ────│                         │
 │                               │                         │
 │── shares confirmation code ──────────────────────────▶ │
 │                               │                         │
 │                               │◀── POST /verify(code) ──│
 │◀── "Registration complete" ───│                         │
```

**Benefits:**
- No spam: agents cannot trigger notifications without a valid invite
- User initiates: the human explicitly decides "I want to add an agent"
- Invite codes are single-use and time-limited
- Confirmation code handshake still validates that the agent holds the private key matching its public key

### How invite codes work

| Property | Value |
|---|---|
| **Generated by** | User, from the Permission Slip web dashboard (returns an invite URL the agent POSTs to) |
| **Format** | `PS-XXXX-XXXX` (8 alphanumeric characters, uppercase, no ambiguous chars) |
| **Lifetime** | Configurable by user: 15 minutes (default), up to 24 hours |
| **Usage** | Single-use — consumed when an agent successfully registers with it |
| **Storage** | HMAC-SHA256 hash stored in database when `INVITE_HMAC_KEY` is set (SHA-256 fallback); same pattern as confirmation codes |
| **Brute-force protection** | 5 failed attempts locks out the invite permanently |

### What the invite URL replaces

The old `POST /v1/agents/register` endpoint (which required an `approver` username) is replaced by `POST /invite/{invite_code}`. The invite code in the URL path maps to the user who generated the invite — the agent does not need to know the approver's username.

The registration notification changes from "unknown agent wants access" (push notification requiring immediate action) to "invited agent completed registration" (informational update on the dashboard). This is a much less alarming, much more expected interaction.

### What stays the same

- **Confirmation code handshake:** After the invite is validated, Permission Slip generates a confirmation code that the user communicates to the agent. The agent submits the code via `POST /v1/agents/{agent_id}/verify`. This step proves the agent holds the private key matching the public key it registered.
- **Ed25519 key pairs:** Agents still generate key pairs and sign all requests.
- **Brute-force lockout:** 5 failed verification attempts lock out the registration.
- **TTL enforcement:** Both invites and registrations expire after their configured TTL.

---

## Consequences

### Positive

- **Eliminates spam vector:** No unsolicited registration notifications
- **Aligns with user-first trust model:** The resource owner initiates trust
- **Simpler mental model:** "I create an invite, I give it to my agent" is intuitive
- **Reduces phishing risk:** Users only see registration activity for agents they explicitly invited
- **Dashboard becomes the control point:** Users manage invites and see registration status from one place

### Negative

- **Extra step for the user:** User must generate an invite before the agent can register (vs. agent just sending a request). This is an intentional trade-off — security over convenience.
- **Invite distribution is out-of-band:** The user must share the invite URL with the agent via some external channel (copy-paste, config file, chat, etc.). This is the same model used for SSH keys, deploy tokens, and OAuth device codes. The invite URL bundles the API endpoint and the invite code into a single value — the agent just POSTs to it.

### Neutral

- **Spec changes required:** API endpoints, notification payloads, architecture diagrams, database schema, and terminology all need updates. These are documented in the accompanying spec changes in this PR.
- **Agent SDK impact:** Agents that previously called `POST /v1/agents/register` with an `approver` username must now accept an invite URL and POST directly to it. Agents may be fully autonomous with no human operator — they only need the invite URL to register. This is a breaking change to the registration API, but the protocol is pre-1.0 and no agents are deployed yet.

---

## Alternatives Considered

### 1. Rate limiting agent-initiated registration

Add aggressive rate limits (e.g., 1 registration request per user per hour) to reduce spam.

**Rejected because:** Even one unsolicited notification is a problem. Rate limiting reduces volume but doesn't eliminate the fundamental issue that agents can target arbitrary users. It also punishes legitimate use (a user who wants to register multiple agents quickly).

### 2. CAPTCHA or proof-of-work on registration

Require agents to solve a challenge before the registration request is accepted.

**Rejected because:** Agents are automated systems — CAPTCHAs don't make sense for machine-to-machine communication. Proof-of-work adds latency without addressing the core trust inversion (the agent still decides who to contact).

### 3. Allowlist of approved agent IDs

Users pre-register agent IDs they're willing to accept registrations from.

**Rejected because:** Agent IDs are derived from public keys, which are generated at registration time. The user can't know the agent ID before the agent generates its key pair. This creates a chicken-and-egg problem.

### 4. Keep agent-initiated but require email verification

Agent sends registration request, but instead of a push notification, the user must click a link in an email to even see the request.

**Rejected because:** This adds friction without changing the fundamental model. The agent can still spam the user's email inbox. The invite code approach is strictly better — the user never receives anything they didn't ask for.
