package expedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetHotel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/v3/properties/content/12345" {
			t.Errorf("path = %s, want /v3/properties/content/12345", got)
		}
		q := r.URL.Query()
		if got := q.Get("language"); got != "en-US" {
			t.Errorf("language = %q, want %q", got, "en-US")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"property_id": "12345",
			"name":        "Grand Hotel",
			"star_rating": 4,
			"amenities":   []string{"wifi", "pool"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.get_hotel"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.get_hotel",
		Parameters:  json.RawMessage(`{"property_id":"12345"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["property_id"] != "12345" {
		t.Errorf("property_id = %v, want 12345", data["property_id"])
	}
	if data["name"] != "Grand Hotel" {
		t.Errorf("name = %v, want Grand Hotel", data["name"])
	}
}

func TestGetHotel_WithRateParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("checkin"); got != "2024-06-15" {
			t.Errorf("checkin = %q, want %q", got, "2024-06-15")
		}
		if got := q.Get("checkout"); got != "2024-06-17" {
			t.Errorf("checkout = %q, want %q", got, "2024-06-17")
		}
		if got := q.Get("occupancy"); got != "2" {
			t.Errorf("occupancy = %q, want %q", got, "2")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"property_id": "12345"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.get_hotel"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.get_hotel",
		Parameters:  json.RawMessage(`{"property_id":"12345","checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetHotel_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.get_hotel"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing property_id",
			params: `{}`,
		},
		{
			name:   "empty property_id",
			params: `{"property_id":""}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.get_hotel",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestGetHotel_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "resource_not_found",
			"message": "Property not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.get_hotel"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.get_hotel",
		Parameters:  json.RawMessage(`{"property_id":"99999"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
