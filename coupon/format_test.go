package coupon

import "testing"

func TestValidFreeProCouponHexFormat(t *testing.T) {
	good := "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
	if !ValidFreeProCouponHexFormat(good) {
		t.Error("expected valid")
	}
	if ValidFreeProCouponHexFormat("ABCdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789") {
		t.Error("uppercase should be invalid for strict check")
	}
	if ValidFreeProCouponHexFormat("short") {
		t.Error("short string invalid")
	}
	if ValidFreeProCouponHexFormat(good + "0") {
		t.Error("too long invalid")
	}
}
