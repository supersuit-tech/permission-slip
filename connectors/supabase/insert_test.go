package supabase

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInsert_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/v1/users" {
			t.Errorf("expected path /rest/v1/users, got %s", r.URL.Path)
		}
		if r.Header.Get("Prefer") != "return=representation" {
			t.Errorf("expected Prefer: return=representation, got %q", r.Header.Get("Prefer"))
		}

		body, _ := io.ReadAll(r.Body)
		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 row in body, got %d", len(rows))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &insertAction{conn: conn}

	params, _ := json.Marshal(insertParams{
		Table: "users",
		Rows: []map[string]any{
			{"name": "Alice", "email": "alice@example.com"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.insert",
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

func TestInsert_MissingRows(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &insertAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"table": "users"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.insert",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing rows")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestInsert_MissingTable(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &insertAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"rows": []map[string]any{{"name": "Alice"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.insert",
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

func TestInsert_TableNotInAllowlist(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &insertAction{conn: conn}

	params, _ := json.Marshal(insertParams{
		Table:         "secrets",
		Rows:          []map[string]any{{"key": "val"}},
		AllowedTables: []string{"users"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.insert",
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

func TestInsert_WithOnConflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("on_conflict") != "email" {
			t.Errorf("expected on_conflict=email, got %q", r.URL.Query().Get("on_conflict"))
		}
		// PostgREST requires the resolution directive for upsert behavior.
		prefer := r.Header.Get("Prefer")
		if prefer == "" {
			t.Error("expected Prefer header to be set for upsert")
		}
		if !strings.Contains(prefer, "resolution=merge-duplicates") {
			t.Errorf("expected Prefer to contain resolution=merge-duplicates, got %q", prefer)
		}
		if !strings.Contains(prefer, "return=representation") {
			t.Errorf("expected Prefer to contain return=representation, got %q", prefer)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &insertAction{conn: conn}

	params, _ := json.Marshal(insertParams{
		Table:      "users",
		Rows:       []map[string]any{{"name": "Alice", "email": "alice@example.com"}},
		OnConflict: "email",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "supabase.insert",
		Parameters:  params,
		Credentials: validCredsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
