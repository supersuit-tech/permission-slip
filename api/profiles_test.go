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

func TestGetProfile_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected error code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
}

func TestGetProfile_TraceIDPresent(t *testing.T) {
	t.Parallel()
	// Verify trace ID middleware runs before session auth
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.TraceID == "" {
		t.Error("expected trace_id in error response")
	}
}

func TestGetProfile_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ID != uid {
		t.Errorf("expected id %q, got %q", uid, resp.ID)
	}
	if resp.Username != "u_"+uid[:8] {
		t.Errorf("expected username %q, got %q", "u_"+uid[:8], resp.Username)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	// No profile inserted for this user

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrProfileNotFound {
		t.Errorf("expected error code %q, got %q", ErrProfileNotFound, errResp.Error.Code)
	}
}

func TestGetProfile_NilDB(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret} // DB is nil
	router := NewRouter(deps)

	uid := testhelper.GenerateUID(t)
	r := authenticatedRequest(t, http.MethodGet, "/profile", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserID_WithContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-id")
	uid := UserID(ctx)
	if uid != "test-user-id" {
		t.Errorf("expected 'test-user-id', got %q", uid)
	}
}

// --- PATCH /profile tests ---

func TestUpdateProfile_SetEmailAndPhone(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"email":"alice@example.com","phone":"+15551234567"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Email == nil || *resp.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %v", resp.Email)
	}
	if resp.Phone == nil || *resp.Phone != "+15551234567" {
		t.Errorf("expected phone '+15551234567', got %v", resp.Phone)
	}
}

func TestUpdateProfile_PartialUpdate_OnlyEmail(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Set initial phone
	phone := "+15559999999"
	if err := db.UpdateProfileFields(context.Background(), tx, uid, nil, &phone, nil); err != nil {
		t.Fatalf("setup: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Only send email — phone should remain unchanged
	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"email":"new@example.com"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Email == nil || *resp.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got %v", resp.Email)
	}
	if resp.Phone == nil || *resp.Phone != "+15559999999" {
		t.Errorf("expected phone '+15559999999' to be preserved, got %v", resp.Phone)
	}
}

func TestUpdateProfile_ClearField_WithNull(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Set initial email
	email := "old@example.com"
	if err := db.UpdateProfileFields(context.Background(), tx, uid, &email, nil, nil); err != nil {
		t.Fatalf("setup: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Send null to clear email
	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"email":null}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Email != nil {
		t.Errorf("expected email to be nil (cleared), got %v", resp.Email)
	}
}

func TestUpdateProfile_InvalidEmail(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"email":"not-an-email"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfile_InvalidPhone(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"phone":"12345"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfile_WrongType_RejectsNumber(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Send a number instead of a string — should get a 400, not silently ignored.
	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"email":42}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong type, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Notification preferences tests ---

func TestGetNotificationPreferences_Defaults(t *testing.T) {
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
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Preferences) != 4 {
		t.Fatalf("expected 4 preferences, got %d", len(resp.Preferences))
	}
	// All channels default to enabled when no preferences are set.
	// SMS should not be available on the free plan.
	for _, p := range resp.Preferences {
		if !p.Enabled {
			t.Errorf("expected channel %q to default to enabled", p.Channel)
		}
		if p.Channel == "sms" && p.Available {
			t.Error("expected SMS to be unavailable on free plan")
		}
		if p.Channel != "sms" && !p.Available {
			t.Errorf("expected channel %q to be available", p.Channel)
		}
	}
}

func TestUpdateNotificationPreferences_Toggle(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Disable email
	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"email","enabled":false}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp notificationPreferencesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, p := range resp.Preferences {
		switch p.Channel {
		case "email":
			if p.Enabled {
				t.Error("expected email to be disabled")
			}
		case "web-push", "sms":
			if !p.Enabled {
				t.Errorf("expected %q to remain enabled", p.Channel)
			}
		}
	}
}

func TestUpdateNotificationPreferences_InvalidChannel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"carrier-pigeon","enabled":true}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_EmptyPreferences(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNotificationPreferences_DuplicateChannel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPut, "/profile/notification-preferences", uid,
		`{"preferences":[{"channel":"email","enabled":true},{"channel":"email","enabled":false}]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Marketing opt-in tests ---

func TestUpdateProfile_SetMarketingOptIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"marketing_opt_in":true}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !resp.MarketingOptIn {
		t.Error("expected marketing_opt_in to be true")
	}
}

func TestUpdateProfile_MarketingOptIn_NullRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/profile", uid,
		`{"marketing_opt_in":null}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null marketing_opt_in, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetProfile_IncludesMarketingOptIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/profile", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// Default should be false.
	if resp.MarketingOptIn {
		t.Error("expected marketing_opt_in to default to false")
	}
}
