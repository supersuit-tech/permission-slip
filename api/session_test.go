package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-secret-at-least-32-chars-long!"

func makeToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

// authenticatedRequest creates an HTTP request with a valid HS256 access JWT
// for the given user ID. Use this in any test that needs an authenticated request.
func authenticatedRequest(t *testing.T, method, path, userID string) *http.Request {
	t.Helper()
	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// authenticatedRequestWithEmail is like authenticatedRequest but sets the JWT
// email claim (used by endpoints that pass UserEmail to connectors).
func authenticatedRequestWithEmail(t *testing.T, method, path, userID, email string) *http.Request {
	t.Helper()
	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// authenticatedJSONRequest creates an authenticated HTTP request with a JSON body.
// Use this for POST/PUT/PATCH endpoints that accept JSON payloads.
func authenticatedJSONRequest(t *testing.T, method, path, userID, body string) *http.Request {
	t.Helper()
	r := authenticatedRequest(t, method, path, userID)
	r.Body = io.NopCloser(strings.NewReader(body))
	r.ContentLength = int64(len(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func sessionTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := UserID(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": uid})
	}
}

func TestRequireSession_ValidToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	r := authenticatedRequest(t, http.MethodGet, "/profile", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body["user_id"] != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected user_id '00000000-0000-0000-0000-000000000001', got %q", body["user_id"])
	}
}

func TestAllowQueryParamToken_ValidToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := AllowQueryParamToken(RequireSession(deps)(sessionTestHandler()))

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/oauth/google/authorize?access_token="+token, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body["user_id"] != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected user_id '00000000-0000-0000-0000-000000000001', got %q", body["user_id"])
	}
}

func TestAllowQueryParamToken_HeaderTakesPrecedence(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := AllowQueryParamToken(RequireSession(deps)(sessionTestHandler()))

	// Header token for user 001, query param token for user 002.
	headerToken := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	queryToken := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000002",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/profile?access_token="+queryToken, nil)
	r.Header.Set("Authorization", "Bearer "+headerToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	// Header should win.
	if body["user_id"] != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected user_id '00000000-0000-0000-0000-000000000001', got %q", body["user_id"])
	}
}

func TestAllowQueryParamToken_ExpiredToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := AllowQueryParamToken(RequireSession(deps)(sessionTestHandler()))

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/oauth/google/authorize?access_token="+token, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAllowQueryParamToken_MalformedToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := AllowQueryParamToken(RequireSession(deps)(sessionTestHandler()))

	r := httptest.NewRequest(http.MethodGet, "/oauth/google/authorize?access_token=not-a-jwt", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAllowQueryParamToken_UnsupportedAlgorithm(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := AllowQueryParamToken(RequireSession(deps)(sessionTestHandler()))

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/oauth/google/authorize?access_token="+signed, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAllowQueryParamToken_StripsTokenFromURL(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}

	// Use a custom handler that inspects the downstream request URL and RequestURI.
	inspectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("access_token should have been stripped from r.URL"))
			return
		}
		if strings.Contains(r.RequestURI, "access_token") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("access_token should have been stripped from r.RequestURI"))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := AllowQueryParamToken(RequireSession(deps)(inspectHandler))

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/oauth/google/authorize?access_token="+token+"&state=abc", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireSession_NoQueryParamFallback(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	// RequireSession alone (without AllowQueryParamToken) should NOT accept query params.
	handler := RequireSession(deps)(sessionTestHandler())

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/profile?access_token="+token, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (RequireSession should not accept query param), got %d", w.Code)
	}
}

func TestRequireSession_MissingAuthHeader(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected error code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
}

func TestRequireSession_MalformedAuthHeader(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected error code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
}

func TestRequireSession_ExpiredToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_WrongSecret(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	token := makeToken(t, "wrong-secret-wrong-secret-wrong!", jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_UnsupportedAlgorithm_HS384(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_MissingSubClaim(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_MissingExpClaim(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	token := makeToken(t, testJWTSecret, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		// no "exp" claim
	})

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_EmptySecret(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: ""}
	handler := RequireSession(deps)(sessionTestHandler())

	// Send a structurally valid HS256 token (signed with any key).
	// The middleware should detect alg=HS256 but then see empty secret → 500.
	token := makeToken(t, "some-key-for-test-only-not-zero!", jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRequireSession_WrongAlgorithm(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	// Sign with HS384 instead of HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-000000000001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserID_NoContext(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	uid := UserID(r.Context())
	if uid != "" {
		t.Errorf("expected empty string, got %q", uid)
	}
}
