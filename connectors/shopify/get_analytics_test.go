package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetAnalytics_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/reports.json" {
			t.Errorf("path = %s, want /reports.json", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reports": []any{
				map[string]any{"id": 1, "name": "Orders over time", "shopify_ql": "SHOW orders"},
				map[string]any{"id": 2, "name": "Sales over time", "shopify_ql": "SHOW sales"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_analytics"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["reports"]; !ok {
		t.Error("result missing 'reports' key")
	}
}

func TestGetAnalytics_WithQueryParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		q := r.URL.Query()
		if got := q.Get("since"); got != "2024-01-01T00:00:00Z" {
			t.Errorf("since = %q, want 2024-01-01T00:00:00Z", got)
		}
		if got := q.Get("until"); got != "2024-06-30T23:59:59Z" {
			t.Errorf("until = %q, want 2024-06-30T23:59:59Z", got)
		}
		if got := q.Get("fields"); got != "id,name" {
			t.Errorf("fields = %q, want id,name", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reports": []any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_analytics"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{"since":"2024-01-01T00:00:00Z","until":"2024-06-30T23:59:59Z","fields":"id,name"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestGetAnalytics_NoParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reports": []any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_analytics"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestGetAnalytics_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_analytics"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetAnalytics_APIAuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":"[API] This action requires merchant approval for read_reports scope."}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_analytics"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got %T: %v", err, err)
	}
}

func TestGetAnalytics_APIRateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errors":"Exceeded 2 calls per second for api client."}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_analytics"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_analytics",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError for 429, got %T: %v", err, err)
	}
}
