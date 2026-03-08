package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// byoaDeps creates deps with a registry containing both configured and
// unconfigured providers, suitable for BYOA tests.
func byoaDeps(tx db.DBTX) (*Deps, *vault.MockVaultStore) {
	v := vault.NewMockVaultStore()
	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"openid", "email"},
		ClientID:     "platform-client-id",
		ClientSecret: "platform-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	_ = reg.Register(oauth.Provider{
		ID:           "salesforce",
		AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
		TokenURL:     "https://login.salesforce.com/services/oauth2/token",
		Scopes:       []string{"api", "refresh_token"},
		Source:       oauth.SourceManifest,
		// No client credentials — needs BYOA
	})
	return &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    reg,
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}, v
}

// ── POST /v1/oauth/provider-configs ─────────────────────────────────────────

func TestCreateOAuthProviderConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, v := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"provider":"salesforce","client_id":"my-id","client_secret":"my-secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthProviderConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Provider != "salesforce" {
		t.Errorf("expected provider 'salesforce', got %q", resp.Provider)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	// Verify vault stored the secrets.
	if v.SecretCount() != 2 {
		t.Errorf("expected 2 vault secrets (client_id + client_secret), got %d", v.SecretCount())
	}

	// Verify the DB row was created.
	config, err := db.GetOAuthProviderConfig(t.Context(), tx, uid, "salesforce")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config == nil {
		t.Fatal("expected config to exist in DB")
	}
	if config.Provider != "salesforce" {
		t.Errorf("expected provider 'salesforce', got %q", config.Provider)
	}

	// Verify the registry was updated with BYOA credentials.
	p, ok := deps.OAuthProviders.Get("salesforce")
	if !ok {
		t.Fatal("expected salesforce in registry")
	}
	if p.Source != oauth.SourceBYOA {
		t.Errorf("expected BYOA source, got %q", p.Source)
	}
	if !p.HasClientCredentials() {
		t.Error("expected salesforce to have credentials after BYOA registration")
	}
}

func TestCreateOAuthProviderConfig_DuplicateReturnsConflict(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"provider":"salesforce","client_id":"id1","client_secret":"secret1"}`

	// First create should succeed.
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Second create should return 409.
	body2 := `{"provider":"salesforce","client_id":"id2","client_secret":"secret2"}`
	r2 := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("second create: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestCreateOAuthProviderConfig_UnknownProviderReturns404(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"provider":"unknown-provider","client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateOAuthProviderConfig_InvalidProviderIDReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"provider":"INVALID ID!","client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateOAuthProviderConfig_MissingFieldsReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	cases := []struct {
		name string
		body string
	}{
		{"missing provider", `{"client_id":"id","client_secret":"secret"}`},
		{"missing client_id", `{"provider":"salesforce","client_secret":"secret"}`},
		{"missing client_secret", `{"provider":"salesforce","client_id":"id"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateOAuthProviderConfig_CredentialTooLongReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	oversized := strings.Repeat("x", oauthClientCredentialMaxLen+1)

	cases := []struct {
		name string
		body string
	}{
		{"client_id too long", `{"provider":"salesforce","client_id":"` + oversized + `","client_secret":"s"}`},
		{"client_secret too long", `{"provider":"salesforce","client_id":"id","client_secret":"` + oversized + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateOAuthProviderConfig_CredentialTooLongReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// Create a config first so the PUT has something to update.
	createBody := `{"provider":"salesforce","client_id":"cid","client_secret":"csecret"}`
	cr := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, createBody)
	cw := httptest.NewRecorder()
	router.ServeHTTP(cw, cr)
	if cw.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d: %s", cw.Code, cw.Body.String())
	}

	oversized := strings.Repeat("x", oauthClientCredentialMaxLen+1)

	cases := []struct {
		name string
		body string
	}{
		{"client_id too long", `{"client_id":"` + oversized + `","client_secret":"s"}`},
		{"client_secret too long", `{"client_id":"id","client_secret":"` + oversized + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPut, "/v1/oauth/provider-configs/salesforce", uid, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

// ── GET /v1/oauth/provider-configs ──────────────────────────────────────────

func TestListOAuthProviderConfigs_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/provider-configs", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthProviderConfigListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(resp.Configs))
	}
}

func TestListOAuthProviderConfigs_ReturnsCreated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// Create a config first.
	body := `{"provider":"salesforce","client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List should return the created config.
	r2 := authenticatedRequest(t, http.MethodGet, "/v1/oauth/provider-configs", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp oauthProviderConfigListResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(resp.Configs))
	}
	if resp.Configs[0].Provider != "salesforce" {
		t.Errorf("expected provider 'salesforce', got %q", resp.Configs[0].Provider)
	}
}

func TestListOAuthProviderConfigs_IsolatedByUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// User 1 creates a config.
	body := `{"provider":"salesforce","client_id":"id1","client_secret":"secret1"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid1, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// User 2 should see an empty list.
	r2 := authenticatedRequest(t, http.MethodGet, "/v1/oauth/provider-configs", uid2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp oauthProviderConfigListResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Configs) != 0 {
		t.Errorf("expected 0 configs for user 2, got %d", len(resp.Configs))
	}
}

// ── DELETE /v1/oauth/provider-configs/{provider} ────────────────────────────

func TestDeleteOAuthProviderConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, v := byoaDeps(tx)
	router := NewRouter(deps)

	// Create a config.
	body := `{"provider":"salesforce","client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	vaultCountAfterCreate := v.SecretCount()
	if vaultCountAfterCreate != 2 {
		t.Fatalf("expected 2 vault secrets after create, got %d", vaultCountAfterCreate)
	}

	// Delete the config.
	r2 := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/provider-configs/salesforce", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp oauthProviderConfigDeleteResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Provider != "salesforce" {
		t.Errorf("expected provider 'salesforce', got %q", resp.Provider)
	}
	if resp.DeletedAt.IsZero() {
		t.Error("expected non-zero deleted_at")
	}

	// Verify vault secrets were deleted.
	if v.SecretCount() != 0 {
		t.Errorf("expected 0 vault secrets after delete, got %d", v.SecretCount())
	}

	// Verify the DB row was deleted.
	config, err := db.GetOAuthProviderConfig(t.Context(), tx, uid, "salesforce")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config != nil {
		t.Error("expected config to be deleted from DB")
	}

	// Verify the provider reverted in the registry (no longer BYOA).
	p, ok := deps.OAuthProviders.Get("salesforce")
	if !ok {
		t.Fatal("expected salesforce to still be in registry (from manifest)")
	}
	if p.Source == oauth.SourceBYOA {
		t.Error("expected provider source to revert from BYOA after delete")
	}
	if p.HasClientCredentials() {
		t.Error("expected provider to not have credentials after BYOA delete")
	}
}

func TestDeleteOAuthProviderConfig_NotFoundReturns404(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/provider-configs/salesforce", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteOAuthProviderConfig_InvalidIDReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/provider-configs/INVALID!", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── PUT /v1/oauth/provider-configs/{provider} ───────────────────────────────

func TestUpdateOAuthProviderConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, v := byoaDeps(tx)
	router := NewRouter(deps)

	// Create a config first.
	createBody := `{"provider":"salesforce","client_id":"old-id","client_secret":"old-secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, createBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createResp oauthProviderConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}

	// Update the config with new credentials.
	updateBody := `{"client_id":"new-id","client_secret":"new-secret"}`
	r2 := authenticatedJSONRequest(t, http.MethodPut, "/v1/oauth/provider-configs/salesforce", uid, updateBody)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var updateResp oauthProviderConfigResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("unmarshal update: %v", err)
	}
	if updateResp.Provider != "salesforce" {
		t.Errorf("expected provider 'salesforce', got %q", updateResp.Provider)
	}
	if updateResp.CreatedAt != createResp.CreatedAt {
		t.Error("expected created_at to be preserved after update")
	}
	if updateResp.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at after update")
	}
	// Note: updated_at may equal created_at when both happen within the same
	// test transaction. In production, separate requests would have distinct
	// timestamps.

	// Verify vault has exactly 2 secrets (old ones deleted, new ones created).
	if v.SecretCount() != 2 {
		t.Errorf("expected 2 vault secrets after update, got %d", v.SecretCount())
	}

	// Verify registry has new credentials.
	p, ok := deps.OAuthProviders.Get("salesforce")
	if !ok {
		t.Fatal("expected salesforce in registry")
	}
	if p.ClientID != "new-id" {
		t.Errorf("expected registry to have new client ID, got %q", p.ClientID)
	}
}

func TestUpdateOAuthProviderConfig_NotFoundReturns404(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/v1/oauth/provider-configs/salesforce", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateOAuthProviderConfig_InvalidIDReturns400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	body := `{"client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/v1/oauth/provider-configs/INVALID!", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Response includes updated_at ────────────────────────────────────────────

func TestListOAuthProviderConfigs_IncludesUpdatedAt(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// Create a config.
	body := `{"provider":"salesforce","client_id":"id","client_secret":"secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List should include updated_at.
	r2 := authenticatedRequest(t, http.MethodGet, "/v1/oauth/provider-configs", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify updated_at is present in JSON.
	var raw map[string][]map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	configs := raw["configs"]
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if _, ok := configs[0]["updated_at"]; !ok {
		t.Error("expected 'updated_at' field in list response")
	}
}

// ── Provider resolution order ───────────────────────────────────────────────

func TestBYOA_OverridesBuiltInForAuthorize(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// Verify google initially has platform credentials.
	p, _ := deps.OAuthProviders.Get("google")
	if p.ClientID != "platform-client-id" {
		t.Fatalf("expected platform-client-id, got %q", p.ClientID)
	}

	// Register BYOA credentials for google.
	body := `{"provider":"google","client_id":"byoa-client-id","client_secret":"byoa-client-secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the registry now has BYOA credentials.
	p2, ok := deps.OAuthProviders.Get("google")
	if !ok {
		t.Fatal("expected google in registry")
	}
	if p2.ClientID != "byoa-client-id" {
		t.Errorf("expected BYOA client ID, got %q", p2.ClientID)
	}
	if p2.Source != oauth.SourceBYOA {
		t.Errorf("expected BYOA source, got %q", p2.Source)
	}
	// Endpoints should be preserved from built-in.
	if p2.AuthorizeURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Errorf("expected built-in authorize URL preserved, got %q", p2.AuthorizeURL)
	}
}

func TestBYOA_DeleteRevertsToBuiltIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps, _ := byoaDeps(tx)
	router := NewRouter(deps)

	// Register BYOA for google.
	body := `{"provider":"google","client_id":"byoa-id","client_secret":"byoa-secret"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/v1/oauth/provider-configs", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Delete BYOA config.
	r2 := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/provider-configs/google", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Google should revert to built-in (without credentials since env vars
	// aren't set in tests, but the source should be built_in).
	p, ok := deps.OAuthProviders.Get("google")
	if !ok {
		t.Fatal("expected google to still be in registry")
	}
	if p.Source != oauth.SourceBuiltIn {
		t.Errorf("expected built_in source after BYOA delete, got %q", p.Source)
	}
}
