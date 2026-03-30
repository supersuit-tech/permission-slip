//go:build integration

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/vault"
)

// supabaseDatabaseURL returns the Supabase local Postgres URL.
// Integration tests run against the full Supabase stack.
func supabaseDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgresql://postgres:postgres@127.0.0.1:54322/postgres"
}

// setupIntegrationPool creates a connection pool to the Supabase Postgres instance.
// Tests using this pool operate against the real database with real extensions.
func setupIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, supabaseDatabaseURL())
	if err != nil {
		t.Fatalf("failed to connect to Supabase Postgres: %v\n"+
			"Is Supabase running? Try: supabase start", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// setupIntegrationUser creates a real auth.users row and profiles row for
// integration tests. Returns the user ID. The caller should clean up after
// the test using t.Cleanup.
func setupIntegrationUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()

	uid := fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%1e12)
	username := fmt.Sprintf("inttest_%d", time.Now().UnixNano()%1e9)

	// Insert into auth.users (Supabase-managed table).
	_, err := pool.Exec(ctx,
		`INSERT INTO auth.users (id, email, instance_id, aud, role, created_at, updated_at)
		 VALUES ($1, $2, '00000000-0000-0000-0000-000000000000', 'authenticated', 'authenticated', now(), now())`,
		uid, username+"@test.local")
	if err != nil {
		t.Fatalf("failed to create auth.users row: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, username)
	if err != nil {
		t.Fatalf("failed to create profiles row: %v", err)
	}

	t.Cleanup(func() {
		// Delete in reverse FK order.
		pool.Exec(context.Background(), `DELETE FROM credentials WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM profiles WHERE id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = $1`, uid)
	})

	return uid
}

// --- Vault round-trip tests ---

func TestIntegration_VaultCreateSecret_ReturnsValidUUID(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()

	id, err := v.CreateSecret(ctx, pool, "inttest_create", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty vault secret ID")
	}
	// UUID format: 8-4-4-4-12
	if len(id) != 36 {
		t.Errorf("expected UUID-length (36), got %d: %q", len(id), id)
	}

	t.Cleanup(func() {
		v.DeleteSecret(context.Background(), pool, id)
	})
}

func TestIntegration_VaultDecryptedSecretsView(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()

	original := `{"api_key":"sk_live_test123","org":"myorg"}`
	id, err := v.CreateSecret(ctx, pool, "inttest_decrypt", []byte(original))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	t.Cleanup(func() {
		v.DeleteSecret(context.Background(), pool, id)
	})

	// Read back via the decrypted_secrets view.
	got, err := v.ReadSecret(ctx, pool, id)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}
	if string(got) != original {
		t.Errorf("decrypted secret mismatch:\n  got:  %s\n  want: %s", got, original)
	}
}

func TestIntegration_VaultDeleteSecret(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()

	id, err := v.CreateSecret(ctx, pool, "inttest_delete", []byte(`{"k":"v"}`))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}

	// Delete it.
	if err := v.DeleteSecret(ctx, pool, id); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	// Verify it's gone.
	_, err = v.ReadSecret(ctx, pool, id)
	if err == nil {
		t.Fatal("expected error reading deleted secret, got nil")
	}
}

func TestIntegration_VaultStoreAndReadRoundTrip(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	// Store credential via API.
	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	creds := map[string]any{"api_key": "ghp_roundtrip_test", "org": "testorg"}
	body := fmt.Sprintf(`{"service": "github", "credentials": %s}`, mustJSON(t, creds))
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	stored := decodeStoreCredential(t, w.Body.Bytes())

	// Validate the API response fields.
	if !strings.HasPrefix(stored.ID, "cred_") {
		t.Errorf("expected credential ID to start with 'cred_', got %q", stored.ID)
	}
	if stored.Service != "github" {
		t.Errorf("expected service 'github', got %q", stored.Service)
	}
	if stored.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at in API response")
	}

	// Read the vault secret ID from the credential row.
	vaultSecretID, err := db.GetVaultSecretID(ctx, pool, uid, "github", nil)
	if err != nil {
		t.Fatalf("GetVaultSecretID: %v", err)
	}

	// Read and decrypt from the real vault.
	raw, err := v.ReadSecret(ctx, pool, vaultSecretID)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}

	var decrypted map[string]any
	if err := json.Unmarshal(raw, &decrypted); err != nil {
		t.Fatalf("unmarshal decrypted secret: %v", err)
	}

	if decrypted["api_key"] != "ghp_roundtrip_test" {
		t.Errorf("expected api_key 'ghp_roundtrip_test', got %v", decrypted["api_key"])
	}
	if decrypted["org"] != "testorg" {
		t.Errorf("expected org 'testorg', got %v", decrypted["org"])
	}
}

func TestIntegration_VaultStoreAndDeleteRoundTrip(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Store.
	body := `{"service": "slack", "credentials": {"token": "xoxb-delete-test"}}`
	sr := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	sw := httptest.NewRecorder()
	router.ServeHTTP(sw, sr)
	if sw.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", sw.Code, sw.Body.String())
	}
	stored := decodeStoreCredential(t, sw.Body.Bytes())

	// Grab vault secret ID before delete.
	vaultSecretID, err := db.GetVaultSecretID(ctx, pool, uid, "slack", nil)
	if err != nil {
		t.Fatalf("GetVaultSecretID: %v", err)
	}

	// Delete via API.
	dr := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/credentials/%s", stored.ID), uid)
	dw := httptest.NewRecorder()
	router.ServeHTTP(dw, dr)
	if dw.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", dw.Code, dw.Body.String())
	}

	// Verify vault secret is gone.
	_, err = v.ReadSecret(ctx, pool, vaultSecretID)
	if err == nil {
		t.Fatal("expected error reading deleted vault secret, got nil")
	}

	// Verify credential row is gone.
	creds, err := db.ListCredentialsByUser(ctx, pool, uid)
	if err != nil {
		t.Fatalf("ListCredentialsByUser: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(creds))
	}
}

func TestIntegration_VaultTransactionAtomicity(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()

	// Start a transaction, create a vault secret, then roll back.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	id, err := v.CreateSecret(ctx, tx, "inttest_atomicity", []byte(`{"test":"atomicity"}`))
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("CreateSecret in tx: %v", err)
	}

	// Roll back the transaction.
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Verify the vault secret does not exist after rollback.
	_, err = v.ReadSecret(ctx, pool, id)
	if err == nil {
		t.Fatal("expected vault secret to be rolled back, but it still exists")
	}
}

// --- Helpers ---

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}

// --- GetDecryptedCredentials with real vault ---

func TestIntegration_GetDecryptedCredentials(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	// Store credential via API so both credential row and vault secret exist.
	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "stripe", "credentials": {"api_key": "sk_test_e2e", "account_id": "acct_123"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("store: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Use GetDecryptedCredentials with the real vault reader.
	creds, err := db.GetDecryptedCredentials(ctx, pool, v.ReadSecret, uid, "stripe", nil)
	if err != nil {
		t.Fatalf("GetDecryptedCredentials: %v", err)
	}
	if creds["api_key"] != "sk_test_e2e" {
		t.Errorf("expected api_key 'sk_test_e2e', got %v", creds["api_key"])
	}
	if creds["account_id"] != "acct_123" {
		t.Errorf("expected account_id 'acct_123', got %v", creds["account_id"])
	}
}

// --- Credential list with real vault ---

func TestIntegration_CredentialList_WithRealVault(t *testing.T) {
	pool := setupIntegrationPool(t)
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Store multiple credentials for different services.
	services := []struct {
		service string
		creds   string
	}{
		{"github", `{"token":"ghp_test123"}`},
		{"slack", `{"token":"xoxb_test456"}`},
		{"stripe", `{"api_key":"sk_test789","account_id":"acct_abc"}`},
	}
	for _, svc := range services {
		body := fmt.Sprintf(`{"service":%q,"credentials":%s}`, svc.service, svc.creds)
		r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("store %s: expected 201, got %d: %s", svc.service, w.Code, w.Body.String())
		}
	}

	// List all credentials.
	listReq := authenticatedRequest(t, http.MethodGet, "/credentials", uid)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", listW.Code, listW.Body.String())
	}

	var listResp credentialListResponse
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}

	if len(listResp.Credentials) != 3 {
		t.Fatalf("expected 3 credentials, got %d", len(listResp.Credentials))
	}

	// Verify each credential has proper metadata and no secrets leaked.
	foundServices := make(map[string]bool)
	for _, cred := range listResp.Credentials {
		if !strings.HasPrefix(cred.ID, "cred_") {
			t.Errorf("expected ID prefix 'cred_', got %q", cred.ID)
		}
		if cred.Service == "" {
			t.Error("expected non-empty service")
		}
		if cred.CreatedAt.IsZero() {
			t.Error("expected non-zero created_at")
		}
		foundServices[cred.Service] = true
	}

	for _, svc := range services {
		if !foundServices[svc.service] {
			t.Errorf("service %q not found in list response", svc.service)
		}
	}

	// Verify the raw list response body does NOT contain any secret values.
	rawBody := listW.Body.String()
	secretValues := []string{"ghp_test123", "xoxb_test456", "sk_test789", "acct_abc"}
	for _, secret := range secretValues {
		if strings.Contains(rawBody, secret) {
			t.Errorf("list response leaks secret value %q", secret)
		}
	}
}

// --- Duplicate credential vault cleanup ---

func TestIntegration_DuplicateCredential_NoOrphanedVaultSecret(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Store a credential for "github".
	body := `{"service":"github","credentials":{"token":"ghp_first"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first store: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Count vault secrets before the duplicate attempt.
	var countBefore int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM vault.secrets WHERE name LIKE 'cred_%'`,
	).Scan(&countBefore); err != nil {
		t.Fatalf("count vault secrets before: %v", err)
	}

	// Attempt to store a duplicate credential for the same service.
	dupBody := `{"service":"github","credentials":{"token":"ghp_second"}}`
	dupR := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, dupBody)
	dupW := httptest.NewRecorder()
	router.ServeHTTP(dupW, dupR)

	if dupW.Code != http.StatusConflict {
		t.Fatalf("duplicate store: expected 409, got %d: %s", dupW.Code, dupW.Body.String())
	}

	// Count vault secrets after the duplicate attempt.
	// The transaction should have rolled back, so no orphaned vault secret.
	var countAfter int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM vault.secrets WHERE name LIKE 'cred_%'`,
	).Scan(&countAfter); err != nil {
		t.Fatalf("count vault secrets after: %v", err)
	}

	if countAfter != countBefore {
		t.Errorf("orphaned vault secret: count before=%d, after=%d (expected equal)", countBefore, countAfter)
	}

	// Verify the original credential's vault secret is still intact.
	creds, err := db.GetDecryptedCredentials(ctx, pool, v.ReadSecret, uid, "github", nil)
	if err != nil {
		t.Fatalf("GetDecryptedCredentials: %v", err)
	}
	if creds["token"] != "ghp_first" {
		t.Errorf("expected original token 'ghp_first', got %v", creds["token"])
	}
}

// --- Large credential payload ---

func TestIntegration_LargeCredentialPayload_RoundTrip(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()
	uid := setupIntegrationUser(t, pool)

	deps := &Deps{DB: pool, Vault: v, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Build a ~10KB credentials payload.
	largeCreds := make(map[string]any)
	largeCreds["api_key"] = "sk_live_" + strings.Repeat("a", 200)
	largeCreds["webhook_secret"] = "whsec_" + strings.Repeat("b", 200)
	for i := 0; i < 50; i++ {
		largeCreds[fmt.Sprintf("field_%03d", i)] = strings.Repeat("x", 150)
	}

	credsJSON, err := json.Marshal(largeCreds)
	if err != nil {
		t.Fatalf("marshal large creds: %v", err)
	}
	t.Logf("large credential payload size: %d bytes", len(credsJSON))

	body := fmt.Sprintf(`{"service":"large-svc","credentials":%s}`, credsJSON)
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("store large credential: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Read back and verify every field round-trips correctly.
	decrypted, err := db.GetDecryptedCredentials(ctx, pool, v.ReadSecret, uid, "large-svc", nil)
	if err != nil {
		t.Fatalf("GetDecryptedCredentials: %v", err)
	}

	// Verify key fields.
	expectedAPIKey := "sk_live_" + strings.Repeat("a", 200)
	if decrypted["api_key"] != expectedAPIKey {
		t.Errorf("api_key mismatch: got length %d, want %d",
			len(fmt.Sprint(decrypted["api_key"])), len(expectedAPIKey))
	}

	// Verify field count.
	if len(decrypted) != len(largeCreds) {
		t.Errorf("expected %d fields, got %d", len(largeCreds), len(decrypted))
	}

	// Spot-check a middle field.
	expectedMiddle := strings.Repeat("x", 150)
	if decrypted["field_025"] != expectedMiddle {
		t.Errorf("field_025 mismatch: expected %d chars, got %v",
			len(expectedMiddle), decrypted["field_025"])
	}
}

// --- Full E2E: GoTrue signup + real vault credential store ---

func TestIntegration_FullE2E_GoTrueAuth_VaultCredentials(t *testing.T) {
	// This test exercises the ENTIRE stack with no mocks:
	// 1. Sign up via real GoTrue (creates auth.users row)
	// 2. Get a real ES256 JWT validated via real JWKS
	// 3. Create a profile via POST /onboarding
	// 4. Store a credential (encrypted in real Supabase Vault)
	// 5. List credentials (verify metadata, no secret leakage)
	// 6. Decrypt via GetDecryptedCredentials
	// 7. Delete the credential and verify vault secret is gone
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool := setupIntegrationPool(t)
	v := vault.NewSupabaseVaultStore()
	anonKey := supabaseAnonKey(t)

	// Step 1: Sign up via GoTrue.
	email := fmt.Sprintf("inttest_e2e_%d@test.local", time.Now().UnixNano())
	signupBody := fmt.Sprintf(`{"email":%q,"password":"testpassword123!"}`, email)
	signupReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/signup",
		strings.NewReader(signupBody))
	if err != nil {
		t.Fatalf("create signup request: %v", err)
	}
	signupReq.Header.Set("Content-Type", "application/json")
	signupReq.Header.Set("apikey", anonKey)

	signupResp, err := http.DefaultClient.Do(signupReq)
	if err != nil {
		t.Fatalf("signup: %v\nIs Supabase running? Try: supabase start", err)
	}
	defer signupResp.Body.Close()

	if signupResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(signupResp.Body)
		t.Fatalf("signup returned %d: %s", signupResp.StatusCode, body)
	}

	var signup struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(signupResp.Body).Decode(&signup); err != nil {
		t.Fatalf("decode signup: %v", err)
	}
	if signup.AccessToken == "" || signup.User.ID == "" {
		t.Fatal("signup did not return access_token or user ID")
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Delete orphaned vault secrets (no FK from credentials → vault.secrets).
		rows, _ := pool.Query(cleanupCtx,
			`SELECT vault_secret_id FROM credentials WHERE user_id = $1`, signup.User.ID)
		if rows != nil {
			for rows.Next() {
				var secretID string
				if err := rows.Scan(&secretID); err == nil {
					pool.Exec(cleanupCtx,
						`DELETE FROM vault.secrets WHERE id = $1`, secretID)
				}
			}
			rows.Close()
		}
		pool.Exec(cleanupCtx, `DELETE FROM credentials WHERE user_id = $1`, signup.User.ID)
		pool.Exec(cleanupCtx, `DELETE FROM profiles WHERE id = $1`, signup.User.ID)
		// Clean up auth user.
		delReq, _ := http.NewRequestWithContext(cleanupCtx, http.MethodDelete,
			fmt.Sprintf("http://127.0.0.1:54321/auth/v1/admin/users/%s", signup.User.ID), nil)
		delReq.Header.Set("apikey", anonKey)
		delReq.Header.Set("Authorization", "Bearer "+supabaseServiceRoleKey(t))
		if resp, err := http.DefaultClient.Do(delReq); err == nil {
			resp.Body.Close()
		}
	})

	// Step 2: Set up router with real JWKS + real Vault.
	// Include both HS256 and ES256 auth paths since the local Supabase token
	// algorithm depends on the GoTrue version (v1 uses HS256, v2+ uses ES256).
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)
	deps := &Deps{
		DB:                pool,
		Vault:             v,
		JWKSCache:         cache,
		SupabaseJWTSecret: supabaseLocalJWTSecret(),
	}
	router := NewRouter(deps)

	// Helper to create authenticated requests using the real GoTrue token.
	authReq := func(method, path string) *http.Request {
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("Authorization", "Bearer "+signup.AccessToken)
		return r
	}
	authJSONReq := func(method, path, body string) *http.Request {
		r := authReq(method, path)
		r.Body = io.NopCloser(strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	// Step 3: Create a profile via POST /onboarding.
	username := fmt.Sprintf("e2e_%d", time.Now().UnixNano()%1e9)
	onboardReq := authJSONReq(http.MethodPost, "/onboarding",
		fmt.Sprintf(`{"username":%q}`, username))
	onboardW := httptest.NewRecorder()
	router.ServeHTTP(onboardW, onboardReq)

	// Accept 201 (new profile created) or 200 (profile already existed via trigger).
	if onboardW.Code != http.StatusCreated && onboardW.Code != http.StatusOK {
		t.Fatalf("onboarding: expected 200 or 201, got %d: %s", onboardW.Code, onboardW.Body.String())
	}

	// Step 4: Store a credential (encrypted in real vault).
	credBody := `{"service":"openai","credentials":{"api_key":"sk-e2e-test-key","org_id":"org-e2e"}}`
	storeReq := authJSONReq(http.MethodPost, "/credentials", credBody)
	storeW := httptest.NewRecorder()
	router.ServeHTTP(storeW, storeReq)

	if storeW.Code != http.StatusCreated {
		t.Fatalf("store credential: expected 201, got %d: %s", storeW.Code, storeW.Body.String())
	}

	var stored credentialSummary
	if err := json.Unmarshal(storeW.Body.Bytes(), &stored); err != nil {
		t.Fatalf("unmarshal store response: %v", err)
	}
	if !strings.HasPrefix(stored.ID, "cred_") {
		t.Errorf("expected ID prefix 'cred_', got %q", stored.ID)
	}
	if stored.Service != "openai" {
		t.Errorf("expected service 'openai', got %q", stored.Service)
	}

	// Step 5: List credentials and verify no secret leakage.
	listReq := authReq(http.MethodGet, "/credentials")
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list credentials: expected 200, got %d: %s", listW.Code, listW.Body.String())
	}

	rawList := listW.Body.String()
	if strings.Contains(rawList, "sk-e2e-test-key") {
		t.Error("list response leaks credential secret")
	}

	var listResp credentialListResponse
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listResp.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(listResp.Credentials))
	}

	// Step 6: Decrypt via GetDecryptedCredentials.
	decrypted, err := db.GetDecryptedCredentials(ctx, pool, v.ReadSecret,
		signup.User.ID, "openai", nil)
	if err != nil {
		t.Fatalf("GetDecryptedCredentials: %v", err)
	}
	if decrypted["api_key"] != "sk-e2e-test-key" {
		t.Errorf("expected api_key 'sk-e2e-test-key', got %v", decrypted["api_key"])
	}
	if decrypted["org_id"] != "org-e2e" {
		t.Errorf("expected org_id 'org-e2e', got %v", decrypted["org_id"])
	}

	// Step 7: Delete the credential and verify vault secret is gone.
	vaultSecretID, err := db.GetVaultSecretID(ctx, pool, signup.User.ID, "openai", nil)
	if err != nil {
		t.Fatalf("GetVaultSecretID: %v", err)
	}

	delReq := authReq(http.MethodDelete, "/credentials/"+stored.ID)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("delete credential: expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify the vault secret is actually gone.
	_, readErr := v.ReadSecret(ctx, pool, vaultSecretID)
	if readErr == nil {
		t.Error("expected vault secret to be deleted, but it still exists")
	}

	// Verify credential row is gone.
	creds, err := db.ListCredentialsByUser(ctx, pool, signup.User.ID)
	if err != nil {
		t.Fatalf("ListCredentialsByUser after delete: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(creds))
	}
}

// --- Vault secret key validation ---

func TestIntegration_VaultSecretEncrypted(t *testing.T) {
	pool := setupIntegrationPool(t)
	ctx := context.Background()
	v := vault.NewSupabaseVaultStore()

	plaintext := "super-secret-api-key-12345"
	id, err := v.CreateSecret(ctx, pool, "inttest_encrypted", []byte(plaintext))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	t.Cleanup(func() {
		v.DeleteSecret(context.Background(), pool, id)
	})

	// Read the raw encrypted value from vault.secrets (not decrypted_secrets).
	var rawSecret string
	err = pool.QueryRow(ctx,
		`SELECT secret FROM vault.secrets WHERE id = $1`, id,
	).Scan(&rawSecret)
	if err != nil {
		t.Fatalf("query raw vault.secrets: %v", err)
	}

	// The raw value should NOT be the plaintext — it should be encrypted.
	if rawSecret == plaintext {
		t.Error("vault.secrets contains plaintext — encryption is not working")
	}
	if strings.Contains(rawSecret, plaintext) {
		t.Error("vault.secrets contains plaintext substring — encryption is not working")
	}
}
