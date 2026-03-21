package coupon

import "testing"

func TestExpectedFreeProCouponHex(t *testing.T) {
	// md5("alice-mysecret") = 84835a9a9c497ac4856515e2bc46f714
	const secret = "mysecret"
	got := ExpectedFreeProCouponHex("alice", secret)
	want := "84835a9a9c497ac4856515e2bc46f714"
	if got != want {
		t.Errorf("ExpectedFreeProCouponHex = %q, want %q", got, want)
	}
}

func TestFreeProCouponMatches(t *testing.T) {
	if !FreeProCouponMatches("alice", "mysecret", "84835A9A9C497AC4856515E2BC46F714") {
		t.Error("expected match (case-insensitive hex)")
	}
	if FreeProCouponMatches("alice", "mysecret", "84835a9a9c497ac4856515e2bc46f713") {
		t.Error("expected mismatch for wrong digest")
	}
	if FreeProCouponMatches("alice", "mysecret", "not-hex") {
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
