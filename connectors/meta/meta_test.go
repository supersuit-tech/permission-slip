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
		"meta.create_instagram_story",
		"meta.get_page_insights",
		"meta.list_instagram_posts",
		"meta.reply_instagram_comment",
		"meta.create_ad",
		"meta.create_ad_campaign",
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
	if len(m.Actions) != 12 {
		t.Errorf("expected 12 actions, got %d", len(m.Actions))
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
	err := checkResponse(401, nil, body)
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
	err := checkResponse(429, nil, body)
	if err == nil {
		t.Fatal("expected error for code 4")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_ErrorCode17_UserRateLimit(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "User request limit reached",
			"type":    "OAuthException",
			"code":    17,
		},
	})
	header := http.Header{"Retry-After": []string{"120"}}
	err := checkResponse(429, header, body)
	if err == nil {
		t.Fatal("expected error for code 17")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_RateLimitRetryAfterHeader(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "Too many calls",
			"code":    4,
		},
	})
	header := http.Header{"Retry-After": []string{"90"}}
	err := checkResponse(429, header, body)
	if err == nil {
		t.Fatal("expected error")
	}
	var rlErr *connectors.RateLimitError
	if !connectors.AsRateLimitError(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got: %T", err)
	}
	if rlErr.RetryAfter.Seconds() != 90 {
		t.Errorf("expected RetryAfter=90s, got %v", rlErr.RetryAfter)
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
	err := checkResponse(400, nil, body)
	if err == nil {
		t.Fatal("expected error for code 100")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	err := checkResponse(200, nil, []byte(`{"id":"123"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsValidGraphID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		valid bool
	}{
		{"123456789", true},
		{"123456_789012", true},
		{"ig_123", true},
		{"page123", true},
		{"", false},
		{"../etc/passwd", false},
		{"123?inject=true", false},
		{"123/456", false},
		{"123\n456", false},
		{"id with spaces", false},
	}

	for _, tc := range tests {
		if got := isValidGraphID(tc.input); got != tc.valid {
			t.Errorf("isValidGraphID(%q) = %v, want %v", tc.input, got, tc.valid)
		}
	}
}

func TestCreatePagePost_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPagePostAction{conn: conn}

	params, _ := json.Marshal(createPagePostParams{
		PageID:  "../other-endpoint",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in page_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeletePagePost_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deletePagePostAction{conn: conn}

	params, _ := json.Marshal(deletePagePostParams{
		PostID: "123?delete_all=true",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.delete_page_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for query injection in post_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
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
