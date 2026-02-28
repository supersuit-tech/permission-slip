//go:build integration

package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- JWKS endpoint tests ---

func TestIntegration_JWKSEndpoint_ReturnsKeys(t *testing.T) {
	// Verify that the real Supabase JWKS endpoint is accessible and returns
	// at least one EC P-256 key.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch JWKS: %v\nIs Supabase running? Try: supabase start", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("JWKS endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		t.Fatalf("decode JWKS: %v", err)
	}

	if len(jwks.Keys) == 0 {
		t.Fatal("JWKS endpoint returned 0 keys")
	}

	// Look for at least one EC P-256 key.
	var foundEC bool
	for _, k := range jwks.Keys {
		if k.Kty == "EC" && k.Crv == "P-256" {
			foundEC = true
			if k.Kid == "" {
				t.Error("EC key has empty kid")
			}
			if k.X == "" || k.Y == "" {
				t.Errorf("EC key kid=%q has empty x or y", k.Kid)
			}
		}
	}
	if !foundEC {
		t.Error("no EC P-256 keys found in JWKS response")
	}
}

func TestIntegration_JWKSCache_FetchesRealKeys(t *testing.T) {
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)

	// Trigger a fetch by looking up a known kid format. We don't know the
	// exact kid ahead of time, so first fetch all keys, then verify one exists.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Force a fetch by calling fetchLocked directly.
	cache.mu.Lock()
	err := cache.fetchLocked(ctx)
	cache.mu.Unlock()
	if err != nil {
		t.Fatalf("fetchLocked: %v", err)
	}

	cache.mu.RLock()
	keyCount := len(cache.keys)
	cache.mu.RUnlock()

	if keyCount == 0 {
		t.Fatal("JWKS cache fetched 0 keys")
	}
}

// --- ES256 JWT validation with real Supabase JWKS ---

func TestIntegration_ES256_JWTValidation_ViaSignup(t *testing.T) {
	// Sign up a user via Supabase Auth, then use the resulting JWT to make
	// an authenticated request. This tests the full ES256 validation path.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	email := fmt.Sprintf("inttest_es256_%d@test.local", time.Now().UnixNano())

	// Sign up with auto-confirm (Supabase local has enable_confirmations = false).
	signupBody := fmt.Sprintf(`{"email": %q, "password": "testpassword123!"}`, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/signup",
		strings.NewReader(signupBody))
	if err != nil {
		t.Fatalf("create signup request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", supabaseAnonKey(t))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("signup returned %d: %s", resp.StatusCode, body)
	}

	var signupResp struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&signupResp); err != nil {
		t.Fatalf("decode signup response: %v", err)
	}

	if signupResp.AccessToken == "" {
		t.Fatal("signup did not return an access_token")
	}
	if signupResp.User.ID == "" {
		t.Fatal("signup did not return a user ID")
	}

	t.Cleanup(func() {
		// Clean up auth user via admin API.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		deleteReq, _ := http.NewRequestWithContext(cleanupCtx, http.MethodDelete,
			fmt.Sprintf("http://127.0.0.1:54321/auth/v1/admin/users/%s", signupResp.User.ID), nil)
		deleteReq.Header.Set("apikey", supabaseAnonKey(t))
		deleteReq.Header.Set("Authorization", "Bearer "+supabaseServiceRoleKey(t))
		http.DefaultClient.Do(deleteReq)
	})

	// Set up the JWKS cache pointing at the real endpoint.
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)

	pool := setupIntegrationPool(t)
	deps := &Deps{DB: pool, JWKSCache: cache}

	// The RequireSession middleware should accept this token.
	handler := RequireSession(deps)(sessionTestHandler())

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("Authorization", "Bearer "+signupResp.AccessToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["user_id"] != signupResp.User.ID {
		t.Errorf("expected user_id %q, got %q", signupResp.User.ID, body["user_id"])
	}
}

func TestIntegration_ExpiredJWT_Rejected(t *testing.T) {
	// Verify that an expired ES256 token is rejected via the JWKS validation
	// path. We confirm the real JWKS endpoint is reachable, then inject a test
	// key into the cache so we can sign an expired token the cache will accept.
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)

	// Force-fetch real keys to prove the JWKS endpoint is reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cache.mu.Lock()
	if err := cache.fetchLocked(ctx); err != nil {
		cache.mu.Unlock()
		t.Fatalf("fetch JWKS: %v\nIs Supabase running? Try: supabase start", err)
	}
	cache.mu.Unlock()

	// Inject a test key so we can sign tokens the cache will accept.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	const kid = "inttest-expired-kid"
	cache.mu.Lock()
	cache.keys[kid] = &privKey.PublicKey
	cache.mu.Unlock()

	pool := setupIntegrationPool(t)
	deps := &Deps{DB: pool, JWKSCache: cache}
	handler := RequireSession(deps)(sessionTestHandler())

	expiredToken := makeES256Token(t, privKey, kid, map[string]any{
		"sub": "00000000-0000-0000-0000-000000000099",
		"aud": SupabaseAudAuthenticated,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired ES256 token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegration_WrongAudience_ES256_Rejected(t *testing.T) {
	// Verify that a valid ES256 token with aud="anon" (instead of "authenticated")
	// is rejected via the JWKS validation path. We confirm the real JWKS endpoint
	// is reachable, then inject a test key so we can sign a wrong-audience token
	// the cache will accept.
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cache.mu.Lock()
	if err := cache.fetchLocked(ctx); err != nil {
		cache.mu.Unlock()
		t.Fatalf("fetch JWKS: %v\nIs Supabase running? Try: supabase start", err)
	}
	cache.mu.Unlock()

	// Inject a test key so we can sign tokens the cache will accept.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	const kid = "inttest-wrongaud-kid"
	cache.mu.Lock()
	cache.keys[kid] = &privKey.PublicKey
	cache.mu.Unlock()

	pool := setupIntegrationPool(t)
	deps := &Deps{DB: pool, JWKSCache: cache}
	handler := RequireSession(deps)(sessionTestHandler())

	// Sign a valid-structure token but with aud="anon" instead of "authenticated".
	wrongAudToken := makeES256Token(t, privKey, kid, map[string]any{
		"sub": "00000000-0000-0000-0000-000000000098",
		"aud": "anon",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("Authorization", "Bearer "+wrongAudToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for ES256 token with wrong audience, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegration_SignedOutUser_TokenBehavior(t *testing.T) {
	// Sign up a user via GoTrue, get a valid token, then sign out.
	// Documents post-logout JWT behavior: the token remains cryptographically
	// valid (200) unless server-side session revocation is enabled (401).
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	email := fmt.Sprintf("inttest_signout_%d@test.local", time.Now().UnixNano())
	anonKey := supabaseAnonKey(t)

	// Sign up.
	signupBody := fmt.Sprintf(`{"email": %q, "password": "testpassword123!"}`, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/signup",
		strings.NewReader(signupBody))
	if err != nil {
		t.Fatalf("create signup request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", anonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("signup returned %d: %s", resp.StatusCode, body)
	}

	var signupResp struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&signupResp); err != nil {
		t.Fatalf("decode signup: %v", err)
	}
	if signupResp.AccessToken == "" {
		t.Fatal("signup did not return an access_token")
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		deleteReq, _ := http.NewRequestWithContext(cleanupCtx, http.MethodDelete,
			fmt.Sprintf("http://127.0.0.1:54321/auth/v1/admin/users/%s", signupResp.User.ID), nil)
		deleteReq.Header.Set("apikey", anonKey)
		deleteReq.Header.Set("Authorization", "Bearer "+supabaseServiceRoleKey(t))
		http.DefaultClient.Do(deleteReq)
	})

	// Verify the token works before signing out.
	jwksURL := "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	cache := NewJWKSCache(jwksURL)
	pool := setupIntegrationPool(t)
	deps := &Deps{DB: pool, JWKSCache: cache}
	handler := RequireSession(deps)(sessionTestHandler())

	preReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	preReq.Header.Set("Authorization", "Bearer "+signupResp.AccessToken)
	preW := httptest.NewRecorder()
	handler.ServeHTTP(preW, preReq)

	if preW.Code != http.StatusOK {
		t.Fatalf("token should be valid before signout, got %d: %s", preW.Code, preW.Body.String())
	}

	// Sign out via GoTrue.
	logoutReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/logout",
		nil)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}
	logoutReq.Header.Set("apikey", anonKey)
	logoutReq.Header.Set("Authorization", "Bearer "+signupResp.AccessToken)

	logoutResp, err := http.DefaultClient.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	logoutResp.Body.Close()

	if logoutResp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout returned %d, expected 204", logoutResp.StatusCode)
	}

	// Note: GoTrue's default /logout invalidates the session server-side but
	// the JWT itself remains cryptographically valid until it expires.
	// This test documents that behavior — the token may still be accepted by
	// our middleware since we validate the JWT signature and claims, not the
	// GoTrue session state. If GoTrue is configured with "global" scope logout
	// or if we add server-side session validation, this assertion should flip
	// to expect 401.
	postReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	postReq.Header.Set("Authorization", "Bearer "+signupResp.AccessToken)
	postW := httptest.NewRecorder()
	handler.ServeHTTP(postW, postReq)

	// JWT-based validation: token is still cryptographically valid after logout.
	// Document this as expected behavior. If server-side session revocation is
	// added, change this to expect 401.
	t.Logf("post-logout token status: %d (JWT remains cryptographically valid until exp)", postW.Code)
	if postW.Code != http.StatusOK && postW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 200 (JWT still valid) or 401 (session revoked), got %d: %s",
			postW.Code, postW.Body.String())
	}
}

// --- Session management tests ---

func TestIntegration_RefreshTokenRotation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	email := fmt.Sprintf("inttest_refresh_%d@test.local", time.Now().UnixNano())

	// Sign up.
	signupBody := fmt.Sprintf(`{"email": %q, "password": "testpassword123!"}`, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/signup",
		strings.NewReader(signupBody))
	if err != nil {
		t.Fatalf("create signup request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", supabaseAnonKey(t))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()

	var signupResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	json.NewDecoder(resp.Body).Decode(&signupResp)

	if signupResp.RefreshToken == "" {
		t.Fatal("signup did not return a refresh_token")
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		deleteReq, _ := http.NewRequestWithContext(cleanupCtx, http.MethodDelete,
			fmt.Sprintf("http://127.0.0.1:54321/auth/v1/admin/users/%s", signupResp.User.ID), nil)
		deleteReq.Header.Set("apikey", supabaseAnonKey(t))
		deleteReq.Header.Set("Authorization", "Bearer "+supabaseServiceRoleKey(t))
		http.DefaultClient.Do(deleteReq)
	})

	// Use the refresh token to get a new access token.
	refreshBody := fmt.Sprintf(`{"refresh_token": %q}`, signupResp.RefreshToken)
	refreshReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/token?grant_type=refresh_token",
		strings.NewReader(refreshBody))
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshReq.Header.Set("apikey", supabaseAnonKey(t))

	refreshResp, err := http.DefaultClient.Do(refreshReq)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	defer refreshResp.Body.Close()

	if refreshResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(refreshResp.Body)
		t.Fatalf("refresh returned %d: %s", refreshResp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(refreshResp.Body).Decode(&tokenResp)

	if tokenResp.AccessToken == "" {
		t.Error("refresh did not return a new access_token")
	}
	if tokenResp.RefreshToken == "" {
		t.Error("refresh did not return a new refresh_token")
	}

	// The new tokens should be different from the originals.
	if tokenResp.AccessToken == signupResp.AccessToken {
		t.Error("refreshed access_token is identical to the original")
	}
	if tokenResp.RefreshToken == signupResp.RefreshToken {
		t.Error("refreshed refresh_token is identical to the original (rotation not working)")
	}
}

// --- User lifecycle tests ---

func TestIntegration_UserLifecycle_SignupCreatesAuthUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	email := fmt.Sprintf("inttest_lifecycle_%d@test.local", time.Now().UnixNano())

	signupBody := fmt.Sprintf(`{"email": %q, "password": "testpassword123!"}`, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:54321/auth/v1/signup",
		strings.NewReader(signupBody))
	if err != nil {
		t.Fatalf("create signup request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", supabaseAnonKey(t))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("signup returned %d: %s", resp.StatusCode, body)
	}

	var signupResp struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	json.NewDecoder(resp.Body).Decode(&signupResp)

	if signupResp.User.ID == "" {
		t.Fatal("signup did not return user ID")
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		deleteReq, _ := http.NewRequestWithContext(cleanupCtx, http.MethodDelete,
			fmt.Sprintf("http://127.0.0.1:54321/auth/v1/admin/users/%s", signupResp.User.ID), nil)
		deleteReq.Header.Set("apikey", supabaseAnonKey(t))
		deleteReq.Header.Set("Authorization", "Bearer "+supabaseServiceRoleKey(t))
		http.DefaultClient.Do(deleteReq)
	})

	// Verify the auth.users row exists in the database.
	pool := setupIntegrationPool(t)
	var exists bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)`,
		signupResp.User.ID).Scan(&exists)
	if err != nil {
		t.Fatalf("query auth.users: %v", err)
	}
	if !exists {
		t.Error("auth.users row not found after signup")
	}
}

// --- Helpers for Supabase keys ---

// supabaseAnonKey returns the Supabase anon key for the local instance.
// The default local dev anon key is well-known and stable.
func supabaseAnonKey(t *testing.T) string {
	t.Helper()
	// Standard Supabase local dev anon key (public, safe to commit).
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0"
}

// supabaseServiceRoleKey returns the Supabase service_role key for the local instance.
// Used for admin operations (user deletion in cleanup).
func supabaseServiceRoleKey(t *testing.T) string {
	t.Helper()
	// Standard Supabase local dev service_role key (public, safe to commit).
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU"
}

// supabaseLocalJWTSecret returns the standard Supabase local dev JWT secret.
// Used for HS256 token validation. Safe to commit — this is the default
// secret from `supabase init` and is documented in the Supabase docs.
func supabaseLocalJWTSecret() string {
	if s := os.Getenv("SUPABASE_JWT_SECRET"); s != "" {
		return s
	}
	return "super-secret-jwt-token-with-at-least-32-characters-long"
}
