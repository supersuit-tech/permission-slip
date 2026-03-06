package amadeus

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestPriceFlight_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/shopping/flight-offers/pricing" {
			t.Errorf("path = %s, want /v1/shopping/flight-offers/pricing", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		data, ok := reqBody["data"].(map[string]any)
		if !ok {
			t.Fatal("expected data object in request body")
		}
		if data["type"] != "flight-offers-pricing" {
			t.Errorf("type = %v, want flight-offers-pricing", data["type"])
		}
		offers, ok := data["flightOffers"].([]any)
		if !ok || len(offers) != 1 {
			t.Fatalf("expected 1 flight offer, got %v", data["flightOffers"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"type":         "flight-offers-pricing",
				"flightOffers": []map[string]any{{"id": "1", "price": map[string]any{"total": "150.00", "currency": "USD"}}},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.price_flight"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.price_flight",
		Parameters:  json.RawMessage(`{"flight_offer":{"id":"1","type":"flight-offer","source":"GDS"}}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if _, ok := data["data"]; !ok {
		t.Error("expected data in response")
	}
}

func TestPriceFlight_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.price_flight"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing flight_offer", `{}`},
		{"null flight_offer", `{"flight_offer":null}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.price_flight",
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

func TestPriceFlight_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(amadeusErrorResponse(400, 477, "INVALID FORMAT", "flight offer is no longer valid"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.price_flight"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.price_flight",
		Parameters:  json.RawMessage(`{"flight_offer":{"id":"1","type":"flight-offer"}}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
