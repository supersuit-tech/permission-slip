package coupon

import (
	"crypto/md5" //nolint:gosec // MD5 is required for deterministic per-username coupon codes (not a security primitive).
	"encoding/hex"
	"strings"
)

// ExpectedFreeProCouponHex returns the lowercase hex MD5 digest for
// md5(username + "-" + secret), matching the redeem API. Username is trimmed;
// secret is used as-is (including empty, which yields a predictable digest — do
// not deploy with an empty COUPON_SECRET).
func ExpectedFreeProCouponHex(username, secret string) string {
	s := strings.TrimSpace(username) + "-" + secret
	sum := md5.Sum([]byte(s)) //nolint:gosec
	return hex.EncodeToString(sum[:])
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
