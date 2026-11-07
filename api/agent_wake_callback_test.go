package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

func TestRequestApproval_WithCallbackURL_StoresAndWakesPerRequest(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	defaultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("wake should not hit default registered webhook URL")
		w.WriteHeader(http.StatusOK)
	}))
	defer defaultSrv.Close()

	var perRequestHits int
	perRequestSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		perRequestHits++
		w.WriteHeader(http.StatusOK)
	}))
	defer perRequestSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = perRequestSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(context.Background(), tx, "test", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, defaultSrv.URL+"/hooks", vaultID, "openclaw"); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	perRequestURL := perRequestSrv.URL + "/hooks"
	body := `{"request_id":"` + testhelper.GenerateID(t, "req_") + `","action":{"type":"email.send","parameters":{"to":"a@b.com","subject":"s","body":"b"}},"context":{"description":"x"},"callback_url":"` + perRequestURL + `"}`
	req := signedJSONRequest(t, http.MethodPost, "/approvals/request", body, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if !createResp.PushWakeConfigured {
		t.Fatal("expected push_wake_configured with per-request callback_url")
	}

	appr, err := db.GetApprovalByIDAndAgent(context.Background(), tx, createResp.ApprovalID, agentID)
	if err != nil || appr == nil || appr.WakeCallbackURL == nil || *appr.WakeCallbackURL != perRequestURL {
		t.Fatalf("stored callback: %v err=%v", appr, err)
	}

	notifyAgentApprovalResolved(deps, appr)

	deadline := time.Now().Add(5 * time.Second)
	for perRequestHits == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if perRequestHits != 1 {
		t.Fatalf("per-request webhook hits = %d, want 1", perRequestHits)
	}
}

func TestRequestApproval_CallbackURLWithoutRegisteredWebhook(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	done := make(chan struct{}, 1)
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case done <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer hookSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = hookSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(context.Background(), tx, "test", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	// Register token only — no default callback URL (simulates hostname-only preregistration removal).
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, hookSrv.URL+"/hooks", vaultID, "openclaw"); err != nil {
		t.Fatal(err)
	}
	testhelper.MustExec(t, tx, `UPDATE agents SET webhook_url = NULL WHERE agent_id = $1`, agentID)

	hasWebhook, err := db.AgentHasWebhook(context.Background(), tx, agentID)
	if err != nil || hasWebhook {
		t.Fatalf("AgentHasWebhook = %v, err=%v", hasWebhook, err)
	}

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	callbackURL := hookSrv.URL + "/hooks"
	body := `{"request_id":"` + testhelper.GenerateID(t, "req_") + `","action":{"type":"email.send","parameters":{"to":"a@b.com","subject":"s","body":"b"}},"context":{"description":"x"},"callback_url":"` + callbackURL + `"}`
	req := signedJSONRequest(t, http.MethodPost, "/approvals/request", body, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if !createResp.PushWakeConfigured {
		t.Fatal("expected push_wake_configured from callback_url alone")
	}

	appr, _ := db.GetApprovalByIDAndAgent(context.Background(), tx, createResp.ApprovalID, agentID)
	notifyAgentApprovalResolved(deps, appr)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("wake webhook not received for per-request-only callback")
	}
}

func TestRequestApproval_RejectsPublicCallbackURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"request_id":"` + testhelper.GenerateID(t, "req_") + `","action":{"type":"email.send","parameters":{"to":"a@b.com","subject":"s","body":"b"}},"context":{},"callback_url":"http://8.8.8.8/hooks"}`
	req := signedJSONRequest(t, http.MethodPost, "/approvals/request", body, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequestApproval_DelayedWakeUsesStoredCallbackURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, _, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	var receivedPath string
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer hookSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = hookSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(context.Background(), tx, "test", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	storedCallback := hookSrv.URL + "/hooks"
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, "http://127.0.0.1:9/hooks", vaultID, "openclaw"); err != nil {
		t.Fatal(err)
	}

	approvalID := testhelper.GenerateID(t, "appr_")
	testhelper.MustExec(t, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, wake_callback_url)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{}', 'pending', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+1 hour'), $4)`,
		approvalID, agentID, uid, storedCallback)

	appr, err := db.GetApprovalByID(context.Background(), tx, approvalID)
	if err != nil || appr == nil {
		t.Fatal(err)
	}

	deps := &Deps{DB: tx, Vault: mockVault}
	NotifyAgentApprovalResolvedSync(deps, appr)

	deadline := time.Now().Add(5 * time.Second)
	for receivedPath == "" && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if receivedPath != "/hooks/wake" {
		t.Fatalf("wake path = %q, want /hooks/wake", receivedPath)
	}
}
