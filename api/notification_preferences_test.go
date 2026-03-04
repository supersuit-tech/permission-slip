package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestUpdateNotificationPreferences_EnableSMS_FreeTier_Rejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"sms","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrSMSRequiresPaidPlan {
		t.Errorf("expected error code %q, got %q", ErrSMSRequiresPaidPlan, resp.Error.Code)
	}
	// Verify error includes plan details for the frontend.
	if resp.Error.Details == nil {
		t.Fatal("expected error details with plan info")
	}
	if resp.Error.Details["current_plan"] != db.PlanFree {
		t.Errorf("expected current_plan=%q, got %v", db.PlanFree, resp.Error.Details["current_plan"])
	}
	if resp.Error.Details["required_plan"] != db.PlanPayAsYouGo {
		t.Errorf("expected required_plan=%q, got %v", db.PlanPayAsYouGo, resp.Error.Details["required_plan"])
	}
}

func TestUpdateNotificationPreferences_EnableSMS_PaidTier_Allowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"sms","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_DisableSMS_FreeTier_Allowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Disabling SMS should work even on free tier (no plan check needed).
	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"sms","enabled":false}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_EnableSMS_NoSubscription_Rejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	// No subscription row — should be treated as not allowed.

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"sms","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_EnableEmail_FreeTier_Allowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Non-SMS channels should not be plan-gated.
	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"email","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_MixedChannels_FreeTier_SMSBlocked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Enabling email (allowed) + SMS (blocked) in a single request should fail.
	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"email","enabled":true},{"channel":"sms","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetNotificationPreferences_SMSAvailable_PaidTier(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile/notification-preferences", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp notificationPreferencesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, p := range resp.Preferences {
		if !p.Available {
			t.Errorf("expected all channels available on paid plan, got %q unavailable", p.Channel)
		}
	}
}

func TestGetNotificationPreferences_SMSUnavailable_FreeTier(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile/notification-preferences", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp notificationPreferencesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, p := range resp.Preferences {
		if p.Channel == "sms" {
			if p.Available {
				t.Error("expected SMS unavailable on free tier")
			}
		} else {
			if !p.Available {
				t.Errorf("expected %q available on free tier", p.Channel)
			}
		}
	}
}

func TestUpdateNotificationPreferences_EnableMobilePush_FreeTier_Allowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// mobile-push is not plan-gated, so it should work on the free tier.
	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"mobile-push","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetNotificationPreferences_IncludesMobilePush(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile/notification-preferences", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp notificationPreferencesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	found := false
	for _, p := range resp.Preferences {
		if p.Channel == "mobile-push" {
			found = true
			if !p.Available {
				t.Error("expected mobile-push to be available on free tier")
			}
			if !p.Enabled {
				t.Error("expected mobile-push to default to enabled")
			}
		}
	}
	if !found {
		t.Error("expected mobile-push channel in preferences response")
	}
}
