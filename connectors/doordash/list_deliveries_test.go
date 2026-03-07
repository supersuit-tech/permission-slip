package doordash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDeliveries_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/drive/v2/deliveries" {
			t.Errorf("path = %s, want /drive/v2/deliveries", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":     []map[string]string{{"external_delivery_id": "d-1", "status": "delivered"}},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.list_deliveries"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.list_deliveries",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["has_more"] != false {
		t.Errorf("has_more = %v, want false", data["has_more"])
	}
}

func TestListDeliveries_WithQueryParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "5" {
			t.Errorf("limit = %q, want 5", q.Get("limit"))
		}
		if q.Get("starting_after") != "cursor123" {
			t.Errorf("starting_after = %q, want cursor123", q.Get("starting_after"))
		}
		if q.Get("status") != "delivered" {
			t.Errorf("status = %q, want delivered", q.Get("status"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":     []map[string]string{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["doordash.list_deliveries"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.list_deliveries",
		Parameters:  json.RawMessage(`{"limit": 5, "starting_after": "cursor123", "status": "delivered"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListDeliveries_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.list_deliveries"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.list_deliveries",
		Parameters:  json.RawMessage(`{"limit": 0}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListDeliveries_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["doordash.list_deliveries"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "doordash.list_deliveries",
		Parameters:  json.RawMessage(`{"status": "bogus"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
