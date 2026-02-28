package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// setupInviteTest creates a test DB transaction, inserts a user, and builds a
// router with default Deps. Returns the router and the authenticated user ID.
// Pass optional modifier functions to customise Deps (e.g. set BaseURL).
func setupInviteTest(t *testing.T, modDeps ...func(*Deps)) (http.Handler, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	for _, fn := range modDeps {
		fn(deps)
	}
	return NewRouter(deps), uid
}

func TestCreateRegistrationInvite_Success(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp createInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify invite code format: PS-XXXX-XXXX
	codePattern := regexp.MustCompile(`^PS-[A-Z2-9]{4}-[A-Z2-9]{4}$`)
	if !codePattern.MatchString(resp.InviteCode) {
		t.Errorf("invite code %q does not match expected format PS-XXXX-XXXX", resp.InviteCode)
	}

	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}

	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}

	if !strings.HasPrefix(resp.ID, "ri_") {
		t.Errorf("expected ID to start with 'ri_', got %q", resp.ID)
	}

	if resp.ExpiresAt.IsZero() {
		t.Error("expected non-zero expires_at")
	}

	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	// Default TTL is 15 minutes
	expectedExpiry := resp.CreatedAt.Add(15 * time.Minute)
	if diff := resp.ExpiresAt.Sub(expectedExpiry); diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expires_at not ~15 minutes after created_at: created=%v, expires=%v", resp.CreatedAt, resp.ExpiresAt)
	}

	// No BaseURL configured, so invite_url should be empty
	if resp.InviteURL != "" {
		t.Errorf("expected empty invite_url when BaseURL not configured, got %q", resp.InviteURL)
	}
}

func TestCreateRegistrationInvite_WithBaseURL(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t, func(d *Deps) {
		d.BaseURL = "https://app.permissionslip.dev"
	})

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp createInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.InviteURL == "" {
		t.Fatal("expected non-empty invite_url when BaseURL is configured")
	}

	if !strings.HasPrefix(resp.InviteURL, "https://app.permissionslip.dev/invite/PS-") {
		t.Errorf("unexpected invite_url format: %q", resp.InviteURL)
	}

	// invite_url should contain the same code
	if !strings.Contains(resp.InviteURL, resp.InviteCode) {
		t.Errorf("invite_url %q does not contain invite_code %q", resp.InviteURL, resp.InviteCode)
	}
}

func TestCreateRegistrationInvite_CustomTTL(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{"expires_in": 3600}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp createInviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Custom TTL is 1 hour
	expectedExpiry := resp.CreatedAt.Add(1 * time.Hour)
	if diff := resp.ExpiresAt.Sub(expectedExpiry); diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expires_at not ~1 hour after created_at: created=%v, expires=%v", resp.CreatedAt, resp.ExpiresAt)
	}
}

func TestCreateRegistrationInvite_TTLTooLow(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{"expires_in": 30}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, errResp.Error.Code)
	}
}

func TestCreateRegistrationInvite_TTLTooHigh(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{"expires_in": 100000}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRegistrationInvite_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := strings.NewReader(`{}`)
	r := httptest.NewRequest(http.MethodPost, "/registration-invites", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRegistrationInvite_NilDB(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	uid := testhelper.GenerateUID(t)
	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRegistrationInvite_NoContentType(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	body := strings.NewReader(`{}`)
	r := authenticatedRequest(t, http.MethodPost, "/registration-invites", uid)
	r.Body = io.NopCloser(body)
	// No Content-Type header — should default to JSON
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (JSON default), got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRegistrationInvite_ProfileNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	// Auth user exists in Supabase but has no profile row

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrProfileNotFound {
		t.Errorf("expected error code %q, got %q", ErrProfileNotFound, errResp.Error.Code)
	}
}

func TestCreateRegistrationInvite_UniqueCodesPerCall(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	// Use 5 iterations — well under inviteRateLimit so the test
	// validates uniqueness without being coupled to rate-limit config.
	const iterations = 5
	codes := make(map[string]bool, iterations)
	for i := 0; i < iterations; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("iteration %d: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}

		var resp createInviteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("iteration %d: failed to unmarshal response: %v", i, err)
		}

		if codes[resp.InviteCode] {
			t.Fatalf("duplicate invite code on iteration %d: %s", i, resp.InviteCode)
		}
		codes[resp.InviteCode] = true
	}
}

func TestGenerateInviteCode_Format(t *testing.T) {
	t.Parallel()
	pattern := regexp.MustCompile(`^PS-[A-Z2-9]{4}-[A-Z2-9]{4}$`)
	for i := 0; i < 100; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if !pattern.MatchString(code) {
			t.Errorf("iteration %d: code %q does not match PS-XXXX-XXXX format", i, code)
		}
	}
}

func TestGenerateInviteCode_NoAmbiguousChars(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		// Strip the PS- prefix and hyphens, check remaining chars
		chars := strings.ReplaceAll(strings.TrimPrefix(code, "PS-"), "-", "")
		for _, c := range chars {
			if c == '0' || c == 'O' || c == '1' || c == 'I' {
				t.Errorf("iteration %d: code %q contains ambiguous character %q", i, code, string(c))
			}
		}
	}
}

func TestHashInviteCode_Deterministic(t *testing.T) {
	t.Parallel()
	h1 := hashCodeHex("PS-ABCD-EFGH", "")
	h2 := hashCodeHex("PS-ABCD-EFGH", "")
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %q vs %q", h1, h2)
	}

	h3 := hashCodeHex("PS-XXXX-YYYY", "")
	if h1 == h3 {
		t.Error("different inputs produced same hash")
	}
}

func TestHashInviteCode_HMAC(t *testing.T) {
	t.Parallel()
	key := "test-hmac-key-for-invites"

	// HMAC mode is deterministic
	h1 := hashCodeHex("PS-ABCD-EFGH", key)
	h2 := hashCodeHex("PS-ABCD-EFGH", key)
	if h1 != h2 {
		t.Errorf("same input+key produced different hashes: %q vs %q", h1, h2)
	}

	// Different codes produce different hashes
	h3 := hashCodeHex("PS-XXXX-YYYY", key)
	if h1 == h3 {
		t.Error("different inputs produced same HMAC hash")
	}

	// HMAC hash differs from plain SHA-256 hash
	plain := hashCodeHex("PS-ABCD-EFGH", "")
	if h1 == plain {
		t.Error("HMAC hash should differ from plain SHA-256 hash")
	}

	// Different keys produce different hashes
	h4 := hashCodeHex("PS-ABCD-EFGH", "different-key")
	if h1 == h4 {
		t.Error("different keys should produce different hashes")
	}
}

func TestCreateRegistrationInvite_RateLimitExceeded(t *testing.T) {
	t.Parallel()
	router, uid := setupInviteTest(t)

	// Create inviteRateLimit invites to exhaust the quota.
	for i := 0; i < inviteRateLimit; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup invite %d: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}

	// The next request should be rate-limited.
	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrRateLimited {
		t.Errorf("expected error code %q, got %q", ErrRateLimited, errResp.Error.Code)
	}
	if !errResp.Error.Retryable {
		t.Error("expected retryable=true for rate limit error")
	}
}

func TestCreateRegistrationInvite_RateLimitPerUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	userA := testhelper.GenerateUID(t)
	userB := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, userA, "u_"+userA[:8])
	testhelper.InsertUser(t, tx, userB, "u_"+userB[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Exhaust user A's quota.
	for i := 0; i < inviteRateLimit; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", userA, `{}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("userA invite %d: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}

	// User B should still be able to create invites.
	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", userB, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("userB: expected 201 (not rate limited), got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuildInviteURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		baseURL  string
		code     string
		expected string
	}{
		{
			name:     "basic",
			baseURL:  "https://app.permissionslip.dev",
			code:     "PS-ABCD-EFGH",
			expected: "https://app.permissionslip.dev/invite/PS-ABCD-EFGH",
		},
		{
			name:     "with trailing slash",
			baseURL:  "https://app.permissionslip.dev/",
			code:     "PS-ABCD-EFGH",
			expected: "https://app.permissionslip.dev/invite/PS-ABCD-EFGH",
		},
		{
			name:     "invalid base URL",
			baseURL:  "://invalid",
			code:     "PS-ABCD-EFGH",
			expected: "",
		},
		{
			name:     "relative URL without scheme",
			baseURL:  "example.com",
			code:     "PS-ABCD-EFGH",
			expected: "",
		},
		{
			name:     "empty base URL",
			baseURL:  "",
			code:     "PS-ABCD-EFGH",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildInviteURL(tt.baseURL, tt.code)
			if got != tt.expected {
				t.Errorf("buildInviteURL(%q, %q) = %q, want %q", tt.baseURL, tt.code, got, tt.expected)
			}
		})
	}
}
