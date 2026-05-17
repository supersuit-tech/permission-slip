package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/auth"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func authTestApp(deps *Deps) http.Handler {
	mux := http.NewServeMux()
	authInner := http.NewServeMux()
	RegisterAuthRoutes(authInner, deps)
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", authInner))
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", NewRouter(deps)))
	return mux
}

func TestAuthFlow_SignupLoginRefreshProfileLogout(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret, DevMode: true}
	h := authTestApp(deps)

	email := fmt.Sprintf("authtest_%d@example.com", time.Now().UnixNano())
	password := "testpassword12"

	signupBody := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader([]byte(signupBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var tok1 authTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tok1); err != nil {
		t.Fatalf("signup decode: %v", err)
	}
	if tok1.AccessToken == "" || tok1.RefreshToken == "" {
		t.Fatal("signup: missing tokens")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tok1.AccessToken)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("profile after signup: want 200, got %d: %s", w.Code, w.Body.String())
	}

	loginBody := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var tokLogin authTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tokLogin); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	if tokLogin.AccessToken == "" || tokLogin.RefreshToken == "" {
		t.Fatal("login: missing tokens")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokLogin.AccessToken)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("profile: want 200, got %d: %s", w.Code, w.Body.String())
	}

	refreshBody := fmt.Sprintf(`{"refresh_token":%q}`, tokLogin.RefreshToken)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader([]byte(refreshBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var tok2 authTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tok2); err != nil {
		t.Fatalf("refresh decode: %v", err)
	}
	if tok2.RefreshToken == "" || tok2.RefreshToken == tokLogin.RefreshToken {
		t.Fatal("refresh: expected rotated refresh token")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader([]byte(
		fmt.Sprintf(`{"refresh_token":%q}`, tokLogin.RefreshToken))))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("reuse old refresh: want 401, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tok2.AccessToken)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("profile after refresh: want 200, got %d: %s", w.Code, w.Body.String())
	}

	logoutBody := fmt.Sprintf(`{"refresh_token":%q}`, tok2.RefreshToken)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader([]byte(logoutBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: want 204, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader([]byte(
		fmt.Sprintf(`{"refresh_token":%q}`, tok2.RefreshToken))))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthRefresh_ExpiredSessionRejected(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret, DevMode: true}
	ctx := t.Context()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u1")
	hash := auth.HashRefreshToken("plaintext-refresh-token")
	if err := db.CreateAuthSession(ctx, tx, "as_testexp", uid, hash, time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("CreateAuthSession: %v", err)
	}

	h := authTestApp(deps)
	body := `{"refresh_token":"plaintext-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for expired session, got %d: %s", w.Code, w.Body.String())
	}
}
