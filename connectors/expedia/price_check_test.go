package expedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestPriceCheck_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/v3/properties/availability/room-abc-123/price-check" {
			t.Errorf("path = %s, want /v3/properties/availability/room-abc-123/price-check", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "available",
			"rate": map[string]any{
				"total":    "399.98",
				"currency": "USD",
			},
			"cancellation_policy": map[string]string{
				"type": "free_cancellation",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.price_check"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.price_check",
		Parameters:  json.RawMessage(`{"room_id":"room-abc-123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "available" {
		t.Errorf("status = %v, want available", data["status"])
	}
}

func TestPriceCheck_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.price_check"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing room_id",
			params: `{}`,
		},
		{
			name:   "empty room_id",
			params: `{"room_id":""}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.price_check",
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

func TestPriceCheck_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "invalid_input",
			"message": "Room is no longer available",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.price_check"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.price_check",
		Parameters:  json.RawMessage(`{"room_id":"expired-room"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
