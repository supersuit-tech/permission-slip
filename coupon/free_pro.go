package coupon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ExpectedFreeProCouponHex returns the lowercase hex HMAC-SHA256(secret, username)
// for the trimmed profile username. The coupon acts as a capability token — keep
// COUPON_SECRET long and random.
func ExpectedFreeProCouponHex(username, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.TrimSpace(username)))
	return hex.EncodeToString(mac.Sum(nil))
}

// FreeProCouponMatches reports whether couponHex equals the expected digest
// for the username and secret. Comparison is constant-time on equal-length
// hex strings; lengths are normalized first.
func FreeProCouponMatches(username, secret, couponHex string) bool {
	want := ExpectedFreeProCouponHex(username, secret)
	got := strings.TrimSpace(strings.ToLower(couponHex))
	if len(got) != len(want) {
		return false
	}
	var v byte
	for i := 0; i < len(want); i++ {
		v |= want[i] ^ got[i]
	}
	return v == 0
}
