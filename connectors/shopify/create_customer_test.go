package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/customers.json" {
			t.Errorf("path = %s, want /customers.json", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		customer, ok := body["customer"].(map[string]any)
		if !ok {
			t.Error("missing 'customer' in request body")
		}
		if customer["email"] != "alice@example.com" {
			t.Errorf("email = %v, want alice@example.com", customer["email"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{
				"id":    5001,
				"email": "alice@example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_customer"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_customer",
		Parameters:  json.RawMessage(`{"email": "alice@example.com", "first_name": "Alice"}`),
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

func TestCreateCustomer_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_customer"]

	tests := []struct {
		name   string
		params string
	}{
		{"no fields", `{}`},
		{"empty email only", `{"email": ""}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.create_customer",
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

func TestCreateCustomer_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"email":["is invalid"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_customer"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_customer",
		Parameters:  json.RawMessage(`{"email": "notvalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_InvalidEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_customer"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_customer",
		Parameters:  json.RawMessage(`{"email": "not-an-email"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
