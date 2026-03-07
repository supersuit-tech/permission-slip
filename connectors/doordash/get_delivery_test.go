package doordash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetDelivery_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/drive/v2/deliveries/d-123" {
			t.Errorf("path = %s, want /drive/v2/deliveries/d-123", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"external_delivery_id": "d-123",
			"status":               "enroute_to_pickup",
			"tracking_url":         "https://track.doordash.com/d-123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.get_delivery"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.get_delivery",
		Parameters:  json.RawMessage(`{"delivery_id": "d-123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "enroute_to_pickup" {
		t.Errorf("status = %v, want enroute_to_pickup", data["status"])
	}
}

func TestGetDelivery_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.get_delivery"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing delivery_id", params: `{}`},
		{name: "empty delivery_id", params: `{"delivery_id": ""}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "doordash.get_delivery",
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

func TestGetDelivery_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Delivery not found"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.get_delivery"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.get_delivery",
		Parameters:  json.RawMessage(`{"delivery_id": "nonexistent"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError (mapped from 404), got %T: %v", err, err)
	}
}
