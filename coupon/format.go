package coupon

import "regexp"

var freeProCouponHexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidFreeProCouponHexFormat reports whether s is exactly 64 lowercase hex digits.
func ValidFreeProCouponHexFormat(s string) bool {
	return freeProCouponHexPattern.MatchString(s)
}
