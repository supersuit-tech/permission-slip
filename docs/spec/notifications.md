# Notifications

This document specifies how Permission Slip notifies users when agents request approval for registration or actions.

> **Architecture Note:** In the Permission Slip architecture, all notifications are sent by the Permission Slip service. External services (Gmail, Stripe, etc.) are not involved in the notification flow — they are unaware of Permission Slip. References to "service" in this document refer to the Permission Slip service unless otherwise noted.

---

## Table of Contents

1. [Overview](#overview)
2. [Delivery Mechanisms](#delivery-mechanisms)
3. [Notification Types](#notification-types)
4. [Webhook Notifications](#webhook-notifications)
5. [Mobile Push Notifications](#mobile-push-notifications)
6. [Approver Preferences](#approver-preferences)
7. [Deep Linking](#deep-linking)
8. [Security Considerations](#security-considerations)

> **Note for developers:** For instructions on adding a new notification channel to the server, see the `notify` package documentation and the README's "Adding a notification channel" section.

---

## Overview

When an agent submits an action for approval (or requests registration), the service MUST notify the designated approver in real-time. This specification defines how services deliver these notifications to approvers.

**Notification Triggers:**

1. **Action Approval Request:** When a registered agent submits a **one-off** action for approval

**Important:** Actions executed under a **standing approval** do NOT trigger notifications. The standing approval itself is the pre-authorization — no per-request notification is needed. See [ADR-002](../adr/002-standing-approvals.md) and [Terminology](terminology.md) for standing approval details.

**Important:** Agent registration does NOT trigger unsolicited push notifications. Registration is user-initiated via invite codes (see [ADR-005](../adr/005-user-initiated-registration.md)). The user generates an invite from the dashboard, and registration status updates (pending, confirmation code ready, complete) are shown on the dashboard via polling. No push notification is needed because the user is already expecting the registration — they initiated it.

**Notification Purpose:**

- Alert the approver that a one-off action is pending their review
- Provide enough context to understand what action is being requested
- Enable quick navigation to the approval UI
- Respect the approval request TTL

---

## Approval Request Time-To-Live (TTL)

Approval requests have a limited lifetime to ensure security and timely resolution.

**TTL Requirements:**

- Services SHOULD provide at least 5 minutes TTL by default for approval requests
- Services MAY use longer TTLs (15 minutes, 30 minutes, 1 hour, etc.) based on risk level or approver preferences
- Registration requests MAY have longer TTLs (e.g., 24 hours) since they are less time-sensitive
- All TTLs MUST be communicated via the `expires_at` field in notification payloads

**TTL Considerations for Notification Delivery:**

- **Retry Logic:** Services MUST check if an approval has expired before retrying notification delivery
- **Channel Selection:** When multiple channels are available, concurrent delivery maximizes the chance of reaching the approver within the TTL window
- **Expired Notifications:** Receivers SHOULD clear or mark expired notifications to avoid confusion

**Rationale:** Short TTLs reduce the attack window for compromised agent credentials or intercepted approval URLs. Longer TTLs improve approver convenience but increase security risk.

---

## Delivery Mechanisms

Permission Slip supports four notification channels. Channels are active only when their required configuration is present; multiple channels may be active simultaneously and notifications are fanned out to all of them.

### Webhook Notifications (Required)

**Webhooks** are HTTP POST requests sent from the service to a URL registered by the approver.

**Characteristics:**
- Simple HTTP-based delivery
- Works for web apps, mobile apps with backend, desktop apps, or notification services
- No third-party service dependencies
- Reliable and easy to implement
- Suitable for all notification use cases

**Requirement Level:** Services MUST support webhook notifications.

### Web Push Notifications (Optional)

**Web push** delivers browser notifications via the [Web Push Protocol](https://www.rfc-editor.org/rfc/rfc8030) using VAPID authentication. Subscriptions are stored per browser; the service sends encrypted payloads directly to the browser push service (FCM for Chrome, Mozilla for Firefox, etc.).

**Configuration:** Set `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, and `VAPID_SUBJECT`. Keys are auto-generated in development mode. See the README for key generation instructions.

**Requirement Level:** Services MAY support web push as an optional enhancement.

### Mobile Push Notifications (Optional)

**Mobile push notifications** deliver alerts to iOS and Android devices via the [Expo Push Service](https://docs.expo.dev/push-notifications/overview/), which routes to APNs (iOS) and FCM (Android). Device tokens are registered on login via the Permission Slip mobile app.

**Configuration:** Always enabled when a database is available. Set `EXPO_ACCESS_TOKEN` for higher rate limits (authenticated mode); unauthenticated mode is used otherwise.

**Requirement Level:** Services MAY support mobile push as an optional enhancement.

### Email Notifications (Optional)

**Email notifications** deliver approval requests to the approver's registered email address. Two providers are supported: SendGrid and SMTP.

**Configuration:** Set `NOTIFICATION_EMAIL_PROVIDER` to `twilio-sendgrid` or `smtp`, plus the provider-specific credentials. See `.env.example` for all variables.

**Requirement Level:** Services MAY support email as an optional enhancement.

### SMS Notifications (Optional)

**SMS notifications** deliver approval requests as text messages via Twilio. SMS delivery is gated behind paid subscription tiers when billing is enabled.

**Configuration:** Set `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, and `TWILIO_FROM_NUMBER`. All three are required; partial configuration disables the channel.

**Requirement Level:** Services MAY support SMS as an optional enhancement.

---

## Notification Types

Services send notifications for two event types: agent registration and approval requests.

### Agent Registration Status Update (Informational)

> **Changed in [ADR-005](../adr/005-user-initiated-registration.md):** Agent registration is now user-initiated via invite codes. Registration no longer triggers unsolicited push notifications. Instead, registration status updates are delivered to the user's dashboard. Webhook notifications for registration events are OPTIONAL — services MAY send them for integrations that want to react to registration completion, but they are not required for the core approval flow.

Sent when an agent completes registration using an invite code the user generated. This is an **informational** notification — the user already expects it because they initiated the invite.

**Trigger:** Agent successfully calls `POST /v1/agents/{agent_id}/verify` with a valid confirmation code

**Payload Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Always `"agent_registration_complete"` |
| `agent_id` | string | Yes | Agent's derived ID |
| `agent_name` | string | No | Human-readable agent name (from agent metadata) |
| `metadata` | object | No | Agent registration metadata as defined in the core API (for example, name, version, capabilities) |
| `approver` | string | Yes | Username/identifier of the approver who created the invite |
| `registered_at` | string | Yes | ISO 8601 timestamp (RFC 3339 format, UTC) when registration completed |
| `timestamp` | string | Yes | ISO 8601 timestamp (RFC 3339 format, UTC) when notification was sent |

**Example Payload:**

```json
{
  "type": "agent_registration_complete",
  "agent_id": "agent_x7K9mP4nQ8rT2vW5yZ1aC3bD6eF9gH0jK4lM7nP0qR3",
  "agent_name": "My AI Assistant",
  "metadata": {
    "name": "My AI Assistant",
    "version": "1.0.0",
    "capabilities": ["email", "calendar"]
  },
  "approver": "alice",
  "registered_at": "2026-02-12T09:22:00Z",
  "timestamp": "2026-02-12T09:22:01Z"
}
```

### Action Approval Notification

Sent when a registered agent submits an action for approval.

**Trigger:** `POST /v1/approvals/request`

**Payload Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Always `"approval_request"` |
| `approval_id` | string | Yes | Unique approval identifier |
| `agent_id` | string | Yes | Agent's derived ID |
| `agent_name` | string | No | Human-readable agent name |
| `action` | object | Yes | Action summary |
| `action.type` | string | Yes | Action type identifier |
| `action.summary` | string | Yes | Brief human-readable action description |
| `context` | object | No | Additional context for approver |
| `context.description` | string | No | Detailed action description |
| `context.risk_level` | string | No | Risk level: `low`, `medium`, `high` (see Risk Level Values below) |
| `approver` | string | Yes | Username/identifier of the approver (format defined by service; typically alphanumeric string matching the approver's account identifier) |
| `expires_at` | string | Yes | ISO 8601 timestamp (RFC 3339 format, UTC) when approval expires, e.g., `2026-02-12T09:25:00Z` |
| `approval_url` | string | Yes | URL to approval UI |
| `alternative_urls` | object | No | Optional alternative URLs for cross-platform deep linking (deeplink for mobile apps, web for browsers) |
| `timestamp` | string | Yes | ISO 8601 timestamp (RFC 3339 format, UTC) when notification was sent |

**Example Payload:**

```json
{
  "type": "approval_request",
  "approval_id": "appr_xyz789",
  "agent_id": "agent_x7K9mP4nQ8rT2vW5yZ1aC3bD6eF9gH0jK4lM7nP0qR3",
  "agent_name": "My AI Assistant",
  "action": {
    "type": "email.send",
    "summary": "Send welcome email to new user"
  },
  "context": {
    "description": "Send welcome email to recipient@example.com with subject 'Welcome'",
    "risk_level": "low"
  },
  "approver": "alice",
  "expires_at": "2026-02-12T09:25:00Z",
  "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_xyz789",
  "alternative_urls": {
    "deeplink": "permissionslip://approve/appr_xyz789",
    "web": "https://app.permissionslip.dev/approve/appr_xyz789"
  },
  "timestamp": "2026-02-12T09:20:00Z"
}
```

**Important:** 
- The `action` object SHOULD NOT include sensitive parameters (e.g., email body, API keys, passwords)
- The `action.summary` and `context.description` provide human-readable context
- Services SHOULD limit `action.summary` to 200 characters for compatibility with push notification displays
- The approver fetches full details via the approval UI

**Note:** The `agent_name` field is the primary identifier for agent display in notifications. For payloads that also include `metadata.name`, both fields SHOULD contain the same value.

**Risk Level Values:** The `risk_level` field uses lowercase string values: `low`, `medium`, or `high`. Services MAY define additional custom risk levels but SHOULD document them clearly for approvers. Receivers SHOULD handle unknown risk levels gracefully (default to `medium` treatment if unknown).

---

## Webhook Notifications

Webhooks deliver notifications via HTTP POST requests to a URL registered by the approver.

### Webhook Registration

Approvers register webhook URLs in their service account settings. Services MUST allow approvers to configure at least one webhook URL.

**Configuration Options:**

- **Webhook URL:** HTTPS endpoint to receive notifications (MUST be HTTPS; services MUST reject non-HTTPS URLs, localhost, and private IP ranges; validation MUST occur at configuration time AND before each delivery - see [Webhook Security](#webhook-security) for full requirements)
- **Secret:** Shared secret for signature verification (MUST be at least 32 bytes of cryptographically secure random data, encoded using standard Base64 for storage and transmission; generated by service or provided by approver; receivers MUST decode from Base64 to obtain raw key bytes before using in HMAC operations)
- **Events:** Which notification types to receive (registration, approvals, or both)

**Generating Webhook Secrets (Go Example):**

```go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateWebhookSecret creates a cryptographically secure webhook secret
// Returns base64-encoded string suitable for storage and transmission
func GenerateWebhookSecret() (string, error) {
	// Generate 32 bytes (256 bits) of secure random data for HMAC-SHA256
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Encode to base64 for storage/transmission
	secretBase64 := base64.StdEncoding.EncodeToString(secretBytes)
	return secretBase64, nil
}

// Example usage:
// secret, err := GenerateWebhookSecret()
// if err != nil {
//     log.Fatal(err)
// }
// fmt.Println("New webhook secret:", secret)
// // Store secret in database (encrypted at rest)
```

**Multiple Webhooks:**

Services MUST support at least one webhook URL. Services MAY support multiple webhook URLs for redundancy. If multiple webhooks are configured, services SHOULD deliver to all URLs concurrently (not sequentially or failover).

**Multiple Webhook Configuration:**

When multiple webhooks are configured:
- Each webhook URL MAY have its own independent shared secret
- All webhooks for the same notification MUST receive the same delivery ID
- Signature verification failures for one webhook do not affect delivery to other webhooks
- Services SHOULD track delivery status independently for each configured webhook

**Example Configuration (Service UI):**

```
Webhook URL: https://notifications.permissionslip.dev/permission-slip
Secret: wh_sec_**************** [Generate New] [Reveal]
Events: [x] Agent Registration  [x] Approval Requests
[Test Webhook]
```

**Note:** When a secret is first generated, it should be displayed once in full. After saving, it should be masked in the UI.

### Webhook Request Format

Services send webhook notifications as HTTP POST requests with JSON payloads.

**HTTP Request:**

```http
POST /permission-slip HTTP/1.1
Host: notifications.permissionslip.dev
Content-Type: application/json
User-Agent: ExampleService-PermissionSlip/1.0
X-Permission-Slip-Version: v1
X-Permission-Slip-Webhook-Signature: t=1770888000,v1=5f8d9e7c6b4a3f2e1d0c9b8a7f6e5d4c3b2a1f0e9d8c7b6a5f4e3d2c1b0a9f8e
X-Permission-Slip-Delivery: 550e8400-e29b-41d4-a716-446655440000

{
  "type": "approval_request",
  "approval_id": "appr_xyz789",
  "agent_id": "agent_x7K9mP4n...",
  "agent_name": "My AI Assistant",
  "action": {
    "type": "email.send",
    "summary": "Send welcome email to new user"
  },
  "approver": "alice",
  "expires_at": "2026-02-12T09:25:00Z",
  "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_xyz789",
  "timestamp": "2026-02-12T09:20:00Z"
}
```

**Request Headers:**

| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes | Always `application/json` |
| `User-Agent` | Yes | Service identifier, format: `{ServiceName}-PermissionSlip/{ServiceVersion}`, e.g., `ExampleService-PermissionSlip/1.0` |
| `X-Permission-Slip-Version` | Yes | Protocol version (currently `v1`, matching API endpoint versioning scheme) |
| `X-Permission-Slip-Webhook-Signature` | Yes | HMAC signature for verification; uses Unix timestamp (seconds) - see [Signature Verification](#signature-verification) |
| `X-Permission-Slip-Delivery` | Yes | Unique delivery ID (UUIDv4 format per RFC 4122) for idempotency |

**Note:** Standard HTTP headers such as `Content-Length` and `Host` MUST be included as required by HTTP/1.1 specifications (RFC 7230).

### Webhook Response

The webhook receiver SHOULD respond with a 2xx status code to acknowledge receipt.

**Success Response:**

```http
HTTP/1.1 200 OK
```

**Error Response:**

If the webhook receiver returns a non-2xx status code, the service SHOULD retry delivery with exponential backoff.

**Retry Policy (Recommended):**

- HTTP timeout: 10 seconds per request (balances reliability with timely retries)
- Retry up to 3 times on network errors or 5xx responses
- Do NOT retry on 4xx responses except 429 Too Many Requests (see below)
- Exponential backoff: 1s, 2s, 4s
- **429 Too Many Requests Handling:**
  - Check for `Retry-After` header in response (value in seconds or HTTP date)
  - If `Retry-After` is present and retry time is within approval TTL, wait specified duration before retry
  - If `Retry-After` is missing, treat as 5xx and use exponential backoff
  - If `Retry-After` exceeds approval TTL, do not retry (approval will expire)
  - Calculation: `if (current_time + retry_after_seconds) > expires_at` then do not retry
  - Example: Approval expires at 09:25:00 UTC, current time is 09:22:00 UTC, `Retry-After: 240` (4 minutes) → retry would complete at 09:26:00 UTC, which exceeds expiration → do not retry
- **Before each retry:** Check if approval has expired (`expires_at` < current time); do not retry if expired
- **TTL edge case:** If the remaining TTL is less than the next retry backoff duration, services SHOULD skip that retry attempt and move to alternative notification channels (if available) or log the delivery failure
- Log all delivery attempts (success and failure) for audit purposes
- If all retries fail within the TTL window, services SHOULD notify the approver through alternative channels (if configured) or log the failure for manual review

**Concurrent Delivery:**

When multiple notification channels are configured (e.g., webhook + iOS push), services SHOULD deliver notifications concurrently to minimize latency. Sequential delivery wastes precious time in the 5-minute TTL window.

**Per-Channel Retry:**

When using concurrent delivery with multiple channels, services SHOULD track delivery status per channel and retry only failed channels, not all channels. For example, if webhook succeeds but iOS push fails, only retry iOS push.

**Per-Channel Retry Counters:**

When using concurrent delivery, services SHOULD implement per-channel retry counters. Each channel has independent retry limits (up to 3 attempts) and backoff timers. This maximizes delivery success probability within the TTL window.

**Error Responses:**

Webhook receivers MAY return structured error responses for debugging:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests, try again in 60 seconds",
  "retry_after": 60
}
```

Services SHOULD log these error responses but MUST NOT expose them to attackers.

### Signature Verification

Services MUST sign webhook payloads to allow receivers to verify authenticity.

**Signature Algorithm:** HMAC-SHA256

**Signature Format:**

```
X-Permission-Slip-Webhook-Signature: t=<timestamp>,v1=<signature>
```

Where:
- `t`: Unix timestamp in seconds (not milliseconds) when signature was generated
- `v1`: HMAC-SHA256 signature (hex-encoded, lowercase)

**Note:** The signature uses Unix timestamps (seconds), while payload timestamps use ISO 8601 format. This is intentional - Unix timestamps are standard for HMAC-based webhook signatures.

**Timestamp Synchronization:** The Unix timestamp `t` in the signature header represents when the signature was computed and MAY differ slightly from the `timestamp` field in the JSON payload (which represents notification creation time). Receivers MUST validate the signature timestamp for replay protection but SHOULD NOT require exact matching with the payload timestamp.

**Signature Input:**

```
{timestamp}.{json_payload}
```

Where:
- `{timestamp}`: Unix timestamp from signature header
- `{json_payload}`: Raw JSON payload (as sent in request body)

**Computing the Signature:**

```
signature = HMAC_SHA256(secret_bytes, "{timestamp}.{json_payload}")
```

**Note on Secret Encoding:**
- When represented as text (for example, in configuration files or environment variables), the shared secret MUST be encoded using standard base64 (for readability and storage)
- When computing or verifying HMAC signatures, the base64-encoded secret string MUST be decoded to the original raw bytes first
- Example: A 32-byte secret base64-encoded as a string must be decoded before use: `base64.StdEncoding.DecodeString(secret)`

**Verification Steps:**

1. Extract `t` (timestamp) and `v1` (signature) from `X-Permission-Slip-Webhook-Signature` header
2. Verify timestamp is not older than 5 minutes and not more than 60 seconds in the future to prevent replay attacks - reject if `timestamp < now - 300` or `timestamp > now + 60`
3. Construct signature input: `"{t}.{raw_json_body}"`
4. Compute expected signature: `HMAC_SHA256(webhook_secret, signature_input)`
5. Compare expected signature with `v1` using constant-time comparison
6. Reject if signatures do not match

**Go Example (Webhook Receiver):**

```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func verifyWebhookSignature(r *http.Request, secret string) ([]byte, error) {
	// Read signature header
	sigHeader := r.Header.Get("X-Permission-Slip-Webhook-Signature")
	if sigHeader == "" {
		return nil, fmt.Errorf("missing signature header")
	}
	
	// Parse signature header: t=<timestamp>,v1=<signature>
	parts := strings.Split(sigHeader, ",")
	var timestamp int64
	var signature string
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		switch key {
		case "t":
			var err error
			timestamp, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
		case "v1":
			signature = value
		}
	}
	
	if timestamp == 0 || signature == "" {
		return nil, fmt.Errorf("invalid signature format")
	}
	
	// Verify timestamp (not older than 5 minutes, not more than 60s in future)
	now := time.Now().Unix()
	if timestamp < now-300 || timestamp > now+60 {
		return nil, fmt.Errorf("signature expired")
	}
	
	// Read raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	
	// Decode the shared secret from base64 before using it as the HMAC key
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("invalid shared secret encoding: %w", err)
	}
	
	// Compute expected signature
	signedPayload := fmt.Sprintf("%d.%s", timestamp, body)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedPayload))
	expectedMAC := mac.Sum(nil)
	
	// Decode provided hex signature and compare in constant time
	sigBytes, err := hex.DecodeString(signature)
	if err != nil || len(sigBytes) != len(expectedMAC) || !hmac.Equal(expectedMAC, sigBytes) {
		return nil, fmt.Errorf("signature mismatch")
	}
	
	// Return body bytes so caller can parse JSON
	return body, nil
}

// Example webhook handler implementation
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Step 1: Verify signature and read body
	// Note: Return generic errors to prevent timing attacks that reveal valid signatures
	// webhookSecret is loaded from configuration (base64-encoded string)
	webhookSecret := getWebhookSecretFromConfig() // Implement based on your config system
	body, err := verifyWebhookSignature(r, webhookSecret)
	if err != nil {
		// Log detailed error internally for debugging
		log.Printf("Webhook signature verification failed: %v", err)
		// Return generic error to client (timing-attack safe)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	
	// Step 2: Parse notification payload
	var notification struct {
		Type         string `json:"type"`
		ApprovalID   string `json:"approval_id,omitempty"`
		RequestID    string `json:"request_id,omitempty"`
		AgentID      string `json:"agent_id,omitempty"`
		AgentName    string `json:"agent_name,omitempty"`
		ApprovalURL  string `json:"approval_url,omitempty"`
		RegistrationURL string `json:"registration_url,omitempty"`
		ExpiresAt    string `json:"expires_at,omitempty"`
		Timestamp    string `json:"timestamp"`
	}
	
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if notification.Type == "" || notification.Timestamp == "" {
		log.Printf("Missing required fields in payload")
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	
	// Type-specific validation of required fields
	switch notification.Type {
	case "agent_registration":
		if notification.RequestID == "" || notification.AgentID == "" || notification.RegistrationURL == "" {
			log.Printf("Missing required registration fields")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
	case "approval_request":
		if notification.ApprovalID == "" || notification.AgentID == "" || notification.ApprovalURL == "" {
			log.Printf("Missing required approval fields")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
	case "test":
		// Test notifications have minimal requirements
	default:
		// Unknown type will be caught later
	}
	
	// Check if notification has expired
	if notification.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, notification.ExpiresAt)
		if err != nil {
			log.Printf("Invalid expires_at format: %v", err)
			http.Error(w, "Invalid timestamp", http.StatusBadRequest)
			return
		}
		
		if time.Now().After(expiresAt) {
			log.Printf("Notification already expired: %s", notification.ExpiresAt)
			// Still return 200 to prevent retries, but don't process
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	
	// Step 3: Check for duplicate delivery (idempotency)
	deliveryID := r.Header.Get("X-Permission-Slip-Delivery")
	if deliveryID != "" && isDuplicate(deliveryID) {
		log.Printf("Duplicate delivery detected: %s", deliveryID)
		w.WriteHeader(http.StatusOK) // Acknowledge without reprocessing
		return
	}
	
	// Step 4: Process notification based on type
	switch notification.Type {
	case "agent_registration":
		handleAgentRegistration(notification)
	case "approval_request":
		handleApprovalRequest(notification)
	case "test":
		log.Printf("Test notification received successfully")
	default:
		log.Printf("Unknown notification type: %s", notification.Type)
		http.Error(w, "Unknown notification type", http.StatusBadRequest)
		return
	}
	
	// Step 5: Store delivery ID to prevent duplicate processing
	if deliveryID != "" {
		storeDeliveryID(deliveryID)
	}
	
	w.WriteHeader(http.StatusOK)
}

// Helper function stubs (implement based on your storage mechanism)
func getWebhookSecretFromConfig() string {
	// Load webhook secret from configuration, environment, or secure storage
	// Must return base64-encoded secret string
	return "" // Placeholder - implement based on your config system
}

func isDuplicate(deliveryID string) bool {
	// Check if deliveryID exists in cache/database (last 24h)
	return false // Placeholder
}

func storeDeliveryID(deliveryID string) {
	// Store deliveryID with TTL of 24 hours
}

func handleAgentRegistration(notification interface{}) {
	// Process agent registration notification
	// Example: Send email, push to mobile app, update UI
}

func handleApprovalRequest(notification interface{}) {
	// Process approval request notification
	// Example: Send alert, update pending approvals list
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
```

### Idempotency

Webhook receivers MAY receive duplicate notifications due to retries or network issues.

**Idempotency Handling:**

- Use the `X-Permission-Slip-Delivery` header (unique delivery ID) to detect duplicates
- Store delivery IDs for at least 24 hours
- If a duplicate is detected, return `200 OK` without reprocessing

**Implementation Note:** Delivery ID storage MUST be atomic and SHOULD use distributed locking or compare-and-set operations if webhooks are delivered from multiple worker instances. This prevents race conditions where retries arrive before the first delivery is marked complete.

### Testing Webhooks

Services SHOULD provide a way for approvers to test webhook configuration.

**Test Notification Payload:**

```json
{
  "type": "test",
  "message": "This is a test notification from Permission Slip",
  "timestamp": "2026-02-12T09:20:00Z",
  "webhook_id": "wh_abc123",
  "test_id": "test_def456",
  "service_name": "ExampleService",
  "approver": "alice"
}
```

**Note:** The notification type `"test"` is reserved for testing purposes. Services implementing webhook testing SHOULD use the `"test"` notification type as defined above. The `"test"` type MUST NOT be used for production notifications.

**Test Behavior:**
- Uses same signature verification as real notifications
- Includes `X-Permission-Slip-Delivery` header for idempotency testing
- Services SHOULD verify webhook endpoint responds successfully before allowing approver to save configuration
- Returns success/failure to approver in UI (including response time)
- Does not create actual approval requests

**Recommended UI:**
```
[Test Webhook] button → sends test notification → shows result:
  ✅ Test successful (received 200 OK in 234ms)
  ❌ Test failed (connection timeout)
```

---

## Mobile Push Notifications

Services MAY support mobile push notifications via Apple Push Notification Service (APNs) for iOS and Firebase Cloud Messaging (FCM) for Android.

### Device Registration

Approvers register mobile device tokens in their service account settings.

**Configuration Options:**

- **Platform:** iOS (APNs) or Android (FCM)
- **Device Token:** Push notification token from the mobile app
- **Events:** Which notification types to receive (registration, approvals, or both)

### Push Notification Payload

Mobile push notifications MUST include only minimal data to alert the approver. Full approval details are fetched via the approval URL after the approver opens the app.

**Size Limits:**
- iOS APNs: Maximum 4KB payload
- Android FCM: Maximum 4KB payload (notification + data)
- Keep payloads well under these limits for reliability

**Priority Settings:**
- iOS: Use `apns-priority: 10` (immediate) for all approval notifications
- Android: Use `priority: high` for all approval notifications
- Rationale: Approval requests are time-sensitive (5-minute TTL)

#### Agent Registration Examples

> **Note:** Since registration is user-initiated ([ADR-005](../adr/005-user-initiated-registration.md)), push notifications for registration events are **optional and informational**. The primary delivery mechanism for registration status is the dashboard. These examples show optional push notifications that services MAY send when an invited agent completes registration.

**iOS (APNs) Payload:**

```json
{
  "aps": {
    "alert": {
      "title": "Agent Registered",
      "body": "My AI Assistant has been registered with your account"
    },
    "sound": "default"
  },
  "permission_slip": {
    "type": "agent_registration_complete",
    "agent_id": "agent_x7K9mP4n..."
  }
}
```

**Android (FCM) Payload:**

```json
{
  "notification": {
    "title": "Agent Registered",
    "body": "My AI Assistant has been registered with your account"
  },
  "data": {
    "type": "agent_registration_complete",
    "agent_id": "agent_x7K9mP4n..."
  },
  "priority": "normal"
}
```

#### Approval Request Examples

**iOS (APNs) Payload:**

```json
{
  "aps": {
    "alert": {
      "title": "Approve Action",
      "body": "My AI Assistant: Send welcome email to new user"
    },
    "sound": "default",
    "badge": 1
  },
  "permission_slip": {
    "type": "approval_request",
    "approval_id": "appr_xyz789",
    "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_xyz789"
  }
}
```

**Android (FCM) Payload:**

```json
{
  "notification": {
    "title": "Approve Action",
    "body": "My AI Assistant: Send welcome email to new user"
  },
  "data": {
    "type": "approval_request",
    "approval_id": "appr_xyz789",
    "approval_url": "https://app.permissionslip.dev/permission-slip/approve/appr_xyz789"
  },
  "priority": "high"
}
```

**Security:** Push notification payloads MUST NOT include:
- Sensitive action parameters
- Confirmation codes
- Authentication tokens
- User data or email content

### Push Notification Best Practices

1. **Keep payloads minimal:** Only include IDs and URLs
2. **Use deep links:** Enable direct navigation to approval UI
3. **Set appropriate priority:** Use high priority for time-sensitive approvals
4. **Handle expiration:** Clear expired notifications from notification center
5. **Batch when possible:** If multiple approvals arrive quickly, services MAY batch them
6. **Badge count management (iOS):**
   - Increment badge for each new pending action
   - Decrement when actions are approved, denied, or expired
   - Services SHOULD track per-device badge counts
   - Set badge to 0 when no actions are pending review
7. **Notification grouping:**
   - If multiple actions arrive within 30 seconds, services SHOULD group them
   - Example: "3 actions pending from My AI Assistant" instead of 3 separate notifications
   - Helps prevent notification fatigue
8. **Sound customization:**
   - Services MAY allow approvers to configure different sounds for different risk levels
   - Example: Critical sound for `high` risk, default for `low` risk

---

## Approver Preferences

Services MUST allow approvers to configure notification preferences in their account settings.

### Required Configuration Options

**Notification Channels:**
- Webhook URL (with secret)
- Mobile device tokens (iOS and/or Android)

**Event Filters:**
- Agent registration notifications (on/off)
- Approval request notifications (on/off)
- Per-agent notification settings (optional)

**Delivery Settings:**
- All enabled notification channels (webhook, iOS, Android) receive notifications concurrently as described in [Delivery Mechanisms](#delivery-mechanisms)
- Services MAY allow approvers to label channels for organizational purposes (e.g., "work phone", "personal laptop") but these labels do NOT affect delivery behavior
- Do Not Disturb schedule (optional): Specifies time windows when push notifications are suppressed
  - Only affects push notifications (webhooks are not suppressed)
  - Approvals that arrive during DND are NOT queued
  - Approvers are responsible for checking pending approvals manually
  - Example: DND from 11:00 PM to 7:00 AM local time

### Example Configuration UI

```
Notification Preferences
========================

Channels (all enabled channels receive notifications concurrently):
[x] Webhook
  URL: https://notifications.permissionslip.dev/permission-slip
  Secret: wh_sec_****************
  [Test Webhook]

[x] iOS Push Notifications
  Device: John's iPhone
  Token: apns_****************
  [Remove Device]

[ ] Android Push Notifications
  [Add Android Device]

Events:
[x] Agent Registration Requests
[x] Approval Requests
```

---

## Deep Linking

Notification recipients MUST be able to quickly navigate from notification to approval UI.

### Approval URL Format

Services define their own approval URL structure. The URL MUST:
- Be HTTPS (secure)
- Include approval ID or authentication token
- Open directly to approval UI (not a landing page)
- Work on mobile and desktop

**Recommended URL Patterns:**

**For registration:**
```
https://app.permissionslip.dev/permission-slip/approve?token=<jwt_token>
```

**For approval requests:**
```
https://app.permissionslip.dev/permission-slip/approve/<approval_id>
```

**URL Security:**

Approval URLs SHOULD use short-lived, single-use tokens to prevent hijacking:
- **Registration URLs:** Use JWT tokens whose lifetime MUST NOT exceed the registration request's `expires_at`, and SHOULD be as short as practical (for example, ~15 minutes) when combined with session-based auth or a refresh mechanism
- **Approval URLs:** Use single-use approval IDs that expire with the approval (5+ minutes)
- **Session-based auth / refresh:** After initial URL access, establish an authenticated session or equivalent mechanism so that, if the URL token expires while the registration remains valid, the approver can still view and act on the request without needing a new notification link
- **Token rotation:** If the same URL is accessed multiple times, services SHOULD consider rotating tokens and MUST ensure that expired or used tokens cannot be replayed

Services MUST ensure that intercepting a notification URL does not grant long-term access to the approver's account.

### Token Security Requirements

Approval URLs and webhook secrets are sensitive credentials that require explicit security controls:

**Webhook Secrets:**
- MUST be at least 32 bytes of cryptographically secure random data
- MUST be stored encrypted at rest in the service database
- MUST use standard Base64 encoding for transmission and storage (decode to raw bytes before HMAC operations)
- SHOULD be displayed in full only once during initial generation
- SHOULD be masked in UI after saving (e.g., `wh_sec_****************`)
- Services SHOULD provide secret rotation without downtime (accept both old and new secrets during rotation window)

**Approval URL Tokens:**
- MUST have expiration matching or shorter than approval TTL
- SHOULD be single-use for registration URLs (invalidate after first successful access)
- MAY allow multiple accesses for approval URLs within the TTL window (to support page refreshes)
- MUST be invalidated immediately when approval is completed or expired
- MUST use cryptographically secure random generation (not predictable sequences)

**Implementation Note:** Short-lived tokens reduce risk but do not eliminate it. Services SHOULD implement rate limiting on approval endpoints and monitor for suspicious access patterns.

### Universal Links / App Links

For mobile apps, services SHOULD support universal links (iOS) and App Links (Android) to enable direct app opening from notifications.

**iOS Universal Link:**
```
https://app.permissionslip.dev/permission-slip/approve/appr_xyz789
```

Associated domain: `app.permissionslip.dev`

**Android App Link:**
```
https://app.permissionslip.dev/permission-slip/approve/appr_xyz789
```

Intent filter matches `app.permissionslip.dev` with path `/permission-slip/approve/*`

**Fallback:** If mobile app is not installed, URL opens in mobile web browser.

### Custom URL Schemes (Optional)

Services MAY support custom URL schemes for mobile app deep linking.

**iOS:**
```
permissionslip://approve/appr_xyz789
```

**Android:**
```
permissionslip://approve/appr_xyz789
```

**Note:** Custom URL schemes are less reliable than universal links/app links and should be used as fallback only.

---

## Security Considerations

### Webhook Security

1. **Always use HTTPS:** Webhook URLs MUST use HTTPS to protect payloads in transit
2. **Validate webhook URLs:** Services MUST reject:
   - Non-HTTPS URLs (http://)
   - localhost and loopback addresses (127.0.0.1, ::1)
   - IPv4 private ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
   - IPv4 link-local: 169.254.0.0/16
   - IPv6 private ranges: fc00::/7 (Unique Local Addresses), fe80::/10 (Link-local)
   - Cloud metadata endpoints: 169.254.169.254 (AWS, GCP, Azure)
3. **SSRF Protection:**
   - Perform URL validation AFTER DNS resolution to prevent DNS rebinding
   - Do NOT follow HTTP redirects (or validate redirect targets with same rules)
   - Set maximum response size limit (e.g., 1MB) to prevent resource exhaustion
   - Verify webhook endpoints before saving configuration (send test notification)
4. **Verify signatures:** Receivers MUST verify HMAC signatures to prevent spoofing
5. **Check timestamps:** Reject signatures older than 5 minutes to prevent replay attacks
6. **Use constant-time comparison:** Prevent timing attacks when comparing signatures
7. **Store secrets securely:** Webhook secrets MUST be stored encrypted at rest; minimum 32 bytes of cryptographically secure random data
8. **Rotate secrets periodically:** Services SHOULD allow secret rotation without downtime
9. **Rate limiting:** Webhook receivers SHOULD implement rate limiting to prevent abuse (recommended: 100 requests per minute per approver)

### Push Notification Security

1. **Minimal payloads:** Do NOT include sensitive data in push notifications
2. **Fetch on demand:** Approver fetches full approval details via authenticated API
3. **Token management:** Protect device tokens as secrets
4. **Revocation:** Allow approvers to revoke device tokens
5. **Certificate security:** Protect APNs certificates and FCM API keys

### Privacy Considerations

1. **PII in notifications:** Avoid including personally identifiable information in notification payloads
2. **Consent:** Obtain approver consent before sending push notifications
3. **Audit logging:** Log notification delivery attempts for security auditing
4. **Data retention:** Do not store notification payloads longer than necessary

### Monitoring and Debugging

**Notification History:**

Services SHOULD provide a notification history view for approvers to:
- View recent notification delivery attempts (last 7-30 days)
- See delivery status (success, failed, retried)
- Debug webhook delivery issues
- Monitor for missed notifications

**Recommended History Fields:**
- Notification type (registration, approval)
- Timestamp
- Delivery method (webhook URL, iOS device name, Android device name)
- Status (delivered, failed, expired)
- Response code (for webhooks)
- Retry count

**Example UI:**
```
Notification History
====================================
Feb 12, 09:20 AM EST  Approval Request   Webhook (notifications.permissionslip.dev)  [Delivered] 200
Feb 12, 09:15 AM EST  Agent Registration iOS (John's iPhone)                  [Delivered]
Feb 12, 09:10 AM EST  Approval Request   Webhook (notifications.permissionslip.dev)  [Failed] timeout, 3 retries
```

**Note:** Timestamps SHOULD be displayed in approver's local timezone with timezone indicator. Use accessible status indicators (not just emoji).

**Failure Alerts:**

Services SHOULD alert approvers when:
- Webhook delivery fails consistently (e.g., 3+ failures in a row)
- No successful delivery method is configured
- Webhook endpoint returns 410 Gone (endpoint permanently removed)

**Alerting Mechanism:** When all configured notification channels fail, services SHOULD use alternative out-of-band communication (e.g., email to approver's account email address, SMS if available, or in-app notification on next login). Services MUST provide a way for approvers to view pending approvals even when notifications fail (e.g., via a dashboard or API endpoint).

---

## Related Documentation

The following specifications are part of the Permission Slip protocol:

- [API Specification](api.md) - Registration and approval endpoints
- [Authentication](authentication.md) - Agent signature verification
- [Terminology](terminology.md) - Core protocol concepts

**Note:** Check the repository for the current status of related specifications.
