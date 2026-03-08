package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetFollowers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/users/99/followers") {
			t.Errorf("path = %s, want prefix /users/99/followers", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "1", "name": "Alice", "username": "alice"},
			},
			"meta": map[string]any{"result_count": 1},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.get_followers"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_followers",
		Parameters:  json.RawMessage(`{"user_id":"99"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || len(result.Data) == 0 {
		t.Fatal("Execute() returned empty result")
	}
}

func TestGetFollowers_MissingUserID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.get_followers"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_followers",
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

func TestGetFollowing_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/users/99/following") {
			t.Errorf("path = %s, want prefix /users/99/following", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "2", "name": "Bob", "username": "bob"},
			},
			"meta": map[string]any{"result_count": 1},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.get_following"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_following",
		Parameters:  json.RawMessage(`{"user_id":"99"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || len(result.Data) == 0 {
		t.Fatal("Execute() returned empty result")
	}
}
