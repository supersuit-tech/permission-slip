package amadeus

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchFlights_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v2/shopping/flight-offers" {
			t.Errorf("path = %s, want /v2/shopping/flight-offers", r.URL.Path)
		}

		q := r.URL.Query()
		if got := q.Get("originLocationCode"); got != "SFO" {
			t.Errorf("originLocationCode = %q, want %q", got, "SFO")
		}
		if got := q.Get("destinationLocationCode"); got != "LAX" {
			t.Errorf("destinationLocationCode = %q, want %q", got, "LAX")
		}
		if got := q.Get("departureDate"); got != "2024-06-15" {
			t.Errorf("departureDate = %q, want %q", got, "2024-06-15")
		}
		if got := q.Get("adults"); got != "2" {
			t.Errorf("adults = %q, want %q", got, "2")
		}
		if got := q.Get("travelClass"); got != "BUSINESS" {
			t.Errorf("travelClass = %q, want %q", got, "BUSINESS")
		}
		if got := q.Get("nonStop"); got != "true" {
			t.Errorf("nonStop = %q, want %q", got, "true")
		}
		if got := q.Get("max"); got != "5" {
			t.Errorf("max = %q, want %q", got, "5")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"type": "flight-offer", "id": "1", "source": "GDS"},
			},
			"dictionaries": map[string]any{
				"carriers": map[string]string{"UA": "UNITED AIRLINES"},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_flights"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_flights",
		Parameters:  json.RawMessage(`{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","adults":2,"cabin":"BUSINESS","nonstop":true,"max_results":5}`),
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
	if _, ok := data["dictionaries"]; !ok {
		t.Error("expected dictionaries in response")
	}
}

func TestSearchFlights_Defaults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("adults"); got != "1" {
			t.Errorf("adults = %q, want %q (default)", got, "1")
		}
		if got := q.Get("max"); got != "10" {
			t.Errorf("max = %q, want %q (default)", got, "10")
		}
		if got := q.Get("travelClass"); got != "" {
			t.Errorf("travelClass = %q, want empty (not set)", got)
		}
		if got := q.Get("nonStop"); got != "" {
			t.Errorf("nonStop = %q, want empty (not set)", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_flights"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_flights",
		Parameters:  json.RawMessage(`{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchFlights_WithReturnDate(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("returnDate"); got != "2024-06-20" {
			t.Errorf("returnDate = %q, want %q", got, "2024-06-20")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_flights"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_flights",
		Parameters:  json.RawMessage(`{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","return_date":"2024-06-20","adults":1}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchFlights_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.search_flights"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing origin", `{"destination":"LAX","departure_date":"2024-06-15","adults":1}`},
		{"invalid origin format", `{"origin":"sfo","destination":"LAX","departure_date":"2024-06-15","adults":1}`},
		{"missing destination", `{"origin":"SFO","departure_date":"2024-06-15","adults":1}`},
		{"invalid destination format", `{"origin":"SFO","destination":"la","departure_date":"2024-06-15","adults":1}`},
		{"missing departure_date", `{"origin":"SFO","destination":"LAX","adults":1}`},
		{"invalid departure_date", `{"origin":"SFO","destination":"LAX","departure_date":"not-a-date","adults":1}`},
		{"invalid return_date", `{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","return_date":"bad","adults":1}`},
		{"adults too high", `{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","adults":10}`},
		{"invalid cabin", `{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","adults":1,"cabin":"LUXURY"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.search_flights",
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

func TestSearchFlights_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(amadeusErrorResponse(400, 477, "INVALID FORMAT", "invalid date format"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_flights"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_flights",
		Parameters:  json.RawMessage(`{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","adults":1}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchFlights_RateLimit(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(amadeusErrorResponse(429, 0, "RATE LIMIT", "rate limit exceeded"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.search_flights"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.search_flights",
		Parameters:  json.RawMessage(`{"origin":"SFO","destination":"LAX","departure_date":"2024-06-15","adults":1}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}
