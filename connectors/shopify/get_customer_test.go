package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/customers/5001.json" {
			t.Errorf("path = %s, want /customers/5001.json", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{
				"id":    5001,
				"email": "alice@example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_customer"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_customer",
		Parameters:  json.RawMessage(`{"customer_id": 5001}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["customer"]; !ok {
		t.Error("result missing 'customer' key")
	}
}

func TestGetCustomer_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_customer"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing customer_id", `{}`},
		{"zero customer_id", `{"customer_id": 0}`},
		{"negative customer_id", `{"customer_id": -1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.get_customer",
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

func TestGetCustomer_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_customer"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_customer",
		Parameters:  json.RawMessage(`{"customer_id": 999999}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}
