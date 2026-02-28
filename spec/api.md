# API Specification

This document defines the complete HTTP API for the Permission Slip protocol.

---

## Table of Contents

1. [Glossary](#glossary)
2. [Base URLs & Discovery](#base-urls--discovery)
3. [Action Types](#action-types)
4. [Authentication](#authentication)
5. [Registration Endpoints](#registration-endpoints)
6. [Approval Endpoints](#approval-endpoints)
7. [Token Usage](#token-usage)
8. [Error Handling](#error-handling)
9. [Complete Examples](#complete-examples)
10. [Protocol Version](#protocol-version)
11. [Related Documentation](#related-documentation)

---

## Glossary

For detailed definitions of core terms (Agent, Approver, Service, Action, Token, etc.), see **[Terminology](terminology.md)**.

---

## Base URLs & Discovery

### Service Discovery

Services implementing Permission Slip MUST publish a discovery endpoint at:

```
https://<service-domain>/.well-known/permission-slip
```

**Response:**

```json
{
  "base_url": "https://api.example.com/permission-slip",
  "version": "1.0",
  "supported_algorithms": ["Ed25519", "ECDSA-P256"],
  "supported_token_algorithms": ["ES256"],
  "jwks_uri": "https://api.example.com/.well-known/permission-slip-jwks.json"
}
```

**Fields:**

- `base_url` (string, required): Base URL for all Permission Slip protocol endpoints. **Does NOT include the version suffix** (e.g., `/v1`).
- `version` (string, required): Protocol version implemented
- `supported_algorithms` (array, required): Agent signature algorithms supported for request signing
- `supported_token_algorithms` (array, required): JWT signature algorithms used for approval tokens
- `jwks_uri` (string, required): Absolute URL to the service's JSON Web Key Set (JWKS) for token verification. If a `.well-known` location is used, it MUST be at the origin root per RFC 8615 (e.g., `https://<service-domain>/.well-known/permission-slip-jwks.json`); otherwise, any absolute URL MAY be used.

**Important:** The `base_url` applies ONLY to Permission Slip protocol endpoints (registration, approval requests, etc.). Service-specific action endpoints (e.g., `/v1/emails/send`) are defined by the service's own API and are NOT under the Permission Slip `base_url`.

**Versioning Strategy:**

The `base_url` does NOT include the version suffix. Agents construct full endpoint URLs by appending the versioned path to the `base_url`.

**Example:**

- Discovery response: `"base_url": "https://api.example.com/permission-slip"`
- Agent constructs registration endpoint: `{base_url}/v1/agents/register`
- Full URL: `https://api.example.com/permission-slip/v1/agents/register`

**Explicit Construction Rule:**

```
Full endpoint URL = {base_url} + "/v1/" + {endpoint_path}
```

Where:
- `{base_url}` = Value from discovery response (no trailing slash, no version)
- `{endpoint_path}` = Specific endpoint path (e.g., `agents/register`, `approvals/request`)

**Examples:**

| Discovery `base_url` | Endpoint Path | Full URL |
|---------------------|---------------|----------|
| `https://api.example.com/permission-slip` | `agents/register` | `https://api.example.com/permission-slip/v1/agents/register` |
| `https://service.com/ps` | `approvals/request` | `https://service.com/ps/v1/approvals/request` |
| `https://example.com` | `agents/agent_abc123/verify` | `https://example.com/v1/agents/agent_abc123/verify` |

**Rationale:**

This design allows services to support multiple protocol versions simultaneously (e.g., `/v1/...` and `/v2/...` under the same `base_url`) without requiring a new discovery endpoint for each version.

All subsequent Permission Slip protocol endpoints in this specification are documented relative to `{base_url}/v1`.

---

## Action Types

Action types identify what operation an agent is requesting approval for. This specification defines the structure and naming conventions for action types, but does not mandate specific type definitions (see [Action Type Registry](https://github.com/supersuit-ai/permission-slip/issues/4) for standard type proposals).

### Action Type Structure

Every action in an approval request MUST include:

```json
{
  "action": {
    "type": "email.send",
    "version": "1",
    "parameters": {
      "from": "user@example.com",
      "to": ["recipient@example.com"]
    }
  }
}
```

**Fields:**

- `type` (string, required): Action type identifier
- `version` (string, optional): Action type version (default: "1" if omitted)
- `parameters` (object, required): Action-specific parameters (service-defined)

### Naming Conventions

Action types follow one of two naming patterns:

#### **Standard Types (Reserved for Future Standardization)**

Types with **exactly two segments** (one dot separator) are reserved for future protocol standardization.

**Format:** `<category>.<operation>`

**Examples:**
- `email.send`
- `payment.charge`
- `data.delete`
- `calendar.create`

**Constraints:**
- MUST match regex: `^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`
- Exactly **two segments** separated by a single dot
- Total length MUST be ≤ 64 characters
- Only lowercase letters, digits, and underscores (plus the single dot separator)
- Services MAY use standard types without formal definition, but should be aware these may be standardized in future protocol versions

**Recognition:** If an action type matches the regex above (exactly two segments, no reverse-DNS domain), it is a **standard type**.

#### **Custom Types (Service-Defined)**

Types with **four or more segments** using reverse-DNS notation are service-specific.

**Format:** `<reverse-domain>.<category>.<operation>` (where `<reverse-domain>` consists of 2 or more dot-separated segments, e.g., `com.example`, so the total segments are ≥ 4)

**Examples:**
- `com.example.deploy.production` (4 segments)
- `io.github.repo.delete` (4 segments)
- `com.acme.internal.admin_action` (4 segments)

**Constraints:**
- MUST have **four or more segments** (minimum: 2-segment domain + category + operation = 4 total)
- MUST start with a valid reverse-DNS domain (e.g., `com.example`, `io.github`)
- Domain portion (first 2+ segments) MUST match regex: `^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`
- After domain: category and operation segments (lowercase letters, digits, underscores only)
- Total length MUST be ≤ 128 characters
- Services have full control over custom type semantics

**Recognition:** If an action type has four or more segments and starts with a reverse-DNS domain, it is a **custom type**.

### Versioning

The `version` field enables breaking changes to action type semantics.

**Version Format:**
- MUST be a positive integer as a string: `"1"`, `"2"`, `"3"`, etc.
- Default: `"1"` if omitted

**When to increment:**
- Breaking change to parameter structure or semantics
- Non-breaking additions (new optional parameters) do NOT require version increment

**Example Evolution:**

```json
// Version 1 (original)
{
  "type": "email.send",
  "version": "1",
  "parameters": {
    "to": ["user@example.com"],
    "subject": "Hello"
  }
}

// Version 2 (breaking change: 'to' is now an object)
{
  "type": "email.send",
  "version": "2",
  "parameters": {
    "recipients": {
      "to": ["user@example.com"],
      "cc": [],
      "bcc": []
    },
    "subject": "Hello"
  }
}
```

Services MUST explicitly support each version they accept and reject unsupported versions with error code `invalid_request`.

### Validation

Services MUST validate action types in approval requests:

1. **Format:** Verify type matches standard or custom naming pattern
2. **Version:** Check version is supported (if service version-aware)
3. **Parameters:** Validate parameters match expected schema for the type/version

**Error Responses:**

- Invalid type format → `400 Bad Request`, error code `invalid_action_type`
- Unsupported type → `400 Bad Request`, error code `unsupported_action_type`
- Unsupported version → `400 Bad Request`, error code `unsupported_action_version`

---

## Authentication

All Permission Slip protocol API requests under `base_url` (registration and approval endpoints, excluding discovery) MUST include a cryptographic signature proving agent identity. This requirement does not apply to service-defined action endpoints invoked using tokens as described in [Token Usage](#token-usage).

### Request Signature Header

```
X-Permission-Slip-Signature: agent_id="<agent_id>", algorithm="<algorithm>", timestamp="<unix_timestamp>", signature="<base64url_signature>"
```

**Fields:**

- `agent_id`: The agent's identifier (derived from public key)
- `algorithm`: Signature algorithm (`Ed25519` or `ECDSA-P256`)
- `timestamp`: Unix timestamp (seconds since epoch)
- `signature`: Base64url-encoded signature (no padding). The bytes prior to base64url encoding MUST be:
  - for `Ed25519`: the 64-byte raw Ed25519 signature as defined in RFC 8032.
  - for `ECDSA-P256`: a 64-byte IEEE P1363 fixed-length encoding `r || s`, where `r` and `s` are each 32-byte, big-endian integers representing the ECDSA values, normalized to a "low-S" form (s ≤ n/2) as in SEC 1.

### Canonical Request Format

The signature is computed over a canonical representation of the request:

```
<HTTP_METHOD>\n
<PATH>\n
<QUERY_STRING>\n
<TIMESTAMP>\n
<BODY_HASH>
```

**Canonicalization Rules:**

Services and agents MUST follow these exact rules to ensure signature compatibility:

1. **HTTP_METHOD**: Uppercase HTTP verb (e.g., `POST`, `GET`)

2. **PATH**: URL path component without query string (e.g., `/v1/agents/register`)

3. **QUERY_STRING**: Canonical query string representation
   - If no query parameters: empty string `""`
   - If query parameters exist:
     - Percent-encode parameter names and values per **RFC 3986** (unreserved characters MUST NOT be encoded)
     - Unreserved characters: `A-Z a-z 0-9 - . _ ~`
     - Spaces MUST be encoded as `%20` (NOT `+`)
     - Use uppercase hex digits in percent-encoding (e.g., `%2F` not `%2f`)
     - Sort parameters by name (lexicographic/byte order)
     - For repeated parameter names, sort values within that name
     - Join as `name1=value1&name2=value2`
   - Example: `?z=2&a=1&a=3` → `"a=1&a=3&z=2"`
   - Example: `?name=hello world` → `"name=hello%20world"` (NOT `"name=hello+world"`)

4. **TIMESTAMP**: Unix timestamp in seconds (as appears in signature header)

5. **BODY_HASH**: SHA-256 hash of the request body
   - For JSON requests, canonicalize using **RFC 8785 JSON Canonicalization Scheme (JCS)**:
     - Services and agents MUST use an RFC 8785-compliant canonicalization library
     - JCS ensures deterministic serialization across languages by specifying exact rules for:
       - Key sorting (lexicographic, Unicode code point order)
       - Number representation (no leading zeros, exponential notation rules)
       - String escaping (minimal escaping, Unicode normalization)
       - No NaN, Infinity, or -Infinity values (not supported in JCS)
     - **Implementation:**
       - Use a language-specific RFC 8785 library (see examples below)
       - Do NOT use ad-hoc approaches like `json.dumps(sort_keys=True)`
       - Compute SHA-256 hash over the UTF-8 bytes of the JCS-canonicalized JSON
       - Encode the hash as lowercase hexadecimal (64 characters)
     - **Library recommendations:**
       - **Python:** `pip install jcs` → `import jcs; canonical = jcs.canonicalize(obj)`
       - **JavaScript/Node.js:** `npm install canonicalize` → `const canonicalize = require('canonicalize'); const canonical = canonicalize(obj);`
       - **Go:** `go get github.com/cyberphone/json-canonicalization/go/json` → use `Serialize()` function
       - **Java:** Maven/Gradle: `org.glassfish:jakarta.json` or `cyberphone/json-canonicalization`
     - **Why JCS is required:** Simple key-sorting approaches fail on edge cases (Unicode, number formats like `1.0` vs `1`, HTML escaping) that cause signature verification failures between implementations
   - For empty body (e.g., GET requests):
     - Hash the empty string: `sha256("") = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`

**Example Construction:**

```javascript
const crypto = require('crypto');
const canonicalize = require('canonicalize');

const method = "POST";
const path = "/v1/agents/register";

// RFC 3986-compliant percent-encoder (avoids locale issues and encodes !'()*)
function rfc3986Encode(str) {
  return encodeURIComponent(str).replace(/[!'()*]/g, c =>
    '%' + c.charCodeAt(0).toString(16).toUpperCase()
  );
}

// Parse query string without '+' → space conversion (no x-www-form-urlencoded semantics)
function parseQueryStringRaw(qs) {
  if (!qs) return [];
  return qs.split('&').filter(Boolean).map(part => {
    const idx = part.indexOf('=');
    const rawKey = idx === -1 ? part : part.slice(0, idx);
    const rawVal = idx === -1 ? '' : part.slice(idx + 1);
    const key = decodeURIComponent(rawKey);
    const value = decodeURIComponent(rawVal);
    return [key, value];
  });
}

// Canonicalize query string (if present)
const params = parseQueryStringRaw(originalQueryString);
const sortedParams = params.slice().sort((a, b) => {
  if (a[0] < b[0]) return -1;
  if (a[0] > b[0]) return 1;
  if (a[1] < b[1]) return -1;
  if (a[1] > b[1]) return 1;
  return 0;
});
const query = sortedParams.map(([k, v]) =>
  `${rfc3986Encode(k)}=${rfc3986Encode(v)}`
).join('&');

const timestamp = "1707667200";

// Canonicalize body using RFC 8785 (JCS)
const canonicalBody = canonicalize(requestBody);
const bodyHash = crypto.createHash('sha256')
  .update(canonicalBody, 'utf8')
  .digest('hex');

const canonicalRequest = `${method}\n${path}\n${query}\n${timestamp}\n${bodyHash}`;

// Sign canonicalRequest with agent's private key
// agentPrivateKey must be PEM-encoded PKCS8 format (see authentication.md "Key Pair Generation")
// For Ed25519, this looks like:
//   -----BEGIN PRIVATE KEY-----
//   MC4CAQAwBQYDK2VwBCIEI...
//   -----END PRIVATE KEY-----
const signature = crypto.sign(null, Buffer.from(canonicalRequest, 'utf-8'), {
  key: agentPrivateKey,  // String containing PEM-encoded private key
  format: 'pem',
  type: 'pkcs8',
});
```

### Signature Verification

Services MUST:

1. Verify `timestamp` is within 300 seconds (5 minutes) of current time to limit the replay window
2. Reconstruct the canonical request
3. Verify the signature:
   - For requests from already-registered agents, use the agent's registered public key.
   - For `POST /v1/agents/register`, use the `public_key` provided in the request body and ensure that the claimed `agent_id` is derived from that key.
4. Reject requests with invalid or expired signatures

**Replay Protection:**

The timestamp requirement limits the replay window to 300 seconds but does not prevent a captured signed request from being replayed multiple times within that window.

Services MUST implement replay protection by:

1. **Request ID Tracking (All Signed POST Endpoints):**
   - All signed POST Permission Slip protocol endpoints MUST require a `request_id` field in the request body (UUID v4 recommended)
   - Services MUST track `request_id` values and reject duplicate requests with `409 Conflict`, error code `duplicate_request_id`
   - The `request_id` serves as an idempotency key for the entire request
   - Storage requirements:
     - Store `request_id` values for at least 5 minutes (300 seconds) after the request timestamp
     - Use atomic check-and-set operations to prevent race conditions
     - Implementation options: database unique constraint, Redis SET NX, distributed lock
   - Expiration: Purge stored `request_id` values after `request_timestamp + 300 seconds` to prevent unbounded storage growth

2. **Signature Hash Tracking (Optional Additional Layer):**
   - Services MAY additionally track signature hashes to detect exact replay attempts
   - Store a hash of the `X-Permission-Slip-Signature` header value
   - Reject duplicate signature hashes within the timestamp window
   - This provides defense-in-depth beyond request IDs

**Implementation Notes:**

- `request_id` tracking prevents both malicious replays and accidental duplicate submissions
- For idempotent operations (e.g., registration), duplicate `request_id` should return the original response if the operation already succeeded, or reject if it's still in progress
- Services SHOULD use distributed storage (Redis, database) for `request_id` tracking in multi-instance deployments

**Error responses:**

- Invalid signature → `401 Unauthorized` with error code `invalid_signature`
- Expired timestamp → `401 Unauthorized` with error code `timestamp_expired`
- Duplicate request ID → `409 Conflict` with error code `duplicate_request_id`

---

## Registration Endpoints

### POST /v1/agents/register

Register a new agent with the service.

**Request:**

```http
POST /v1/agents/register
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "6f1a7c30-9b2e-4d91-8c3f-2a4b6c7d8e9f",
  "agent_id": "agent_x7K9mP4n...",
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFPZj8V7K2mT9Xw4nL5rQ1pY3vN8cM6hU0oB4eW2sR7k",
  "approver": "alice",
  "metadata": {
    "name": "My AI Assistant",
    "version": "1.0.0",
    "capabilities": ["email", "calendar"]
  },
  "registration_ttl": 300
}
```

**Fields:**

- `request_id` (string, required): Unique request identifier for replay protection (UUID v4 recommended)
- `agent_id` (string, required): Agent identifier (derived from public key). MUST match the `agent_id` in the `X-Permission-Slip-Signature` header.
- `public_key` (string, required): Agent's public key in OpenSSH format
- `approver` (string, required): Service username who will approve agent actions
- `metadata` (object, optional): Agent metadata
  - `name` (string, optional): Human-readable agent name
  - `version` (string, optional): Agent version
  - `capabilities` (array, optional): Agent capabilities
- `registration_ttl` (integer, optional): Registration URL TTL in seconds (default: 300)

**Validation:**

Services MUST verify that the `agent_id` in the request body matches the `agent_id` in the `X-Permission-Slip-Signature` header. If they do not match, return `400 Bad Request` with error code `agent_id_mismatch`.

**Response (200 OK):**

```json
{
  "registration_url": "https://example.com/permission-slip/approve?token=eyJ...",
  "alternative_urls": {
    "deeplink": "example://permission-slip/approve?token=eyJ...",
    "web": "https://accounts.example.com/permission-slip/approve?token=eyJ..."
  },
  "approver": "alice",
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Fields:**

- `registration_url` (string, required): Primary registration URL (universal link). This is the canonical URL that clients MUST support.
- `alternative_urls` (object, optional): Optional alternative URL formats for specific platforms. If omitted, clients MUST use `registration_url`.
  - `deeplink` (string, optional): Native app deep link
  - `web` (string, optional): Web-only URL
- `approver` (string, required): Username who must approve
- `expires_at` (string, required): ISO 8601 timestamp when registration expires
- `verification_required` (boolean, required): Always `true` (confirmation code required)

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Malformed request body
- `400 Bad Request` - `invalid_public_key`: Public key format invalid
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `404 Not Found` - `approver_not_found`: Approver username not found
- `409 Conflict` - `agent_already_registered`: Agent already registered with this approver

---

### POST /v1/agents/{agent_id}/verify

Complete agent registration by submitting the confirmation code shown to the approver.

**Path Parameters:**

- `agent_id` (string, required): Must match the `agent_id` in the signature header

**Request:**

```http
POST /v1/agents/agent_x7K9mP4n.../verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "req_verify_g7h8i9j0k1l2",
  "confirmation_code": "XK7-M9P"
}
```

**Validation:**

Services MUST verify that the `agent_id` in the URL path matches the `agent_id` in the `X-Permission-Slip-Signature` header. If they do not match, return `400 Bad Request` with error code `agent_id_mismatch`.

**Fields:**

- `request_id` (string, required): Unique request identifier for replay protection (UUID v4 recommended)
- `confirmation_code` (string, required): 6-character confirmation code. See [Confirmation Code Format](#confirmation-code-format) for detailed specification.

**Response (200 OK):**

```json
{
  "status": "approved",
  "registered_at": "2026-02-11T13:20:15Z"
}
```

**Fields:**

- `status` (string, required): Registration status (`approved`)
- `registered_at` (string, required): ISO 8601 timestamp of registration completion

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Missing or malformed confirmation code
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `401 Unauthorized` - `invalid_code`: Confirmation code incorrect or expired
- `404 Not Found` - `agent_not_found`: Agent registration not found or expired
- `410 Gone` - `registration_expired`: Registration TTL elapsed

---

## Approval Endpoints

### POST /v1/approvals/request

Request approval for a specific action.

**Request:**

```http
POST /v1/approvals/request
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "agent_id": "agent_x7K9mP4n...",
  "request_id": "req_a1b2c3d4e5f6",
  "approver": "alice",
  "action": {
    "type": "email.send",
    "version": "1",
    "parameters": {
      "from": "alice@example.com",
      "to": ["recipient@example.com"],
      "subject": "Hello World",
      "body": "This is a test email."
    }
  },
  "context": {
    "description": "Send welcome email to new user",
    "risk_level": "low",
    "details": {
      "recipient_count": 1
    }
  }
}
```

**Fields:**

- `agent_id` (string, required): Agent identifier. MUST match the `agent_id` in the `X-Permission-Slip-Signature` header.
- `request_id` (string, required): Unique request identifier (UUID v4 recommended)
- `approver` (string, required): Service username who must approve this action
- `action` (object, required): Action to be approved
  - `type` (string, required): Action type identifier
  - `version` (string, optional): Action type version (default: "1")
  - `parameters` (object, required): Action-specific parameters
- `context` (object, required): Human-readable context for approval UI
  - `description` (string, required): Brief action summary
  - `risk_level` (string, optional): `low`, `medium`, or `high`
  - `details` (object, optional): Additional context for UI display

**Validation:**

Services MUST verify that the `agent_id` in the request body matches the `agent_id` in the `X-Permission-Slip-Signature` header. If they do not match, return `400 Bad Request` with error code `agent_id_mismatch`.

**Response (200 OK):**

```json
{
  "approval_id": "appr_xyz789",
  "approval_url": "https://example.com/permission-slip/approve/appr_xyz789",
  "alternative_urls": {
    "deeplink": "example://permission-slip/approve/appr_xyz789",
    "web": "https://accounts.example.com/permission-slip/approve/appr_xyz789"
  },
  "status": "pending",
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Fields:**

- `approval_id` (string, required): Unique approval identifier
- `approval_url` (string, required): Primary approval URL (universal link)
- `alternative_urls` (object, optional): Optional alternative URL formats for specific platforms
- `status` (string, required): Approval status (`pending`)
- `expires_at` (string, required): ISO 8601 timestamp when approval expires
- `verification_required` (boolean, required): Always `true` (confirmation code required)

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Malformed request
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `403 Forbidden` - `agent_not_authorized`: Agent not registered with this approver
- `404 Not Found` - `agent_not_found`: Agent not registered
- `404 Not Found` - `approver_not_found`: Approver username not found
- `409 Conflict` - `duplicate_request_id`: Request ID already used

---

### POST /v1/approvals/{approval_id}/verify

Verify approval and retrieve the single-use token.

**Request:**

```http
POST /v1/approvals/appr_xyz789/verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "req_verify_m3n4o5p6q7r8",
  "confirmation_code": "RK3-P7M"
}
```

**Validation:**

Services MUST verify that the `agent_id` in the `X-Permission-Slip-Signature` header matches the agent associated with the `approval_id`. If they do not match, return `403 Forbidden` with error code `agent_not_authorized`.

**Fields:**

- `request_id` (string, required): Unique request identifier for replay protection (UUID v4 recommended)
- `confirmation_code` (string, required): 6-character confirmation code. See [Confirmation Code Format](#confirmation-code-format) for detailed specification.

**Response (200 OK - Approved):**

```json
{
  "status": "approved",
  "approved_at": "2026-02-11T13:20:45Z",
  "token": {
    "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...<see Token Structure section for payload details>",
    "expires_at": "2026-02-11T13:25:45Z",
    "scope": "email.send",
    "scope_version": "1"
  }
}
```

**Fields:**

- `status` (string, required): Approval status (`approved`)
- `approved_at` (string, required): ISO 8601 timestamp of approval
- `token` (object, required): Single-use token
  - `access_token` (string, required): JWT token
  - `expires_at` (string, required): ISO 8601 token expiration
  - `scope` (string, required): Action type
  - `scope_version` (string, required): Action version

**Response (403 Forbidden - Denied):**

When the approver explicitly denies the approval request, the service MUST return:

```json
{
  "error": {
    "code": "approval_denied",
    "message": "User denied the approval request",
    "retryable": false,
    "details": {
      "denied_at": "2026-02-11T13:20:30Z"
    },
    "trace_id": "trace_xyz789"
  }
}
```

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Missing or malformed confirmation code
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `401 Unauthorized` - `invalid_code`: Confirmation code incorrect
- `403 Forbidden` - `agent_not_authorized`: Agent is not authorized for this approval
- `403 Forbidden` - `approval_denied`: User denied the approval request
- `404 Not Found` - `approval_not_found`: Approval ID not found
- `410 Gone` - `approval_expired`: Approval TTL elapsed

---

### POST /v1/approvals/{approval_id}/cancel

Cancel a pending approval request.

**Request:**

```http
POST /v1/approvals/appr_xyz789/cancel
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "req_cancel_s9t0u1v2w3x4"
}
```

**Validation:**

Services MUST verify that the `agent_id` in the `X-Permission-Slip-Signature` header matches the agent associated with the `approval_id`. If they do not match, return `403 Forbidden` with error code `agent_not_authorized`.

**Fields:**

- `request_id` (string, required): Unique request identifier for replay protection (UUID v4 recommended)

**Response (200 OK):**

```json
{
  "status": "cancelled",
  "cancelled_at": "2026-02-11T13:21:00Z"
}
```

**Error Responses:**

- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `403 Forbidden` - `agent_not_authorized`: Agent not authorized for this approval
- `404 Not Found` - `approval_not_found`: Approval ID not found
- `409 Conflict` - `approval_already_resolved`: Approval already approved/denied

---

## Token Usage

After receiving an approval token, the agent uses it to perform the approved action.

### Token Structure (JWT)

**Header:**

```json
{
  "alg": "ES256",
  "typ": "JWT"
}
```

**Payload:**

```json
{
  "sub": "agent_x7K9mP4n...",
  "aud": "example.com",
  "approver": "alice",
  "approval_id": "appr_xyz789",
  "scope": "email.send",
  "scope_version": "1",
  "params_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "iat": 1707667245,
  "exp": 1707667545,
  "jti": "tok_unique123"
}
```

**Claims:**

- `sub` (string, required): Agent ID
- `aud` (string, required): Service domain
- `approver` (string, required): Username who approved
- `approval_id` (string, required): Approval request ID
- `scope` (string, required): Action type
- `scope_version` (string, required): Action version
- `params_hash` (string, required): SHA-256 hash of the UTF-8 encoded bytes of the **parameters JSON object** canonicalized using RFC 8785 (JCS). The parameters JSON object MUST contain exactly the action-specific parameters from the approval request (no additional fields).
- `iat` (integer, required): Issued at (Unix timestamp)
- `exp` (integer, required): Expires at (Unix timestamp)
- `jti` (string, required): Unique token ID

### Using the Token

The agent presents the token in the `Authorization` header when calling **service-specific action endpoints**.

**Important:** Action endpoints (e.g., `/v1/emails/send`) are NOT part of the Permission Slip protocol. They are defined by the service's own API and may use a different base URL than the Permission Slip `base_url`.

**Example Request:**

```http
POST https://api.example.com/v1/emails/send
Authorization: Bearer eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "from": "alice@example.com",
  "to": ["recipient@example.com"],
  "subject": "Hello World",
  "body": "This is a test email."
}
```

**Note:** The action endpoint (`https://api.example.com/v1/emails/send`) is separate from the Permission Slip base URL (`https://api.example.com/permission-slip/v1`). Services define where action endpoints live in their own API documentation.

### Token Signature Verification (JWKS)

Services MUST publish their token signing public keys at the `jwks_uri` specified in the discovery response.

**JWKS Endpoint:**

```http
GET https://api.example.com/.well-known/permission-slip-jwks.json
```

**Note:** Per RFC 8615, if you use a `.well-known` URL, it MUST be at the origin root (`https://<domain>/.well-known/...`), not nested under a path. Otherwise, services MAY use any URL for the JWKS endpoint and MUST specify the full URL in the `jwks_uri` discovery field.

**Response:**

```json
{
  "keys": [
    {
      "kty": "EC",
      "use": "sig",
      "crv": "P-256",
      "kid": "key-2024-01",
      "x": "base64url...",
      "y": "base64url...",
      "alg": "ES256"
    }
  ]
}
```

Agents MAY fetch the JWKS if they choose to locally validate approval token signatures; however, the service-side action endpoint/resource server is responsible for verifying approval tokens. Services MAY rotate keys by adding new keys to the JWKS and including a `kid` (key ID) header parameter in issued tokens.

### Service Verification Requirements

Services MUST:

1. **Verify JWT signature** using service's public key
2. **Check expiration** (`exp` claim)
3. **Verify audience** (`aud` matches service domain)
4. **Verify scope** matches the endpoint being called
5. **Verify params_hash** matches the actual request parameters
6. **Check single-use** by tracking `jti` (MUST be atomic):
   - Perform an **atomic check-and-set operation** to mark the `jti` as consumed
   - The check (if used) and set (mark consumed) MUST be a single atomic operation to prevent race conditions
   - Implementation options:
     - Database: Use `INSERT ... ON CONFLICT` or equivalent upsert with unique constraint
     - Redis: Use `SET NX` (set if not exists) command
     - In-memory: Use atomic compare-and-swap (CAS) or mutex-protected check-then-set
   - If `jti` already exists (token already used): reject with `403 Forbidden` and error code `token_already_used`
   - If `jti` successfully inserted: proceed with action
   - Cache consumed `jti` values until token `exp` time, then purge
   - Under concurrent requests with the same `jti`, exactly one MUST succeed; all others MUST fail with `token_already_used`

**Parameter Hash Verification:**

> **⚠️ CRITICAL: RFC 8785 Canonicalization Required**
>
> **DO NOT use `json.dumps(sort_keys=True)` or similar ad-hoc approaches!**
>
> Simple key-sorting approaches fail on edge cases and will cause signature verification failures between implementations:
> - **Unicode:** Different escaping rules for non-ASCII characters (e.g., `\u0020` vs ` `)
> - **Numbers:** Inconsistent representation (`1.0` vs `1`, exponential notation)
> - **Whitespace:** Varying space/newline insertion
> - **Key ordering:** Unicode code point order vs. string sort order
>
> **You MUST use an RFC 8785 (JCS) compliant library:**
> - **JavaScript (Frontend):** `npm install canonicalize` → `const canonicalize = require('canonicalize'); const canonical = canonicalize(obj);`
> - **Go (Backend):** `go get github.com/cyberphone/json-canonicalization/go/json`
>
> Failure to use RFC 8785 will result in non-interoperable implementations.

```go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	jsoncanon "github.com/cyberphone/json-canonicalization/go/json"
)

// verifyParameters checks that request body parameters match the params_hash in the JWT.
// This prevents parameter tampering attacks where an attacker modifies the request body
// after approval but reuses the original JWT token.
//
// Note on type safety: This example uses map[string]interface{} for tokenClaims to keep
// the code generic. In production, consider defining a struct for JWT claims:
//   type ApprovalClaims struct {
//     Sub         string `json:"sub"`
//     ParamsHash  string `json:"params_hash"`
//     // ... other fields
//   }
// and unmarshal the JWT payload into that struct for compile-time type safety.
func verifyParameters(w http.ResponseWriter, r *http.Request, tokenClaims map[string]interface{}) bool {
	// Helper to send a spec-conformant JSON error response.
	writeJSONError := func(code, message string, retryable bool, status int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// Construct the JSON error object explicitly to ensure proper formatting.
		resp := map[string]map[string]interface{}{
			"error": {
				"code":      code,
				"message":   message,
				"retryable": retryable,
			},
		}
		if data, err := json.Marshal(resp); err == nil {
			_, _ = w.Write(data)
		} else {
			// Fallback: minimal JSON if marshaling somehow fails.
			_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"Failed to encode error response.","retryable":false}}`))
		}
	}

	// Read and preserve the request body so other handlers can still access it.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError("invalid_parameters", "Failed to read request body.", false, http.StatusForbidden)
		return false
	}
	// Restore the body for downstream handlers.
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Validate that the request body is valid JSON without altering it
	if !json.Valid(bodyBytes) {
		writeJSONError("invalid_parameters", "Request body is not valid JSON.", false, http.StatusForbidden)
		return false
	}

	// Canonicalize using RFC 8785 (JCS) directly from the original request bytes
	canonicalParams, err := jsoncanon.Transform(bodyBytes)
	if err != nil {
		writeJSONError("invalid_parameters", "Failed to canonicalize request parameters.", false, http.StatusForbidden)
		return false
	}

	// Compute hash
	hash := sha256.Sum256(canonicalParams)
	computedHash := hex.EncodeToString(hash[:])
	
	// NOTE: To verify your RFC 8785 implementation, see the test vector below.
	// The test input should produce hash: 818f3ff05a25a71bb9bbd2bca14cd7653f60748267581099a6a09cd6a83c9f1f

	// Safely extract params_hash claim from token
	rawHash, ok := tokenClaims["params_hash"]
	tokenHash, okType := rawHash.(string)
	if !ok || !okType || tokenHash == "" {
		writeJSONError("invalid_parameters", "params_hash claim is missing or not a string.", false, http.StatusForbidden)
		return false
	}

	// Compare with token claim
	if computedHash != tokenHash {
		// Reject: parameters were tampered with
		writeJSONError("invalid_parameters", "Request parameters do not match the params_hash claim.", false, http.StatusForbidden)
		return false
	}
	return true
}
```

**Test Vector:**

To verify your RFC 8785 implementation is correct, use this test:

```json
{
  "from": "alice@example.com",
  "to": ["bob@example.com"],
  "subject": "Test with unicode: café",
  "amount": 1.0
}
```

**Expected canonical form (UTF-8 bytes):**
```json
{"amount":1,"from":"alice@example.com","subject":"Test with unicode: café","to":["bob@example.com"]}
```

**Expected SHA-256 hash (lowercase hex):**
```
818f3ff05a25a71bb9bbd2bca14cd7653f60748267581099a6a09cd6a83c9f1f
```

This is a real test vector; implementations MUST compute this exact hash for the canonical form shown above.

**Error Responses:**

- `401 Unauthorized` - `invalid_token`: Token signature invalid or expired
- `403 Forbidden` - `token_already_used`: Token has already been consumed
- `403 Forbidden` - `insufficient_scope`: Token scope doesn't match action
- `403 Forbidden` - `invalid_parameters`: Parameters don't match `params_hash`

### Token Lifecycle

Approval tokens are short-lived and single-use. Understanding their lifecycle helps agents handle edge cases gracefully.

**Token Properties:**

- **Expiration:** Tokens typically expire 5 minutes after issuance (service-defined via `exp` claim)
- **Single-use:** Each token can only be used once, enforced via the `jti` claim
- **No refresh tokens:** The Permission Slip protocol does not support refresh tokens by design

**What Happens When a Token Expires:**

If an agent retrieves a token but doesn't use it before expiration:

1. The token becomes invalid (service rejects it with `401 Unauthorized`, error code `invalid_token`)
2. The agent MUST request a new approval from the user
3. There is no mechanism to refresh or extend an expired token

**What Happens When a Token is Already Used:**

If an agent attempts to use the same token twice:

1. The first request succeeds (action is performed)
2. Subsequent requests with the same token are rejected with `403 Forbidden`, error code `token_already_used`
3. The agent MUST request a new approval if another action is needed

**Retry Guidance:**

**For expired tokens:**
- DO request a new approval from the user
- DO NOT retry the same token with exponential backoff (it will never succeed)

**For already-used tokens:**
- DO NOT retry with the same token
- DO request a new approval if the action needs to be performed again
- DO check if the first request actually succeeded before requesting new approval (avoid duplicate actions)

**For transient errors (network failures, 500 errors):**
- DO retry with exponential backoff while the token is still valid
- Track retry attempts to avoid exhausting token expiration time
- If retries exceed token TTL, request a new approval

**Best Practices:**

1. **Use tokens immediately** after retrieval to minimize expiration risk
2. **Implement idempotency** on the service side using the token's `jti` to prevent duplicate actions
3. **Handle `token_already_used` gracefully** by checking if the original action succeeded before re-requesting approval
4. **Set reasonable token TTLs** (services should balance security with usability; 5 minutes is recommended)

---

## Confirmation Code Format

Confirmation codes are displayed to approvers after they approve a request (registration or action approval). The agent must collect this code from the user and submit it to complete the verification flow.

### Canonical Format

**Services MUST generate confirmation codes in the following canonical format:**

- **Length:** Exactly 6 characters
- **Character set:** Uppercase alphanumeric, excluding confusing characters
  - **Allowed:** `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (32 characters)
  - **Excluded:** `0` (zero), `O` (letter O), `1` (one), `I` (letter I) — these are visually confusing
- **Display format:** `XXX-XXX` (two groups of 3, separated by hyphen) for readability
- **Storage format:** `XXXXXX` (6 characters, no hyphen)

**Example codes:**
- Display: `XK7-M9P`, `RK3-P7M`, `A23-BCD`
- Storage: `XK7M9P`, `RK3P7M`, `A23BCD`

### Client Submission

**Agents/clients MAY submit confirmation codes in either format:**

- With hyphen: `XXX-XXX` (e.g., `XK7-M9P`)
- Without hyphen: `XXXXXX` (e.g., `XK7M9P`)
- Mixed case: `xk7-m9p`, `XK7m9p` (services MUST normalize to uppercase)

**Services MUST normalize submitted codes before validation:**

1. **Remove hyphens:** Strip all `-` characters
2. **Convert to uppercase:** Normalize to uppercase letters
3. **Validate length:** Ensure exactly 6 characters remain
4. **Validate character set:** Ensure all characters are in the allowed set
5. **Compare:** Match against stored code

**Normalization Example (Go - Backend):**

```go
package main

import (
	"errors"
	"strings"
)

// normalizeConfirmationCode normalizes and validates a user-submitted confirmation code.
// Returns the normalized code (uppercase, no hyphens) or an error if invalid.
func normalizeConfirmationCode(submittedCode string) (string, error) {
	// Remove hyphens and convert to uppercase
	normalized := strings.ToUpper(strings.ReplaceAll(submittedCode, "-", ""))
	
	// Validate length
	if len(normalized) != 6 {
		return "", errors.New("confirmation code must be exactly 6 characters")
	}
	
	// Validate character set
	allowedChars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, c := range normalized {
		if !strings.ContainsRune(allowedChars, c) {
			return "", errors.New("confirmation code contains invalid characters")
		}
	}
	
	return normalized, nil
}

// Example usage in an HTTP handler (production pattern):
// Note: extractSubmittedCode helper extracts the confirmation_code field from the request body.
// Implementation example:
//   var body map[string]interface{}
//   json.NewDecoder(r.Body).Decode(&body)
//   return body["confirmation_code"].(string)
func handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	// Extract submitted code from request
	submitted := extractSubmittedCode(r) // e.g., "xk7-m9p"
	storedCode := "XK7M9P"                // Retrieved from database
	
	normalized, err := normalizeConfirmationCode(submitted)
	if err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Invalid confirmation code format"}}`, http.StatusBadRequest)
		return
	}
	
	if normalized != storedCode {
		http.Error(w, `{"error":{"code":"invalid_code","message":"Incorrect confirmation code"}}`, http.StatusUnauthorized)
		return
	}
	
	// Code matches - proceed with verification
	w.WriteHeader(http.StatusOK)
	// ... return success response
}
```

### Generation Requirements

**Services MUST generate confirmation codes using a cryptographically secure random number generator:**

**Go Example (Backend):**

```go
package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
)

const allowedChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateConfirmationCode creates a cryptographically secure random 6-character
// confirmation code using the allowed character set (base32-like, without 0/O/1/I/L).
func generateConfirmationCode() (string, error) {
	code := make([]byte, 6)
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(allowedChars))))
		if err != nil {
			return "", err
		}
		code[i] = allowedChars[n.Int64()]
	}
	return string(code), nil
}

// Example usage in an HTTP handler (production pattern):
func handleGenerateApproval(w http.ResponseWriter, r *http.Request) {
	// Generate code
	code, err := generateConfirmationCode() // e.g., "XK7M9P"
	if err != nil {
		// Production error handling - return HTTP error instead of crashing
		http.Error(w, `{"error":{"code":"internal_error","message":"Failed to generate confirmation code"}}`, http.StatusInternalServerError)
		// Log the error for monitoring/debugging
		fmt.Fprintf(os.Stderr, "generateConfirmationCode failed: %v\n", err)
		return
	}
	
	// Format for display to approver
	displayCode := code[:3] + "-" + code[3:] // "XK7-M9P"
	
	// Store code in database and return approval response
	// ... (implementation details omitted)
	
	// Example: include in response to approver's app
	fmt.Fprintf(w, `{"confirmation_code":"%s"}`, displayCode)
}
```

**JavaScript Example (Frontend):**

```javascript
const crypto = require('crypto');

const ALLOWED_CHARS = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';

function generateConfirmationCode() {
  let code = '';
  for (let i = 0; i < 6; i++) {
    const randomIndex = crypto.randomInt(0, ALLOWED_CHARS.length);
    code += ALLOWED_CHARS[randomIndex];
  }
  return code;
}

// Generate code
const code = generateConfirmationCode();  // e.g., "XK7M9P"

// Format for display
const displayCode = `${code.slice(0, 3)}-${code.slice(3)}`;  // "XK7-M9P"
```

### Brute-Force Protection

**Services MUST implement rate limiting on confirmation code verification to prevent brute-force attacks:**

**Requirements:**

1. **Attempt limit:** Maximum 5 incorrect attempts per approval/registration
2. **Lockout:** After 5 failed attempts, mark the approval/registration as expired/invalid
3. **No retry after lockout:** User must restart the approval flow (request new approval)
4. **Timing attacks:** Use constant-time comparison for code validation
5. **Logging:** Log failed attempts for security monitoring

**Implementation Example (Go - Backend):**

```go
package main

import (
	"crypto/subtle"
	"errors"
)

// Note: This example omits database and logging helper functions for brevity.
// In a complete implementation, you would need:
//   - getApproval(approvalID string) (*Approval, error)
//   - saveApproval(approval *Approval) error
//   - logSecurityEvent(eventType, approvalID string)

func verifyConfirmationCode(approvalID, submittedCode string) (*Approval, error) {
	// Retrieve approval record from database
	approval, err := getApproval(approvalID)
	if err != nil {
		return nil, err
	}
	
	// Check if already locked out
	if approval.FailedAttempts >= 5 {
		return nil, errors.New("too many failed attempts")
	}
	
	// Normalize submitted code
	normalized, err := normalizeConfirmationCode(submittedCode)
	if err != nil {
		return nil, err
	}
	
	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte(normalized), []byte(approval.ConfirmationCode)) != 1 {
		// Increment failed attempts
		approval.FailedAttempts++
		saveApproval(approval)
		
		// Lock out after 5 attempts
		if approval.FailedAttempts >= 5 {
			approval.Status = "expired"
			saveApproval(approval)
			logSecurityEvent("confirmation_code_lockout", approvalID)
		}
		
		return nil, errors.New("incorrect confirmation code")
	}
	
	// Code is correct
	approval.Status = "approved"
	saveApproval(approval)
	return approval, nil
}
```

**Error Responses:**

- Incorrect code (attempts < 5): `401 Unauthorized`, error code `invalid_code`
- Too many attempts (attempts ≥ 5): `410 Gone`, error code `approval_expired` (or `registration_expired`)

### Security Considerations

1. **Entropy:** 6 characters from 32-character set = 32^6 = ~1 billion combinations
   - With 5-attempt limit: ~1 in 200 million chance of brute-force success
   - Acceptable for short-lived codes whose TTL matches the approval/registration TTL (default 5 minutes / 300 seconds)

2. **Code lifetime:** Confirmation codes MUST expire no later than the approval/registration TTL (`registration_ttl`, default 300 seconds), and SHOULD use the same TTL value

3. **Single-use:** Codes MUST be invalidated after successful verification

4. **Display security:** Codes shown in approver's app only (not transmitted to agent until agent submits for verification)

5. **Phishing resistance:** Out-of-band verification (user sees code in different device/app than agent) provides phishing resistance

---

## Error Handling

### Error Response Format

All error responses follow this structure:

```json
{
  "error": {
    "code": "error_code_here",
    "message": "Human-readable error message",
    "retryable": false,
    "details": {
      "additional": "context"
    },
    "trace_id": "trace_xyz789"
  }
}
```

**Fields:**

- `code` (string, required): Machine-readable error code
- `message` (string, required): Human-readable description
- `retryable` (boolean, required): Whether client should retry
- `details` (object, optional): Additional error context
- `trace_id` (string, optional): Server-generated trace/correlation ID for debugging and support requests. This is distinct from the client-supplied `request_id` in approval requests.

For rate limiting errors, include:

```json
{
  "error": {
    "code": "rate_limited",
    "message": "Too many requests",
    "retryable": true,
    "retry_after": 60,
    "trace_id": "trace_xyz789"
  }
}
```

**Additional field:**

- `retry_after` (integer, required for `rate_limited`): Seconds until retry allowed

---

### Error Codes

| Error Code | HTTP Status | Retryable | Description |
|------------|-------------|-----------|-------------|
| `invalid_request` | 400 | `false` | Malformed request (missing fields, invalid JSON) |
| `invalid_action_type` | 400 | `false` | Action type format is invalid |
| `unsupported_action_type` | 400 | `false` | Action type is not supported by this service |
| `unsupported_action_version` | 400 | `false` | Action version is not supported by this service |
| `agent_id_mismatch` | 400 | `false` | Agent ID in request body/path doesn't match signature header |
| `invalid_public_key` | 400 | `false` | Public key format invalid |
| `invalid_signature` | 401 | `false` | Signature verification failed |
| `timestamp_expired` | 401 | `false` | Request timestamp outside acceptable window (>5 minutes) |
| `invalid_code` | 401 | `false` | Confirmation code incorrect or expired |
| `invalid_token` | 401 | `false` | JWT token invalid or expired |
| `agent_not_found` | 404 | `false` | Agent not registered |
| `approver_not_found` | 404 | `false` | Approver username not found |
| `approval_not_found` | 404 | `false` | Approval ID not found |
| `agent_not_authorized` | 403 | `false` | Agent not registered with specified approver |
| `approval_denied` | 403 | `false` | User explicitly denied the approval |
| `token_already_used` | 403 | `false` | Single-use token already consumed |
| `insufficient_scope` | 403 | `false` | Token lacks required permissions |
| `invalid_parameters` | 403 | `false` | Request parameters don't match token `params_hash` |
| `agent_already_registered` | 409 | `false` | Agent already registered with this approver |
| `duplicate_request_id` | 409 | `false` | Request ID already used |
| `approval_already_resolved` | 409 | `false` | Approval already approved/denied/cancelled |
| `registration_expired` | 410 | `false` | Registration TTL elapsed |
| `approval_expired` | 410 | `false` | Approval TTL elapsed |
| `rate_limited` | 429 | `true` | Too many requests (includes `retry_after`) |
| `internal_error` | 500 | `true` | Server error (retry with exponential backoff) |
| `service_unavailable` | 503 | `true` | Service temporarily unavailable |

---

## Complete Examples

### Example 1: Agent Registration Flow

**Step 1: Agent initiates registration**

```http
POST https://api.example.com/permission-slip/v1/agents/register
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "req_reg_a1b2c3d4",
  "agent_id": "agent_abc123",
  "public_key": "ssh-ed25519 AAAAC3Nza...",
  "approver": "alice",
  "metadata": {
    "name": "My Assistant"
  }
}
```

**Response:**

```json
{
  "registration_url": "https://example.com/permission-slip/approve?token=eyJ...",
  "approver": "alice",
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Step 2: User approves (out of band)**

- User "alice" clicks the URL on their phone
- App shows: "Approve 'My Assistant'?"
- Alice taps "Approve"
- App displays confirmation code: `XK7-M9P`

**Step 3: Agent verifies registration**

```http
POST https://api.example.com/permission-slip/v1/agents/agent_abc123/verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667220", signature="..."

{
  "request_id": "req_verify_e5f6g7h8",
  "confirmation_code": "XK7-M9P"
}
```

**Response:**

```json
{
  "status": "approved",
  "registered_at": "2026-02-11T13:20:15Z"
}
```

---

### Example 2: Approval Request Flow

**Step 1: Agent requests approval**

```http
POST https://api.example.com/permission-slip/v1/approvals/request
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667300", signature="..."

{
  "agent_id": "agent_abc123",
  "request_id": "req_xyz789",
  "approver": "alice",
  "action": {
    "type": "email.send",
    "version": "1",
    "parameters": {
      "from": "alice@example.com",
      "to": ["bob@example.com"],
      "subject": "Meeting tomorrow",
      "body": "Let's meet at 3pm."
    }
  },
  "context": {
    "description": "Send meeting invitation to Bob",
    "risk_level": "low"
  }
}
```

**Response:**

```json
{
  "approval_id": "appr_abc456",
  "approval_url": "https://example.com/permission-slip/approve/appr_abc456",
  "alternative_urls": {
    "deeplink": "example://permission-slip/approve/appr_abc456"
  },
  "status": "pending",
  "expires_at": "2026-02-11T13:30:00Z",
  "verification_required": true
}
```

**Step 2: User approves (out of band)**

- Push notification sent to alice
- Alice reviews: "Send email to bob@example.com?"
- Alice approves
- Confirmation code shown: `RK3-P7M`

**Step 3: Agent verifies and retrieves token**

```http
POST https://api.example.com/permission-slip/v1/approvals/appr_abc456/verify
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667320", signature="..."

{
  "request_id": "req_verify_i9j0k1l2",
  "confirmation_code": "RK3-P7M"
}
```

**Response:**

```json
{
  "status": "approved",
  "approved_at": "2026-02-11T13:25:20Z",
  "token": {
    "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-02-11T13:30:20Z",
    "scope": "email.send",
    "scope_version": "1"
  }
}
```

**Step 4: Agent uses token to perform action**

The agent calls the service's **own action endpoint** (NOT a Permission Slip endpoint) using the token:

```http
POST https://api.example.com/v1/emails/send
Authorization: Bearer eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "from": "alice@example.com",
  "to": ["bob@example.com"],
  "subject": "Meeting tomorrow",
  "body": "Let's meet at 3pm."
}
```

**Note:** This endpoint (`/v1/emails/send`) is part of the service's own API, not the Permission Slip protocol. The service defines this endpoint's behavior, request/response format, and base URL in their own API documentation.

**Response:**

```json
{
  "message_id": "msg_sent123",
  "status": "sent"
}
```

---

## Protocol Version

This specification describes **Permission Slip Protocol v1.0**.

Services implementing this version MUST return `"version": "1.0"` in the discovery endpoint response.

---

## Related Documentation

- [Authentication Specification](authentication.md) - Cryptographic details for key generation and signature verification
- [Action Type Registry](https://github.com/supersuit-ai/permission-slip/issues/4) - Standard action type definitions (deferred)
