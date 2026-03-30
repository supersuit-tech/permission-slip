package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestOnboarding_BillingDisabled_AssignsPayAsYouGo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: false}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid,
		`{"username":"billing_off"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	sub, err := db.GetSubscriptionByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription to be created during onboarding, got nil")
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan_id=%s when billing disabled, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
}

func TestOnboarding_BillingEnabled_AssignsFreePlan(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid,
		`{"username":"billing_on"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	sub, err := db.GetSubscriptionByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription to be created during onboarding, got nil")
	}
	if sub.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s when billing enabled, got %s", db.PlanFree, sub.PlanID)
	}
}
