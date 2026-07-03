package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

func TestPutAgentWebhook_RejectsPublicURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"url":"http://8.8.8.8/hooks","token":"secret"}`
	r := signedJSONRequest(t, http.MethodPut, "/agent/webhook", body, privKey, agentID)
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

func TestPutAgentWebhook_SuccessWithTestDelivery(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	var receivedAuth string
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/hooks/wake" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer hookSrv.Close()

	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = hookSrv.Client()
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"url":"` + hookSrv.URL + `/hooks","token":"hook-secret"}`
	r := signedJSONRequest(t, http.MethodPut, "/agent/webhook", body, privKey, agentID)
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
}

func TestNotifyAgentWake_DispatchesToHooks(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, _, _ := GenerateEd25519OpenSSHKey()
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
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, hookSrv.URL+"/hooks", vaultID); err != nil {
		t.Fatal(err)
	}

	approvalID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApproval(t, tx, approvalID, agentID, uid)
	appr, err := db.GetApprovalByID(context.Background(), tx, approvalID)
	if err != nil || appr == nil {
		t.Fatalf("GetApprovalByID: %v", err)
	}

	deps := &Deps{DB: tx, Vault: mockVault}
	notifyAgentApprovalResolved(deps, appr)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("wake webhook not received")
	}
}

func TestValidatePrivateNetworkURL_LocalhostAllowed(t *testing.T) {
	if err := connectors.ValidatePrivateNetworkURL("http://127.0.0.1:1/hooks", "url"); err != nil {
		t.Fatal(err)
	}
}
