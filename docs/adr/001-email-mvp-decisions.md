# ADR-001: Email MVP Architectural Decisions

**Status:** Accepted
**Date:** 2026-02-14
**Authors:** SuperSuit team

## Context

Permission Slip is a middle-man service that mediates between AI agents and external APIs. We need to ship an MVP that demonstrates the core value proposition — human-approved AI agent actions — with a concrete, useful vertical. This ADR captures the key decisions made for that MVP.

---

## Decision 1: Gmail as the first integration

### Status
Accepted

### Context
We need one external service to build the MVP around. Candidates included email (Gmail), payments (Stripe), calendar (Google Calendar), and file storage (Google Drive).

### Decision
Build the MVP around Gmail, exposing two action types: `email.read` and `email.send`.

### Rationale
- **High frequency, low risk starting point.** Email is something agents need to do constantly (summarize inbox, send replies, draft messages) — it demonstrates clear value immediately.
- **Rich approval UX.** Email actions have human-readable parameters (recipients, subject, body) that make the approval screen intuitive. Users can see exactly what will be sent.
- **OAuth is well-documented.** Gmail API + Google OAuth 2.0 is mature, well-supported, and familiar to developers.
- **Two complementary actions.** `email.read` (retrieve) and `email.send` (mutate) exercise both read and write paths through the system, validating the full architecture.

### Alternatives considered
- **Stripe** — Higher stakes (money), more friction for testing, narrower audience.
- **Calendar** — Lower perceived value; "create an event" doesn't feel urgent enough to justify an approval flow.
- **Multi-service from day one** — Spreads effort thin; better to nail one integration deeply.

---

## Decision 2: Middle-man architecture (not an open protocol)

### Status
Accepted

### Context
Earlier iterations of Permission Slip envisioned an open protocol that every external service would implement. This required service-side SDKs, client-side SDKs, and broad adoption before it became useful.

### Decision
Permission Slip is a centralized service that acts as a proxy. External services (Gmail, etc.) are unaware of Permission Slip — they receive normal API calls made with the user's stored credentials.

### Rationale
- **Works today.** No adoption required from external services.
- **Single integration point for agents.** One API to learn, not one per service.
- **Credential isolation by design.** Agents never see credentials because they never talk to external services.
- **Consistent UX.** Same approval flow regardless of the underlying service.
- **New connectors expand capability without protocol changes.**

### Consequences
- Permission Slip is a single point of failure and a high-value target — must be hardened.
- We bear operational cost of maintaining connectors for each external service.
- We hold user credentials — vault security is critical.

---

## Decision 3: Actions as the core primitive

### Status
Accepted

### Context
We needed to decide what agents actually submit to Permission Slip. Options ranged from free-form API proxying to structured, pre-defined operations.

### Decision
Agents submit structured, pre-defined **actions** (e.g., `email.send` with a typed parameter schema). They cannot make arbitrary API calls.

### Rationale
- **Security boundary.** Constraining agents to known action types prevents entire classes of attacks (arbitrary API calls, parameter injection, scope creep).
- **Human-readable approvals.** Structured actions render cleanly in the approval UI ("Send email to bob@example.com" vs. raw JSON).
- **Validation before approval.** Parameter schemas catch malformed requests before they reach the user.
- **Extensibility.** New action types can be added without changing the core protocol.

### Consequences
- Every new capability requires defining an action type and schema.
- Agents that need operations we haven't defined yet are blocked until we add them.
- Custom action types (reverse-DNS naming) provide an escape hatch for service-specific operations.

---

## Decision 4: Per-agent, per-action constraints

### Status
Accepted

### Context
Users need more granularity than just "agent X can use `email.send`." They want to say "agent X can send emails, but only to people at @mycompany.com."

### Decision
Users configure **constraints** on each action for each agent. Constraints are enforced at request validation time, before the approval prompt is shown.

### Constraints for `email.read`:
- Subject filter, sender filter, max results, date range

### Constraints for `email.send`:
- Recipient whitelist/patterns, max attachment size, allowed attachment types, max recipients, body length limit

### Rationale
- **Defense in depth.** Constraints reject out-of-bounds requests before they reach the user, reducing approval fatigue.
- **Per-agent scoping.** Different agents can have different boundaries for the same action type.
- **All optional.** Unconfigured constraints impose no restriction — progressive tightening, not mandatory lockdown.

### Consequences
- Constraint enforcement adds validation logic to the request path.
- Constraint schemas need to be defined per action type.
- Users need a UI to configure constraints — adds surface area to the web interface.

---

## Decision 5: Ed25519 signatures for agent identity

### Status
Accepted

### Context
Agents need to prove their identity on every request. Options included API keys, OAuth client credentials, mTLS, and asymmetric signatures.

### Decision
Agents hold an Ed25519 key pair. Every request includes an `X-Permission-Slip-Signature` header with the agent ID, timestamp, and signature over the canonical request.

### Rationale
- **No shared secrets.** Permission Slip stores only the public key — a breach doesn't compromise agent identity.
- **Request integrity.** Signatures cover method, path, query, timestamp, and body hash — tampering is detectable.
- **Replay protection.** 5-minute timestamp window + `request_id` deduplication.
- **Ed25519 specifically.** Fast, small keys (32 bytes), deterministic signatures, no parameter choice footguns. ECDSA P-256 supported as a fallback.

### Consequences
- Agents must manage a private key (generation, storage, rotation).
- SDK must implement canonical request construction and signing.
- Signature verification adds latency to every request (minimal for Ed25519).

---

## Decision 6: Single-use JWT tokens for action execution

### Status
Accepted

### Context
After approval, the agent needs something to present to trigger execution. Options included session tokens, long-lived API keys, or single-use tokens.

### Decision
Approved actions produce a **single-use JWT** (ES256) that the agent presents back to `POST /v1/approvals/request` for execution. The token is:
- **Single-use:** tracked by `jti` claim with atomic check-and-set
- **Time-bound:** expires with the approval TTL (default 5 minutes)
- **Parameter-bound:** `params_hash` (SHA-256 of JCS-canonicalized parameters) prevents tampering
- **Not a refresh token:** no renewal mechanism by design

### Rationale
- **Minimal blast radius.** A leaked token can only do one thing, one time, for a few minutes.
- **No scope creep.** The token is bound to the exact parameters that were approved.
- **Auditable.** Every token use is a single, traceable event.
- **No refresh tokens** forces the agent back through the approval flow for every action — this is a feature, not a limitation, for the one-off flow.

### Consequences
- Agents must use tokens promptly after receiving them.
- Failed execution (network error) while the token is still valid can be retried; after expiry, a new approval is needed.
- `jti` tracking requires atomic storage (Redis `SET NX`, DB upsert, etc.).

**Note:** [ADR-002](002-standing-approvals.md) introduces **standing approvals** as a complement to single-use tokens. For repetitive, pre-authorized actions, standing approvals bypass the token mechanism entirely — the standing approval itself is the authorization. Single-use tokens remain the authorization mechanism for one-off actions.

---

## Decision 7: Confirmation codes for out-of-band verification

### Status
Accepted

### Context
After a user approves a request in the app, we need a mechanism for the agent to prove that the user actually approved. Without this, a compromised notification channel could forge approvals.

### Decision
After approval, the user sees a **6-character confirmation code** (e.g., `XK7-M9P`). The agent must submit this code to complete verification. The code uses a 32-character alphabet (uppercase alphanumeric minus `0`, `O`, `1`, `I`), displayed as `XXX-XXX`.

### Rationale
- **Out-of-band verification.** The code is shown on the user's device; the agent must obtain it through a separate channel — this prevents approval forgery.
- **Phishing resistance.** User sees the code on a different device/app than the one the agent operates on.
- **Brute-force resistant.** 32^6 (~1 billion) combinations with a 5-attempt lockout = ~1 in 200 million chance of guessing.
- **Human-friendly.** Ambiguous characters removed, hyphen-separated groups, case-insensitive input.

### Consequences
- Adds a step to the flow (user must communicate code back to agent).
- Code TTL is tied to approval TTL — expired approval = expired code.
- Lockout after 5 failed attempts requires restarting the approval flow.

---

## Decision 8: RFC 8785 (JCS) for JSON canonicalization

### Status
Accepted

### Context
Both request signing (`BODY_HASH`) and token verification (`params_hash`) require computing a deterministic hash of JSON data. Naive approaches (`json.dumps(sort_keys=True)`) fail on edge cases: Unicode normalization, number formatting (`1.0` vs `1`), key ordering with non-ASCII characters.

### Decision
All JSON canonicalization MUST use **RFC 8785 JSON Canonicalization Scheme (JCS)**, implemented via language-specific libraries:
- **Go:** `github.com/cyberphone/json-canonicalization`
- **JavaScript:** `canonicalize` (npm)
- **Python:** `jcs` (pip)

### Rationale
- **Interoperability.** JCS guarantees identical byte output across languages — agents and the server will always compute the same hash.
- **Standard, not ad-hoc.** RFC 8785 handles all edge cases (Unicode, numbers, escaping) that custom approaches miss.
- **Proven.** Well-tested libraries available for all major languages.

### Consequences
- Every implementation language needs a JCS library as a dependency.
- Test vectors (included in the API spec) must be validated during integration testing.
- Slightly more complex than naive key-sorting, but eliminates an entire class of signature verification bugs.

---

## Decision 9: Push notifications as the primary approval channel

### Status
Accepted

### Context
Users need to be notified when an agent requests an action. Options included email, SMS, polling, webhooks, and push notifications.

### Decision
The primary notification channel is **push notifications** to the user's device/browser. Webhook delivery is supported as a secondary channel for programmatic consumers.

### Rationale
- **Low latency.** Push notifications arrive in seconds — critical for keeping the agent's 5-minute TTL window viable.
- **User is in control.** Notification appears on the user's device; they act when ready.
- **Rich rendering.** Push notifications can include the action summary, agent name, and risk level.
- **Webhooks as escape hatch.** Supports automation and custom UIs for power users.

### Consequences
- Requires push notification infrastructure (APNs, FCM, or web push).
- Users must have notifications enabled — if disabled, approvals rely on the user opening the web interface.
- 5-minute TTL means missed notifications = expired requests.

---

## Decision 10: 5-minute default TTL for approvals and tokens

### Status
Accepted

### Context
Approval requests and tokens need a time limit. Too short and users can't respond in time; too long and stale approvals become a security risk.

### Decision
Default TTL is **5 minutes** (300 seconds) for both approval requests and tokens. The timestamp window for signature verification is also 5 minutes.

### Rationale
- **Long enough** for a user to receive a notification, review the action, and approve on their phone.
- **Short enough** that a forgotten or unnoticed request expires quickly, limiting the window for stale approvals.
- **Consistent.** Same 5-minute window for signatures, approvals, and tokens simplifies reasoning about expiration.
- **Configurable.** Per-request `registration_ttl` allows overriding for specific flows.

### Consequences
- Users who are slow to respond will see expired requests and need to ask the agent to retry.
- Agents must handle the `approval_expired` error gracefully and re-request if needed.
- In poor network conditions, latency can eat into the 5-minute window.

---

## Summary

| # | Decision | Key driver |
|---|----------|-----------|
| 1 | Gmail as first integration | High frequency, rich approval UX, mature OAuth |
| 2 | Middle-man architecture | Works today, no adoption required from services |
| 3 | Actions as core primitive | Security boundary + human-readable approvals |
| 4 | Per-agent, per-action constraints | Defense in depth, reduce approval fatigue |
| 5 | Ed25519 signatures | No shared secrets, request integrity |
| 6 | Single-use JWT tokens | Minimal blast radius, parameter-bound |
| 7 | Confirmation codes | Out-of-band verification, phishing resistance |
| 8 | RFC 8785 (JCS) | Cross-language interoperability |
| 9 | Push notifications | Low latency, rich rendering |
| 10 | 5-minute TTL | Balance usability vs. security window |
