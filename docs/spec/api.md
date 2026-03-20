# API Specification

This document defines the complete HTTP API for the Permission Slip service.

> **Architecture Note:** All endpoints in this document are provided by the Permission Slip service. Agents communicate exclusively with Permission Slip — never with external services directly. When an approved action is executed, Permission Slip calls the external service's API using the user's stored credentials and returns the result to the agent. References to "service" in this document refer to the Permission Slip service unless otherwise noted.

---

## Table of Contents

1. [Glossary](#glossary)
2. [Base URL](#base-url)
3. [Action Types](#action-types)
4. [Authentication](#authentication)
5. [Registration Endpoints](#registration-endpoints)
6. [Connector Endpoints](#connector-endpoints)
7. [Capability Discovery](#capability-discovery)
8. [Credential Endpoints](#credential-endpoints)
9. [Approval Endpoints](#approval-endpoints)
10. [Token Usage](#token-usage)
11. [Standing Approvals](#standing-approvals)
12. [Error Handling](#error-handling)
13. [Complete Examples](#complete-examples)
14. [Protocol Version](#protocol-version)
15. [Related Documentation](#related-documentation)

---

## Glossary

For detailed definitions of core terms (Agent, Approver, Service, Action, Token, etc.), see **[Terminology](terminology.md)**.

---

## Base URL

Permission Slip is a hosted SaaS. All API endpoints are served at:

```
https://app.permissionslip.dev/permission-slip/v1
```

Agents interact only with the Permission Slip service — never with external services directly. When an approved action is executed, Permission Slip calls the external service's API using the user's stored credentials and returns the result to the agent.

All endpoints in this specification are documented relative to this base URL. For example, `POST /v1/approvals/request` means:

```
POST https://app.permissionslip.dev/permission-slip/v1/approvals/request
```

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

Services MUST explicitly support each version they accept and reject unsupported versions with error code `unsupported_action_version`.

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

All Permission Slip protocol API requests (registration, approval, and standing approval endpoints) MUST include a cryptographic signature proving agent identity, with two exceptions: connector endpoints (`GET /v1/connectors`) are unauthenticated, and credential endpoints use session authentication. This requirement does not apply to action execution using bearer tokens as described in [Token Usage](#token-usage).

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

```go
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	
	// Install: go get github.com/cyberphone/json-canonicalization/go/json
	jsoncanon "github.com/cyberphone/json-canonicalization/go/json"
)

func signRequest(
	method, path, rawQuery string,
	timestamp int64, // Unix timestamp in seconds
	requestBody interface{}, // nil for empty body (e.g., GET requests)
	privateKey ed25519.PrivateKey,
) (string, error) {
	// Validate inputs
	if method == "" {
		return "", fmt.Errorf("method cannot be empty")
	}
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if timestamp <= 0 {
		return "", fmt.Errorf("timestamp must be positive")
	}
	
	// RFC 3986-compliant percent-encoding for query parameters
	// Parse and canonicalize query string
	var canonicalQuery string
	if rawQuery != "" {
		// Parse query parameters using spec-compliant rules:
		// - Split on '&' only (do not treat ';' as a separator)
		// - Percent-decode each name and value
		type queryPair struct {
			key string
			val string
		}
		
		var params []queryPair
		parts := strings.Split(rawQuery, "&")
		for _, part := range parts {
			if part == "" {
				continue
			}
			
			var rawKey, rawVal string
			if idx := strings.Index(part, "="); idx >= 0 {
				rawKey = part[:idx]
				rawVal = part[idx+1:]
			} else {
				rawKey = part
				rawVal = ""
			}
			
			decodedKey, err := url.QueryUnescape(rawKey)
			if err != nil {
				return "", fmt.Errorf("failed to decode query key: %w", err)
			}
			decodedVal, err := url.QueryUnescape(rawVal)
			if err != nil {
				return "", fmt.Errorf("failed to decode query value: %w", err)
			}
			
			params = append(params, queryPair{key: decodedKey, val: decodedVal})
		}
		
		// Sort parameters by key first, then by value
		sort.Slice(params, func(i, j int) bool {
			if params[i].key == params[j].key {
				return params[i].val < params[j].val
			}
			return params[i].key < params[j].key
		})
		
		var pairs []string
		for _, p := range params {
			// RFC 3986 encoding (unreserved: A-Z a-z 0-9 - . _ ~)
			encodedKey := url.QueryEscape(p.key)
			encodedVal := url.QueryEscape(p.val)
			pairs = append(pairs, encodedKey+"="+encodedVal)
		}
		canonicalQuery = strings.Join(pairs, "&")
	}
	
	// Canonicalize body using RFC 8785 (JCS)
	var bodyHash string
	if requestBody != nil {
		jsonBytes, err := json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal body: %w", err)
		}
		canonicalBody, err := jsoncanon.Transform(jsonBytes)
		if err != nil {
			return "", fmt.Errorf("failed to canonicalize body: %w", err)
		}
		hash := sha256.Sum256(canonicalBody)
		bodyHash = hex.EncodeToString(hash[:])
	} else {
		// Empty body: SHA-256 hash of empty string
		// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
		hash := sha256.Sum256([]byte{})
		bodyHash = hex.EncodeToString(hash[:])
	}
	
	// Construct canonical request
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%d\n%s",
		strings.ToUpper(method),
		path,
		canonicalQuery,
		timestamp,
		bodyHash,
	)
	
	// Sign with Ed25519 private key
	signature := ed25519.Sign(privateKey, []byte(canonicalRequest))
	
	// Base64url encode without padding (RFC 4648 §5)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	
	return signatureB64, nil
}

// Example usage with error handling:
// timestamp := time.Now().Unix()
// signatureB64, err := signRequest("POST", "/invite/PS-R7K3-X9M4", "", timestamp, requestBody, privateKey)
// if err != nil {
//     return fmt.Errorf("failed to sign request: %w", err)
// }
// header := fmt.Sprintf(`agent_id="%s", algorithm="Ed25519", timestamp="%d", signature="%s"`,
//     agentID, timestamp, signatureB64)
```

### Signature Verification

Services MUST:

1. Verify `timestamp` is within 300 seconds (5 minutes) of current time to limit the replay window
2. Reconstruct the canonical request
3. Verify the signature:
   - For requests from already-registered agents, use the agent's registered public key.
   - For `POST /invite/{invite_code}` (agent registration), use the `public_key` provided in the request body.
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

> **Design Note:** Agent registration is **user-initiated**. The user generates a registration invite from the Permission Slip dashboard, which produces an invite URL (e.g., `https://app.permissionslip.dev/invite/PS-R7K3-X9M4`). The user shares this URL with the agent, and the agent POSTs directly to it to register. Registration requests without a valid invite code are rejected immediately. See [ADR-005](../adr/005-user-initiated-registration.md) for the rationale behind this design.

### POST /invite/{invite_code}

Register a new agent with Permission Slip by POSTing to the invite URL. The invite code in the URL path identifies which user the agent is registering with — the agent does not need to know the user's username.

**Path Parameters:**

- `invite_code` (string, required): Invite code from the URL path. Format: `PS-XXXX-XXXX` (8 alphanumeric characters).

**Request:**

```http
POST /invite/PS-R7K3-X9M4
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "6f1a7c30-9b2e-4d91-8c3f-2a4b6c7d8e9f",
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFPZj8V7K2mT9Xw4nL5rQ1pY3vN8cM6hU0oB4eW2sR7k",
  "metadata": {
    "name": "My AI Assistant",
    "version": "1.0.0",
    "capabilities": ["email", "calendar"]
  }
}
```

**Fields:**

- `request_id` (string, required): Unique request identifier for replay protection (UUID v4 recommended)
- `public_key` (string, required): Agent's public key in OpenSSH format. Permission Slip derives the `agent_id` from this key.
- `metadata` (object, optional): Agent metadata
  - `name` (string, optional): Human-readable agent name
  - `version` (string, optional): Agent version
  - `capabilities` (array, optional): Agent capabilities

**Validation:**

1. Services MUST validate the `invite_code` (from the URL path) against stored invite hashes. If the code is invalid, expired, already consumed, or locked out, return the appropriate error (see below).
2. Services MUST verify the signature matches the `public_key` in the request body.

**Response (200 OK):**

```json
{
  "agent_id": 42,
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Fields:**

- `agent_id` (integer, required): Server-assigned agent identifier. The agent uses this in subsequent API calls (e.g., `POST /v1/agents/{agent_id}/verify`).
- `expires_at` (string, required): ISO 8601 timestamp when registration expires (inherited from the invite's remaining TTL)
- `verification_required` (boolean, required): Always `true` (confirmation code required)

**Behavior:**

On successful validation, Permission Slip:
1. Consumes the invite code (single-use)
2. Creates a pending agent registration linked to the invite's creator (the user)
3. Generates a confirmation code and displays it to the user on their dashboard
4. The user communicates the confirmation code to the agent out-of-band

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Malformed request body
- `400 Bad Request` - `invalid_public_key`: Public key format invalid
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `401 Unauthorized` - `invalid_invite_code`: Invite code is incorrect
- `404 Not Found` - `invite_not_found`: No matching invite exists
- `409 Conflict` - `agent_already_registered`: Agent is already registered
- `410 Gone` - `invite_expired`: Invite code has expired
- `423 Locked` - `invite_locked`: Invite locked due to too many failed attempts (5 max)

---

### POST /v1/agents/{agent_id}/verify

Complete agent registration by submitting the confirmation code shown to the user on their dashboard after the agent registered via `POST /invite/{invite_code}`.

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

## Connector Endpoints

Connectors represent available integrations (e.g., Gmail, Stripe, GitHub). These endpoints are **unauthenticated** — agents use them to discover what actions are available and what credentials users need to set up.

### GET /v1/connectors

List all available connectors.

**Request:**

```http
GET /v1/connectors
```

**Response (200 OK):**

```json
{
  "connectors": [
    {
      "id": "gmail",
      "name": "Gmail",
      "description": "Read, send, and manage emails via the Gmail API",
      "actions": [
        {
          "action_type": "email.send",
          "name": "Send Email",
          "description": "Send an email on behalf of the user",
          "risk_level": "medium",
          "parameters_schema": {
            "type": "object",
            "properties": {
              "to": { "type": "array", "items": { "type": "string" } },
              "subject": { "type": "string" },
              "body": { "type": "string" }
            },
            "required": ["to", "subject", "body"]
          }
        },
        {
          "action_type": "email.read",
          "name": "Read Emails",
          "description": "Read emails matching a filter",
          "risk_level": "low",
          "parameters_schema": { "..." : "..." }
        }
      ],
      "required_credentials": [
        {
          "service": "github",
          "auth_type": "api_key"
        }
      ]
    }
  ]
}
```

**Fields:**

- `connectors` (array, required): List of available connectors
  - `id` (string): Connector identifier
  - `name` (string): Human-readable name
  - `description` (string or null): Connector description
  - `actions` (array): Actions available on this connector
    - `action_type` (string): Action type identifier (used in approval requests)
    - `name` (string): Human-readable action name
    - `description` (string or null): Action description
    - `risk_level` (string or null): `low`, `medium`, or `high`
    - `parameters_schema` (object or null): JSON Schema describing action parameters
  - `required_credentials` (array): Credential types the user must set up to use this connector. The `service` field is a join key to `credentials.service` — it identifies which stored credential Permission Slip uses when executing actions for this connector. A connector may require credentials for services other than itself (e.g., a `google-workspace` connector might need both `gmail` and `google-calendar` credentials).
    - `service` (string): Service name that maps to a stored credential (e.g., `"github"`, `"stripe"`)
    - `auth_type` (string): `api_key`, `basic`, or `custom`

---

### GET /v1/connectors/{connector_id}

Get details for a specific connector.

**Path Parameters:**

- `connector_id` (string, required): Connector identifier

**Request:**

```http
GET /v1/connectors/gmail
```

**Response (200 OK):**

Same structure as a single element of the `connectors` array in the list response.

**Error Responses:**

- `404 Not Found` - `connector_not_found`: Connector ID not found

---

## Capability Discovery

After registration, agents need to know what they can do on behalf of their user. The capability discovery endpoint provides a complete, agent-specific view in a single call.

### GET /v1/agents/{agent_id}/capabilities

Returns everything the agent needs to know: enabled connectors, available actions with full parameter schemas, active standing approvals, and credential readiness.

This is the primary discovery endpoint for agents. Unlike `GET /v1/connectors` (which returns the global catalog), this endpoint returns only what's relevant to the authenticated agent — connectors the user has enabled, actions the agent can perform, and which actions can execute immediately via standing approval.

**Path Parameters:**

- `agent_id` (integer, required): Must match the `agent_id` in the signature header

**Request:**

```http
GET /v1/agents/42/capabilities
X-Permission-Slip-Signature: agent_id="42", algorithm="Ed25519", timestamp="1707667200", signature="..."
```

**Response (200 OK):**

```json
{
  "agent_id": 42,
  "connectors": [
    {
      "id": "gmail",
      "name": "Gmail",
      "description": "Send and manage emails via Gmail API",
      "credentials_ready": true,
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
              "to": { "type": "array", "items": { "type": "string", "format": "email" } },
              "subject": { "type": "string", "maxLength": 998 },
              "body": { "type": "string" }
            }
          },
          "standing_approvals": [
            {
              "standing_approval_id": "sa_def456",
              "constraints": { "recipient_pattern": "*@mycompany.com" },
              "max_executions": 100,
              "executions_remaining": 88,
              "expires_at": "2026-05-15T00:00:00Z"
            }
          ]
        },
        {
          "action_type": "email.read",
          "name": "Read Email",
          "description": "Read emails from Gmail inbox",
          "risk_level": "low",
          "parameters_schema": { "..." : "..." },
          "standing_approvals": []
        }
      ]
    },
    {
      "id": "stripe",
      "name": "Stripe",
      "description": "Payment processing via Stripe API",
      "credentials_ready": false,
      "credentials_setup_url": "https://app.permissionslip.dev/connect/stripe",
      "actions": [
        {
          "action_type": "payment.charge",
          "name": "Charge Payment",
          "description": "Charge a customer's payment method",
          "risk_level": "high",
          "parameters_schema": { "..." : "..." },
          "standing_approvals": []
        }
      ]
    }
  ]
}
```

**Fields:**

- `agent_id` (integer): The authenticated agent's identifier
- `connectors` (array): Connectors enabled for this agent
  - `id` (string): Connector identifier
  - `name` (string): Human-readable name
  - `description` (string or null): Connector description
  - `credentials_ready` (boolean): Whether the user has stored the required credentials. `true` means actions can be executed; `false` means execution will fail with `credentials_not_found`. No credential values are ever exposed.
  - `credentials_setup_url` (string or null): URL where the user can set up credentials. Only included when `credentials_ready` is `false`.
  - `actions` (array): Available actions with full schemas
    - `action_type` (string): Action type identifier (use as `action.type` in approval requests)
    - `name` (string): Human-readable action name
    - `description` (string or null): Action description
    - `risk_level` (string or null): `low`, `medium`, or `high`
    - `parameters_schema` (object or null): JSON Schema for parameter validation
    - `standing_approvals` (array): Active standing approvals for this action type. If non-empty, the agent can execute immediately under matching constraints. If empty, one-off approval is required.
      - `standing_approval_id` (string): Standing approval identifier
      - `constraints` (object): Parameter constraint boundaries
      - `max_executions` (integer or null): Execution cap (`null` = unlimited)
      - `executions_remaining` (integer or null): Remaining executions (`null` = unlimited)
      - `expires_at` (string): ISO 8601 expiration timestamp

**Agent Workflow After Calling This Endpoint:**

1. **Check `connectors`**: What services are available?
2. **Check `credentials_ready`**: If `false`, prompt user to visit `credentials_setup_url`
3. **Check `standing_approvals`** for desired action: If non-empty with matching constraints, `POST /v1/approvals/request` will auto-approve and execute immediately
4. **If no matching standing approval**: Use one-off flow via `POST /v1/approvals/request` (creates a pending approval for user review)

**Error Responses:**

- `400 Bad Request` - `agent_id_mismatch`: Agent ID in path doesn't match signature header
- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `404 Not Found` - `agent_not_found`: Agent not registered

---

## Credential Endpoints

Credentials store references to user-provided service keys (API keys, OAuth tokens, etc.) that Permission Slip uses to execute actions on external services. The actual secret material is stored in a secrets vault — the API only exposes metadata.

These endpoints use **session authentication** (Supabase Auth). The user must be logged in via the web UI — requests are authenticated using the user's session token, not the `X-Permission-Slip-Signature` header used by agents. Agents do not call these endpoints directly.

### GET /v1/credentials

List the authenticated user's stored credentials.

**Request:**

```http
GET /v1/credentials
```

**Response (200 OK):**

```json
{
  "credentials": [
    {
      "id": "cred_abc123",
      "service": "gmail",
      "label": null,
      "created_at": "2026-02-10T09:15:00Z"
    },
    {
      "id": "cred_def456",
      "service": "stripe",
      "label": "production",
      "created_at": "2026-02-11T14:30:00Z"
    }
  ]
}
```

**Fields:**

- `credentials` (array, required): List of the user's stored credentials
  - `id` (string): Credential identifier
  - `service` (string): Service name (e.g., `"gmail"`, `"stripe"`)
  - `label` (string or null): Optional label for distinguishing multiple credentials for the same service (e.g., `"work"`, `"personal"`)
  - `created_at` (string): ISO 8601 timestamp

---

### POST /v1/credentials

Store a new credential for the authenticated user.

**Request:**

```http
POST /v1/credentials
Content-Type: application/json

{
  "service": "gmail",
  "credentials": {
    "access_token": "ya29.a0AfH6SM...",
    "refresh_token": "1//0dx..."
  },
  "label": "work"
}
```

**Fields:**

- `service` (string, required): Service name
- `credentials` (object, required): Service-specific credential fields (e.g., `{ "api_key": "sk_live_..." }` or `{ "access_token": "...", "refresh_token": "..." }`). Stored in the secrets vault — never returned by the API after creation.
- `label` (string, optional): Label to distinguish multiple credentials for the same service

**Response (201 Created):**

```json
{
  "id": "cred_ghi789",
  "service": "gmail",
  "label": "work",
  "created_at": "2026-02-15T10:00:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - `invalid_request`: Missing required fields
- `409 Conflict` - `duplicate_credential`: Credential already exists for this user/service/label combination

---

### DELETE /v1/credentials/{credential_id}

Delete a stored credential.

**Path Parameters:**

- `credential_id` (string, required): Credential identifier

**Request:**

```http
DELETE /v1/credentials/cred_abc123
```

**Response (200 OK):**

```json
{
  "id": "cred_abc123",
  "deleted_at": "2026-02-11T15:00:00Z"
}
```

**Fields:**

- `id` (string): Credential identifier that was deleted
- `deleted_at` (string): ISO 8601 timestamp of deletion

**Error Responses:**

- `403 Forbidden` - `agent_not_authorized`: Not authorized to delete this credential
- `404 Not Found` - `credential_not_found`: Credential ID not found or does not belong to the authenticated user

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
- `payment_method_id` (string, optional): Payment method ID for payment-required actions. Forwarded to the connector when a standing approval auto-approves.
- `amount_cents` (integer, optional): Transaction amount in cents. Required when `payment_method_id` is provided.

**Validation:**

Services MUST verify that the `agent_id` in the request body matches the `agent_id` in the `X-Permission-Slip-Signature` header. If they do not match, return `400 Bad Request` with error code `agent_id_mismatch`.

**Response (200 OK — Pending Approval):**

When no matching standing approval exists, a pending approval is created for user review:

```json
{
  "approval_id": "appr_xyz789",
  "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_xyz789",
  "alternative_urls": {
    "deeplink": "permissionslip://permission-slip/approve/appr_xyz789",
    "web": "https://accounts.permissionslip.dev/permission-slip/approve/appr_xyz789"
  },
  "status": "pending",
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Response (200 OK — Auto-Approved via Standing Approval):**

When a matching standing approval exists, the action is auto-approved and executed immediately:

```json
{
  "status": "approved",
  "result": {
    "emails": [...]
  },
  "standing_approval_id": "sa_abc123",
  "executions_remaining": null
}
```

**Fields (pending response):**

- `approval_id` (string, required): Unique approval identifier
- `approval_url` (string, required): Primary approval URL (universal link)
- `alternative_urls` (object, optional): Optional alternative URL formats for specific platforms
- `status` (string, required): Approval status (`pending`)
- `expires_at` (string, required): ISO 8601 timestamp when approval expires
- `verification_required` (boolean, required): Always `true` (confirmation code required)

**Fields (auto-approved response):**

- `status` (string, required): Approval status (`approved`)
- `result` (object, required): Action result from the external service
- `standing_approval_id` (string, required): Which standing approval authorized this execution
- `executions_remaining` (integer or null, required): Remaining executions (`null` = unlimited)

> **Execution slot consumption on error:** When a standing approval matches, the execution slot is consumed _before_ the connector action runs. If the connector fails (network timeout, upstream error, etc.), the slot is still consumed. On untyped 500 errors, the response includes `executions_remaining` and `standing_approval_id` in the error `details` so agents can track quota erosion. Agents should monitor `executions_remaining` and request a new standing approval if the quota runs low due to transient failures.

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

After receiving an approval token from the one-off flow, the agent uses it to perform the approved action. For actions covered by a **standing approval**, no token is needed — see [Standing Approvals](#standing-approvals) below.

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
  "aud": "permissionslip.dev",
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

After receiving an approval token from the one-off flow, the agent presents it in a subsequent `POST /v1/approvals/request` call. Permission Slip validates the token, then executes the action by calling the appropriate external service API using the user's stored credentials.

**Note:** For actions covered by a standing approval, no token is needed. When `POST /v1/approvals/request` is called and a matching standing approval exists, the action is auto-approved and executed immediately — the response includes the result inline with `status: "approved"`. See [Standing Approvals](#standing-approvals) for details.

### Token Signature Verification (JWKS)

Permission Slip publishes its token signing public keys at:

**JWKS Endpoint:**

```http
GET https://app.permissionslip.dev/.well-known/permission-slip-jwks.json
```

**Response:**

```json
{
  "keys": [
    {
      "kty": "EC",
      "use": "sig",
      "crv": "P-256",
      "kid": "key-2026-01",
      "x": "base64url...",
      "y": "base64url...",
      "alg": "ES256"
    }
  ]
}
```

Agents MAY fetch the JWKS if they choose to locally validate approval token signatures; however, Permission Slip is responsible for verifying approval tokens when they are presented for action execution. Permission Slip MAY rotate keys by adding new keys to the JWKS and including a `kid` (key ID) header parameter in issued tokens.

### Token Verification Requirements

Permission Slip MUST:

1. **Verify JWT signature** using service's public key
2. **Check expiration** (`exp` claim)
3. **Verify audience** (`aud` matches service domain)
4. **Verify scope** matches the `action.type` in the request body
5. **Verify params_hash** matches the `action.parameters` object in the request body
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

// verifyParameters checks that the action.parameters object in the request body matches
// the params_hash claim in the JWT. This prevents parameter tampering attacks where an
// attacker modifies parameters after approval but reuses the original JWT token.
//
// The request body has the structure: {"request_id": "...", "action": {"type": "...", "parameters": {...}}}
// Only the action.parameters object is hashed — not the full body.
func verifyParameters(w http.ResponseWriter, r *http.Request, tokenClaims map[string]interface{}) bool {
	// Helper to send a spec-conformant JSON error response.
	writeJSONError := func(code, message string, retryable bool, status int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
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
			_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"Failed to encode error response.","retryable":false}}`))
		}
	}

	// Read and preserve the request body so other handlers can still access it.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError("invalid_parameters", "Failed to read request body.", false, http.StatusForbidden)
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Parse the request body to extract action.parameters
	var body struct {
		Action struct {
			Parameters json.RawMessage `json:"parameters"`
		} `json:"action"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil || body.Action.Parameters == nil {
		writeJSONError("invalid_parameters", "Request body missing action.parameters.", false, http.StatusForbidden)
		return false
	}

	// Canonicalize action.parameters using RFC 8785 (JCS)
	canonicalParams, err := jsoncanon.Transform(body.Action.Parameters)
	if err != nil {
		writeJSONError("invalid_parameters", "Failed to canonicalize action parameters.", false, http.StatusForbidden)
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
		writeJSONError("invalid_parameters", "Action parameters do not match the params_hash claim.", false, http.StatusForbidden)
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

Approval tokens are short-lived and single-use. They apply to the **one-off approval flow** only. Standing approvals bypass the token mechanism entirely — see [Standing Approvals](#standing-approvals).

**Token Properties:**

- **Expiration:** Tokens typically expire 5 minutes after issuance (service-defined via `exp` claim)
- **Single-use:** Each token can only be used once, enforced via the `jti` claim
- **No refresh tokens:** The one-off flow does not support refresh tokens by design

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

## Standing Approvals

Standing approvals allow agents to execute pre-authorized actions without the one-off approval flow (no push notification, no confirmation code, no single-use token). See [ADR-002](../adr/002-standing-approvals.md) for full design details and [Terminology](terminology.md) for definitions.

### GET /v1/agents/{agent_id}/standing-approvals

List active standing approvals for an agent. Agents use this to discover which actions they can execute immediately without the one-off approval flow.

**Path Parameters:**

- `agent_id` (string, required): Must match the `agent_id` in the signature header

**Request:**

```http
GET /v1/agents/agent_x7K9mP4n.../standing-approvals
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4n...", algorithm="Ed25519", timestamp="1707667200", signature="..."
```

**Response (200 OK):**

```json
{
  "standing_approvals": [
    {
      "standing_approval_id": "sa_abc123",
      "action_type": "email.read",
      "action_version": "1",
      "constraints": {
        "sender": "*@github.com",
        "max_results": 10
      },
      "status": "active",
      "max_executions": null,
      "execution_count": 42,
      "starts_at": "2026-02-10T00:00:00Z",
      "expires_at": "2026-03-12T00:00:00Z"
    },
    {
      "standing_approval_id": "sa_def456",
      "action_type": "email.send",
      "action_version": "1",
      "constraints": {
        "to": "*@mycompany.com"
      },
      "status": "active",
      "max_executions": 50,
      "execution_count": 3,
      "starts_at": "2026-02-14T00:00:00Z",
      "expires_at": "2026-02-21T00:00:00Z"
    }
  ]
}
```

**Fields:**

- `standing_approvals` (array, required): List of active standing approvals for this agent
  - `standing_approval_id` (string): Standing approval identifier
  - `action_type` (string): Action type this approval covers
  - `action_version` (string): Action version
  - `constraints` (object or null): Parameter bounds enforced at execution time
  - `status` (string): Always `active` (only active standing approvals are returned)
  - `max_executions` (integer or null): Execution cap (`null` = unlimited)
  - `execution_count` (integer): Number of times this standing approval has been used
  - `starts_at` (string): ISO 8601 timestamp when the approval became active
  - `expires_at` (string): ISO 8601 timestamp when the approval expires (max 90 days from `starts_at`)

**Error Responses:**

- `401 Unauthorized` - `invalid_signature`: Signature verification failed
- `400 Bad Request` - `agent_id_mismatch`: Agent ID in path doesn't match signature header
- `404 Not Found` - `agent_not_found`: Agent not registered

---

### Standing Approval Matching via POST /v1/approvals/request

Standing approval matching and execution is handled automatically by `POST /v1/approvals/request`. When an agent submits an approval request, Permission Slip checks for matching standing approvals before creating a pending approval:

**Execution Flow:**

1. **Verify agent signature** (Ed25519, same as all requests)
2. **Match standing approval:** Find an active standing approval for this agent + action type
3. **Validate constraints:** Verify parameters fall within the standing approval's constraint bounds
4. **Check expiration:** Verify standing approval has not expired
5. **Check execution count:** If `max_executions` is set, verify count has not been exceeded
6. **Execute action:** Call external service API using stored credentials
7. **Increment execution count** and **create audit log entry**
8. **Return result** to agent with `status: "approved"`

If a matching standing approval is found, the response includes the action result inline (see the auto-approved response format in the [POST /v1/approvals/request](#post-v1approvalsrequest) section above).

**Fallthrough Behavior:**

If no standing approval matches (wrong action type, parameters outside constraints, expired, or exhausted), Permission Slip creates a pending approval as usual and returns `status: "pending"`. The agent then proceeds with the standard one-off approval flow (user review, confirmation code, token).

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
// confirmation code using the allowed character set (base32-like, without 0/O/1/I).
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
| `standing_approval_exhausted` | 403 | `false` | Standing approval execution count exceeded |
| `constraint_violation` | 403 | `false` | Parameters violate standing approval constraints |
| `no_matching_standing_approval` | 404 | `false` | No active standing approval matches this agent/action/parameters |
| `standing_approval_expired` | 410 | `false` | Standing approval has expired |
| `credential_not_found` | 404 | `false` | Credential ID not found |
| `credentials_not_found` | 404 | `false` | No stored credentials found for the required service |
| `connector_not_found` | 404 | `false` | Connector ID not found |
| `duplicate_credential` | 409 | `false` | Credential already exists for this user/service/label |
| `upstream_error` | 502 | `true` | External service returned an error during action execution |
| `rate_limited` | 429 | `true` | Too many requests (includes `retry_after`) |
| `internal_error` | 500 | `true` | Server error (retry with exponential backoff) |
| `service_unavailable` | 503 | `true` | Service temporarily unavailable |

---

## Complete Examples

### Example 1: Agent Registration Flow

> Registration is user-initiated via invite codes. See [ADR-005](../adr/005-user-initiated-registration.md).

**Step 1: User generates an invite (dashboard)**

- User "alice" opens Permission Slip dashboard
- Clicks "Add Agent"
- Permission Slip generates invite URL: `https://app.permissionslip.dev/invite/PS-R7K3-X9M4`
- Alice copies the URL and shares it with the agent (or the agent's operator)

**Step 2: Agent POSTs to the invite URL**

```http
POST https://app.permissionslip.dev/invite/PS-R7K3-X9M4
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "request_id": "req_reg_a1b2c3d4",
  "public_key": "ssh-ed25519 AAAAC3Nza...",
  "metadata": {
    "name": "My Assistant"
  }
}
```

**Response:**

```json
{
  "agent_id": 42,
  "expires_at": "2026-02-11T13:25:00Z",
  "verification_required": true
}
```

**Step 3: User sees confirmation code (dashboard)**

- Alice's dashboard updates to show a pending registration from "My Assistant"
- Dashboard displays confirmation code: `XK7-M9P`
- Alice communicates the code to the agent

**Step 4: Agent verifies registration**

```http
POST https://app.permissionslip.dev/permission-slip/v1/agents/agent_abc123/verify
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

**Step 5: Registration complete**

- Alice's dashboard updates to show "My Assistant" as a registered agent
- The agent can now submit approval requests

---

### Example 2: Approval Request Flow

**Step 1: Agent requests approval**

```http
POST https://app.permissionslip.dev/permission-slip/v1/approvals/request
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
  "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_abc456",
  "alternative_urls": {
    "deeplink": "permissionslip://permission-slip/approve/appr_abc456"
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
POST https://app.permissionslip.dev/permission-slip/v1/approvals/appr_abc456/verify
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

**Step 4: Agent uses token to execute action**

The agent presents the token back to Permission Slip via `POST /v1/approvals/request` with the token included. Permission Slip validates the JWT, verifies the `params_hash` claim against the `action.parameters` object, then calls the Gmail API using the user's stored credentials and returns the result.

**Response:**

```json
{
  "result": {
    "message_id": "msg_sent123",
    "status": "sent"
  }
}
```

---

## Protocol Version

This specification describes **Permission Slip Protocol v1.0**.

---

## Related Documentation

- [Authentication Specification](authentication.md) - Cryptographic details for key generation and signature verification
- [Action Type Registry](https://github.com/supersuit-ai/permission-slip/issues/4) - Standard action type definitions (deferred)
