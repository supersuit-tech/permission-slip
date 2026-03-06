package amadeus

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestBookFlight_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/booking/flight-orders" {
			t.Errorf("path = %s, want /v1/booking/flight-orders", r.URL.Path)
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
		if data["type"] != "flight-order" {
			t.Errorf("type = %v, want flight-order", data["type"])
		}
		travelers, ok := data["travelers"].([]any)
		if !ok || len(travelers) != 1 {
			t.Fatalf("expected 1 traveler, got %v", data["travelers"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"type":              "flight-order",
				"id":                "ORDER123",
				"associatedRecords": []map[string]string{{"reference": "ABC123", "originSystemCode": "GDS"}},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.book_flight"]

	params := `{
		"flight_offer": {"id":"1","type":"flight-offer"},
		"travelers": [{"name":"John Doe","dateOfBirth":"1990-01-15","gender":"MALE","contact":{"email":"john@example.com","phone":"+14155551234"}}],
		"payment_method_id": "pm_abc123"
	}`

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.book_flight",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	inner, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object in result")
	}
	if inner["id"] != "ORDER123" {
		t.Errorf("id = %v, want ORDER123", inner["id"])
	}
}

func TestBookFlight_WithRemarks(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		data := reqBody["data"].(map[string]any)
		remarks, ok := data["remarks"].(map[string]any)
		if !ok {
			t.Fatal("expected remarks in request body")
		}
		general, ok := remarks["general"].([]any)
		if !ok || len(general) != 1 {
			t.Fatal("expected 1 general remark")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"type": "flight-order", "id": "ORDER456"},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.book_flight"]

	params := `{
		"flight_offer": {"id":"1"},
		"travelers": [{"name":"Jane","dateOfBirth":"1985-05-20","gender":"FEMALE","contact":{"email":"jane@example.com","phone":"+14155559999"}}],
		"payment_method_id": "pm_xyz",
		"remarks": "Window seat preferred"
	}`

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.book_flight",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestBookFlight_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["amadeus.book_flight"]

	baseTraveler := `[{"name":"John","dateOfBirth":"1990-01-15","gender":"MALE","contact":{"email":"j@e.com","phone":"+1"}}]`

	tests := []struct {
		name   string
		params string
	}{
		{"missing flight_offer", `{"travelers":` + baseTraveler + `,"payment_method_id":"pm_1"}`},
		{"null flight_offer", `{"flight_offer":null,"travelers":` + baseTraveler + `,"payment_method_id":"pm_1"}`},
		{"missing travelers", `{"flight_offer":{"id":"1"},"payment_method_id":"pm_1"}`},
		{"empty travelers", `{"flight_offer":{"id":"1"},"travelers":[],"payment_method_id":"pm_1"}`},
		{"missing payment_method_id", `{"flight_offer":{"id":"1"},"travelers":` + baseTraveler + `}`},
		{"traveler missing name", `{"flight_offer":{"id":"1"},"travelers":[{"dateOfBirth":"1990-01-15","gender":"MALE","contact":{"email":"j@e.com","phone":"+1"}}],"payment_method_id":"pm_1"}`},
		{"traveler missing email", `{"flight_offer":{"id":"1"},"travelers":[{"name":"John","dateOfBirth":"1990-01-15","gender":"MALE","contact":{"phone":"+1"}}],"payment_method_id":"pm_1"}`},
		{"traveler missing phone", `{"flight_offer":{"id":"1"},"travelers":[{"name":"John","dateOfBirth":"1990-01-15","gender":"MALE","contact":{"email":"j@e.com"}}],"payment_method_id":"pm_1"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "amadeus.book_flight",
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

func TestBookFlight_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(amadeusErrorResponse(400, 477, "INVALID FORMAT", "flight offer expired"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["amadeus.book_flight"]

	params := `{
		"flight_offer": {"id":"1"},
		"travelers": [{"name":"John","dateOfBirth":"1990-01-15","gender":"MALE","contact":{"email":"j@e.com","phone":"+1"}}],
		"payment_method_id": "pm_1"
	}`

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "amadeus.book_flight",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
