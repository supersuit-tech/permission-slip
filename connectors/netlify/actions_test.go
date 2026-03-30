package netlify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListSites_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_netlify_token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_netlify_token")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "site_123", "name": "my-site"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestListSites_OAuthCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test_oauth_token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_oauth_token")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "site_123", "name": "my-site"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: oauthCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestListDeployments_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "deploy_123", "state": "ready"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_deployments"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_deployments",
		Parameters:  json.RawMessage(`{"site_id":"site_123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestListDeployments_MissingSiteID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["netlify.list_deployments"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_deployments",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetDeployment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "deploy_123",
			"state": "ready",
			"url":   "https://my-site.netlify.app",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.get_deployment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.get_deployment",
		Parameters:  json.RawMessage(`{"deploy_id":"deploy_123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestGetDeployment_MissingID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["netlify.get_deployment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.get_deployment",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestTriggerDeployment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "build_123",
			"state": "building",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.trigger_deployment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.trigger_deployment",
		Parameters:  json.RawMessage(`{"site_id":"site_123","branch":"main"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestTriggerDeployment_MissingSiteID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["netlify.trigger_deployment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.trigger_deployment",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestRollbackDeployment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "deploy_old",
			"state": "ready",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.rollback_deployment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.rollback_deployment",
		Parameters:  json.RawMessage(`{"site_id":"site_123","deploy_id":"deploy_old"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestRollbackDeployment_MissingParams(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["netlify.rollback_deployment"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing site_id", params: `{"deploy_id":"d"}`},
		{name: "missing deploy_id", params: `{"site_id":"s"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "netlify.rollback_deployment",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSetEnvVar_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"key": "DATABASE_URL", "scopes": []string{"builds", "functions"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.set_env_var"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.set_env_var",
		Parameters:  json.RawMessage(`{"account_slug":"my-team","site_id":"site_123","key":"DATABASE_URL","values":[{"value":"postgres://...","context":"all"}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestSetEnvVar_MissingParams(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["netlify.set_env_var"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing account_slug", params: `{"site_id":"s","key":"K","values":[{"value":"V","context":"all"}]}`},
		{name: "missing site_id", params: `{"account_slug":"a","key":"K","values":[{"value":"V","context":"all"}]}`},
		{name: "missing key", params: `{"account_slug":"a","site_id":"s","values":[{"value":"V","context":"all"}]}`},
		{name: "missing values", params: `{"account_slug":"a","site_id":"s","key":"K"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "netlify.set_env_var",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestDeleteEnvVar_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.delete_env_var"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.delete_env_var",
		Parameters:  json.RawMessage(`{"account_slug":"my-team","key":"OLD_VAR"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestListEnvVars_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"key": "DATABASE_URL", "scopes": []string{"builds"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_env_vars"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_env_vars",
		Parameters:  json.RawMessage(`{"account_slug":"my-team","site_id":"site_123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestNetlify_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestNetlify_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid token",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "bad_token"}),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestNetlify_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
		}
	}
}

func TestNetlify_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["netlify.list_sites"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "netlify.list_sites",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
