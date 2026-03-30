package square

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListCatalog_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/catalog/list" {
			t.Errorf("path = %s, want /catalog/list", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"objects": []map[string]any{
				{"type": "ITEM", "id": "ITEM1", "item_data": map[string]any{"name": "Latte"}},
				{"type": "ITEM", "id": "ITEM2", "item_data": map[string]any{"name": "Mocha"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if _, ok := data["objects"]; !ok {
		t.Error("result missing 'objects' key")
	}
}

func TestListCatalog_WithTypes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		types := r.URL.Query().Get("types")
		if types != "ITEM,CATEGORY" {
			t.Errorf("types = %q, want %q", types, "ITEM,CATEGORY")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"objects": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{"types": "ITEM,CATEGORY"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListCatalog_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")

		resp := map[string]any{"objects": []any{}}
		if cursor == "" {
			resp["cursor"] = "next_page_token"
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	// First page — should get a cursor back.
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["cursor"] != "next_page_token" {
		t.Errorf("cursor = %v, want next_page_token", data["cursor"])
	}

	// Second page with cursor — should not have cursor in response.
	result, err = action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{"cursor": "next_page_token"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data2 map[string]any
	json.Unmarshal(result.Data, &data2)
	if _, ok := data2["cursor"]; ok {
		t.Error("cursor should not be present on last page")
	}
}

func TestListCatalog_NullObjectsNormalized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Square returns null objects when catalog is empty.
		w.Write([]byte(`{"objects": null}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	json.Unmarshal(result.Data, &data)
	if string(data["objects"]) != "[]" {
		t.Errorf("objects = %s, want [] (null should be normalized to empty array)", string(data["objects"]))
	}
}

func TestListCatalog_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "AUTHENTICATION_ERROR", "code": "UNAUTHORIZED", "detail": "Invalid token"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_catalog",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestListCatalog_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_catalog"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "square.list_catalog",
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
