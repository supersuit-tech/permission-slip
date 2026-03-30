package expedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCancelBooking_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if got := r.URL.Path; got != "/v3/itineraries/itin-789/rooms/room-abc-123" {
			t.Errorf("path = %s, want /v3/itineraries/itin-789/rooms/room-abc-123", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.cancel_booking"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.cancel_booking",
		Parameters:  json.RawMessage(`{"itinerary_id":"itin-789","room_id":"room-abc-123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", data["status"])
	}
	if data["itinerary_id"] != "itin-789" {
		t.Errorf("itinerary_id = %v, want itin-789", data["itinerary_id"])
	}
}

func TestCancelBooking_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.cancel_booking"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing itinerary_id",
			params: `{"room_id":"room-123"}`,
		},
		{
			name:   "missing room_id",
			params: `{"itinerary_id":"itin-789"}`,
		},
		{
			name:   "both missing",
			params: `{}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.cancel_booking",
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

func TestCancelBooking_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "request_unauthenticated",
			"message": "Invalid credentials",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.cancel_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.cancel_booking",
		Parameters:  json.RawMessage(`{"itinerary_id":"itin-789","room_id":"room-123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
