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

func decodeTestConnection(t *testing.T, body []byte) testCredentialConnectionResponse {
	t.Helper()
	var resp testCredentialConnectionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal test connection response: %v", err)
	}
	return resp
}

func TestTestCredentialConnection_Success(t *testing.T) {
	old := testProtonBridgeConnection
	testProtonBridgeConnection = func(_ context.Context, _ connectors.Credentials, _ time.Duration) error {
		return nil
	}
	t.Cleanup(func() { testProtonBridgeConnection = old })

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service":"protonmail","credentials":{"username":"user@proton.me","password":"bridge-pass"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials/test-connection", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeTestConnection(t, w.Body.Bytes())
	if !resp.OK {
		t.Errorf("expected ok=true, got %+v", resp)
	}
}

func TestTestCredentialConnection_AuthFailure(t *testing.T) {
	old := testProtonBridgeConnection
	testProtonBridgeConnection = func(_ context.Context, _ connectors.Credentials, _ time.Duration) error {
		return &connectors.AuthError{Message: "Wrong Bridge password"}
	}
	t.Cleanup(func() { testProtonBridgeConnection = old })

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service":"protonmail","credentials":{"username":"user@proton.me","password":"bad"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials/test-connection", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTestCredentialConnection_UnsupportedService(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service":"github","credentials":{"api_key":"x"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials/test-connection", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTestCredentialConnection_StoredCredentialPersistsHealth(t *testing.T) {
	old := testProtonBridgeConnection
	testProtonBridgeConnection = func(_ context.Context, _ connectors.Credentials, _ time.Duration) error {
		return nil
	}
	t.Cleanup(func() { testProtonBridgeConnection = old })

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	credID := testhelper.GenerateID(t, "cred_")

	mockVault := vault.NewMockVaultStore()
	secretID, err := mockVault.CreateSecret(t.Context(), tx, credID, []byte(`{"username":"user@proton.me","password":"bridge-pass"}`))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "protonmail", secretID)

	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service":"protonmail","credential_id":"` + credID + `"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials/test-connection", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	health, err := db.GetProtonmailHealth(t.Context(), tx, credID)
	if err != nil {
		t.Fatalf("GetProtonmailHealth: %v", err)
	}
	if health == nil || health.Status != db.ProtonmailHealthOK {
		t.Fatalf("expected health ok, got %+v", health)
	}
}
