package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

func TestGetAgentWebhookForUser_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whunauth_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/agents/%d/webhook", agentID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgentWebhookForUser_OwnershipRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ownerUID := testhelper.GenerateUID(t)
	otherUID := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, otherUID, "whother_"+otherUID[:8])
	agentID := testhelper.InsertUserWithAgent(t, tx, ownerUID, "whowner_"+ownerUID[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/webhook", agentID), otherUID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owned agent, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgentWebhookForUser_Unconfigured(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whnone_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/webhook", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentWebhookStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Configured {
		t.Fatalf("expected configured=false, got %+v", resp)
	}
	assertNoTokenInWebhookResponse(t, w.Body.Bytes())
}

func TestPutAgentWebhookForUser_HappyPath(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whput_"+uid[:8])

	var receivedAuth string
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer hookSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = hookSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := []byte(fmt.Sprintf(`{"url":"%s/hooks","token":"hook-secret"}`, hookSrv.URL))
	r := authenticatedRequestWithBody(t, http.MethodPut, fmt.Sprintf("/agents/%d/webhook", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if receivedAuth != "Bearer hook-secret" {
		t.Fatalf("Authorization = %q", receivedAuth)
	}

	var resp agentWebhookStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Configured || resp.Test == nil || !resp.Test.Success {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	assertNoTokenInWebhookResponse(t, w.Body.Bytes())
}

func TestPutAgentWebhookForUser_InvalidPublicURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whpub_"+uid[:8])

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := []byte(`{"url":"http://8.8.8.8/hooks","token":"secret"}`)
	r := authenticatedRequestWithBody(t, http.MethodPut, fmt.Sprintf("/agents/%d/webhook", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	if errResp.Error.Code != ErrInvalidWebhookURL {
		t.Fatalf("code = %q, want %q", errResp.Error.Code, ErrInvalidWebhookURL)
	}
}

func TestPutAgentWebhookForUser_VaultUnavailable(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "wh503_"+uid[:8])

	deps := &Deps{DB: tx, Vault: nil, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := []byte(`{"url":"http://127.0.0.1:1/hooks","token":"secret"}`)
	r := authenticatedRequestWithBody(t, http.MethodPut, fmt.Sprintf("/agents/%d/webhook", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAgentWebhookForUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whdel_"+uid[:8])

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(ctx, tx, "agent_webhook", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetAgentWebhook(ctx, tx, agentID, "http://127.0.0.1:1/hooks", vaultID, "openclaw"); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/webhook", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp["cleared"] {
		t.Fatalf("expected cleared=true, got %+v", resp)
	}

	cfg, err := db.GetAgentWebhookConfig(ctx, tx, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil && cfg.WebhookURL != nil && *cfg.WebhookURL != "" {
		t.Fatal("expected webhook cleared from DB")
	}
}

func TestGetAgentWebhookForUser_TestWake(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "whtest_"+uid[:8])

	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer hookSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = hookSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(ctx, tx, "agent_webhook", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetAgentWebhook(ctx, tx, agentID, hookSrv.URL+"/hooks", vaultID, "openclaw"); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/webhook?test=true", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentWebhookStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Test == nil || !resp.Test.Success {
		t.Fatalf("expected successful test wake, got %+v", resp)
	}
	assertNoTokenInWebhookResponse(t, w.Body.Bytes())
}

func assertNoTokenInWebhookResponse(t *testing.T, body []byte) {
	t.Helper()
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["token"]; ok {
		t.Fatal("GET response must not include token field")
	}
}
