package api

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// signatureHeader is the HTTP header name for the agent signature.
	signatureHeader = "X-Permission-Slip-Signature"

	// signatureTimestampWindow is the maximum age of a signature timestamp.
	signatureTimestampWindow = 300 * time.Second
)

// ErrSigTimestampExpired is returned when a signature timestamp falls outside
// the acceptable window (±5 minutes).
var ErrSigTimestampExpired = errors.New("timestamp expired")

// emptyBodyHash is the SHA-256 hash of an empty body.
const emptyBodyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// ParsedSignature holds the parsed fields from the X-Permission-Slip-Signature header.
type ParsedSignature struct {
	AgentID   int64
	Algorithm string
	Timestamp int64
	Signature []byte
}

// ParseSignatureHeader parses the X-Permission-Slip-Signature header value.
// Format: agent_id="<id>", algorithm="<alg>", timestamp="<ts>", signature="<sig>"
func ParseSignatureHeader(header string) (*ParsedSignature, error) {
	fields := make(map[string]string)
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			return nil, fmt.Errorf("malformed field: %q", part)
		}
		key := strings.TrimSpace(part[:eq])
		val := strings.TrimSpace(part[eq+1:])
		val = strings.Trim(val, `"`)
		fields[key] = val
	}

	agentIDStr, ok := fields["agent_id"]
	if !ok || agentIDStr == "" {
		return nil, errors.New("missing agent_id")
	}
	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil || agentID <= 0 {
		return nil, fmt.Errorf("invalid agent_id: %q", agentIDStr)
	}

	algorithm, ok := fields["algorithm"]
	if !ok || algorithm == "" {
		return nil, errors.New("missing algorithm")
	}

	timestampStr, ok := fields["timestamp"]
	if !ok || timestampStr == "" {
		return nil, errors.New("missing timestamp")
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %q", timestampStr)
	}

	sigStr, ok := fields["signature"]
	if !ok || sigStr == "" {
		return nil, errors.New("missing signature")
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	return &ParsedSignature{
		AgentID:   agentID,
		Algorithm: algorithm,
		Timestamp: timestamp,
		Signature: sig,
	}, nil
}

// ParseEd25519PublicKey parses an OpenSSH-format Ed25519 public key string
// (e.g. "ssh-ed25519 AAAAC3..."). Returns the raw 32-byte Ed25519 public key.
func ParseEd25519PublicKey(keyStr string) (ed25519.PublicKey, error) {
	parts := strings.Fields(keyStr)
	if len(parts) < 2 {
		return nil, errors.New("invalid OpenSSH key format: expected at least 2 fields")
	}
	if parts[0] != "ssh-ed25519" {
		return nil, fmt.Errorf("unsupported key type: %q (only ssh-ed25519 is supported)", parts[0])
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid base64 in key: %w", err)
	}

	// OpenSSH wire format: 4 bytes length + "ssh-ed25519" + 4 bytes length + 32 bytes key
	// Minimum: 4 + 11 + 4 + 32 = 51 bytes
	if len(decoded) < 51 {
		return nil, fmt.Errorf("decoded key data too short (%d bytes)", len(decoded))
	}

	// Read key type string length (big-endian uint32)
	typeLen := int(decoded[0])<<24 | int(decoded[1])<<16 | int(decoded[2])<<8 | int(decoded[3])
	if typeLen != 11 || string(decoded[4:15]) != "ssh-ed25519" {
		return nil, errors.New("key type mismatch in wire format")
	}

	// Read the public key data
	keyDataOffset := 4 + typeLen
	if keyDataOffset+4 > len(decoded) {
		return nil, errors.New("truncated key data")
	}
	keyLen := int(decoded[keyDataOffset])<<24 | int(decoded[keyDataOffset+1])<<16 | int(decoded[keyDataOffset+2])<<8 | int(decoded[keyDataOffset+3])
	keyDataStart := keyDataOffset + 4
	if keyLen != ed25519.PublicKeySize || keyDataStart+keyLen > len(decoded) {
		return nil, fmt.Errorf("unexpected Ed25519 key length: %d", keyLen)
	}

	pubKey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pubKey, decoded[keyDataStart:keyDataStart+keyLen])
	return pubKey, nil
}

// BuildCanonicalRequest constructs the canonical string to be signed per the spec:
//
//	<METHOD>\n<PATH>\n<QUERY>\n<TIMESTAMP>\n<BODY_HASH>
func BuildCanonicalRequest(method, path, rawQuery string, timestamp int64, bodyHash string) string {
	// Normalize query: sort params, uppercase percent-encoding.
	canonicalQuery := canonicalizeQuery(rawQuery)
	return fmt.Sprintf("%s\n%s\n%s\n%d\n%s",
		strings.ToUpper(method),
		path,
		canonicalQuery,
		timestamp,
		bodyHash,
	)
}

// canonicalizeQuery sorts query parameters lexicographically and normalizes
// percent-encoding to uppercase hex.
func canonicalizeQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		for _, v := range params[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// HashBody returns the lowercase hex SHA-256 hash of the given body bytes.
// Returns emptyBodyHash for nil/empty body.
func HashBody(body []byte) string {
	if len(body) == 0 {
		return emptyBodyHash
	}
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

// VerifyEd25519Signature verifies the request signature using the provided
// Ed25519 public key. It validates the timestamp window, builds the canonical
// request, and verifies the Ed25519 signature.
func VerifyEd25519Signature(pubKey ed25519.PublicKey, sig *ParsedSignature, r *http.Request, bodyBytes []byte) error {
	if sig.Algorithm != "Ed25519" {
		return fmt.Errorf("unsupported algorithm: %q", sig.Algorithm)
	}

	// Check timestamp window.
	now := time.Now().Unix()
	diff := now - sig.Timestamp
	if diff < 0 {
		diff = -diff
	}
	if diff > int64(signatureTimestampWindow.Seconds()) {
		return ErrSigTimestampExpired
	}

	// Build canonical request.
	bodyHash := HashBody(bodyBytes)
	canonical := BuildCanonicalRequest(r.Method, r.URL.Path, r.URL.RawQuery, sig.Timestamp, bodyHash)

	// Verify Ed25519 signature.
	if !ed25519.Verify(pubKey, []byte(canonical), sig.Signature) {
		return errors.New("signature verification failed")
	}

	return nil
}

// VerifyRegistrationSignature verifies the signature on a registration request
// where the public key is provided in the request body (the agent doesn't have
// an agent_id yet, or is using a temporary one). The agent_id in the header may
// be 0 or any placeholder for registration; the caller can ignore it.
func VerifyRegistrationSignature(pubKeyStr string, r *http.Request, bodyBytes []byte) (*ParsedSignature, error) {
	headerVal := r.Header.Get(signatureHeader)
	if headerVal == "" {
		return nil, errors.New("missing signature header")
	}

	sig, err := ParseSignatureHeader(headerVal)
	if err != nil {
		return nil, fmt.Errorf("parse signature header: %w", err)
	}

	pubKey, err := ParseEd25519PublicKey(pubKeyStr)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	if err := VerifyEd25519Signature(pubKey, sig, r, bodyBytes); err != nil {
		return nil, err
	}

	return sig, nil
}

// VerifyAgentSignature verifies the signature on a request from a registered/pending
// agent. It parses the header, looks up the agent's stored public key, and verifies.
func VerifyAgentSignature(storedPubKey string, r *http.Request, bodyBytes []byte) (*ParsedSignature, error) {
	headerVal := r.Header.Get(signatureHeader)
	if headerVal == "" {
		return nil, errors.New("missing signature header")
	}

	sig, err := ParseSignatureHeader(headerVal)
	if err != nil {
		return nil, fmt.Errorf("parse signature header: %w", err)
	}

	pubKey, err := ParseEd25519PublicKey(storedPubKey)
	if err != nil {
		return nil, fmt.Errorf("parse stored public key: %w", err)
	}

	if err := VerifyEd25519Signature(pubKey, sig, r, bodyBytes); err != nil {
		return nil, err
	}

	return sig, nil
}

// GenerateEd25519OpenSSHKey generates a new Ed25519 key pair and returns the
// public key in OpenSSH format (e.g. "ssh-ed25519 AAAA...") and the raw
// private key. This is primarily useful for testing.
func GenerateEd25519OpenSSHKey() (pubKeySSH string, privKey ed25519.PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", nil, err
	}
	return FormatEd25519OpenSSH(pub), priv, nil
}

// FormatEd25519OpenSSH formats an Ed25519 public key in OpenSSH wire format.
func FormatEd25519OpenSSH(pub ed25519.PublicKey) string {
	keyType := "ssh-ed25519"
	typeBytes := []byte(keyType)

	// Build wire format: uint32(len("ssh-ed25519")) + "ssh-ed25519" + uint32(32) + <key bytes>
	wireLen := 4 + len(typeBytes) + 4 + len(pub)
	wire := make([]byte, wireLen)
	offset := 0

	// Type string length (big-endian)
	wire[offset] = byte(len(typeBytes) >> 24)
	wire[offset+1] = byte(len(typeBytes) >> 16)
	wire[offset+2] = byte(len(typeBytes) >> 8)
	wire[offset+3] = byte(len(typeBytes))
	offset += 4
	copy(wire[offset:], typeBytes)
	offset += len(typeBytes)

	// Key data length (big-endian)
	wire[offset] = byte(len(pub) >> 24)
	wire[offset+1] = byte(len(pub) >> 16)
	wire[offset+2] = byte(len(pub) >> 8)
	wire[offset+3] = byte(len(pub))
	offset += 4
	copy(wire[offset:], pub)

	return keyType + " " + base64.StdEncoding.EncodeToString(wire)
}

// SignRequestAt creates a signed HTTP request with the X-Permission-Slip-Signature
// header using an explicit Unix timestamp. Used for testing — callers can craft
// signatures with timestamps in the past or future.
// agentID can be 0 for registration requests.
func SignRequestAt(privKey ed25519.PrivateKey, agentID int64, r *http.Request, bodyBytes []byte, timestamp int64) {
	bodyHash := HashBody(bodyBytes)
	canonical := BuildCanonicalRequest(r.Method, r.URL.Path, r.URL.RawQuery, timestamp, bodyHash)
	sig := ed25519.Sign(privKey, []byte(canonical))

	// Ensure agentID is at least 1 for valid header format (registration uses a placeholder)
	headerAgentID := agentID
	if headerAgentID <= 0 {
		headerAgentID = math.MaxInt64 // placeholder for registration
	}

	headerVal := fmt.Sprintf(`agent_id="%d", algorithm="Ed25519", timestamp="%d", signature="%s"`,
		headerAgentID,
		timestamp,
		base64.RawURLEncoding.EncodeToString(sig),
	)
	r.Header.Set(signatureHeader, headerVal)
}

// SignRequest creates a signed HTTP request with the X-Permission-Slip-Signature header
// using the current time. Used for testing. agentID can be 0 for registration requests.
func SignRequest(privKey ed25519.PrivateKey, agentID int64, r *http.Request, bodyBytes []byte) {
	SignRequestAt(privKey, agentID, r, bodyBytes, time.Now().Unix())
}
