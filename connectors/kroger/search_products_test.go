package kroger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchProducts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/products" {
			t.Errorf("path = %s, want /products", got)
		}
		if got := r.URL.Query().Get("filter.term"); got != "milk" {
			t.Errorf("filter.term = %q, want %q", got, "milk")
		}
		if got := r.URL.Query().Get("filter.limit"); got != "10" {
			t.Errorf("filter.limit = %q, want %q", got, "10")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_access_token_123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_access_token_123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"productId": "0001111041700", "description": "Kroger 2% Milk"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_products"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"milk"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	items, ok := data["data"].([]any)
	if !ok || len(items) != 1 {
		t.Errorf("expected 1 item, got %v", data["data"])
	}
}

func TestSearchProducts_WithLocation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter.locationId"); got != "01400376" {
			t.Errorf("filter.locationId = %q, want %q", got, "01400376")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"bread","location_id":"01400376"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchProducts_MissingTerm(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
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

func TestSearchProducts_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchProducts_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Unauthorized"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestSearchProducts_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Rate limit exceeded"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestSearchProducts_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestSearchProducts_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_products",
		Parameters:  json.RawMessage(`{"term":"milk","limit":100}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
