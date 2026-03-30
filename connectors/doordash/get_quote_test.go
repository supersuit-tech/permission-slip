package doordash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetQuote_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/drive/v2/quotes" {
			t.Errorf("path = %s, want /drive/v2/quotes", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if reqBody["pickup_address"] != "901 Market St, San Francisco, CA 94103" {
			t.Errorf("pickup_address = %v, want 901 Market St", reqBody["pickup_address"])
		}
		if reqBody["dropoff_address"] != "123 Main St, San Francisco, CA 94105" {
			t.Errorf("dropoff_address = %v, want 123 Main St", reqBody["dropoff_address"])
		}
		if _, ok := reqBody["external_delivery_id"]; !ok {
			t.Error("missing external_delivery_id in request body")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"external_delivery_id": "d-123",
			"fee":                  595,
			"currency":             "USD",
			"delivery_time":        "2024-03-15T14:30:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.get_quote"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "doordash.get_quote",
		Parameters: json.RawMessage(`{
			"pickup_address": "901 Market St, San Francisco, CA 94103",
			"dropoff_address": "123 Main St, San Francisco, CA 94105",
			"pickup_phone": "+15551234567",
			"dropoff_phone": "+15559876543",
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
	if data["fee"] != float64(595) {
		t.Errorf("fee = %v, want 595", data["fee"])
	}
	if data["currency"] != "USD" {
		t.Errorf("currency = %v, want USD", data["currency"])
	}
}

func TestGetQuote_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.get_quote"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing pickup_address", params: `{"dropoff_address":"x","pickup_phone":"+1","dropoff_phone":"+1"}`},
		{name: "missing dropoff_address", params: `{"pickup_address":"x","pickup_phone":"+1","dropoff_phone":"+1"}`},
		{name: "missing pickup_phone", params: `{"pickup_address":"x","dropoff_address":"x","dropoff_phone":"+1"}`},
		{name: "missing dropoff_phone", params: `{"pickup_address":"x","dropoff_address":"x","pickup_phone":"+1"}`},
		{name: "negative order_value", params: `{"pickup_address":"x","dropoff_address":"x","pickup_phone":"+1","dropoff_phone":"+1","order_value":-1}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "doordash.get_quote",
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

func TestGetQuote_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Internal server error"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.get_quote"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "doordash.get_quote",
		Parameters: json.RawMessage(`{
			"pickup_address": "x",
			"dropoff_address": "x",
			"pickup_phone": "+1",
			"dropoff_phone": "+1"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestGetQuote_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.get_quote"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType: "doordash.get_quote",
		Parameters: json.RawMessage(`{
			"pickup_address": "x",
			"dropoff_address": "x",
			"pickup_phone": "+1",
			"dropoff_phone": "+1"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
