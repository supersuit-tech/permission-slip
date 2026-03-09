package supabase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestRead_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/rest/v1/users" {
			t.Errorf("expected path /rest/v1/users, got %s", r.URL.Path)
		}
		if r.Header.Get("apikey") == "" {
			t.Error("expected apikey header to be set")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header to be set")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{Table: "users"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
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
	rows, ok := data["rows"].([]any)
	if !ok {
		t.Fatal("expected rows in result")
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	count, ok := data["count"].(float64)
	if !ok || int(count) != 2 {
		t.Errorf("expected count 2, got %v", data["count"])
	}
}

func TestRead_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "eq.active" {
			t.Errorf("expected filter status=eq.active, got %q", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("age") != "gte.18" {
			t.Errorf("expected filter age=gte.18, got %q", r.URL.Query().Get("age"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{
		Table:   "users",
		Filters: map[string]string{"status": "eq.active", "age": "gte.18"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRead_WithSelect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("select") != "id,name" {
			t.Errorf("expected select=id,name, got %q", r.URL.Query().Get("select"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{
		Table:  "users",
		Select: "id,name",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRead_MissingTable(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
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

func TestRead_TableNotInAllowlist(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{
		Table:         "secrets",
		AllowedTables: []string{"users", "posts"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
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

func TestRead_TableInAllowlist(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{
		Table:         "users",
		AllowedTables: []string{"users", "posts"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRead_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Relation \"nonexistent\" does not exist",
			"code":    "42P01",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{Table: "nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for undefined table, got: %T", err)
	}
}

func TestRead_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Invalid API key",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &readAction{conn: conn}

	params, _ := json.Marshal(readParams{Table: "users"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
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

func TestRead_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.read",
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
