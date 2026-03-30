package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreatePagePost_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/123456/feed" {
			t.Errorf("expected path /123456/feed, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer EAAtest-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body createPagePostRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Message != "Hello from my Page!" {
			t.Errorf("expected message 'Hello from my Page!', got %q", body.Message)
		}
		if body.Link != "https://example.com" {
			t.Errorf("expected link 'https://example.com', got %q", body.Link)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createPagePostResponse{ID: "123456_789012"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(createPagePostParams{
		PageID:  "123456",
		Message: "Hello from my Page!",
		Link:    "https://example.com",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "123456_789012" {
		t.Errorf("expected id '123456_789012', got %q", data["id"])
	}
}

func TestCreatePagePost_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"message": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing page_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePagePost_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"page_id": "123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePagePost_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid OAuth access token",
				"type":    "OAuthException",
				"code":    190,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(createPagePostParams{
		PageID:  "123456",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestCreatePagePost_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Too many calls",
				"type":    "OAuthException",
				"code":    4,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(createPagePostParams{
		PageID:  "123456",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCreatePagePost_InvalidLinkURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(createPagePostParams{
		PageID:  "123456",
		Message: "Check this out",
		Link:    "not-a-url",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid link URL")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePagePost_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPagePostAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
