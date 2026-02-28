# Permission Slip Protocol - Terminology

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
- Registers with services by providing a public key
- Makes approval requests for one-off actions
- Uses single-use tokens to execute approved actions

### Service
A backend system that implements the Permission Slip server protocol. Services manage agent registrations, approval requests, and token issuance.

**Examples:**
- Gmail (email service)
- Expedia (travel booking service)
- Stripe (payment service)

**Key characteristics:**
- Provides Permission Slip protocol endpoints
- Verifies agent signatures
- Issues single-use tokens after approval
- Executes approved actions when presented with valid tokens

### Approver
A human with rights to approve agent actions on specific resources. An approver does not need to own the resource, but must have sufficient permissions to perform the requested action.

**Key characteristics:**
- Reviews approval requests in real-time
- Approves or denies specific one-off actions
- Must be on the agent owner's "approvers list"
- One approver per approval request

**Note:** Multiple approvers may be authorized to approve requests for the same agent, but only one approver is needed per request.

### Agent Owner
The person who registered the agent with a service. The agent owner has administrative control over the agent's lifecycle.

**Responsibilities:**
- Approve agent registration with services (agent cannot register without owner approval)
- Manage agent deregistration
- Maintain the "approvers list" (who can approve agent actions)
- Review agent activity logs
- Revoke agent access if compromised

---

## Core Concepts

### Registration
The one-time setup process where an agent proves its identity to a service by providing its public key. An agent must register separately with each service it intends to use.

**Key characteristics:**
- Agent initiates registration by providing public key
- Agent owner must approve the registration (agent cannot self-register)
- Establishes agent identity with a service
- Provides public key for signature verification
- Links agent to an agent owner
- Required before making approval requests

### Approval Request
An asynchronous request for permission to perform a specific one-off action. The request includes detailed information about the action, parameters, and context to help the approver make an informed decision.

**Key characteristics:**
- Describes a specific action with concrete parameters
- Includes risk level and context
- Has a configurable TTL (default: 5 minutes)
- Does not block the agent while waiting for approval
- Can expire if not approved within the TTL

**Example:** "Send 100 emails to marketing-list@example.com with subject 'Product Launch'"

### Token
A single-use credential issued to an agent after an approval request is approved. The token authorizes the agent to execute the specific approved action exactly once.

**Key characteristics:**
- Single-use: consumed after one API call
- Time-bound: expires at the approval request's TTL if not used
- Action-specific: tied to the exact parameters in the approval request
- Scope cannot exceed what was approved in the original request
- Not reusable for similar or related actions

**Important:** Permission Slip tokens are NOT a replacement for OAuth. For long-lived, recurring access, services should implement OAuth in addition to Permission Slip.

### Time-to-Live (TTL)
The maximum duration an approval request remains valid before expiring. The TTL also determines token expiration if the token is not used.

**Default:** 5 minutes  
**Customization:** Services may allow custom TTLs per approval request

The TTL balances two concerns:
- Short enough to minimize risk if an approval is granted but not immediately used
- Long enough to accommodate real-world approval delays (user checking their phone, reviewing details)

---

## Protocol Model

Permission Slip follows a **"break glass" approval model** for one-off agent actions:

1. **Agent makes request:** "I need to do X right now"
2. **Service generates approval:** Creates approval request with specific parameters and TTL
3. **Approver reviews:** Sees exactly what will happen, approves or denies
4. **Service issues token:** Single-use token tied to the approved action
5. **Agent executes:** Uses token once to trigger the action
6. **Token consumed:** Cannot be reused, even for identical future requests

**This is NOT:**
- A replacement for OAuth (for recurring access)
- A way to grant broad, long-lived permissions
- A token the agent can cache and reuse

---

## Security Concepts

### Public/Private Key Pair
Agents use asymmetric cryptography to prove their identity and sign requests.

**Recommendations:**
- Use the same key pair across all services (simpler identity management)
- Agents may use separate keys per service if they prefer isolation
- The protocol does not enforce either approach

**Supported algorithms:**
- Ed25519 (preferred)
- ECDSA P-256

### Signature Verification
Services verify that approval requests are signed by the agent's private key (matching the registered public key). This prevents impersonation and ensures request integrity.

---

## Multi-Service Operations

When an agent needs approval from multiple services in a single user interaction, the client library may aggregate multiple independent approval requests into one UI screen for better user experience.

**Key characteristics:**
- Each service issues its own approval request independently
- No protocol-level "batch approval" concept
- UI aggregation is a client library feature, not part of the protocol
- Each approval results in a separate single-use token

**Example:** An agent booking a trip might request:
- Expedia approval: "Book flight LAX→NYC on 3/15"
- Gmail approval: "Send confirmation email to user@example.com"

The client library shows both on one screen, but they are two separate protocol transactions with two separate tokens.

---

## Glossary

| Term | Definition |
|------|------------|
| **Agent** | Automated system requesting approval |
| **Service** | Backend implementing Permission Slip server protocol |
| **Approver** | Human authorized to approve agent actions |
| **Agent Owner** | Person who registered the agent, has admin control |
| **Registration** | One-time setup providing public key to a service |
| **Approval Request** | Async request for one-off action approval |
| **Token** | Single-use credential issued after approval |
| **TTL** | Time-to-live for approval requests and tokens (default: 5 minutes) |
| **Break Glass** | Model where approval is for immediate, one-off actions only |
| **Public Key** | Agent's identity credential, shared with services during registration |
| **Private Key** | Agent's secret key, used to sign approval requests |

---

## Next Steps

See other specification documents (planned or in progress) for details on:
- **Authentication** - Key generation, registration, and signature verification
- **API** - Complete endpoint documentation and request/response formats
- **Errors** - Error handling and status codes
