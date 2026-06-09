package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

// --- Decode helpers ---

func decodeCredentialList(t *testing.T, body []byte) credentialListResponse {
	t.Helper()
	var resp credentialListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal credential list response: %v", err)
	}
	return resp
}

func decodeStoreCredential(t *testing.T, body []byte) credentialSummary {
	t.Helper()
	var resp credentialSummary
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal store credential response: %v", err)
	}
	return resp
}

func decodeDeleteCredential(t *testing.T, body []byte) deleteCredentialResponse {
	t.Helper()
	var resp deleteCredentialResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal delete credential response: %v", err)
	}
	return resp
}

func decodeErrorResponse(t *testing.T, body []byte) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	return resp
}

// ── GET /credentials ────────────────────────────────────────────────────────

func TestListCredentials_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/credentials", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialList(t, w.Body.Bytes())
	if len(resp.Credentials) != 0 {
		t.Errorf("expected 0 credentials, got %d", len(resp.Credentials))
	}
}

func TestListCredentials_ReturnsUserCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	cred1 := testhelper.GenerateID(t, "cred_")
	cred2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, cred1, uid, "github")
	testhelper.InsertCredentialWithLabel(t, tx, cred2, uid, "slack", "Work Slack")

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/credentials", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialList(t, w.Body.Bytes())
	if len(resp.Credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(resp.Credentials))
	}
}

func TestListCredentials_IsolatedPerUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid1, "github")
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid2, "slack")

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// User 1 should only see their credential
	r := authenticatedRequest(t, http.MethodGet, "/credentials", uid1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialList(t, w.Body.Bytes())
	if len(resp.Credentials) != 1 {
		t.Fatalf("expected 1 credential for user1, got %d", len(resp.Credentials))
	}
	if resp.Credentials[0].Service != "github" {
		t.Errorf("expected service 'github', got %q", resp.Credentials[0].Service)
	}
}

func TestListCredentials_RequiresAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/credentials", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /credentials ───────────────────────────────────────────────────────

func TestStoreCredential_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_test123"}, "label": "Personal"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStoreCredential(t, w.Body.Bytes())
	if !strings.HasPrefix(resp.ID, "cred_") {
		t.Errorf("expected id to start with 'cred_', got %q", resp.ID)
	}
	if resp.Service != "github" {
		t.Errorf("expected service 'github', got %q", resp.Service)
	}
	if resp.Label == nil || *resp.Label != "Personal" {
		t.Errorf("expected label 'Personal', got %v", resp.Label)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	// Verify a vault secret was created.
	if mockVault.SecretCount() != 1 {
		t.Errorf("expected 1 vault secret, got %d", mockVault.SecretCount())
	}
}

func TestStoreCredential_WithoutLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "stripe", "stripe", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "stripe", "credentials": {"api_key": "sk_test_123"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStoreCredential(t, w.Body.Bytes())
	if resp.Label != nil {
		t.Errorf("expected nil label, got %v", resp.Label)
	}
}

func TestStoreCredential_MissingService(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"credentials": {"api_key": "test"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_UnknownService_LegacyPermissive(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// No connector_required_credentials row: store still succeeds (settings / arbitrary services).
	body := `{"service": "not_registered_anywhere", "credentials": {"api_key": "x"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_ExtraKeyRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_x", "org": "nope"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_MultiFieldCustom(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	fields := []byte(`[{"key":"a","label":"A","secret":true,"required":true},{"key":"b","label":"B","secret":false,"required":true}]`)
	testhelper.InsertConnectorWithStaticCredential(t, tx, "twofield", "twofield", "custom", fields)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "twofield", "credentials": {"a": "secretval", "b": "plain"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_MissingCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_InvalidServiceFormat(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "GitHub", "credentials": {"api_key": "test"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_DuplicateConflict(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_test1"}, "label": "work"}`

	// First store should succeed.
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first store: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Second store with same service+label should conflict.
	body2 := `{"service": "github", "credentials": {"api_key": "ghp_test2"}, "label": "work"}`
	r2 := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	errResp := decodeErrorResponse(t, w2.Body.Bytes())
	if errResp.Error.Code != ErrDuplicateCredential {
		t.Errorf("expected error code %q, got %q", ErrDuplicateCredential, errResp.Error.Code)
	}
}

func TestStoreCredential_RequiresAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/credentials", strings.NewReader(`{"service": "x", "credentials": {"api_key": "v"}}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_ServiceTooLong(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	longService := "a" + strings.Repeat("b", 128) // 129 chars
	body := fmt.Sprintf(`{"service": %q, "credentials": {"api_key": "v"}}`, longService)
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── PATCH /credentials/{credential_id} ────────────────────────────────────────

func storeTestCredential(t *testing.T, router http.Handler, uid, body string) credentialSummary {
	t.Helper()
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	return decodeStoreCredential(t, w.Body.Bytes())
}

func patchCredential(t *testing.T, router http.Handler, uid, credID, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/credentials/%s", credID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestUpdateCredential_RenameLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid, `{"service": "github", "credentials": {"api_key": "ghp_test"}, "label": "work"}`)

	w := patchCredential(t, router, uid, stored.ID, `{"label": "personal"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeStoreCredential(t, w.Body.Bytes())
	if resp.Label == nil || *resp.Label != "personal" {
		t.Errorf("expected label 'personal', got %v", resp.Label)
	}
	if resp.ID != stored.ID {
		t.Errorf("expected same credential id %q, got %q", stored.ID, resp.ID)
	}
}

func TestUpdateCredential_ClearLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid, `{"service": "github", "credentials": {"api_key": "ghp_test"}, "label": "work"}`)

	w := patchCredential(t, router, uid, stored.ID, `{"label": null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeStoreCredential(t, w.Body.Bytes())
	if resp.Label != nil {
		t.Errorf("expected nil label, got %v", resp.Label)
	}
}

func TestUpdateCredential_UpdateSecret(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	fields := []byte(`[{"key":"username","label":"Username","secret":false,"required":true},{"key":"password","label":"Password","secret":true,"required":true}]`)
	testhelper.InsertConnectorWithStaticCredential(t, tx, "proton", "proton", "custom", fields)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid, `{"service": "proton", "credentials": {"username": "alice", "password": "oldpass"}}`)

	w := patchCredential(t, router, uid, stored.ID, `{"credentials": {"password": "newpass"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	vaultID, err := db.GetVaultSecretIDByCredentialID(context.Background(), tx, stored.ID)
	if err != nil {
		t.Fatalf("GetVaultSecretIDByCredentialID: %v", err)
	}
	raw, err := mockVault.ReadSecret(context.Background(), tx, vaultID)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}
	var creds map[string]string
	if err := json.Unmarshal(raw, &creds); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if creds["username"] != "alice" {
		t.Errorf("expected username unchanged 'alice', got %q", creds["username"])
	}
	if creds["password"] != "newpass" {
		t.Errorf("expected password 'newpass', got %q", creds["password"])
	}
}

func TestUpdateCredential_PartialFieldMerge(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	fields := []byte(`[{"key":"host","label":"Host","secret":false,"required":true},{"key":"port","label":"Port","secret":false,"required":true},{"key":"password","label":"Password","secret":true,"required":true}]`)
	testhelper.InsertConnectorWithStaticCredential(t, tx, "bridge", "bridge", "custom", fields)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid, `{"service": "bridge", "credentials": {"host": "127.0.0.1", "port": "1143", "password": "secret"}}`)

	w := patchCredential(t, router, uid, stored.ID, `{"credentials": {"host": "", "port": "1025", "password": ""}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	vaultID, err := db.GetVaultSecretIDByCredentialID(context.Background(), tx, stored.ID)
	if err != nil {
		t.Fatalf("GetVaultSecretIDByCredentialID: %v", err)
	}
	raw, err := mockVault.ReadSecret(context.Background(), tx, vaultID)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}
	var creds map[string]string
	if err := json.Unmarshal(raw, &creds); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if creds["host"] != "127.0.0.1" {
		t.Errorf("expected host unchanged, got %q", creds["host"])
	}
	if creds["port"] != "1025" {
		t.Errorf("expected port '1025', got %q", creds["port"])
	}
	if creds["password"] != "secret" {
		t.Errorf("expected password unchanged, got %q", creds["password"])
	}
}

func TestUpdateCredential_NoOpRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid, `{"service": "github", "credentials": {"api_key": "ghp_test"}}`)

	w := patchCredential(t, router, uid, stored.ID, `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	w2 := patchCredential(t, router, uid, stored.ID, `{"credentials": {"api_key": ""}}`)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank-only credentials, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestUpdateCredential_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	w := patchCredential(t, router, uid, "cred_nonexistent", `{"label": "work"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCredential_OtherUserSeesNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	stored := storeTestCredential(t, router, uid1, `{"service": "github", "credentials": {"api_key": "ghp_test"}, "label": "work"}`)

	w := patchCredential(t, router, uid2, stored.ID, `{"label": "hijack"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCredential_LabelCollision(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	storeTestCredential(t, router, uid, `{"service": "github", "credentials": {"api_key": "ghp_1"}, "label": "work"}`)
	stored2 := storeTestCredential(t, router, uid, `{"service": "github", "credentials": {"api_key": "ghp_2"}, "label": "personal"}`)

	w := patchCredential(t, router, uid, stored2.ID, `{"label": "work"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	errResp := decodeErrorResponse(t, w.Body.Bytes())
	if errResp.Error.Code != ErrDuplicateCredential {
		t.Errorf("expected error code %q, got %q", ErrDuplicateCredential, errResp.Error.Code)
	}
}

// ── DELETE /credentials/{credential_id} ─────────────────────────────────────

func TestDeleteCredential_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	// Store a credential via the API so the vault also gets a secret.
	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	storeBody := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	sr := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, storeBody)
	sw := httptest.NewRecorder()
	router.ServeHTTP(sw, sr)
	if sw.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", sw.Code, sw.Body.String())
	}
	stored := decodeStoreCredential(t, sw.Body.Bytes())

	if mockVault.SecretCount() != 1 {
		t.Fatalf("expected 1 vault secret after store, got %d", mockVault.SecretCount())
	}

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/credentials/%s", stored.ID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeDeleteCredential(t, w.Body.Bytes())
	if resp.ID != stored.ID {
		t.Errorf("expected id %q, got %q", stored.ID, resp.ID)
	}
	if resp.DeletedAt.IsZero() {
		t.Error("expected non-zero deleted_at")
	}

	// Verify vault secret was also deleted.
	if mockVault.SecretCount() != 0 {
		t.Errorf("expected 0 vault secrets after delete, got %d", mockVault.SecretCount())
	}
}

func TestDeleteCredential_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/credentials/cred_nonexistent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCredential_OtherUserSeesNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	// Store credential for uid1.
	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	storeBody := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	sr := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid1, storeBody)
	sw := httptest.NewRecorder()
	router.ServeHTTP(sw, sr)
	if sw.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", sw.Code, sw.Body.String())
	}
	stored := decodeStoreCredential(t, sw.Body.Bytes())

	// User 2 tries to delete user 1's credential — should see 404, not 403,
	// to avoid leaking credential existence to non-owners.
	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/credentials/%s", stored.ID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	errResp := decodeErrorResponse(t, w.Body.Bytes())
	if errResp.Error.Code != ErrCredentialNotFound {
		t.Errorf("expected error code %q, got %q", ErrCredentialNotFound, errResp.Error.Code)
	}
}

func TestDeleteCredential_RequiresAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodDelete, "/credentials/cred_abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCredential_VerifyActuallyDeleted(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Store
	storeBody := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	sr := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, storeBody)
	sw := httptest.NewRecorder()
	router.ServeHTTP(sw, sr)
	if sw.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", sw.Code, sw.Body.String())
	}
	stored := decodeStoreCredential(t, sw.Body.Bytes())

	// Delete
	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/credentials/%s", stored.ID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify list is empty
	r2 := authenticatedRequest(t, http.MethodGet, "/credentials", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp := decodeCredentialList(t, w2.Body.Bytes())
	if len(resp.Credentials) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(resp.Credentials))
	}

	// Verify vault secret was also deleted.
	if mockVault.SecretCount() != 0 {
		t.Errorf("expected 0 vault secrets after delete, got %d", mockVault.SecretCount())
	}
}

// ── Integration: store then list ────────────────────────────────────────────

func TestStoreAndListCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Store a credential
	body := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	stored := decodeStoreCredential(t, w.Body.Bytes())

	// List should return it
	r2 := authenticatedRequest(t, http.MethodGet, "/credentials", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	list := decodeCredentialList(t, w2.Body.Bytes())
	if len(list.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list.Credentials))
	}
	if list.Credentials[0].ID != stored.ID {
		t.Errorf("expected id %q, got %q", stored.ID, list.Credentials[0].ID)
	}
	if list.Credentials[0].Service != "github" {
		t.Errorf("expected service 'github', got %q", list.Credentials[0].Service)
	}
}

// ── Vault-specific tests ────────────────────────────────────────────────────

func TestStoreCredential_VaultSecretContainsSerializedCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_secret_value"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if mockVault.SecretCount() != 1 {
		t.Fatalf("expected 1 vault secret, got %d", mockVault.SecretCount())
	}
}

func TestStoreCredential_VaultErrorReturns500(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: &failingVaultStore{}, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	errResp := decodeErrorResponse(t, w.Body.Bytes())
	// Verify the error message does not leak credential data.
	if strings.Contains(errResp.Error.Message, "ghp_test") {
		t.Error("error response leaks credential data")
	}
}

func TestStoreCredential_NilVaultReturns503(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnectorWithStaticCredential(t, tx, "github", "github", "api_key", nil)

	deps := &Deps{DB: tx, Vault: nil, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "github", "credentials": {"api_key": "ghp_test"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCredential_NilVaultReturns503(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, Vault: nil, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/credentials/cred_abc123", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// failingVaultStore always returns errors — used to test error propagation.
type failingVaultStore struct{}

func (f *failingVaultStore) CreateSecret(_ context.Context, _ db.DBTX, _ string, _ []byte) (string, error) {
	return "", fmt.Errorf("vault unavailable")
}
func (f *failingVaultStore) ReadSecret(_ context.Context, _ db.DBTX, _ string) ([]byte, error) {
	return nil, fmt.Errorf("vault unavailable")
}
func (f *failingVaultStore) UpdateSecret(_ context.Context, _ db.DBTX, _ string, _ []byte) error {
	return fmt.Errorf("vault unavailable")
}
func (f *failingVaultStore) DeleteSecret(_ context.Context, _ db.DBTX, _ string) error {
	return fmt.Errorf("vault unavailable")
}
