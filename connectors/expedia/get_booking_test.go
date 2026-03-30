package expedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetBooking_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/v3/itineraries/itin-789" {
			t.Errorf("path = %s, want /v3/itineraries/itin-789", got)
		}
		if got := r.URL.Query().Get("email"); got != "john@example.com" {
			t.Errorf("email = %q, want %q", got, "john@example.com")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"itinerary_id": "itin-789",
			"status":       "confirmed",
			"rooms": []map[string]any{
				{
					"room_id":  "room-abc-123",
					"checkin":  "2024-06-15",
					"checkout": "2024-06-17",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.get_booking"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.get_booking",
		Parameters:  json.RawMessage(`{"itinerary_id":"itin-789","email":"john@example.com"}`),
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
	if data["status"] != "confirmed" {
		t.Errorf("status = %v, want confirmed", data["status"])
	}
}

func TestGetBooking_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.get_booking"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing itinerary_id",
			params: `{"email":"john@example.com"}`,
		},
		{
			name:   "missing email",
			params: `{"itinerary_id":"itin-789"}`,
		},
		{
			name:   "both missing",
			params: `{}`,
		},
		{
			name:   "invalid email",
			params: `{"itinerary_id":"itin-789","email":"not-an-email"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.get_booking",
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

func TestGetBooking_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "resource_not_found",
			"message": "Itinerary not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.get_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.get_booking",
		Parameters:  json.RawMessage(`{"itinerary_id":"nonexistent","email":"john@example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
