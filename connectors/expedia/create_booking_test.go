package expedia

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateBooking_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/v3/itineraries" {
			t.Errorf("path = %s, want /v3/itineraries", got)
		}
		if got := r.URL.Query().Get("room_id"); got != "room-abc-123" {
			t.Errorf("room_id query = %q, want %q", got, "room-abc-123")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		contact, ok := reqBody["contact"].(map[string]any)
		if !ok {
			t.Fatal("missing contact in request body")
		}
		if contact["given_name"] != "John" {
			t.Errorf("given_name = %v, want John", contact["given_name"])
		}
		if contact["family_name"] != "Doe" {
			t.Errorf("family_name = %v, want Doe", contact["family_name"])
		}
		if reqBody["payment_method_id"] != "pm_test_123" {
			t.Errorf("payment_method_id = %v, want pm_test_123", reqBody["payment_method_id"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"itinerary_id": "itin-789",
			"links": map[string]any{
				"retrieve": "/v3/itineraries/itin-789",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.create_booking"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.create_booking",
		Parameters:  json.RawMessage(`{"room_id":"room-abc-123","given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_test_123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["itinerary_id"] != "itin-789" {
		t.Errorf("itinerary_id = %v, want itin-789", data["itinerary_id"])
	}
}

func TestCreateBooking_WithSpecialRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["special_request"] != "Late check-in please" {
			t.Errorf("special_request = %v, want 'Late check-in please'", reqBody["special_request"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"itinerary_id": "itin-789"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.create_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.create_booking",
		Parameters:  json.RawMessage(`{"room_id":"room-abc-123","given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_test_123","special_request":"Late check-in please"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateBooking_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.create_booking"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing room_id",
			params: `{"given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_123"}`,
		},
		{
			name:   "missing given_name",
			params: `{"room_id":"room-123","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_123"}`,
		},
		{
			name:   "missing family_name",
			params: `{"room_id":"room-123","given_name":"John","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_123"}`,
		},
		{
			name:   "missing email",
			params: `{"room_id":"room-123","given_name":"John","family_name":"Doe","phone":"+1234567890","payment_method_id":"pm_123"}`,
		},
		{
			name:   "missing phone",
			params: `{"room_id":"room-123","given_name":"John","family_name":"Doe","email":"john@example.com","payment_method_id":"pm_123"}`,
		},
		{
			name:   "missing payment_method_id",
			params: `{"room_id":"room-123","given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890"}`,
		},
		{
			name:   "invalid email",
			params: `{"room_id":"room-123","given_name":"John","family_name":"Doe","email":"not-an-email","phone":"+1234567890","payment_method_id":"pm_123"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	// Add special_request length test separately since it needs a long string.
	longRequest := `{"room_id":"room-123","given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_123","special_request":"` + strings.Repeat("x", 5001) + `"}`
	tests = append(tests, struct {
		name   string
		params string
	}{
		name:   "special_request too long",
		params: longRequest,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.create_booking",
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

func TestCreateBooking_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "invalid_input",
			"message": "Price has changed, please re-check",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.create_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.create_booking",
		Parameters:  json.RawMessage(`{"room_id":"room-123","given_name":"John","family_name":"Doe","email":"john@example.com","phone":"+1234567890","payment_method_id":"pm_123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
