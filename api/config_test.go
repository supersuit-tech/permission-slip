package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestGetConfig_BillingDisabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "cfg_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: false}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/config", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp configResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.BillingEnabled {
		t.Error("expected billing_enabled=false when BillingEnabled is false")
	}
	if !resp.BYOAEnabled {
		t.Error("expected byoa_enabled=true when BillingEnabled is false (self-hosted)")
	}
}

func TestGetConfig_BillingEnabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "cfg_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/config", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp configResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.BillingEnabled {
		t.Error("expected billing_enabled=true when BillingEnabled is true")
	}
	if resp.BYOAEnabled {
		t.Error("expected byoa_enabled=false when BillingEnabled is true (hosted)")
	}
}

func TestGetConfig_Unauthenticated(t *testing.T) {
	t.Parallel()

	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
