package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestCreatePushSubscription_WebPush(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"endpoint":"https://fcm.googleapis.com/fcm/send/abc","p256dh":"test_key","auth":"test_auth"}`)
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
	if resp.Channel != "web-push" {
		t.Errorf("expected channel web-push, got %q", resp.Channel)
	}
	if resp.Endpoint == nil || *resp.Endpoint != "https://fcm.googleapis.com/fcm/send/abc" {
		t.Errorf("expected endpoint, got %v", resp.Endpoint)
	}
	if resp.ExpoToken != nil {
		t.Errorf("expected nil expo_token, got %v", resp.ExpoToken)
	}
}

func TestCreatePushSubscription_WebPush_ExplicitType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"web-push","endpoint":"https://fcm.googleapis.com/fcm/send/abc2","p256dh":"k","auth":"a"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
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
	if resp.Endpoint != nil {
		t.Errorf("expected nil endpoint for expo, got %v", resp.Endpoint)
	}
}

func TestCreatePushSubscription_Expo_ExpoPushTokenPrefix(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExpoPushToken[abc123xyz]"}`)
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

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
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

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
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

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"apns","expo_token":"ExponentPushToken[x]"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListPushSubscriptions_MixedTypes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create web-push
	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"endpoint":"https://push.example.com/sub1","p256dh":"k","auth":"a"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create web-push: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create expo
	r = authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"type":"expo","expo_token":"ExponentPushToken[list_test]"}`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create expo: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List both
	r = authenticatedRequest(t, http.MethodGet, "/push-subscriptions", uid)
	w = httptest.NewRecorder()
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

	// Verify we have one of each type
	channels := map[string]int{}
	for _, s := range listResp.Subscriptions {
		channels[s.Channel]++
	}
	if channels["web-push"] != 1 {
		t.Errorf("expected 1 web-push, got %d", channels["web-push"])
	}
	if channels["mobile-push"] != 1 {
		t.Errorf("expected 1 mobile-push, got %d", channels["mobile-push"])
	}
}

func TestCreatePushSubscription_WebPush_MissingEndpoint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/push-subscriptions", uid,
		`{"p256dh":"k","auth":"a"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePushSubscription_Expo_Upsert(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
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
