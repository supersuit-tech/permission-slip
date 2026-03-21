package api

import (
	"context"
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

func TestRedeemFreeProCoupon_FromPayAsYouGo_WithoutStripeSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "paidcoupon_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	secret := "test-coupon-secret-paid"
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

	sub, err := db.GetSubscriptionByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanFreePro {
		t.Errorf("DB plan_id: want %s, got %s", db.PlanFreePro, sub.PlanID)
	}
}

func TestRedeemFreeProCoupon_FromPayAsYouGo_RequiresStripeClient(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "needstripe_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)
	custID := "cus_redeem_test"
	subID := "sub_redeem_test"
	if _, err := db.UpdateSubscriptionStripe(context.Background(), tx, uid, &custID, &subID); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	secret := "test-coupon-secret-stripe"
	good := coupon.ExpectedFreeProCouponHex(username, secret)

	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		BillingEnabled:    true,
		CouponSecret:      secret,
		Stripe:            nil,
	}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/redeem-coupon", uid, `{"coupon":"`+good+`"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
