package supabase

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/rest/v1/users" {
			t.Errorf("expected path /rest/v1/users, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "eq.1" {
			t.Errorf("expected filter id=eq.1, got %q", r.URL.Query().Get("id"))
		}

		body, _ := io.ReadAll(r.Body)
		var set map[string]any
		if err := json.Unmarshal(body, &set); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if set["name"] != "Alice Updated" {
			t.Errorf("expected name 'Alice Updated', got %v", set["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice Updated", "email": "alice@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &updateAction{conn: conn}

	params, _ := json.Marshal(updateParams{
		Table:   "users",
		Set:     map[string]any{"name": "Alice Updated"},
		Filters: map[string]string{"id": "eq.1"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.update",
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

func TestUpdate_MissingFilters(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"table": "users",
		"set":   map[string]any{"name": "Hacked"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.update",
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

func TestUpdate_MissingSet(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"table":   "users",
		"filters": map[string]string{"id": "eq.1"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.update",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing set")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdate_TableNotInAllowlist(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateAction{conn: conn}

	params, _ := json.Marshal(updateParams{
		Table:         "secrets",
		Set:           map[string]any{"val": "x"},
		Filters:       map[string]string{"id": "eq.1"},
		AllowedTables: []string{"users"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.update",
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
