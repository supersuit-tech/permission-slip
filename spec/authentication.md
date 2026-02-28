# Authentication

This document specifies how agents establish and prove their identity in the Permission Slip protocol.

---

## Cryptographic Identity

Every agent in the Permission Slip protocol is identified by a cryptographic key pair. The agent's public key serves as its verifiable identity, while the private key is used to sign requests and prove authenticity.

### Key Pair Generation

**Algorithm Requirements:**

- **MUST support:** Ed25519 (RFC 8032)
- **MAY support:** ECDSA with P-256 curve (for legacy compatibility)

**Rationale:** Ed25519 is the recommended default for modern implementations. It offers:
- Simple, parameter-free API (fewer configuration mistakes)
- Fast signing and verification
- Small key size (32 bytes)
- Strong security guarantees

ECDSA P-256 support is optional for implementers who need compatibility with legacy systems or enterprise hardware security modules (HSMs).

**Generation Example (Ed25519):**

```bash
# Using ssh-keygen
ssh-keygen -t ed25519 -f agent_key -N "" -C "permission-slip-agent"

# Generates:
# agent_key       (private key)
# agent_key.pub   (public key)
```

**Go Example (Backend):**

```go
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	
	"golang.org/x/crypto/ssh"
)

// generateEd25519Keys creates a new Ed25519 key pair and returns the public key
// in OpenSSH format. For production use, store the private key securely using
// the pattern shown in the private key PEM encoding section below.
func generateEd25519Keys() error {
	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}
	
	// Format public key as OpenSSH format
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH public key: %w", err)
	}
	sshFormat := string(ssh.MarshalAuthorizedKey(sshPublicKey))
	
	fmt.Print(sshFormat) // Already includes newline
	// ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...
	
	// For PEM-encoded private key storage:
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	
	// Store privatePEM securely (see Key Storage Requirements section)
	_ = privatePEM
	
	return nil
}
```

```javascript
// Using Node.js crypto module
const crypto = require('crypto');

// Generate key pair
const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519', {
  publicKeyEncoding: {
    type: 'spki',
    format: 'pem'
  },
  privateKeyEncoding: {
    type: 'pkcs8',
    format: 'pem'
  }
});

// Convert to OpenSSH format for public key
const sshKey = crypto.createPublicKey(publicKey).export({
  type: 'spki',
  format: 'der'
});

// For OpenSSH format, you'd typically use a library like ssh-keygen or ssh2
// This example shows the raw generation
```

---

### Key Storage Requirements

**Private Key Security:**

Private keys are the agent's identity credentials and MUST be protected from unauthorized access.

**Best Practices (Server-Side Agents):**

1. **File-based storage:**
   ```bash
   # Store private key with restricted permissions
   chmod 600 agent_key
   # Only the agent process owner can read
   ```

2. **Environment-based storage:**
   ```bash
   # Export private key as environment variable
   export AGENT_PRIVATE_KEY="$(cat agent_key)"
   ```

3. **Secrets management systems:**
   - AWS Secrets Manager
   - HashiCorp Vault
   - Kubernetes Secrets
   - Google Secret Manager

4. **Encryption at rest (recommended):**
   ```go
   // Example: Encrypt private key with passphrase using modern encryption
   package main
   
   import (
   	"crypto/rand"
   	"encoding/base64"
   	"fmt"
   	"io"
   	"os"
   	
   	"golang.org/x/crypto/argon2"
   	"golang.org/x/crypto/nacl/secretbox"
   )
   
   func encryptPrivateKey(privateKeyPEM []byte) (string, error) {
   	// IMPORTANT: Passphrase MUST come from a secure source:
   	// - Environment variable
   	// - Secrets management system (Vault, AWS Secrets Manager, etc.)
   	// - Secure configuration (not hardcoded)
   	// NEVER hardcode passphrases in source code or commit them to version control
   	passphrase := []byte(os.Getenv("AGENT_KEY_PASSPHRASE"))
   	
   	// Generate a random salt for key derivation
   	salt := make([]byte, 32)
   	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
   		return "", err
   	}
   	
   	// Derive a 32-byte key using Argon2
   	key := argon2.IDKey(passphrase, salt, 1, 64*1024, 4, 32)
   	
   	// Generate a random nonce
   	var nonce [24]byte
   	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
   		return "", err
   	}
   	
   	// Encrypt the private key using NaCl secretbox
   	var keyArray [32]byte
   	copy(keyArray[:], key)
   	encrypted := secretbox.Seal(nonce[:], privateKeyPEM, &nonce, &keyArray)
   	
   	// Prepend salt to encrypted data and base64 encode
   	result := append(salt, encrypted...)
   	return base64.StdEncoding.EncodeToString(result), nil
   }
   
   // Note: This uses modern encryption (NaCl secretbox + Argon2 KDF).
   // For production, consider using age (filippo.io/age) for file encryption.
   ```

**Note:** Key storage security is implementation-specific and not mandated by this protocol. Choose the approach that fits your threat model and operational environment.

---

### Public Key Format

**Primary Format: OpenSSH**

Public keys MUST be representable in OpenSSH format (the format produced by `ssh-keygen`).

**Format Structure:**
```
<algorithm> <base64-encoded-key> [optional-comment]
```

**Example (Ed25519):**
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFPZj8V7K2mT9Xw4nL5rQ1pY3vN8cM6hU0oB4eW2sR7k permission-slip-agent
```

**Rationale:**
- Familiar to developers (same as GitHub SSH keys)
- Single-line format, easy to transmit
- Widely supported tooling (`ssh-keygen`, `ssh-add`, etc.)
- Compatible with existing infrastructure

**Alternative Formats (Optional):**

Implementations MAY also support:
- **PEM format** (for compatibility with TLS/PKI tools)
- **JWK format** (JSON Web Key, for web-native applications)

However, OpenSSH format is the canonical representation for interoperability.

---

### Agent ID Derivation

An **Agent ID** is a stable, unique identifier derived from the agent's public key.

**Derivation Algorithm:**

1. Obtain the canonical public key bytes (algorithm-specific, see below)
2. Compute SHA-256 hash of the public key bytes
3. Encode the hash using base64url (RFC 4648 §5, no padding)
4. Prefix the result with `agent_`

**Formula:**
```
agent_id = "agent_" + base64url(SHA256(public_key_bytes))
```

**Canonical Public Key Byte Representation:**

For interoperability, implementations MUST use the following byte representations when deriving Agent IDs:

**Ed25519:**
- **Bytes to hash:** Raw 32-byte public key (RFC 8032 §5.1.5)
- **From OpenSSH format:** Decode the base64 portion, skip the format header, extract the 32-byte key
- **From PEM format:** Decode the SubjectPublicKeyInfo, extract the raw 32-byte key from the BIT STRING
- **From JWK format:** Base64url-decode the `x` parameter (32 bytes)

**ECDSA P-256:**
- **Bytes to hash:** SEC1/X9.62 uncompressed point format (65 bytes: `0x04 || x || y`)
  - First byte: `0x04` (uncompressed point indicator)
  - Next 32 bytes: x-coordinate (big-endian)
  - Final 32 bytes: y-coordinate (big-endian)
- **From OpenSSH format:** Decode the base64 portion, skip the format header, extract the curve point
- **From PEM format:** Decode the SubjectPublicKeyInfo, extract the uncompressed point from the BIT STRING
- **From JWK format:** Construct `0x04 || base64url_decode(x) || base64url_decode(y)`

**Rationale:** Using a single canonical byte representation per algorithm ensures that the same public key always produces the same Agent ID, regardless of how the key is encoded or transmitted.

**Example (Ed25519 - Go):**

```go
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

func deriveAgentID(publicKey ed25519.PublicKey) string {
	// publicKey is already the canonical 32-byte raw public key for Ed25519
	
	// Hash the canonical public key bytes
	keyHash := sha256.Sum256(publicKey)
	
	// Base64url encode (no padding)
	agentIDSuffix := base64.URLEncoding.EncodeToString(keyHash[:])
	agentIDSuffix = strings.TrimRight(agentIDSuffix, "=")
	
	// Construct Agent ID
	agentID := fmt.Sprintf("agent_%s", agentIDSuffix)
	
	fmt.Println(agentID)
	// agent_x7K9mP4nQ8rT2vW5yZ1aC3bD6eF9gH0jK4lM7nP0qR3s
	
	return agentID
}
```

```javascript
const crypto = require('crypto');

// For Ed25519: extract the raw 32-byte public key
// If you have an OpenSSH format key, you need to parse and extract the raw bytes
// This example assumes you already have the raw 32-byte Ed25519 public key
const publicKeyBytes = Buffer.from(rawEd25519PublicKey); // 32 bytes

// Hash the canonical public key bytes
const hash = crypto.createHash('sha256').update(publicKeyBytes).digest();

// Base64url encode (no padding)
const agentIdSuffix = hash.toString('base64url').replace(/=/g, '');

// Construct Agent ID
const agentId = `agent_${agentIdSuffix}`;

console.log(agentId);
// agent_x7K9mP4nQ8rT2vW5yZ1aC3bD6eF9gH0jK4lM7nP0qR3s
```

**Example (ECDSA P-256 - Go):**

```go
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

func deriveAgentIDFromECDSA(publicKey *ecdsa.PublicKey) string {
	// Extract the uncompressed point (65 bytes: 0x04 || x || y)
	publicBytes := elliptic.Marshal(elliptic.P256(), publicKey.X, publicKey.Y)
	// publicBytes[0] == 0x04, len(publicBytes) == 65
	
	// Hash the canonical public key bytes
	keyHash := sha256.Sum256(publicBytes)
	
	// Base64url encode (no padding)
	agentIDSuffix := base64.URLEncoding.EncodeToString(keyHash[:])
	agentIDSuffix = strings.TrimRight(agentIDSuffix, "=")
	
	// Construct Agent ID
	agentID := fmt.Sprintf("agent_%s", agentIDSuffix)
	
	fmt.Println(agentID)
	// agent_a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8s9T0u1V2w3X
	
	return agentID
}
```

**Properties:**

- **Deterministic:** The same public key always produces the same Agent ID
- **Unique:** Different public keys produce different IDs (collision-resistant via SHA-256)
- **Self-verifiable:** Anyone with the public key can recompute the Agent ID to verify authenticity
- **Compact:** Fixed length (49 characters including prefix)

**Verification:**

To verify an Agent ID matches a public key:

```go
func verifyAgentID(publicKey interface{}, claimedAgentID string) bool {
	// Derive the expected Agent ID
	expectedID := deriveAgentID(publicKey)
	
	// Compare
	return expectedID == claimedAgentID
}
```

This check ensures that a claimed Agent ID is consistent with a given public key; actual proof that a party controls the corresponding private key is provided separately via signature verification in the authentication protocol.

---

## Transport Security

This section specifies the transport-layer security requirements for all Permission Slip protocol communications. Transport security protects against eavesdropping, tampering, and man-in-the-middle attacks.

### TLS Requirements

**All Permission Slip protocol endpoints MUST be served over HTTPS with TLS.**

**Minimum TLS Version:**

- **RECOMMENDED:** TLS 1.3 (RFC 8446)
- **ACCEPTABLE:** TLS 1.2 (RFC 5246) as a fallback for legacy compatibility

**Rationale:**
- **TLS 1.3** is the modern standard, offering improved security (removes weak cipher suites, simplifies handshake) and performance (faster connection establishment)
- **TLS 1.2** remains acceptable for services supporting older clients or enterprise environments with legacy constraints
- **TLS 1.1 and earlier** MUST NOT be used (obsolete, known vulnerabilities)

**Configuration Best Practices:**

Services SHOULD:

1. **Prefer TLS 1.3** and configure servers to negotiate TLS 1.3 when clients support it
2. **Disable weak cipher suites** (e.g., RC4, DES, 3DES, CBC mode ciphers)
3. **Enable forward secrecy** (use ECDHE or DHE key exchange)
4. **Use strong cipher suites**:
   - For TLS 1.3: `TLS_AES_128_GCM_SHA256`, `TLS_AES_256_GCM_SHA384`, `TLS_CHACHA20_POLY1305_SHA256`
   - For TLS 1.2: `ECDHE-RSA-AES128-GCM-SHA256`, `ECDHE-RSA-AES256-GCM-SHA384`

**Implementation Example (Nginx):**

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    # TLS version
    ssl_protocols TLSv1.3 TLSv1.2;
    
    # Cipher suites (TLS 1.2)
    ssl_ciphers 'ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers on;
    
    # TLS 1.3 cipher suites (default secure, no config needed)
    
    # Certificate
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
}
```

**Implementation Example (Node.js with Express):**

```javascript
const https = require('https');
const express = require('express');
const fs = require('fs');

const app = express();

const httpsOptions = {
  key: fs.readFileSync('/path/to/key.pem'),
  cert: fs.readFileSync('/path/to/cert.pem'),
  
  // TLS configuration
  minVersion: 'TLSv1.2',  // Minimum version
  maxVersion: 'TLSv1.3',  // Prefer TLS 1.3
  
  // Cipher suites for TLS 1.2
  ciphers: [
    'ECDHE-RSA-AES128-GCM-SHA256',
    'ECDHE-RSA-AES256-GCM-SHA384'
  ].join(':'),
  
  honorCipherOrder: true
};

https.createServer(httpsOptions, app).listen(443);
```

**Implementation Example (Go with net/http):**

```go
package main

import (
	"crypto/tls"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	// Add your handlers here
	
	server := &http.Server{
		Addr:    ":443",
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
			PreferServerCipherSuites: true,
		},
	}
	
	log.Fatal(server.ListenAndServeTLS("/path/to/cert.pem", "/path/to/key.pem"))
}
```

**Agent Implementation:**

Agents SHOULD:

1. **Prefer TLS 1.3** if supported by their HTTP client library
2. **Validate server certificates** against trusted root CAs (do not disable certificate verification in production)
3. **Fail hard on TLS errors** (connection drops, invalid certificates) rather than falling back to insecure connections

Most modern HTTP client libraries (e.g., `requests` in Python, `axios` in Node.js, `reqwest` in Rust) handle TLS 1.2/1.3 negotiation automatically with secure defaults.

---

### Certificate Validation

**Services MUST use valid TLS certificates from a trusted Certificate Authority (CA).**

**Certificate Requirements:**

- **Issued by a trusted CA** (e.g., Let's Encrypt, DigiCert, AWS Certificate Manager)
- **Valid for the service domain** (matches `Host` header or service discovery domain)
- **Not expired** (valid `notBefore` and `notAfter` dates)
- **Complete certificate chain** provided (intermediate CA certificates included)

**Agent Validation Requirements:**

Agents MUST:

1. **Verify the certificate chain** up to a trusted root CA
2. **Check certificate expiration** (reject expired certificates)
3. **Validate hostname** matches the request URL (prevents domain mismatch attacks)
4. **Reject self-signed certificates** in production environments

**Exception:** Self-signed certificates MAY be used in local development or internal testing environments, but MUST NOT be used in production deployments.

**Implementation Example (JavaScript - Frontend):**

```javascript
const axios = require('axios');

// Default: certificate verification enabled
const response = await axios.post(
    'https://api.example.com/permission-slip/v1/approvals/request',
    payload,
    { headers }
    // Certificate verification is enabled by default in Node.js
);

// To use a custom CA bundle (e.g., for internal PKI):
const https = require('https');
const fs = require('fs');

const agent = new https.Agent({
  ca: fs.readFileSync('/path/to/ca-bundle.crt')
});

const response = await axios.post(
    'https://internal-api.example.com/permission-slip/v1/approvals/request',
    payload,
    { headers, httpsAgent: agent }
);

// WARNING: Never do this in production!
// process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';  // INSECURE
```

---

### Certificate Pinning (Optional)

**Certificate pinning** is an optional security enhancement where agents validate that the service's certificate matches a known, expected certificate or public key. This protects against compromised CAs or man-in-the-middle attacks using fraudulent certificates.

**When to Use Certificate Pinning:**

Certificate pinning is RECOMMENDED for:
- **High-security environments** (financial services, healthcare, government)
- **Agents with long-lived deployments** (mobile apps, embedded systems)
- **Services with static infrastructure** (minimal certificate rotation)

Certificate pinning is NOT RECOMMENDED for:
- **Agents with frequent updates** (web apps, short-lived scripts)
- **Services with frequent certificate rotation** (Let's Encrypt 90-day certs without automation)
- **Multi-tenant environments** where service domains change frequently

**Pinning Strategies:**

**1. Pin the leaf certificate (service certificate):**

**Pros:** Most specific, highest security  
**Cons:** Must update pin on every certificate renewal (typically 90 days for Let's Encrypt)

**JavaScript Example (Agent/Client-Side):**

```javascript
// Example: Pin the SHA-256 hash of the service's certificate
// This is client-side validation performed by the agent before making requests to the service
const crypto = require('crypto');
const https = require('https');
const tls = require('tls');

const EXPECTED_CERT_HASH = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

function getCertHash(hostname) {
  return new Promise((resolve, reject) => {
    const socket = tls.connect(443, hostname, () => {
      const cert = socket.getPeerCertificate();
      const certDER = cert.raw;
      const hash = crypto.createHash('sha256').update(certDER).digest('hex');
      socket.end();
      resolve(hash);
    });
    socket.on('error', reject);
  });
}

// Before making requests, verify the certificate hash
const actualHash = await getCertHash('api.example.com');
if (actualHash !== EXPECTED_CERT_HASH) {
  throw new Error("Certificate pinning validation failed!");
}

// Proceed with request
const axios = require('axios');
const response = await axios.post('https://api.example.com/permission-slip/v1/approvals/request', payload);
```

**2. Pin the public key (SPKI hash):**

**Pros:** Survives certificate renewal (as long as the key pair doesn't change)  
**Cons:** Less common, requires extracting public key

**JavaScript Example (Agent/Client-Side):**

```javascript
// Example: Pin the SHA-256 hash of the Subject Public Key Info (SPKI)
// This is client-side validation performed by the agent before making requests to the service
const crypto = require('crypto');
const tls = require('tls');

const EXPECTED_SPKI_HASH = "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2";

function getSPKIHash(hostname) {
  return new Promise((resolve, reject) => {
    const socket = tls.connect(443, hostname, () => {
      const cert = socket.getPeerCertificate();
      // Get the public key info
      const spki = cert.pubkey;
      const hash = crypto.createHash('sha256').update(spki).digest('hex');
      socket.end();
      resolve(hash);
    });
    socket.on('error', reject);
  });
}

const actualHash = await getSPKIHash('api.example.com');
if (actualHash !== EXPECTED_SPKI_HASH) {
  throw new Error("Certificate pinning validation failed!");
}
```

**3. Pin the intermediate or root CA certificate:**

**Pros:** Most flexible, survives certificate and key rotation  
**Cons:** Least specific, doesn't protect against compromised service keys

This approach is useful when the service uses a specific internal CA or long-lived intermediate CA.

**Recommendation:**

For most implementations, **public key (SPKI) pinning** offers the best balance of security and operational flexibility. Services SHOULD document their pinning policy and provide the expected SPKI hash(es) in their developer documentation.

**Graceful Degradation:**

When implementing certificate pinning:

1. **Use backup pins** (pin both current and next certificate/key to allow rotation)
2. **Implement pin expiration** (re-fetch pins periodically from a secure source)
3. **Provide a pin update mechanism** (e.g., over-the-air updates for mobile apps)
4. **Monitor pinning failures** and alert on anomalies (could indicate attack or misconfiguration)

**Example with Backup Pin:**

```javascript
// Pin both current and next public key (during rotation period)
const EXPECTED_SPKI_HASHES = [
  "a1b2c3d4...",  // Current key
  "f9e8d7c6..."   // Next key (during rotation)
];

const actualHash = await getSPKIHash('api.example.com');
if (!EXPECTED_SPKI_HASHES.includes(actualHash)) {
  throw new Error("Certificate pinning validation failed!");
}
```

---

### Encryption Beyond TLS

**TLS is sufficient for Permission Slip protocol security. No additional encryption layer is required.**

**Rationale:**

The Permission Slip protocol relies on **TLS for confidentiality and integrity**:

- **Confidentiality:** TLS encrypts all data in transit (request bodies, headers, response payloads)
- **Integrity:** TLS ensures data is not tampered with during transmission
- **Authentication:** TLS certificate validation ensures the client is communicating with the legitimate service

**Additional encryption beyond TLS would:**
- Add complexity without meaningful security benefit
- Increase implementation burden for both services and agents
- Introduce potential for cryptographic implementation errors
- Degrade performance (double encryption overhead)

**What TLS Protects Against:**

- **Eavesdropping:** Third parties cannot read request/response data
- **Tampering:** Attackers cannot modify data in transit without detection
- **Man-in-the-middle attacks:** Certificate validation ensures agents connect to the legitimate service

**What TLS Does NOT Protect Against:**

- **Compromised endpoints:** If the service or agent is compromised, TLS cannot protect data at rest or in memory
- **Application-layer vulnerabilities:** SQL injection, XSS, etc. (outside the scope of transport security)

**End-to-End Security Model:**

Permission Slip's security model layers multiple protections:

1. **Transport layer (TLS):** Protects data in transit
2. **Request signing:** Proves agent identity and prevents request tampering
3. **Approval workflow:** Ensures human oversight for sensitive actions
4. **Single-use tokens:** Limits blast radius of compromised tokens

This layered approach provides defense in depth without requiring additional encryption schemes.

**When Additional Encryption Might Be Needed:**

In specialized scenarios (e.g., zero-knowledge architectures, end-to-end encrypted workflows), services MAY implement application-layer encryption for specific action parameters. However:

- This is outside the scope of the Permission Slip protocol specification
- Services implementing this MUST document it in their own API documentation
- The protocol itself (registration, approval requests, token exchange) remains protected by TLS alone

**Example of Application-Layer Encryption (Out of Scope):**

If a service wanted to implement end-to-end encryption for action parameters:

```json
{
  "action": {
    "type": "email.send",
    "version": "1",
    "parameters": {
      "encrypted_payload": "base64-encoded-encrypted-data",
      "encryption_algorithm": "AES-256-GCM",
      "recipient_key_id": "user_abc123"
    }
  }
}
```

This is a service-specific design choice and not a requirement of Permission Slip.

---

### Security Considerations

**For Services:**

1. **Use automated certificate renewal** (e.g., certbot for Let's Encrypt) to avoid expiration
2. **Monitor TLS configuration** with tools like SSL Labs' SSL Server Test
3. **Rotate certificates proactively** before expiration (30-day buffer recommended)
4. **Log TLS handshake failures** for security monitoring
5. **Disable HTTP (port 80)** or redirect to HTTPS immediately

**For Agents:**

1. **Never disable certificate verification** in production code
2. **Handle TLS errors gracefully** (log, alert, retry with exponential backoff)
3. **Update TLS libraries regularly** to patch vulnerabilities
4. **Test against services with expired/invalid certificates** in staging environments
5. **Document pinning strategy** if implemented

**For Both:**

- **Assume TLS alone is sufficient** for Permission Slip protocol security
- **Do not transmit sensitive data over non-HTTPS channels** (e.g., embedding tokens in HTTP query strings, using webhooks without TLS)
- **Use secure random number generation** for cryptographic operations (e.g., generating nonces, confirmation codes)

---

## Request Signing

All Permission Slip protocol API requests (registration, approval requests, token verification) MUST include a cryptographic signature proving the agent's identity. This section specifies how agents construct and sign requests, and how services verify those signatures.

### Signature Header Format

Agents MUST include the following header with every signed request:

```
X-Permission-Slip-Signature: agent_id="<agent_id>", algorithm="<algorithm>", timestamp="<unix_timestamp>", signature="<base64url_signature>"
```

**Header Components:**

- `agent_id` (string, required): The agent's identifier (derived from public key as specified in [Agent ID Derivation](#agent-id-derivation))
- `algorithm` (string, required): Signature algorithm used (`Ed25519` or `ECDSA-P256`)
- `timestamp` (string, required): Unix timestamp in seconds (integer as string, e.g., `"1707667200"`)
- `signature` (string, required): Base64url-encoded signature bytes (no padding per RFC 4648 §5)

**Signature Byte Format:**

The signature bytes (before base64url encoding) MUST be:

- **For Ed25519:** 64-byte raw Ed25519 signature as defined in RFC 8032
- **For ECDSA-P256:** 64-byte IEEE P1363 fixed-length encoding (`r || s`), where:
  - `r` and `s` are each 32-byte big-endian integers
  - `s` MUST be normalized to "low-S" form (`s ≤ n/2`) as specified in SEC 1
  - This differs from DER encoding used in some contexts; implementations must convert if necessary

**Example Header:**

```
X-Permission-Slip-Signature: agent_id="agent_x7K9mP4nQ8rT2vW5yZ1aC3bD6eF9gH0jK4lM7nP0qR3s", algorithm="Ed25519", timestamp="1707667200", signature="dGhpc19pc19hX2Jhc2U2NHVybF9lbmNvZGVkX3NpZ25hdHVyZV93aXRoX25vX3BhZGRpbmc"
```

---

### Canonical Request Format

The signature is computed over a **canonical representation** of the HTTP request. This ensures that both agent and service construct the same string to sign/verify.

**Canonical Request Structure:**

```
<METHOD>\n
<PATH>\n
<QUERY_STRING>\n
<TIMESTAMP>\n
<BODY_HASH>
```

Where:
- `<METHOD>`: HTTP method (uppercase, e.g., `POST`, `GET`)
- `<PATH>`: URL path without query string (e.g., `/v1/agents/register`)
- `<QUERY_STRING>`: Canonicalized query string (see rules below)
- `<TIMESTAMP>`: Unix timestamp from the signature header
- `<BODY_HASH>`: SHA-256 hash of the request body (see rules below)
- `\n`: Literal newline character (line feed, `0x0A`)

**Newlines:** The canonical request consists of exactly 5 lines separated by newline characters (`\n`). Each component is on its own line, with no trailing newline after the last component.

---

### Canonicalization Rules

Both agents and services MUST follow these exact rules to ensure signature compatibility.

#### 1. HTTP Method

**Rule:** HTTP verb in uppercase (e.g., `POST`, `GET`, `PUT`, `DELETE`).

**Example:**
- Input: `post`
- Canonical: `POST`

#### 2. Path

**Rule:** URL path component without query string, exactly as it appears in the request URL, including any path prefix from the `base_url` (e.g., `/permission-slip`).

**Example:**
- `base_url`: `https://api.example.com/permission-slip`
- Request URL: `https://api.example.com/permission-slip/v1/agents/register?foo=bar`
- Path (`<PATH>` in the canonical request): `/permission-slip/v1/agents/register` (query string excluded)

**Important:** Do NOT modify the path (no percent-encoding/decoding, no normalization). Use it exactly as transmitted.

#### 3. Query String

**Rule:** Canonical representation of URL query parameters.

**If no query parameters exist:**
- Use an empty string `""` (zero-length string, no characters)

**If query parameters exist:**

1. **Parse and percent-decode** the query string into name-value pairs:
   - Split on `&` to obtain pairs, and on the first `=` within each pair to separate name and value.
   - Percent-decode `%HH` sequences and `+` (if present) to obtain the decoded parameter names and values.
2. From the decoded parameter names and values, **percent-encode** each parameter name and value per **RFC 3986** to produce the canonical form:
   - **Unreserved characters** (MUST NOT be encoded): `A-Z a-z 0-9 - . _ ~`
   - **All other characters** (MUST be percent-encoded): Including space, `=`, `&`, `/`, etc.
   - **Spaces** MUST be encoded as `%20` (NOT `+`)
   - **Hex digits** in percent-encoding MUST be uppercase (e.g., `%2F` not `%2f`)
   - Implementations MUST normalize percent-encoding by re-encoding from the decoded form; they MUST NOT attempt to preserve original percent-encoded sequences verbatim.
3. **Sort** parameters:
   - Primary sort: by parameter name (lexicographic/byte order, case-sensitive)
   - Secondary sort: for parameters with the same name, sort by value (lexicographic/byte order)
4. **Join** as `name1=value1&name2=value2`

**Examples:**

```
Input:  ?z=2&a=1&a=3
Canonical: "a=1&a=3&z=2"

Input:  ?name=hello world&tag=cool!
Canonical: "name=hello%20world&tag=cool%21"

Input:  ?path=/api/v1
Canonical: "path=%2Fapi%2Fv1"

Input:  (no query string)
Canonical: ""
```

**Implementation Guidance:**

Most URL parsing libraries handle percent-decoding automatically when extracting parameters. After parsing, re-encode each name and value using an RFC 3986-compliant percent-encoding function, then sort and join.

**JavaScript/Node.js Example:**

```javascript
const { URLSearchParams } = require('url');

// Parse query string
const params = new URLSearchParams(originalQueryString);

// Sort and encode
const sortedParams = Array.from(params.entries()).sort((a, b) => {
  if (a[0] !== b[0]) return a[0].localeCompare(b[0]);
  return a[1].localeCompare(b[1]);
});

const canonicalQuery = sortedParams.map(([name, value]) => {
  // encodeURIComponent follows RFC 3986 except it encodes more chars (like !)
  // For strict RFC 3986, you may need a custom function
  return `${encodeURIComponent(name)}=${encodeURIComponent(value)}`;
}).join('&');
```

#### 4. Timestamp

**Rule:** Unix timestamp (seconds since epoch) as it appears in the `X-Permission-Slip-Signature` header.

**Format:** Integer seconds as a string (e.g., `"1707667200"`)

**Important:** The timestamp in the canonical request MUST exactly match the `timestamp` field in the signature header.

#### 5. Body Hash

**Rule:** SHA-256 hash of the request body, encoded as lowercase hexadecimal (64 characters).

**For JSON request bodies:**

JSON bodies MUST be canonicalized using **RFC 8785 JSON Canonicalization Scheme (JCS)** before hashing.

**Steps:**

1. **Canonicalize** the JSON using an RFC 8785-compliant library (see list below)
2. **Hash** the UTF-8 encoded bytes of the canonical JSON with SHA-256
3. **Encode** the hash as lowercase hexadecimal

**Why JCS is Required:**

Simple approaches like `json.dumps(sort_keys=True)` fail on edge cases:
- **Unicode:** Different escaping rules for non-ASCII characters
- **Numbers:** `1.0` vs `1`, exponential notation handling
- **Whitespace:** Varying space/newline insertion
- **Key ordering:** Unicode code point order vs. string sort order

RFC 8785 provides deterministic rules for all these cases, ensuring that agents and services compute identical hashes regardless of programming language or JSON library.

**RFC 8785 Library Recommendations:**

| Language | Library | Installation |
|----------|---------|--------------|
| JavaScript (Frontend) | `canonicalize` | `npm install canonicalize` |
| Go (Backend) | `cyberphone/json-canonicalization` | `go get github.com/cyberphone/json-canonicalization/go/json` |

**Go Example (Backend):**

```go
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	
	jsoncanon "github.com/cyberphone/json-canonicalization/go/json"
)

func computeBodyHash(requestBody interface{}) (string, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	
	// Canonicalize with RFC 8785
	canonicalJSON, err := jsoncanon.Transform(jsonBytes)
	if err != nil {
		return "", err
	}
	
	// Hash and encode
	hash := sha256.Sum256(canonicalJSON)
	bodyHash := hex.EncodeToString(hash[:])
	
	// NOTE: See test vector in api.md "Token Usage" section to verify your implementation.
	// Example hash: 3f3c4ee1ffb4e39da4a4923380a6fec415e4513db2af0354a76f432cafd82fb0
	return bodyHash, nil
}
```

**JavaScript Example (Frontend):**

```javascript
const crypto = require('crypto');
const canonicalize = require('canonicalize');

const requestBody = {
  agent_id: "agent_abc123",
  public_key: "ssh-ed25519 AAAAC3Nza...",
  approver: "alice"
};

// Canonicalize with RFC 8785
const canonicalJson = canonicalize(requestBody);  // Returns string

// Hash and encode
const bodyHash = crypto.createHash('sha256')
  .update(canonicalJson, 'utf8')
  .digest('hex');

// NOTE: See test vector in api.md "Token Usage" section to verify your implementation.

console.log(bodyHash);
// 3f3c4ee1ffb4e39da4a4923380a6fec415e4513db2af0354a76f432cafd82fb0
```

**For empty request bodies (e.g., GET requests):**

When there is no request body:
- Hash the empty string
- Result: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

**For non-JSON request bodies:**

If a request uses a non-JSON content type (rare in this protocol), hash the raw bytes of the body without any canonicalization. Encode the hash as lowercase hexadecimal.

---

### Constructing the Canonical Request

**Example Request:**

```http
POST /v1/agents/register?debug=true HTTP/1.1
Host: api.example.com
Content-Type: application/json
X-Permission-Slip-Signature: agent_id="agent_abc123", algorithm="Ed25519", timestamp="1707667200", signature="..."

{
  "agent_id": "agent_abc123",
  "public_key": "ssh-ed25519 AAAAC3Nza...",
  "approver": "alice"
}
```

**Step-by-Step Canonicalization:**

1. **Method:** `POST`
2. **Path:** `/v1/agents/register`
3. **Query string:** `debug=true` (already canonical in this case)
4. **Timestamp:** `1707667200`
5. **Body hash:**
   ```javascript
   const canonicalize = require('canonicalize');
   const crypto = require('crypto');
   
   const body = {"agent_id": "agent_abc123", "public_key": "ssh-ed25519 AAAAC3Nza...", "approver": "alice"};
   const canonicalBody = canonicalize(body);
   const bodyHash = crypto.createHash('sha256').update(canonicalBody, 'utf8').digest('hex');
   // bodyHash = "a1b2c3d4e5f6..." (example)
   ```

**Canonical Request:**

```
POST
/v1/agents/register
debug=true
1707667200
a1b2c3d4e5f6...
```

(5 lines, separated by newlines, no trailing newline)

---

### Signing the Canonical Request

**Steps:**

1. Construct the canonical request string (as above)
2. Encode the canonical request as UTF-8 bytes
3. Sign the UTF-8 bytes using the agent's private key
4. Encode the signature bytes as base64url (no padding)
5. Include in the `X-Permission-Slip-Signature` header

**JavaScript Example (Frontend - Ed25519):**

```javascript
const crypto = require('crypto');

// Construct canonical request
const canonicalRequest = `${method}\n${path}\n${query}\n${timestamp}\n${bodyHash}`;

// Sign
const signatureBytes = crypto.sign(
  null,
  Buffer.from(canonicalRequest, 'utf-8'),
  privateKey
);

// Base64url encode (no padding)
const signature = signatureBytes.toString('base64url').replace(/=/g, '');

// Construct header
const header = `agent_id="${agentId}", algorithm="Ed25519", timestamp="${timestamp}", signature="${signature}"`;
```

---

### Signature Verification

Services MUST verify the signature on every incoming request to Permission Slip protocol endpoints.

**Verification Steps:**

1. **Parse** the `X-Permission-Slip-Signature` header to extract:
   - `agent_id`
   - `algorithm`
   - `timestamp`
   - `signature` (base64url decode to get signature bytes)

2. **Timestamp validation:**
   - Verify the timestamp is within **300 seconds (5 minutes)** of the current server time
   - Reject if outside this window with `401 Unauthorized`, error code `timestamp_expired`
   - This limits the replay window for captured requests

3. **Agent lookup:**
   - For requests from already-registered agents: Retrieve the agent's registered public key from the database
   - For `POST /v1/agents/register` requests: Use the `public_key` provided in the request body
     - Also verify that the `agent_id` in the header is correctly derived from this public key

4. **Reconstruct the canonical request:**
   - Extract method, path, query string from the HTTP request
   - Use the `timestamp` from the signature header
   - Compute the body hash using the same canonicalization rules (RFC 8785 for JSON)

5. **Verify the signature:**
   - Use the agent's public key and the specified algorithm
   - Verify that the signature matches the canonical request

6. **Reject** if any step fails:
   - `401 Unauthorized`, error code `invalid_signature` (signature verification failed)
   - `401 Unauthorized`, error code `timestamp_expired` (timestamp outside window)

**Go Example (Backend - Ed25519):**

```go
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Production Implementation Guide for Helper Functions:
//
// parseSignatureHeader(header string) map[string]string
//   Purpose: Parse X-Permission-Slip-Signature header into components
//   Example implementation:
//     parts := strings.Split(header, ", ")
//     result := make(map[string]string)
//     for _, part := range parts {
//       kv := strings.SplitN(part, "=", 2)
//       if len(kv) == 2 {
//         result[kv[0]] = strings.Trim(kv[1], `"`)
//       }
//     }
//     return result
//   Returns map with keys: agent_id, algorithm, timestamp, signature
//
// constructCanonicalRequest(method, path, query, timestamp string, bodyBytes []byte) string
//   Purpose: Build canonical request string for signature verification
//   Implementation: See api.md section "Authentication" → "Canonical Request Format"
//   (Search for "### Canonical Request Format" in api.md)
//   Steps:
//     1. method.upper() + "\n"
//     2. path + "\n"
//     3. canonicalizeQuery(query) + "\n"  // RFC 3986, sorted
//     4. timestamp + "\n"
//     5. sha256(JCS_canonicalize(bodyBytes)).hex()
//   Returns: multi-line string as shown in spec
//
// IMPORTANT: HTTP request bodies can only be read once. In production middleware,
// you must buffer the body before this function and pass the bytes to both
// constructCanonicalRequest and subsequent handlers. See the verifyParameters
// example in api.md for body buffering pattern.

// verifyRequestSignature validates the Ed25519 signature on an incoming request.
func verifyRequestSignature(r *http.Request, agentPublicKey ed25519.PublicKey, bodyBytes []byte) error {
	// Parse signature header (extracts agent_id, algorithm, timestamp, signature)
	sigHeader := parseSignatureHeader(r.Header.Get("X-Permission-Slip-Signature"))
	
	// Verify timestamp (within 5 minutes)
	timestamp, err := strconv.ParseInt(sigHeader["timestamp"], 10, 64)
	if err != nil {
		return errors.New("invalid_timestamp")
	}
	if abs(time.Now().Unix()-timestamp) > 300 {
		return errors.New("timestamp_expired")
	}
	
	// Reconstruct canonical request using buffered body
	canonicalRequest := constructCanonicalRequest(
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		sigHeader["timestamp"],
		bodyBytes,
	)
	
	// Decode signature with proper error handling
	signatureB64 := sigHeader["signature"]
	// Add base64 padding if needed
	padding := strings.Repeat("=", (4-len(signatureB64)%4)%4)
	signatureBytes, err := base64.URLEncoding.DecodeString(signatureB64 + padding)
	if err != nil {
		return errors.New("invalid_signature_encoding")
	}
	
	// Verify signature
	if !ed25519.Verify(agentPublicKey, []byte(canonicalRequest), signatureBytes) {
		return errors.New("invalid_signature")
	}
	
	return nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
```

---

### Replay Protection

The timestamp requirement (requests valid for 5 minutes) limits the window for replay attacks but does not prevent a captured signed request from being replayed multiple times within that window.

**Required Protections:**

Services MUST implement replay protection by:

1. **Tracking request signatures:** Store a hash of the signature for each request within the 5-minute timestamp window
   - If a duplicate signature is detected, reject with `409 Conflict`, error code `duplicate_request`
   - Expire tracked signatures after 5 minutes (or after the request's timestamp + 300 seconds)

2. **Using request IDs:** For endpoints that include a `request_id` field (e.g., approval requests), use the `request_id` as an idempotency key
   - Reject duplicate `request_id` values with `409 Conflict`, error code `duplicate_request_id`
   - This provides application-level replay protection independent of signatures

**Implementation Example (Request ID Tracking):**

```go
package main

import (
	"encoding/json"
	"net/http"
)

// Production Implementation Guide for Helper Functions:
//
// requestIDExists(requestID string) bool
//   Purpose: Atomically check if request_id has been used
//   Redis implementation:
//     exists, err := redisClient.Get(ctx, "reqid:"+requestID).Result()
//     return err != redis.Nil
//   Database implementation:
//     var count int
//     db.QueryRow("SELECT COUNT(*) FROM request_ids WHERE id = $1", requestID).Scan(&count)
//     return count > 0
//   Note: Must be atomic - concurrent checks for same ID should not both return false
//
// storeRequestID(requestID string)
//   Purpose: Atomically store request_id with TTL to prevent future duplicates
//   Redis implementation (recommended - atomic check-and-set):
//     result := redisClient.SetNX(ctx, "reqid:"+requestID, "1", 5*time.Minute)
//     // SetNX returns false if key already exists (atomic operation)
//   Database implementation:
//     INSERT INTO request_ids (id, expires_at) VALUES ($1, NOW() + INTERVAL '5 minutes')
//     ON CONFLICT (id) DO NOTHING
//     // Requires unique constraint on id column
//   TTL: MUST expire after request_timestamp + 300 seconds
//
// For distributed systems, use Redis or database (not in-memory sync.Map)

// handleApprovalRequest processes an approval request with proper request ID
// validation to prevent replay attacks and duplicate submissions.
func handleApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// ... verify signature ...
	
	// Parse and validate request body
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Invalid JSON body"}}`, http.StatusBadRequest)
		return
	}
	
	// Extract and validate request_id with type checking
	requestIDRaw, ok := body["request_id"]
	if !ok {
		http.Error(w, `{"error":{"code":"missing_request_id","message":"request_id is required"}}`, http.StatusBadRequest)
		return
	}
	requestID, ok := requestIDRaw.(string)
	if !ok || requestID == "" {
		http.Error(w, `{"error":{"code":"invalid_request_id","message":"request_id must be a non-empty string"}}`, http.StatusBadRequest)
		return
	}
	
	// Check if request_id already used (atomic check-and-set)
	// Helper functions should use Redis SETNX, database unique constraint, or sync.Map with mutex
	if requestIDExists(requestID) {
		http.Error(w, `{"error":{"code":"duplicate_request_id","message":"Request ID already used"}}`, http.StatusConflict)
		return
	}
	
	// Store request_id with TTL (at least 5 minutes)
	storeRequestID(requestID)
	
	// Process request
	// ...
}
```

---

### Security Considerations

**For Agents:**

1. **Never reuse signatures:** Each request should have a unique timestamp and therefore a unique signature
2. **Protect private keys:** Store securely (see [Key Storage Requirements](#key-storage-requirements))
3. **Use current timestamps:** Always use the current time, not a cached value
4. **Implement clock skew tolerance:** If signature verification fails with `timestamp_expired`, check system clock synchronization

**For Services:**

1. **Enforce timestamp validation:** Always verify timestamps are within the 300-second window
2. **Use constant-time comparison:** When comparing signatures, use constant-time comparison functions to prevent timing attacks
3. **Implement replay protection:** Track signatures or request IDs to prevent replay attacks
4. **Reject unknown algorithms:** Only accept `Ed25519` and `ECDSA-P256`; reject others with `400 Bad Request`
5. **Log authentication failures:** Monitor for patterns that might indicate attacks
6. **Use RFC 8785 libraries:** Do not implement JSON canonicalization yourself; use tested libraries

**Common Pitfalls:**

1. **Incorrect JSON canonicalization:** Using `json.dumps(sort_keys=True)` instead of RFC 8785
   - Symptom: Signature verification always fails, even with correct keys
   - Cause: Simple key-sorting doesn't handle Unicode, number formats (`1.0` vs `1`), or escape sequences consistently
   - Solution: ALWAYS use an RFC 8785 library (`canonicalize` for JS, `json-canonicalization` for Go, `jcs` for Python)
   - Test: Verify your implementation produces the correct hash for the test vector in api.md

2. **Query string encoding errors:** Encoding spaces as `+` instead of `%20`, or using wrong encoding for special characters
   - Symptom: Signature verification fails on requests with query parameters
   - Cause: Using `application/x-www-form-urlencoded` encoding instead of RFC 3986 percent-encoding
   - Solution: Use RFC 3986 compliant encoder (unreserved chars: `A-Z a-z 0-9 - . _ ~` stay unencoded, space becomes `%20` not `+`)
   - Examples: `hello world` → `hello%20world`, `user@example.com` → `user%40example.com`

3. **Timestamp format errors:** Using milliseconds instead of seconds, or including decimals
   - Symptom: `timestamp_expired` errors or verification failures
   - Cause: JavaScript `Date.now()` returns milliseconds; some languages default to float timestamps
   - Solution: Use integer seconds since epoch (`Math.floor(Date.now() / 1000)` in JS, `time.Now().Unix()` in Go)

4. **Missing newlines in canonical request:** Forgetting newlines or adding extra trailing newlines
   - Symptom: Signature verification always fails
   - Cause: String concatenation errors or template literal issues
   - Solution: Exactly 5 lines separated by `\n`, NO trailing newline. Use template literals or explicit `\n` joins.
   - Example: `"POST\n/path\n\n1234567890\nabcd1234"` (5 lines, last line has no trailing newline)

5. **Base64 padding errors:** Incorrect base64url encoding or decoding of signatures
   - Symptom: `invalid_signature_encoding` errors
   - Cause: Using standard base64 instead of base64url, or incorrect padding removal/addition
   - Solution: Use base64url (no padding for encoding, add padding for decoding if needed): `-` and `_` instead of `+` and `/`

---

## Next Steps

This document will be expanded with:
- Challenge-response authentication flow (if needed for specific use cases)
- Additional algorithm support (RSA, if demand exists)

See also:
- [API Specification](api.md) - Full HTTP API reference
- [Terminology](terminology.md) - Core protocol concepts
