package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/agentwake"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func grokBotOKClient(t *testing.T, capture func(*http.Request, []byte)) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			if capture != nil {
				capture(r, body)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}
}

func TestPutAgentWebhook_GrokBotAcceptsCursorURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	webhookURL := "https://api2.cursor.sh/automations/webhook/wh_test"
	var gotURL, gotAuth string
	var gotBody []byte
	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = grokBotOKClient(t, func(r *http.Request, body []byte) {
		gotURL = r.URL.String()
		gotAuth = r.Header.Get("Authorization")
		gotBody = body
	})
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"provider":"grokbot","url":"` + webhookURL + `","token":"cursor-wh-key"}`
	r := signedJSONRequest(t, http.MethodPut, "/agent/webhook", body, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotURL != webhookURL {
		t.Fatalf("delivered URL = %q, want as-is %q", gotURL, webhookURL)
	}
	if gotAuth != "Bearer cursor-wh-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["source"] != "permission-slip" || payload["status"] != "test" {
		t.Fatalf("payload = %v", payload)
	}

	var resp agentWebhookStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Configured || resp.Provider != agentwake.ProviderGrokBot || resp.WebhookURL != webhookURL {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if resp.Test == nil || !resp.Test.Success {
		t.Fatalf("expected successful grokbot test wake, got %+v", resp.Test)
	}
}

func TestPutAgentWebhook_GrokBotRejectsNonCursorURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"provider":"grokbot","url":"https://example.com/hooks","token":"secret"}`
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

func TestPutAgentWebhook_OpenClawStillRejectsCursorURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"provider":"openclaw","url":"https://api2.cursor.sh/automations/webhook/wh_x","token":"secret"}`
	r := signedJSONRequest(t, http.MethodPut, "/agent/webhook", body, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutAgentWebhook_InvalidProvider(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"provider":"zapier","url":"http://127.0.0.1:1/hooks","token":"secret"}`
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
	if errResp.Error.Code != ErrInvalidRequest {
		t.Fatalf("code = %q, want %q", errResp.Error.Code, ErrInvalidRequest)
	}
}

func TestNotifyAgentWake_GrokBotPostsExactURL(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, _, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	webhookURL := "https://api2.cursor.sh/automations/webhook/wh_live"
	done := make(chan struct{}, 1)
	var gotURL string
	var gotBody []byte
	oldClient := agentWakeHTTPClient
	agentWakeHTTPClient = grokBotOKClient(t, func(r *http.Request, body []byte) {
		gotURL = r.URL.String()
		gotBody = body
		select {
		case done <- struct{}{}:
		default:
		}
	})
	t.Cleanup(func() { agentWakeHTTPClient = oldClient })

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(context.Background(), tx, "test", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, webhookURL, vaultID, agentwake.ProviderGrokBot); err != nil {
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
		t.Fatal("grokbot wake webhook not received")
	}
	if gotURL != webhookURL {
		t.Fatalf("URL = %q, want %q", gotURL, webhookURL)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["approval_id"] != approvalID || payload["source"] != "permission-slip" {
		t.Fatalf("payload = %v", payload)
	}
	if payload["status"] != "pending" {
		t.Fatalf("status = %v, want pending (unresolved fixture)", payload["status"])
	}
}

func TestGetAgentWebhook_ReturnsProvider(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, _ := GenerateEd25519OpenSSHKey()
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	mockVault := vault.NewMockVaultStore()
	vaultID, err := mockVault.CreateSecret(context.Background(), tx, "test", []byte("tok"))
	if err != nil {
		t.Fatal(err)
	}
	url := "https://api2.cursor.sh/automations/webhook/wh_get"
	if err := db.SetAgentWebhook(context.Background(), tx, agentID, url, vaultID, agentwake.ProviderGrokBot); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)
	r := signedJSONRequest(t, http.MethodGet, "/agent/webhook", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentWebhookStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Configured || resp.Provider != agentwake.ProviderGrokBot || resp.WebhookURL != url {
		t.Fatalf("resp = %+v", resp)
	}
}
