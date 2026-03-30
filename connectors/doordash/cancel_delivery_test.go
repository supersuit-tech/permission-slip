package doordash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCancelDelivery_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/drive/v2/deliveries/d-123/cancel" {
			t.Errorf("path = %s, want /drive/v2/deliveries/d-123/cancel", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"external_delivery_id": "d-123",
			"status":               "cancelled",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.cancel_delivery"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.cancel_delivery",
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
	if data["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", data["status"])
	}
}

func TestCancelDelivery_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.cancel_delivery"]

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
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "doordash.cancel_delivery",
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
