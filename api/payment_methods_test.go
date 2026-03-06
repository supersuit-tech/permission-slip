package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestListPaymentMethods_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pm_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/payment-methods", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp paymentMethodListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.PaymentMethods) != 0 {
		t.Errorf("expected 0 payment methods, got %d", len(resp.PaymentMethods))
	}
}

func TestListPaymentMethods_ReturnsUserMethods(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pm_"+uid[:8])

	// Create payment methods directly in DB.
	_, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_stripe_" + uid[:8],
		Label:                 "My Card",
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2027,
		IsDefault:             true,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/payment-methods", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp paymentMethodListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.PaymentMethods) != 1 {
		t.Fatalf("expected 1 payment method, got %d", len(resp.PaymentMethods))
	}
	if resp.PaymentMethods[0].Brand != "visa" {
		t.Errorf("expected brand=visa, got %q", resp.PaymentMethods[0].Brand)
	}
	if resp.PaymentMethods[0].Last4 != "4242" {
		t.Errorf("expected last4=4242, got %q", resp.PaymentMethods[0].Last4)
	}
	if !resp.PaymentMethods[0].IsDefault {
		t.Error("expected is_default=true")
	}
}

func TestListPaymentMethods_IsolatedPerUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "pm1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "pm2_"+uid2[:8])

	_, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid1,
		StripePaymentMethodID: "pm_iso_" + uid1[:8],
		Brand:                 "visa",
		Last4:                 "1234",
		ExpMonth:              6,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// User2 should see 0 methods.
	r := authenticatedRequest(t, http.MethodGet, "/payment-methods", uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp paymentMethodListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.PaymentMethods) != 0 {
		t.Errorf("expected 0 payment methods for user2, got %d", len(resp.PaymentMethods))
	}
}

func TestUpdatePaymentMethod_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmup_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_up_" + uid[:8],
		Label:                 "Old Label",
		Brand:                 "visa",
		Last4:                 "9999",
		ExpMonth:              3,
		ExpYear:               2029,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"label":"New Label","per_transaction_limit":5000}`
	r := authenticatedJSONRequest(t, http.MethodPatch, "/payment-methods/"+pm.ID, uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp paymentMethodResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Label != "New Label" {
		t.Errorf("expected label='New Label', got %q", resp.Label)
	}
	if resp.PerTransactionLimit == nil || *resp.PerTransactionLimit != 5000 {
		t.Errorf("expected per_transaction_limit=5000, got %v", resp.PerTransactionLimit)
	}
}

func TestUpdatePaymentMethod_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmup404_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"label":"test"}`
	r := authenticatedJSONRequest(t, http.MethodPatch, "/payment-methods/00000000-0000-0000-0000-000000000000", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePaymentMethod_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmdel404_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/payment-methods/00000000-0000-0000-0000-000000000000", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePaymentMethod_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmdel_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_del_" + uid[:8],
		Brand:                 "mastercard",
		Last4:                 "7777",
		ExpMonth:              9,
		ExpYear:               2030,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	// No Stripe client configured = detach is skipped (best-effort).
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/payment-methods/"+pm.ID, uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deletePaymentMethodResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Deleted {
		t.Error("expected deleted=true")
	}

	// Verify it's gone.
	fetched, err := db.GetPaymentMethodByID(ctx, tx, uid, pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodByID: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil after delete")
	}
}

func TestUpdatePaymentMethod_NegativeLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmneglim_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_neg_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "1111",
		ExpMonth:              6,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"per_transaction_limit":-100}`
	r := authenticatedJSONRequest(t, http.MethodPatch, "/payment-methods/"+pm.ID, uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePaymentMethod_PerTxExceedsMonthly(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmexceed_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_exceed_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "2222",
		ExpMonth:              6,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"per_transaction_limit":10000,"monthly_limit":5000}`
	r := authenticatedJSONRequest(t, http.MethodPatch, "/payment-methods/"+pm.ID, uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListPaymentMethods_IncludesMaxAllowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmmax_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/payment-methods", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp paymentMethodListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.MaxAllowed != maxPaymentMethodsPerUser {
		t.Errorf("expected max_allowed=%d, got %d", maxPaymentMethodsPerUser, resp.MaxAllowed)
	}
}

func TestConfirmPaymentMethod_InvalidStripeID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pminvid_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"payment_method_id":"not_a_valid_pm_id"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/payment-methods", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid Stripe PM ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConfirmPaymentMethod_LabelTooLong(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmlabel_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	longLabel := make([]byte, 200)
	for i := range longLabel {
		longLabel[i] = 'a'
	}
	body := `{"payment_method_id":"pm_test","label":"` + string(longLabel) + `"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/payment-methods", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long label, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSetupIntent_NoStripe(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmsi_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/payment-methods/setup-intent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConfirmPaymentMethod_NoStripe(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmconf_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, Stripe: nil}
	router := NewRouter(deps)

	body := `{"payment_method_id":"pm_test"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/payment-methods", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConfirmPaymentMethod_MissingID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmconf2_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/payment-methods", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
