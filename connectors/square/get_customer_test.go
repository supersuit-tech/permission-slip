package square

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetSquareCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/customers/CUST123" {
			t.Errorf("path = %s, want /customers/CUST123", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{
				"id":            "CUST123",
				"given_name":    "Alice",
				"email_address": "alice@example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.get_customer"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.get_customer",
		Parameters:  json.RawMessage(`{"customer_id": "CUST123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["customer"]; !ok {
		t.Error("result missing 'customer' key")
	}
}

func TestGetSquareCustomer_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.get_customer"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing customer_id", `{}`},
		{"empty customer_id", `{"customer_id": ""}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.get_customer",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestGetSquareCustomer_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.get_customer"]

	malicious := []string{
		"../other-customer",
		"CUST123/../../admin",
		"CUST%2F..%2Fsecret",
	}
	for _, id := range malicious {
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "square.get_customer",
			Parameters:  []byte(`{"customer_id": "` + id + `"}`),
			Credentials: validCreds(),
		})
		if err == nil {
			t.Errorf("customer_id %q: expected error, got nil", id)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("customer_id %q: expected ValidationError, got %T: %v", id, err, err)
		}
	}
}

func TestGetSquareCustomer_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "NOT_FOUND", "detail": "Customer not found."},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.get_customer"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.get_customer",
		Parameters:  json.RawMessage(`{"customer_id": "NONEXISTENT"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for 404, got %T: %v", err, err)
	}
}
