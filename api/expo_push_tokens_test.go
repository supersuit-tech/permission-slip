package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestCreateExpoPushToken_Valid(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":"ExponentPushToken[abc123def456]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp expoPushTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token != "ExponentPushToken[abc123def456]" {
		t.Errorf("expected token in response, got %q", resp.Token)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestCreateExpoPushToken_ExpoPushTokenPrefix(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":"ExpoPushToken[abc123def456]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for ExpoPushToken prefix, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpoPushToken_EmptyToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":""}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpoPushToken_InvalidPrefix(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":"not-a-valid-token"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid prefix, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpoPushToken_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"token":"ExponentPushToken[idempotent]"}`

	// First registration
	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("first: expected 201, got %d", w.Code)
	}

	// Second registration — same token, should succeed
	r = authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid, body)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("second: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List should show only 1 token
	r = authenticatedRequest(t, http.MethodGet, "/expo-push-tokens", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	var listResp expoPushTokenListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listResp.Tokens) != 1 {
		t.Errorf("expected 1 token after idempotent upsert, got %d", len(listResp.Tokens))
	}
}

func TestListExpoPushTokens_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/expo-push-tokens", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp expoPushTokenListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(resp.Tokens))
	}
}

func TestDeleteExpoPushToken_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create a token first
	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":"ExponentPushToken[todelete]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	var created expoPushTokenResponse
	json.Unmarshal(w.Body.Bytes(), &created)

	// Delete it
	r = authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/expo-push-tokens/%d", created.ID), uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	r = authenticatedRequest(t, http.MethodGet, "/expo-push-tokens", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	var listResp expoPushTokenListResponse
	json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Tokens) != 0 {
		t.Errorf("expected 0 tokens after delete, got %d", len(listResp.Tokens))
	}
}

func TestDeleteExpoPushToken_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/expo-push-tokens/99999", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteExpoPushToken_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// User1 creates a token
	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid1,
		`{"token":"ExponentPushToken[user1device]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	var created expoPushTokenResponse
	json.Unmarshal(w.Body.Bytes(), &created)

	// User2 tries to delete it — should get 404 (not found for their scope)
	r = authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/expo-push-tokens/%d", created.ID), uid2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteExpoPushToken_InvalidID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/expo-push-tokens/not-a-number", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpoPushToken_WhitespaceTrimmed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Token with leading/trailing whitespace should be trimmed
	r := authenticatedJSONRequest(t, http.MethodPost, "/expo-push-tokens", uid,
		`{"token":"  ExponentPushToken[trimtest]  "}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp expoPushTokenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token != "ExponentPushToken[trimtest]" {
		t.Errorf("expected trimmed token, got %q", resp.Token)
	}
}
