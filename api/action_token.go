package api

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	jcs "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// actionTokenAudience is the audience claim for action tokens.
	actionTokenAudience = "permissionslip.dev"

	// actionTokenTTL is the default time-to-live for action tokens.
	// Tokens also cannot outlive the approval's expires_at.
	actionTokenTTL = 5 * time.Minute
)

// ActionTokenClaims contains the JWT claims for an action token.
type ActionTokenClaims struct {
	jwt.RegisteredClaims
	Approver   string `json:"approver"`
	ApprovalID string `json:"approval_id"`
	Scope      string `json:"scope"`
	// ScopeVersion is serialized as a string (e.g. "1") per the spec.
	ScopeVersion string `json:"scope_version"`
	ParamsHash   string `json:"params_hash"`
}

// MintActionToken creates a signed ES256 JWT action token.
func MintActionToken(key *ecdsa.PrivateKey, keyID string, claims ActionTokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(key)
}

// buildActionTokenClaims constructs ActionTokenClaims from an approval and its context.
func buildActionTokenClaims(
	agentID int64,
	approverUsername string,
	approvalID string,
	actionJSON []byte,
	approvalExpiresAt time.Time,
	jti string,
) (ActionTokenClaims, error) {
	scope, scopeVersion, params, err := parseActionFields(actionJSON)
	if err != nil {
		return ActionTokenClaims{}, err
	}

	hash, err := HashParameters(params)
	if err != nil {
		return ActionTokenClaims{}, fmt.Errorf("compute params hash: %w", err)
	}

	now := time.Now().UTC()
	exp := now.Add(actionTokenTTL)
	// Don't let the token outlive the approval.
	if approvalExpiresAt.Before(exp) {
		exp = approvalExpiresAt
	}

	return ActionTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(agentID, 10),
			Audience:  jwt.ClaimStrings{actionTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        jti,
		},
		Approver:     approverUsername,
		ApprovalID:   approvalID,
		Scope:        scope,
		ScopeVersion: scopeVersion,
		ParamsHash:   hash,
	}, nil
}

// parseActionFields unmarshals an action JSON blob once and extracts scope,
// version, and parameters. This avoids double-parsing the same JSON.
func parseActionFields(actionJSON []byte) (scope, scopeVersion string, params json.RawMessage, err error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(actionJSON, &obj); err != nil {
		return "", "", nil, fmt.Errorf("unmarshal action: %w", err)
	}

	// Extract scope (required).
	typeRaw, ok := obj["type"]
	if !ok {
		return "", "", nil, fmt.Errorf("action missing required \"type\" field")
	}
	if err := json.Unmarshal(typeRaw, &scope); err != nil {
		return "", "", nil, fmt.Errorf("action.type is not a string: %w", err)
	}
	if scope == "" {
		return "", "", nil, fmt.Errorf("action.type must be non-empty")
	}

	// Extract version (defaults to "1").
	scopeVersion = "1"
	if versionRaw, ok := obj["version"]; ok {
		var v string
		if err := json.Unmarshal(versionRaw, &v); err != nil {
			return "", "", nil, fmt.Errorf("action.version is not a string: %w", err)
		}
		if v != "" {
			scopeVersion = v
		}
	}

	// Extract parameters (defaults to empty object).
	params, ok = obj["parameters"]
	if !ok || len(params) == 0 || string(params) == "null" {
		params = []byte("{}")
	}

	return scope, scopeVersion, params, nil
}

// HashParameters returns the hex-encoded SHA-256 hash of the JCS (RFC 8785)
// canonicalized form of the given JSON parameters. This is the value stored
// in action tokens as the params_hash claim.
func HashParameters(params json.RawMessage) (string, error) {
	canonical, err := jcsCanonical(params)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(canonical)
	return hex.EncodeToString(h[:]), nil
}

// VerifyParamsHash recomputes the SHA-256 hash of JCS-canonicalized params
// and compares it against expectedHash using constant-time comparison to
// prevent timing side-channel attacks. Returns nil on match, or an error
// describing the mismatch.
func VerifyParamsHash(params json.RawMessage, expectedHash string) error {
	computed, err := HashParameters(params)
	if err != nil {
		return fmt.Errorf("compute params hash: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) != 1 {
		return fmt.Errorf("params_hash mismatch: expected %s, got %s", expectedHash, computed)
	}
	return nil
}

// jcsCanonical returns the JCS (RFC 8785) canonical serialization of a JSON value
// using github.com/cyberphone/json-canonicalization — the reference implementation
// by the RFC author. This handles sorted keys, RFC 8785 number formatting,
// Unicode normalization, and correct string escaping.
func jcsCanonical(raw json.RawMessage) ([]byte, error) {
	return jcs.Transform(raw)
}
