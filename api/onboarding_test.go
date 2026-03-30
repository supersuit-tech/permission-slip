package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestOnboarding_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid,
		`{"username":"alice"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp onboardingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ID != uid {
		t.Errorf("expected id %q, got %q", uid, resp.ID)
	}
	if resp.Username != "alice" {
		t.Errorf("expected username %q, got %q", "alice", resp.Username)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestOnboarding_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "alice2")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Calling onboarding again for an existing user should return the existing profile
	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid,
		`{"username":"different"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for idempotent call, got %d: %s", w.Code, w.Body.String())
	}

	var resp onboardingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	// Should return existing username, not the new one
	if resp.Username != "alice2" {
		t.Errorf("expected original username %q, got %q", "alice2", resp.Username)
	}
}

func TestOnboarding_UsernameTaken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "taken")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid2,
		`{"username":"taken"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrConstraintViolation {
		t.Errorf("expected %q, got %q", ErrConstraintViolation, errResp.Error.Code)
	}
}

func TestOnboarding_InvalidUsername(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		username string
		wantMsg  string
	}{
		{"too short", "ab", "at least 3"},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "at most 32"},
		{"invalid chars", "hello world", "only contain"},
		{"starts with underscore", "_bad", "must start with a letter"},
		{"unicode cyrillic", "аlice", "ASCII"},   // Cyrillic 'а' (U+0430) — homoglyph attack
		{"unicode digits", "１２３abc", "ASCII"},  // fullwidth digits
		{"unicode letters", "café", "ASCII"},      // Latin with diacritical mark
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tx := testhelper.SetupTestDB(t)
			uid := testhelper.GenerateUID(t)

			deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
			router := NewRouter(deps)

			body, _ := json.Marshal(map[string]string{"username": tc.username})
			r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid, string(body))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestOnboarding_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/onboarding", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOnboarding_NilDB(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	uid := testhelper.GenerateUID(t)
	r := authenticatedJSONRequest(t, http.MethodPost, "/onboarding", uid, `{"username":"alice"}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
