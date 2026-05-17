package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "unit-test-jwt-signing-secret-32bytes!"

func TestHashPasswordVerifyPassword_RoundTrip(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	ok, err = VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword wrong: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPassword_StubHashAlwaysFalse(t *testing.T) {
	t.Parallel()
	ok, err := VerifyPassword("x", "test-stub-hash")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("stub hash should not verify")
	}
}

func TestIssueAccessTokenVerifyAccessToken_RoundTrip(t *testing.T) {
	t.Parallel()
	secret := []byte(testSecret)
	now := time.Now().UTC()
	tok, exp, err := IssueAccessToken(secret, "user-uuid-1", "a@b.co", now)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if tok == "" || !exp.After(now) {
		t.Fatalf("unexpected token or expiry: exp=%v", exp)
	}
	sub, email, err := VerifyAccessToken(secret, tok)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	if sub != "user-uuid-1" || email != "a@b.co" {
		t.Fatalf("claims: sub=%q email=%q", sub, email)
	}
}

func TestVerifyAccessToken_Expired(t *testing.T) {
	t.Parallel()
	secret := []byte(testSecret)
	claims := jwt.MapClaims{
		"sub":   "u1",
		"email": "e@x.co",
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		"exp":   time.Now().Add(-time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	_, _, err = VerifyAccessToken(secret, s)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyAccessToken_WrongSecret(t *testing.T) {
	t.Parallel()
	secret := []byte(testSecret)
	tok, _, err := IssueAccessToken(secret, "u1", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	_, _, err = VerifyAccessToken([]byte("wrong-secret-wrong-secret-wrong!"), tok)
	if err == nil {
		t.Fatal("expected verify failure with wrong secret")
	}
}

func TestIssueAccessToken_ShortSecretRejected(t *testing.T) {
	t.Parallel()
	_, _, err := IssueAccessToken([]byte("short"), "u1", "", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestNewOpaqueRefreshToken_HashRefreshToken(t *testing.T) {
	t.Parallel()
	a, err := NewOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("NewOpaqueRefreshToken: %v", err)
	}
	b, err := NewOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("NewOpaqueRefreshToken: %v", err)
	}
	if a == b {
		t.Fatal("expected distinct tokens")
	}
	if len(HashRefreshToken(a)) != 64 || len(HashRefreshToken(b)) != 64 {
		t.Fatalf("expected 64-char hex hashes, got %d and %d", len(HashRefreshToken(a)), len(HashRefreshToken(b)))
	}
}
