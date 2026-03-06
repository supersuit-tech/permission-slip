package amadeus

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchCars_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/shopping/transfer-offers" {
			t.Errorf("path = %s, want /v2/shopping/transfer-offers", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["startLocationCode"] != "LAX" {
			t.Errorf("startLocationCode = %v, want LAX", reqBody["startLocationCode"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"type": "transfer-offer", "id": "1", "vehicle": map[string]string{"category": "SEDAN"}},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_cars"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_cars",
		Parameters:  json.RawMessage(`{"pickup_location":"LAX","pickup_date":"2024-07-01T10:00:00","dropoff_date":"2024-07-05T10:00:00"}`),
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

func TestSearchCars_WithDropoffLocation(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["endLocationCode"] != "SFO" {
			t.Errorf("endLocationCode = %v, want SFO", reqBody["endLocationCode"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_cars"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_cars",
		Parameters:  json.RawMessage(`{"pickup_location":"LAX","pickup_date":"2024-07-01T10:00:00","dropoff_date":"2024-07-05T10:00:00","dropoff_location":"SFO"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchCars_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.search_cars"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing pickup_location", `{"pickup_date":"2024-07-01","dropoff_date":"2024-07-05"}`},
		{"missing pickup_date", `{"pickup_location":"LAX","dropoff_date":"2024-07-05"}`},
		{"missing dropoff_date", `{"pickup_location":"LAX","pickup_date":"2024-07-01"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.search_cars",
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

func TestSearchCars_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(amadeusErrorResponse(500, 141, "SYSTEM ERROR", "service unavailable"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_cars"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_cars",
		Parameters:  json.RawMessage(`{"pickup_location":"LAX","pickup_date":"2024-07-01","dropoff_date":"2024-07-05"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
