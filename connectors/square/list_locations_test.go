package square

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListLocations_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/locations" {
			t.Errorf("path = %s, want /locations", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"locations": []map[string]any{
				{"id": "LOC1", "name": "Main Street", "status": "ACTIVE"},
				{"id": "LOC2", "name": "Downtown", "status": "ACTIVE"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_locations"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_locations",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["locations"]; !ok {
		t.Error("result missing 'locations' key")
	}
}

func TestListLocations_EmptyLocations(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_locations"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_locations",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	// Should normalize null to empty array.
	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var locations []any
	if err := json.Unmarshal(data["locations"], &locations); err != nil {
		t.Fatalf("unmarshal locations: %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected empty locations, got %d", len(locations))
	}
}

func TestListLocations_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.list_locations"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_locations",
		Parameters:  json.RawMessage(`{bad}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListLocations_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "AUTHENTICATION_ERROR", "code": "UNAUTHORIZED"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_locations"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_locations",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
