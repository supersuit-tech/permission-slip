package square

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/customers" {
			t.Errorf("path = %s, want /customers", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		var givenName string
		json.Unmarshal(reqBody["given_name"], &givenName)
		if givenName != "Jane" {
			t.Errorf("given_name = %q, want %q", givenName, "Jane")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{
				"id":            "CUST123",
				"given_name":    "Jane",
				"email_address": "jane@example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_customer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_customer",
		Parameters:  json.RawMessage(`{"given_name": "Jane", "email_address": "jane@example.com"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "CUST123" {
		t.Errorf("customer id = %v, want CUST123", data["id"])
	}
}

func TestCreateCustomer_WithAllOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		wantFields := []string{"given_name", "family_name", "email_address", "phone_number", "company_name", "note"}
		for _, field := range wantFields {
			if _, ok := reqBody[field]; !ok {
				t.Errorf("missing field %q in request body", field)
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{"id": "CUST456"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_customer",
		Parameters: json.RawMessage(`{
			"given_name": "Jane",
			"family_name": "Doe",
			"email_address": "jane@example.com",
			"phone_number": "+15551234567",
			"company_name": "Acme Inc",
			"note": "VIP customer"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateCustomer_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.create_customer"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing given_name", params: `{"family_name": "Doe"}`},
		{name: "empty given_name", params: `{"given_name": ""}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.create_customer",
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

func TestCreateCustomer_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "INVALID_EMAIL_ADDRESS", "detail": "Invalid email", "field": "email_address"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_customer",
		Parameters:  json.RawMessage(`{"given_name": "Jane", "email_address": "not-an-email"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_customer"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "square.create_customer",
		Parameters:  json.RawMessage(`{"given_name": "Jane"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
