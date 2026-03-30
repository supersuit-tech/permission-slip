package square

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListSquareCustomers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/customers" {
			t.Errorf("path = %s, want /customers", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customers": []map[string]any{
				{"id": "C1", "given_name": "Alice"},
				{"id": "C2", "given_name": "Bob"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_customers"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_customers",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["customers"]; !ok {
		t.Error("result missing 'customers' key")
	}
}

func TestListSquareCustomers_WithQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should use POST /customers/search when query is provided.
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/customers/search" {
			t.Errorf("path = %s, want /customers/search", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		query, ok := body["query"].(map[string]any)
		if !ok {
			t.Error("missing 'query' in request body")
		}
		filter, ok := query["filter"].(map[string]any)
		if !ok {
			t.Error("missing 'filter' in query")
		}
		if filter["email_address"] == nil {
			t.Error("missing 'email_address' filter")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"customers": []map[string]any{
				{"id": "C1", "email_address": "alice@example.com"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_customers"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_customers",
		Parameters:  json.RawMessage(`{"query": "alice@example.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListSquareCustomers_EmptyResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.list_customers"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.list_customers",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	// Should normalize null customers to empty array.
	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var customers []any
	if err := json.Unmarshal(data["customers"], &customers); err != nil {
		t.Fatalf("unmarshal customers: %v", err)
	}
	if len(customers) != 0 {
		t.Errorf("expected empty customers, got %d", len(customers))
	}
}

func TestListSquareCustomers_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.list_customers"]

	tests := []struct {
		name   string
		params string
	}{
		{"negative limit", `{"limit": -1}`},
		{"limit too large", `{"limit": 101}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.list_customers",
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
