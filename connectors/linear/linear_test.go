package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestLinearConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "linear" {
		t.Errorf("expected ID 'linear', got %q", c.ID())
	}
}

func TestLinearConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"linear.create_issue",
		"linear.update_issue",
		"linear.add_comment",
		"linear.create_project",
		"linear.search_issues",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestLinearConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "lin_api_abc123"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "wrong credential key",
			creds:   connectors.NewCredentials(map[string]string{"token": "lin_api_abc123"}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestLinearConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "linear" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "linear")
	}
	if m.Name != "Linear" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Linear")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"linear.create_issue",
		"linear.update_issue",
		"linear.add_comment",
		"linear.create_project",
		"linear.search_issues",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "linear" {
		t.Errorf("credential service = %q, want %q", cred.Service, "linear")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestLinearConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestLinearConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*LinearConnector)(nil)
	var _ connectors.ManifestProvider = (*LinearConnector)(nil)
}

// --- doGraphQL tests ---

func TestDoGraphQL_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"viewer": map[string]string{
					"id":   "user-123",
					"name": "Test User",
				},
			},
		},
		wantAuth: "lin_api_test_key_123",
	}

	conn, _ := newTestServer(t, handler)

	var dest struct {
		Viewer struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"viewer"`
	}

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id name } }", nil, &dest)
	if err != nil {
		t.Fatalf("doGraphQL() error = %v", err)
	}
	if dest.Viewer.ID != "user-123" {
		t.Errorf("Viewer.ID = %q, want %q", dest.Viewer.ID, "user-123")
	}
	if dest.Viewer.Name != "Test User" {
		t.Errorf("Viewer.Name = %q, want %q", dest.Viewer.Name, "Test User")
	}
}

func TestDoGraphQL_AuthHeader(t *testing.T) {
	t.Parallel()

	// Verify the auth header uses the API key directly (no "Bearer" prefix).
	handler := &graphQLHandler{
		t:        t,
		response: map[string]any{"data": nil},
		wantAuth: "my-key-no-bearer",
	}

	conn, _ := newTestServer(t, handler)
	creds := connectors.NewCredentials(map[string]string{"api_key": "my-key-no-bearer"})

	_ = conn.doGraphQL(t.Context(), creds, "{ viewer { id } }", nil, nil)
}

func TestDoGraphQL_Variables(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t:        t,
		response: map[string]any{"data": json.RawMessage(`{}`)},
	}

	conn, _ := newTestServer(t, handler)

	vars := map[string]any{
		"teamId": "team-1",
		"title":  "Test Issue",
	}

	err := conn.doGraphQL(t.Context(), validCreds(), "mutation { issueCreate(input: $input) { issue { id } } }", vars, nil)
	if err != nil {
		t.Fatalf("doGraphQL() error = %v", err)
	}
}

func TestDoGraphQL_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	emptyCreds := connectors.NewCredentials(map[string]string{})

	err := conn.doGraphQL(t.Context(), emptyCreds, "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_HTTPUnauthorized(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t:        t,
		response: map[string]any{"error": "unauthorized"},
		status:   http.StatusUnauthorized,
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_HTTPForbidden(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t:        t,
		response: map[string]any{"error": "forbidden"},
		status:   http.StatusForbidden,
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_RateLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	t.Cleanup(server.Close)

	conn := newForTest(server.Client(), server.URL)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter != 30*1e9 { // 30 seconds in nanoseconds
			t.Errorf("RetryAfter = %v, want 30s", rlErr.RetryAfter)
		}
	}
}

func TestDoGraphQL_GraphQLAuthError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{
					"message":    "Authentication required",
					"extensions": map[string]any{"type": "authentication_error"},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for authentication_error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_GraphQLForbiddenError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{
					"message":    "Forbidden",
					"extensions": map[string]any{"type": "forbidden"},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for forbidden")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_GraphQLRateLimitError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{
					"message":    "Rate limited",
					"extensions": map[string]any{"type": "ratelimited"},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for ratelimited")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_GraphQLValidationError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{
					"message":    "Invalid input",
					"extensions": map[string]any{"type": "validation_error"},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "mutation { issueCreate }", nil, nil)
	if err == nil {
		t.Fatal("expected error for validation_error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_GraphQLGenericError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{"message": "Something went wrong"},
			},
		},
	}

	conn, _ := newTestServer(t, handler)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for generic GraphQL error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(server.Close)

	conn := newForTest(server.Client(), server.URL)

	err := conn.doGraphQL(t.Context(), validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestDoGraphQL_Timeout(t *testing.T) {
	t.Parallel()

	// Create a context that's already canceled to simulate a timeout.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := New()

	err := conn.doGraphQL(ctx, validCreds(), "{ viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestMapGraphQLErrors_Empty(t *testing.T) {
	t.Parallel()
	if err := mapGraphQLErrors(nil); err != nil {
		t.Errorf("expected nil for empty errors, got %v", err)
	}
}
