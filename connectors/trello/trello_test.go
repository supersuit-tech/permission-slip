package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "trello" {
		t.Errorf("expected ID 'trello', got %q", c.ID())
	}
}

func TestActions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"trello.create_card",
		"trello.update_card",
		"trello.add_comment",
		"trello.move_card",
		"trello.create_checklist",
		"trello.search_cards",
	}

	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("missing action %q", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), validCreds())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_MissingAPIKey(t *testing.T) {
	t.Parallel()
	c := New()
	creds := connectors.NewCredentials(map[string]string{
		"token": "test-token",
	})
	err := c.ValidateCredentials(context.Background(), creds)
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateCredentials_MissingToken(t *testing.T) {
	t.Parallel()
	c := New()
	creds := connectors.NewCredentials(map[string]string{
		"api_key": "test-key",
	})
	err := c.ValidateCredentials(context.Background(), creds)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateCredentials_Empty(t *testing.T) {
	t.Parallel()
	c := New()
	creds := connectors.NewCredentials(map[string]string{})
	err := c.ValidateCredentials(context.Background(), creds)
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestManifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "trello" {
		t.Errorf("expected manifest ID 'trello', got %q", m.ID)
	}
	if m.Name != "Trello" {
		t.Errorf("expected manifest name 'Trello', got %q", m.Name)
	}
	if len(m.Actions) != 6 {
		t.Errorf("expected 6 actions in manifest, got %d", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 {
		t.Errorf("expected 1 required credential, got %d", len(m.RequiredCredentials))
	}
	if m.RequiredCredentials[0].AuthType != "api_key" {
		t.Errorf("expected auth type 'api_key', got %q", m.RequiredCredentials[0].AuthType)
	}

	// Verify all action schemas parse as valid JSON.
	for _, a := range m.Actions {
		var schema map[string]any
		if err := json.Unmarshal(a.ParametersSchema, &schema); err != nil {
			t.Errorf("action %q has invalid JSON schema: %v", a.ActionType, err)
		}
	}
}

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	err := checkResponse(200, http.Header{}, []byte("OK"))
	if err != nil {
		t.Errorf("expected nil error for 200, got: %v", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "5")
	err := checkResponse(429, h, []byte("Rate limit"))
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	err := checkResponse(401, http.Header{}, []byte("unauthorized"))
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	err := checkResponse(403, http.Header{}, []byte("forbidden"))
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	err := checkResponse(400, http.Header{}, []byte("invalid value"))
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, []byte("not found"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte("server error"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestDo_QueryParamAuth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth is in query params, NOT in headers.
		if r.URL.Query().Get("key") != "test-api-key-123" {
			t.Errorf("expected key in query params, got %q", r.URL.Query().Get("key"))
		}
		if r.URL.Query().Get("token") != "test-token-456" {
			t.Errorf("expected token in query params, got %q", r.URL.Query().Get("token"))
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "me123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/members/me", nil, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["id"] != "me123" {
		t.Errorf("expected id=me123, got %q", resp["id"])
	}
}
