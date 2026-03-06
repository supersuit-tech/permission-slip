package amadeus

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchHotels_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		switch {
		case r.URL.Path == "/v1/reference-data/locations/hotels/by-city":
			if got := r.URL.Query().Get("cityCode"); got != "PAR" {
				t.Errorf("cityCode = %q, want %q", got, "PAR")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"hotelId": "HLPAR001"},
					{"hotelId": "HLPAR002"},
				},
			})

		case r.URL.Path == "/v3/shopping/hotel-offers":
			if got := r.URL.Query().Get("hotelIds"); got != "HLPAR001,HLPAR002" {
				t.Errorf("hotelIds = %q, want %q", got, "HLPAR001,HLPAR002")
			}
			if got := r.URL.Query().Get("checkInDate"); got != "2024-07-01" {
				t.Errorf("checkInDate = %q, want %q", got, "2024-07-01")
			}
			if got := r.URL.Query().Get("checkOutDate"); got != "2024-07-05" {
				t.Errorf("checkOutDate = %q, want %q", got, "2024-07-05")
			}
			if got := r.URL.Query().Get("adults"); got != "2" {
				t.Errorf("adults = %q, want %q", got, "2")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"type": "hotel-offers", "hotel": map[string]string{"hotelId": "HLPAR001", "name": "Hotel Paris"}},
				},
			})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_hotels"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_hotels",
		Parameters:  json.RawMessage(`{"city_code":"PAR","check_in_date":"2024-07-01","check_out_date":"2024-07-05","adults":2}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (hotel list + offers), got %d", callCount)
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

func TestSearchHotels_ByGeocode(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/reference-data/locations/hotels/by-geocode":
			if got := r.URL.Query().Get("latitude"); got != "48.8566" {
				t.Errorf("latitude = %q, want %q", got, "48.8566")
			}
			if got := r.URL.Query().Get("longitude"); got != "2.3522" {
				t.Errorf("longitude = %q, want %q", got, "2.3522")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"hotelId": "HLGEO001"}},
			})

		case r.URL.Path == "/v3/shopping/hotel-offers":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_hotels"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_hotels",
		Parameters:  json.RawMessage(`{"latitude":"48.8566","longitude":"2.3522","check_in_date":"2024-07-01","check_out_date":"2024-07-05"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchHotels_NoHotelsFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_hotels"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_hotels",
		Parameters:  json.RawMessage(`{"city_code":"XXX","check_in_date":"2024-07-01","check_out_date":"2024-07-05"}`),
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
	if !ok || len(items) != 0 {
		t.Fatalf("expected empty data, got %v", data["data"])
	}
}

func TestSearchHotels_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.search_hotels"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing location", `{"check_in_date":"2024-07-01","check_out_date":"2024-07-05"}`},
		{"missing check_in_date", `{"city_code":"PAR","check_out_date":"2024-07-05"}`},
		{"missing check_out_date", `{"city_code":"PAR","check_in_date":"2024-07-01"}`},
		{"partial geocode", `{"latitude":"48.8566","check_in_date":"2024-07-01","check_out_date":"2024-07-05"}`},
		{"invalid rating", `{"city_code":"PAR","check_in_date":"2024-07-01","check_out_date":"2024-07-05","ratings":[6]}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.search_hotels",
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

func TestSearchHotels_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(amadeusErrorResponse(500, 141, "SYSTEM ERROR", "internal error"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_hotels"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_hotels",
		Parameters:  json.RawMessage(`{"city_code":"PAR","check_in_date":"2024-07-01","check_out_date":"2024-07-05"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
