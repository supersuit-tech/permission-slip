package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/coupon"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestRedeemFreeProCoupon_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "couponuser_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	secret := "test-coupon-secret"
	good := coupon.ExpectedFreeProCouponHex(username, secret)

	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		BillingEnabled:    true,
		CouponSecret:      secret,
	}
	router := NewRouter(deps)

	body := `{"coupon":"` + good + `"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/redeem-coupon", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp redeemCouponResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.PlanID != db.PlanFreePro || resp.Status != "redeemed" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRedeemFreeProCoupon_Invalid(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		BillingEnabled:    true,
		CouponSecret:      "secret",
	}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/redeem-coupon", uid, `{"coupon":"deadbeef"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRedeemFreeProCoupon_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "twice_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)
	testhelper.InsertSubscription(t, tx, uid, db.PlanFreePro)

	secret := "s"
	good := coupon.ExpectedFreeProCouponHex(username, secret)

	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		BillingEnabled:    true,
		CouponSecret:      secret,
	}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/redeem-coupon", uid, `{"coupon":"`+good+`"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp redeemCouponResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "already_redeemed" {
		t.Fatalf("expected already_redeemed, got %+v", resp)
	}
}
