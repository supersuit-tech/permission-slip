package supabase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/rest/v1/users" {
			t.Errorf("expected path /rest/v1/users, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "eq.1" {
			t.Errorf("expected filter id=eq.1, got %q", r.URL.Query().Get("id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(deleteParams{
		Table:   "users",
		Filters: map[string]string{"id": "eq.1"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	affected, ok := data["rows_affected"].(float64)
	if !ok || int(affected) != 1 {
		t.Errorf("expected rows_affected 1, got %v", data["rows_affected"])
	}
}

func TestDelete_MissingFilters(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"table": "users",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing filters")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDelete_MissingTable(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"filters": map[string]string{"id": "eq.1"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing table")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDelete_TableNotInAllowlist(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(deleteParams{
		Table:         "secrets",
		Filters:       map[string]string{"id": "eq.1"},
		AllowedTables: []string{"users"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for table not in allowlist")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDelete_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Invalid API key",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(deleteParams{
		Table:   "users",
		Filters: map[string]string{"id": "eq.1"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestDelete_InvalidFilterOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(deleteParams{
		Table:   "users",
		Filters: map[string]string{"status": "badop.active"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid filter operator")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDelete_Forbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "new row violates row-level security policy",
			"code":    "42501",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &deleteAction{conn: conn}

	params, _ := json.Marshal(deleteParams{
		Table:   "users",
		Filters: map[string]string{"id": "eq.1"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.delete",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for forbidden")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
