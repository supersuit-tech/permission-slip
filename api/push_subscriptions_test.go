package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestIsValidExpoToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		token string
		valid bool
	}{
		{"ExponentPushToken[abc123]", true},
		{"ExpoPushToken[abc123]", true},
		{"ExponentPushToken[a]", true},
		{"ExponentPushToken[]", false},         // empty content
		{"ExpoPushToken[]", false},              // empty content
		{"ExponentPushToken[no_bracket", false}, // missing closing bracket
		{"not-a-token", false},
		{"", false},
		{"ExponentPushToken", false},   // no brackets at all
		{"exponentpushtoken[abc]", false}, // wrong case
	}

	for _, tt := range tests {
		if got := isValidExpoToken(tt.token); got != tt.valid {
			t.Errorf("isValidExpoToken(%q) = %v, want %v", tt.token, got, tt.valid)
		}
	}
}

func TestCreatePushSubscription_Expo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[abc123xyz]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp pushSubscriptionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Channel != "mobile-push" {
		t.Errorf("expected channel mobile-push, got %q", resp.Channel)
	}
	if resp.ExpoToken == nil || *resp.ExpoToken != "ExponentPushToken[abc123xyz]" {
		t.Errorf("expected expo_token, got %v", resp.ExpoToken)
	}
}

func TestCreatePushSubscription_Expo_ExpoPushTokenPrefix(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExpoPushToken[abc123xyz]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_MobilePushTypeAlias(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"mobile-push","expo_token":"ExponentPushToken[alias_test]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp pushSubscriptionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Channel != "mobile-push" {
		t.Errorf("expected channel mobile-push, got %q", resp.Channel)
	}
}

func TestCreatePushSubscription_NoType_DefaultsToExpo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"expo_token":"ExponentPushToken[default_type]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo_MissingToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo_InvalidToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"not-a-valid-token"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_InvalidType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"apns","expo_token":"ExponentPushToken[x]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListPushSubscriptions(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create two expo tokens
	for _, token := range []string{"ExponentPushToken[list1]", "ExponentPushToken[list2]"} {
		r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
			`{"type":"expo","expo_token":"`+token+`"}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d: %s", token, w.Code, w.Body.String())
		}
	}

	r := authenticatedRequest(t, http.MethodGet, "/push-subscriptions", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var listResp pushSubscriptionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listResp.Subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(listResp.Subscriptions))
	}
	for _, s := range listResp.Subscriptions {
		if s.Channel != "mobile-push" {
			t.Errorf("expected mobile-push channel, got %q", s.Channel)
		}
	}
}

func TestListPushSubscriptions_ChannelFilter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[filter_test]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create expo: %d: %s", w.Code, w.Body.String())
	}

	// Filter by mobile-push
	r = authenticatedRequest(t, http.MethodGet, "/push-subscriptions?channel=mobile-push", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("filter mobile-push: %d: %s", w.Code, w.Body.String())
	}
	var listResp pushSubscriptionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listResp.Subscriptions) != 1 || listResp.Subscriptions[0].Channel != "mobile-push" {
		t.Errorf("expected 1 mobile-push, got %d", len(listResp.Subscriptions))
	}

	// Invalid channel filter
	r = authenticatedRequest(t, http.MethodGet, "/push-subscriptions?channel=invalid", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid channel, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnregisterExpoPushToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Register a token
	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[unreg_test]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d: %s", w.Code, w.Body.String())
	}

	// Unregister by token
	r = authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions/unregister", uid,
		`{"expo_token":"ExponentPushToken[unreg_test]"}`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("unregister: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	r = authenticatedRequest(t, http.MethodGet, "/push-subscriptions?channel=mobile-push", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	var listResp pushSubscriptionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listResp.Subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions after unregister, got %d", len(listResp.Subscriptions))
	}
}

func TestUnregisterExpoPushToken_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions/unregister", uid,
		`{"expo_token":"ExponentPushToken[nonexistent]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnregisterExpoPushToken_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// user1 registers a token
	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid1,
		`{"type":"expo","expo_token":"ExponentPushToken[user1_token]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d: %s", w.Code, w.Body.String())
	}

	// user2 should not be able to unregister user1's token
	r = authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions/unregister", uid2,
		`{"expo_token":"ExponentPushToken[user1_token]"}`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo_Upsert(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create expo token
	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[upsert_test]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Same token again should succeed (upsert)
	r = authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[upsert_test]"}`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("upsert: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Should still have just 1 subscription
	r = authenticatedRequest(t, http.MethodGet, "/push-subscriptions", uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	var listResp pushSubscriptionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listResp.Subscriptions) != 1 {
		t.Errorf("expected 1 subscription after upsert, got %d", len(listResp.Subscriptions))
	}
}

func TestCreatePushSubscription_Expo_MissingClosingBracket(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[no_closing_bracket"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo_EmptyBrackets(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty brackets, got %d: %s", w.Code, w.Body.String())
	}
}
