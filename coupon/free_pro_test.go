package coupon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func expectedHMAC(username, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.TrimSpace(username)))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestExpectedFreeProCouponHex(t *testing.T) {
	const secret = "mysecret"
	got := ExpectedFreeProCouponHex("alice", secret)
	want := expectedHMAC("alice", secret)
	if got != want {
		t.Errorf("ExpectedFreeProCouponHex = %q, want %q", got, want)
	}
	if len(got) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(got))
	}
}

func TestFreeProCouponMatches(t *testing.T) {
	const secret = "mysecret"
	good := expectedHMAC("alice", secret)
	if !FreeProCouponMatches("alice", secret, strings.ToUpper(good)) {
		t.Error("expected match (case-insensitive hex)")
	}
	if FreeProCouponMatches("alice", secret, good[:63]+"0") {
		t.Error("expected mismatch for wrong digest")
	}
	if FreeProCouponMatches("alice", secret, "not-hex") {
		t.Error("expected mismatch for wrong length")
	}
}

func TestExpectedFreeProCouponHexTrimsUsername(t *testing.T) {
	a := ExpectedFreeProCouponHex("  bob  ", "x")
	b := ExpectedFreeProCouponHex("bob", "x")
	if a != b {
		t.Errorf("trim inconsistency: %q vs %q", a, b)
	}
}
