package kroger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchLocations_ByZip(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/locations" {
			t.Errorf("path = %s, want /locations", got)
		}
		if got := r.URL.Query().Get("filter.zipCode.near"); got != "45202" {
			t.Errorf("filter.zipCode.near = %q, want %q", got, "45202")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_access_token_123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_access_token_123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"locationId": "01400376", "name": "Kroger"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_locations"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_locations",
		Parameters:  json.RawMessage(`{"zip_code":"45202"}`),
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
		t.Errorf("expected 1 location, got %v", data["data"])
	}
}

func TestSearchLocations_ByCoordinates(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter.lat.near"); got == "" {
			t.Error("expected filter.lat.near to be set")
		}
		if got := r.URL.Query().Get("filter.lon.near"); got == "" {
			t.Error("expected filter.lon.near to be set")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_locations"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_locations",
		Parameters:  json.RawMessage(`{"lat":39.1031,"lon":-84.5120}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchLocations_WithChainFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter.chain"); got != "Ralphs" {
			t.Errorf("filter.chain = %q, want %q", got, "Ralphs")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.search_locations"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_locations",
		Parameters:  json.RawMessage(`{"zip_code":"90210","chain":"Ralphs"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchLocations_InvalidRadius(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.search_locations"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_locations",
		Parameters:  json.RawMessage(`{"zip_code":"45202","radius_miles":200}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchLocations_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.search_locations"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.search_locations",
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
