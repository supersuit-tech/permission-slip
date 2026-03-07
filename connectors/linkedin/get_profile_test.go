package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetProfile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/userinfo" {
			t.Errorf("expected path /userinfo, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-linkedin-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		// /v2 endpoint — should NOT have LinkedIn-Version header
		if got := r.Header.Get("LinkedIn-Version"); got != "" {
			t.Errorf("expected no LinkedIn-Version header for v2 endpoint, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userinfoResponse{
			Sub:           "abc123",
			Name:          "Jane Doe",
			Email:         "jane@example.com",
			Picture:       "https://media.licdn.com/photo.jpg",
			GivenName:     "Jane",
			FamilyName:    "Doe",
			EmailVerified: true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getProfileAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_profile",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "abc123" {
		t.Errorf("expected id 'abc123', got %q", data["id"])
	}
	if data["name"] != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %q", data["name"])
	}
	if data["email"] != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got %q", data["email"])
	}
}

func TestGetProfile_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(linkedInErrorResponse{
			Status:  401,
			Message: "Invalid access token",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getProfileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_profile",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestGetProfile_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getProfileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_profile",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}
