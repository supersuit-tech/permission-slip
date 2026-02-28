package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── JCS canonicalization (RFC 8785) ─────────────────────────────────────────

func TestJCSCanonical_SortsKeys(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"b":"2","a":"1","c":"3"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"a":"1","b":"2","c":"3"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_NestedObjects(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"z":{"b":"2","a":"1"},"a":"top"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"a":"top","z":{"a":"1","b":"2"}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_DeeplyNestedObjects(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"c":{"z":{"q":"deep","a":"also"},"b":"mid"},"a":"top"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"a":"top","c":{"b":"mid","z":{"a":"also","q":"deep"}}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_Arrays(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"items":[3,1,2],"name":"test"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// Arrays preserve order; top-level keys sorted.
	expected := `{"items":[3,1,2],"name":"test"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_ArrayOfObjects(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"items":[{"z":"1","a":"2"},{"b":"3"}]}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// Objects inside arrays should also have sorted keys.
	expected := `{"items":[{"a":"2","z":"1"},{"b":"3"}]}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_NoHTMLEscaping(t *testing.T) {
	t.Parallel()
	// json.Marshal would escape < > & to \u003c etc. JCS must not.
	input := json.RawMessage(`{"query":"a<b && c>d"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"query":"a<b && c>d"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_EmptyObject(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("expected {}, got %s", string(got))
	}
}

func TestJCSCanonical_EmptyArray(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`[]`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	if string(got) != "[]" {
		t.Errorf("expected [], got %s", string(got))
	}
}

func TestJCSCanonical_BooleanValues(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"flag":true,"other":false}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"flag":true,"other":false}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

// ── RFC 8785 number formatting ──────────────────────────────────────────────

func TestJCSCanonical_Integers(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"amount":9900,"count":0,"negative":-42}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"amount":9900,"count":0,"negative":-42}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_FloatingPoint(t *testing.T) {
	t.Parallel()
	// RFC 8785 requires shortest representation: 1.5 stays as 1.5.
	input := json.RawMessage(`{"price":1.5}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"price":1.5}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_TrailingZeros(t *testing.T) {
	t.Parallel()
	// 1.00 should be serialized as 1 (shortest representation per RFC 8785).
	input := json.RawMessage(`{"value":1.00}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"value":1}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_ScientificNotation(t *testing.T) {
	t.Parallel()
	// RFC 8785: 1e2 = 100, should be serialized without scientific notation.
	input := json.RawMessage(`{"val":1e2}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"val":100}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_LargeExponent(t *testing.T) {
	t.Parallel()
	// RFC 8785: very large numbers use exponential notation per ES6 rules.
	input := json.RawMessage(`{"big":1e20}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// 1e20 = 100000000000000000000 — within safe integer range, no exponent.
	expected := `{"big":100000000000000000000}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_VeryLargeNumber(t *testing.T) {
	t.Parallel()
	// 1e21 crosses into exponential notation territory per ES6 ToString.
	input := json.RawMessage(`{"huge":1e21}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// RFC 8785 / ES6: 1e21 → "1e+21"
	expected := `{"huge":1e+21}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_NegativeZero(t *testing.T) {
	t.Parallel()
	// RFC 8785: -0 must be serialized as 0.
	input := json.RawMessage(`{"val":-0}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"val":0}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

// ── RFC 8785 Unicode handling ───────────────────────────────────────────────

func TestJCSCanonical_UnicodeStrings(t *testing.T) {
	t.Parallel()
	// Unicode should pass through without unnecessary escaping.
	input := json.RawMessage(`{"name":"日本語"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"name":"日本語"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_Emoji(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"msg":"hello 👋"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"msg":"hello 👋"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_ControlCharEscaping(t *testing.T) {
	t.Parallel()
	// RFC 8785: control characters (U+0000–U+001F) must be escaped.
	// Tab (\t = U+0009) and newline (\n = U+000A) should remain escaped.
	input := json.RawMessage(`{"text":"line1\nline2\ttab"}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// The output should contain escaped control characters.
	if !strings.Contains(string(got), `\n`) || !strings.Contains(string(got), `\t`) {
		t.Errorf("expected escaped control chars, got %s", string(got))
	}
}

func TestJCSCanonical_UnicodeEscapeSequences(t *testing.T) {
	t.Parallel()
	// Input with Unicode escape sequences should produce the same output
	// as the unescaped version.
	escaped := json.RawMessage(`{"key":"\u0041\u0042\u0043"}`)
	unescaped := json.RawMessage(`{"key":"ABC"}`)
	got1, err := jcsCanonical(escaped)
	if err != nil {
		t.Fatalf("jcsCanonical escaped: %v", err)
	}
	got2, err := jcsCanonical(unescaped)
	if err != nil {
		t.Fatalf("jcsCanonical unescaped: %v", err)
	}
	if string(got1) != string(got2) {
		t.Errorf("expected same output for equivalent inputs:\n  escaped:   %s\n  unescaped: %s", got1, got2)
	}
}

func TestJCSCanonical_MixedTypes(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"str":"hello","num":42,"bool":true,"null_val":null,"arr":[1,"two"],"obj":{"nested":true}}`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	// Keys sorted alphabetically.
	expected := `{"arr":[1,"two"],"bool":true,"null_val":null,"num":42,"obj":{"nested":true},"str":"hello"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_WhitespaceStripped(t *testing.T) {
	t.Parallel()
	// Input with extra whitespace should produce compact output.
	input := json.RawMessage(`{  "b" :  "2" , "a" : "1"  }`)
	got, err := jcsCanonical(input)
	if err != nil {
		t.Fatalf("jcsCanonical: %v", err)
	}
	expected := `{"a":"1","b":"2"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestJCSCanonical_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := jcsCanonical(json.RawMessage(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── HashParameters ──────────────────────────────────────────────────────────

func TestHashParameters_Deterministic(t *testing.T) {
	t.Parallel()
	// Same content, different key order → same hash.
	hash1, err := HashParameters(json.RawMessage(`{"b":"2","a":"1"}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	hash2, err := HashParameters(json.RawMessage(`{"a":"1","b":"2"}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("expected same hash for equivalent JSON, got %q vs %q", hash1, hash2)
	}
}

func TestHashParameters_DifferentContent(t *testing.T) {
	t.Parallel()
	hash1, _ := HashParameters(json.RawMessage(`{"a":"1"}`))
	hash2, _ := HashParameters(json.RawMessage(`{"a":"2"}`))
	if hash1 == hash2 {
		t.Error("expected different hashes for different content")
	}
}

func TestHashParameters_EmptyObject(t *testing.T) {
	t.Parallel()
	hash, err := HashParameters(json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	// SHA-256 of "{}" is stable and known.
	expected := sha256.Sum256([]byte("{}"))
	expectedHex := hex.EncodeToString(expected[:])
	if hash != expectedHex {
		t.Errorf("expected hash %q, got %q", expectedHex, hash)
	}
}

func TestHashParameters_Stability(t *testing.T) {
	t.Parallel()
	// Verify a known input produces a known, stable hash. This guards
	// against accidental changes to the canonicalization logic.
	hash, err := HashParameters(json.RawMessage(`{"to":"alice@example.com","subject":"hi"}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	// Precomputed: SHA-256 of JCS(`{"subject":"hi","to":"alice@example.com"}`)
	canonical := `{"subject":"hi","to":"alice@example.com"}`
	expected := sha256.Sum256([]byte(canonical))
	expectedHex := hex.EncodeToString(expected[:])
	if hash != expectedHex {
		t.Errorf("expected stable hash %q, got %q", expectedHex, hash)
	}
}

func TestHashParameters_WhitespaceInsensitive(t *testing.T) {
	t.Parallel()
	// Different formatting of the same JSON should produce the same hash.
	compact := json.RawMessage(`{"a":"1","b":"2"}`)
	spaced := json.RawMessage(`{  "a" : "1" ,  "b" : "2"  }`)
	hash1, err := HashParameters(compact)
	if err != nil {
		t.Fatalf("HashParameters compact: %v", err)
	}
	hash2, err := HashParameters(spaced)
	if err != nil {
		t.Fatalf("HashParameters spaced: %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("expected same hash regardless of whitespace, got %q vs %q", hash1, hash2)
	}
}

func TestHashParameters_HexEncoded(t *testing.T) {
	t.Parallel()
	hash, err := HashParameters(json.RawMessage(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	// SHA-256 hex digest is always 64 characters.
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex string, got %d chars: %q", len(hash), hash)
	}
	// Verify it's valid hex.
	if _, err := hex.DecodeString(hash); err != nil {
		t.Errorf("expected valid hex, got error: %v", err)
	}
}

// ── VerifyParamsHash ────────────────────────────────────────────────────────

func TestVerifyParamsHash_Match(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"to":"alice@example.com","subject":"hi"}`)
	hash, err := HashParameters(params)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	if err := VerifyParamsHash(params, hash); err != nil {
		t.Errorf("expected match, got error: %v", err)
	}
}

func TestVerifyParamsHash_MatchDifferentKeyOrder(t *testing.T) {
	t.Parallel()
	// Hash computed from one key order should verify against another.
	original := json.RawMessage(`{"subject":"hi","to":"alice@example.com"}`)
	hash, err := HashParameters(original)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	reordered := json.RawMessage(`{"to":"alice@example.com","subject":"hi"}`)
	if err := VerifyParamsHash(reordered, hash); err != nil {
		t.Errorf("expected match with reordered keys, got error: %v", err)
	}
}

func TestVerifyParamsHash_Mismatch(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"to":"alice@example.com"}`)
	if err := VerifyParamsHash(params, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected mismatch error, got nil")
	}
}

func TestVerifyParamsHash_TamperedParameters(t *testing.T) {
	t.Parallel()
	// Simulate: approval was for sending to alice, but agent tampers to bob.
	approved := json.RawMessage(`{"to":"alice@example.com","subject":"Meeting"}`)
	hash, err := HashParameters(approved)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	tampered := json.RawMessage(`{"to":"bob@example.com","subject":"Meeting"}`)
	if err := VerifyParamsHash(tampered, hash); err == nil {
		t.Error("expected error for tampered parameters, got nil")
	}
}

func TestVerifyParamsHash_ExtraField(t *testing.T) {
	t.Parallel()
	// Adding an extra field should cause a mismatch.
	original := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, err := HashParameters(original)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	withExtra := json.RawMessage(`{"to":"alice@example.com","bcc":"eve@example.com"}`)
	if err := VerifyParamsHash(withExtra, hash); err == nil {
		t.Error("expected error for extra field, got nil")
	}
}

func TestVerifyParamsHash_MissingField(t *testing.T) {
	t.Parallel()
	// Removing a field should cause a mismatch.
	original := json.RawMessage(`{"to":"alice@example.com","subject":"hi"}`)
	hash, err := HashParameters(original)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	withMissing := json.RawMessage(`{"to":"alice@example.com"}`)
	if err := VerifyParamsHash(withMissing, hash); err == nil {
		t.Error("expected error for missing field, got nil")
	}
}

func TestVerifyParamsHash_InvalidJSON(t *testing.T) {
	t.Parallel()
	err := VerifyParamsHash(json.RawMessage(`{invalid}`), "abc")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestVerifyParamsHash_ErrorContainsBothHashes(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"key":"value"}`)
	expectedHash := "0000000000000000000000000000000000000000000000000000000000000000"
	err := VerifyParamsHash(params, expectedHash)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), expectedHash) {
		t.Errorf("error should contain expected hash, got: %v", err)
	}
}

// ── parseActionFields (scope, version, params in one pass) ──────────────────

func TestParseActionFields_ExtractsTypeAndVersion(t *testing.T) {
	t.Parallel()
	scope, version, _, err := parseActionFields([]byte(`{"type":"email.send","version":"2","parameters":{}}`))
	if err != nil {
		t.Fatalf("parseActionFields: %v", err)
	}
	if scope != "email.send" {
		t.Errorf("expected scope 'email.send', got %q", scope)
	}
	if version != "2" {
		t.Errorf("expected version '2', got %q", version)
	}
}

func TestParseActionFields_DefaultsVersionTo1(t *testing.T) {
	t.Parallel()
	scope, version, _, err := parseActionFields([]byte(`{"type":"email.send","parameters":{}}`))
	if err != nil {
		t.Fatalf("parseActionFields: %v", err)
	}
	if scope != "email.send" {
		t.Errorf("expected scope 'email.send', got %q", scope)
	}
	if version != "1" {
		t.Errorf("expected default version '1', got %q", version)
	}
}

func TestParseActionFields_ExtractsParameters(t *testing.T) {
	t.Parallel()
	_, _, params, err := parseActionFields([]byte(`{"type":"email.send","parameters":{"to":"alice@example.com"}}`))
	if err != nil {
		t.Fatalf("parseActionFields: %v", err)
	}
	var obj map[string]string
	if err := json.Unmarshal(params, &obj); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if obj["to"] != "alice@example.com" {
		t.Errorf("expected to='alice@example.com', got %q", obj["to"])
	}
}

func TestParseActionFields_DefaultsParamsToEmptyObject(t *testing.T) {
	t.Parallel()
	_, _, params, err := parseActionFields([]byte(`{"type":"email.send"}`))
	if err != nil {
		t.Fatalf("parseActionFields: %v", err)
	}
	if string(params) != "{}" {
		t.Errorf("expected {}, got %s", string(params))
	}
}

func TestParseActionFields_NullParamsTreatedAsEmptyObject(t *testing.T) {
	t.Parallel()
	_, _, params, err := parseActionFields([]byte(`{"type":"email.send","parameters":null}`))
	if err != nil {
		t.Fatalf("parseActionFields: %v", err)
	}
	if string(params) != "{}" {
		t.Errorf("expected {}, got %s", string(params))
	}
}

// ── MintActionToken ─────────────────────────────────────────────────────────

func TestMintActionToken_ValidES256(t *testing.T) {
	t.Parallel()
	key, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Now().UTC()
	claims := ActionTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "42",
			Audience:  jwt.ClaimStrings{"permissionslip.dev"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			ID:        "tok_test123",
		},
		Approver:     "alice",
		ApprovalID:   "appr_xyz789",
		Scope:        "email.send",
		ScopeVersion: "1",
		ParamsHash:   "abc123",
	}

	tokenStr, err := MintActionToken(key, "test-key-1", claims)
	if err != nil {
		t.Fatalf("MintActionToken: %v", err)
	}

	// Verify the token can be parsed back with the public key.
	parsed, err := jwt.ParseWithClaims(tokenStr, &ActionTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return &key.PublicKey, nil
		})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !parsed.Valid {
		t.Error("expected valid token")
	}

	parsedClaims := parsed.Claims.(*ActionTokenClaims)
	if parsedClaims.Approver != "alice" {
		t.Errorf("expected approver 'alice', got %q", parsedClaims.Approver)
	}
	if parsedClaims.Scope != "email.send" {
		t.Errorf("expected scope 'email.send', got %q", parsedClaims.Scope)
	}
}

// ── buildActionTokenClaims ──────────────────────────────────────────────────

func TestBuildActionTokenClaims_Success(t *testing.T) {
	t.Parallel()
	actionJSON := []byte(`{"type":"email.send","version":"2","parameters":{"to":"alice@example.com"}}`)
	expiresAt := time.Now().Add(10 * time.Minute)

	claims, err := buildActionTokenClaims(42, "alice", "appr_test", actionJSON, expiresAt, "tok_jti123")
	if err != nil {
		t.Fatalf("buildActionTokenClaims: %v", err)
	}

	if claims.Subject != "42" {
		t.Errorf("expected sub '42', got %q", claims.Subject)
	}
	if claims.Approver != "alice" {
		t.Errorf("expected approver 'alice', got %q", claims.Approver)
	}
	if claims.ApprovalID != "appr_test" {
		t.Errorf("expected approval_id 'appr_test', got %q", claims.ApprovalID)
	}
	if claims.Scope != "email.send" {
		t.Errorf("expected scope 'email.send', got %q", claims.Scope)
	}
	if claims.ScopeVersion != "2" {
		t.Errorf("expected scope_version '2', got %q", claims.ScopeVersion)
	}
	if claims.ParamsHash == "" {
		t.Error("expected non-empty params_hash")
	}
	if claims.ID != "tok_jti123" {
		t.Errorf("expected jti 'tok_jti123', got %q", claims.ID)
	}

	// Token expiry should be min(now+5min, expiresAt). Since expiresAt is 10 min,
	// the token should expire at ~now+5min.
	ttl := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if ttl > 5*time.Minute+time.Second {
		t.Errorf("expected TTL ≤5min, got %v", ttl)
	}
}

func TestBuildActionTokenClaims_CapsAtApprovalExpiry(t *testing.T) {
	t.Parallel()
	actionJSON := []byte(`{"type":"test","parameters":{}}`)
	// Approval expires in 2 minutes — less than the 5-minute token TTL.
	expiresAt := time.Now().Add(2 * time.Minute)

	claims, err := buildActionTokenClaims(1, "bob", "appr_x", actionJSON, expiresAt, "tok_1")
	if err != nil {
		t.Fatalf("buildActionTokenClaims: %v", err)
	}

	// Token should not outlive the approval.
	if claims.ExpiresAt.Time.After(expiresAt.Add(time.Second)) {
		t.Errorf("token exp %v should not exceed approval expiresAt %v",
			claims.ExpiresAt.Time, expiresAt)
	}
}

func TestBuildActionTokenClaims_ParamsHashMatchesHashParameters(t *testing.T) {
	t.Parallel()
	actionJSON := []byte(`{"type":"email.send","parameters":{"to":"bob@example.com","cc":"carol@example.com"}}`)
	expiresAt := time.Now().Add(10 * time.Minute)

	claims, err := buildActionTokenClaims(1, "alice", "appr_1", actionJSON, expiresAt, "tok_1")
	if err != nil {
		t.Fatalf("buildActionTokenClaims: %v", err)
	}

	// Hash from claims should match a direct HashParameters call.
	directHash, err := HashParameters(json.RawMessage(`{"to":"bob@example.com","cc":"carol@example.com"}`))
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}
	if claims.ParamsHash != directHash {
		t.Errorf("claims.ParamsHash %q != direct HashParameters %q", claims.ParamsHash, directHash)
	}
}

// ── End-to-end: mint → verify hash ──────────────────────────────────────────

func TestMintAndVerifyParamsHash_RoundTrip(t *testing.T) {
	t.Parallel()
	key, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	actionJSON := []byte(`{"type":"email.send","parameters":{"to":"alice@example.com","subject":"Test"}}`)
	expiresAt := time.Now().Add(10 * time.Minute)

	claims, err := buildActionTokenClaims(42, "alice", "appr_rt", actionJSON, expiresAt, "tok_rt1")
	if err != nil {
		t.Fatalf("buildActionTokenClaims: %v", err)
	}

	tokenStr, err := MintActionToken(key, "key-1", claims)
	if err != nil {
		t.Fatalf("MintActionToken: %v", err)
	}

	// Parse the token back.
	parsed, err := jwt.ParseWithClaims(tokenStr, &ActionTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return &key.PublicKey, nil
		})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	parsedClaims := parsed.Claims.(*ActionTokenClaims)

	// Verify: same parameters should pass.
	sameParams := json.RawMessage(`{"to":"alice@example.com","subject":"Test"}`)
	if err := VerifyParamsHash(sameParams, parsedClaims.ParamsHash); err != nil {
		t.Errorf("expected verification to pass for same params, got: %v", err)
	}

	// Verify: reordered parameters should also pass (JCS normalizes key order).
	reorderedParams := json.RawMessage(`{"subject":"Test","to":"alice@example.com"}`)
	if err := VerifyParamsHash(reorderedParams, parsedClaims.ParamsHash); err != nil {
		t.Errorf("expected verification to pass for reordered params, got: %v", err)
	}

	// Verify: tampered parameters should fail.
	tamperedParams := json.RawMessage(`{"to":"eve@example.com","subject":"Test"}`)
	if err := VerifyParamsHash(tamperedParams, parsedClaims.ParamsHash); err == nil {
		t.Error("expected verification to fail for tampered params")
	}
}
