package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMetaConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "meta" {
		t.Errorf("expected ID 'meta', got %q", c.ID())
	}
}

func TestMetaConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"meta.create_page_post",
		"meta.delete_page_post",
		"meta.reply_page_comment",
		"meta.create_instagram_post",
		"meta.get_instagram_insights",
		"meta.list_page_posts",
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

func TestMetaConnector_ValidateCredentials_Success(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetaConnector_ValidateCredentials_MissingToken(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{}))
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMetaConnector_ValidateCredentials_EmptyToken(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{
		"access_token": "",
	}))
	if err == nil {
		t.Fatal("expected error for empty access_token")
	}
}

func TestMetaConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "meta" {
		t.Errorf("expected manifest ID 'meta', got %q", m.ID)
	}
	if len(m.Actions) != 6 {
		t.Errorf("expected 6 actions, got %d", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("expected 1 required credential, got %d", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.AuthType != "oauth2" {
		t.Errorf("expected auth_type 'oauth2', got %q", cred.AuthType)
	}
	if cred.OAuthProvider != "meta" {
		t.Errorf("expected oauth_provider 'meta', got %q", cred.OAuthProvider)
	}
	if len(cred.OAuthScopes) == 0 {
		t.Error("expected non-empty oauth scopes")
	}

	// Validate the manifest is well-formed.
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest validation failed: %v", err)
	}
}

func TestCheckResponse_ErrorCode190(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "Invalid access token",
			"type":    "OAuthException",
			"code":    190,
		},
	})
	err := checkResponse(401, body)
	if err == nil {
		t.Fatal("expected error for code 190")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_ErrorCode4(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "Too many calls",
			"type":    "OAuthException",
			"code":    4,
		},
	})
	err := checkResponse(429, body)
	if err == nil {
		t.Fatal("expected error for code 4")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_ErrorCode100(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "Invalid parameter",
			"type":    "OAuthException",
			"code":    100,
		},
	})
	err := checkResponse(400, body)
	if err == nil {
		t.Fatal("expected error for code 100")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	err := checkResponse(200, []byte(`{"id":"123"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoJSON_Timeout(t *testing.T) {
	t.Parallel()

	// Create a server that never responds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate server error.
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for gateway timeout")
	}
}
