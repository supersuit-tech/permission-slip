package amadeus

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchAirports_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/reference-data/locations" {
			t.Errorf("path = %s, want /v1/reference-data/locations", r.URL.Path)
		}
		if got := r.URL.Query().Get("keyword"); got != "san francisco" {
			t.Errorf("keyword = %q, want %q", got, "san francisco")
		}
		if got := r.URL.Query().Get("subType"); got != "AIRPORT,CITY" {
			t.Errorf("subType = %q, want %q", got, "AIRPORT,CITY")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-access-token-123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"type":        "location",
					"subType":     "AIRPORT",
					"name":        "SAN FRANCISCO INTL",
					"iataCode":    "SFO",
					"address":     map[string]string{"cityName": "SAN FRANCISCO", "countryCode": "US"},
				},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_airports"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_airports",
		Parameters:  json.RawMessage(`{"keyword":"san francisco"}`),
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
		t.Fatalf("expected 1 result, got %v", data["data"])
	}
}

func TestSearchAirports_SubtypeFilter(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("subType"); got != "AIRPORT" {
			t.Errorf("subType = %q, want %q", got, "AIRPORT")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_airports"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_airports",
		Parameters:  json.RawMessage(`{"keyword":"SFO","subtype":"AIRPORT"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchAirports_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.search_airports"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing keyword", `{}`},
		{"empty keyword", `{"keyword":""}`},
		{"invalid subtype", `{"keyword":"SFO","subtype":"INVALID"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.search_airports",
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

func TestSearchAirports_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(amadeusErrorResponse(500, 141, "SYSTEM ERROR", "internal server error"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_airports"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_airports",
		Parameters:  json.RawMessage(`{"keyword":"SFO"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
