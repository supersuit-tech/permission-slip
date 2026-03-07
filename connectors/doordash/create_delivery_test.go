package doordash

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDelivery_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/drive/v2/deliveries" {
			t.Errorf("path = %s, want /drive/v2/deliveries", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["external_delivery_id"]; !ok {
			t.Error("missing external_delivery_id in request body")
		}
		if reqBody["pickup_address"] != "901 Market St, San Francisco, CA 94103" {
			t.Errorf("pickup_address = %v", reqBody["pickup_address"])
		}
		if reqBody["dropoff_contact_given_name"] != "Jane" {
			t.Errorf("dropoff_contact_given_name = %v, want Jane", reqBody["dropoff_contact_given_name"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"external_delivery_id": reqBody["external_delivery_id"],
			"tracking_url":         "https://track.doordash.com/abc123",
			"fee":                  595,
			"status":               "created",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.create_delivery"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "doordash.create_delivery",
		Parameters: json.RawMessage(`{
			"pickup_address": "901 Market St, San Francisco, CA 94103",
			"pickup_phone": "+15551234567",
			"dropoff_address": "123 Main St, San Francisco, CA 94105",
			"dropoff_phone": "+15559876543",
			"dropoff_contact_given_name": "Jane",
			"order_value": 2500
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "created" {
		t.Errorf("status = %v, want created", data["status"])
	}
	if data["tracking_url"] != "https://track.doordash.com/abc123" {
		t.Errorf("tracking_url = %v", data["tracking_url"])
	}
}

func TestCreateDelivery_WithItems(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var items []map[string]interface{}
		json.Unmarshal(reqBody["items"], &items)
		if len(items) != 1 {
			t.Errorf("items count = %d, want 1", len(items))
		}
		if items[0]["name"] != "Documents" {
			t.Errorf("items[0].name = %v, want Documents", items[0]["name"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"external_delivery_id": "d-456",
			"status":               "created",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.create_delivery"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "doordash.create_delivery",
		Parameters: json.RawMessage(`{
			"pickup_address": "901 Market St",
			"pickup_phone": "+15551234567",
			"dropoff_address": "123 Main St",
			"dropoff_phone": "+15559876543",
			"dropoff_contact_given_name": "Jane",
			"items": [{"name": "Documents", "quantity": 1, "description": "Legal papers"}]
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateDelivery_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.create_delivery"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing pickup_address", params: `{"pickup_phone":"+1","dropoff_address":"x","dropoff_phone":"+1","dropoff_contact_given_name":"J"}`},
		{name: "missing pickup_phone", params: `{"pickup_address":"x","dropoff_address":"x","dropoff_phone":"+1","dropoff_contact_given_name":"J"}`},
		{name: "missing dropoff_address", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_phone":"+1","dropoff_contact_given_name":"J"}`},
		{name: "missing dropoff_phone", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_address":"x","dropoff_contact_given_name":"J"}`},
		{name: "missing dropoff_contact_given_name", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_address":"x","dropoff_phone":"+1"}`},
		{name: "negative order_value", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_address":"x","dropoff_phone":"+1","dropoff_contact_given_name":"J","order_value":-1}`},
		{name: "item missing name", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_address":"x","dropoff_phone":"+1","dropoff_contact_given_name":"J","items":[{"quantity":1}]}`},
		{name: "item zero quantity", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_address":"x","dropoff_phone":"+1","dropoff_contact_given_name":"J","items":[{"name":"X","quantity":0}]}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "doordash.create_delivery",
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
