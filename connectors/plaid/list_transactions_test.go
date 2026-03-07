package plaid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/transactions/get" {
			t.Errorf("path = %s, want /transactions/get", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["start_date"] != "2026-01-01" {
			t.Errorf("start_date = %v, want 2026-01-01", body["start_date"])
		}
		if body["end_date"] != "2026-03-01" {
			t.Errorf("end_date = %v, want 2026-03-01", body["end_date"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"transactions": []map[string]any{
				{
					"transaction_id": "txn123",
					"amount":         42.50,
					"name":           "Coffee Shop",
					"category":       []string{"Food and Drink", "Coffee Shop"},
					"date":           "2026-02-15",
				},
			},
			"total_transactions": 1,
			"request_id":         "req789",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.list_transactions"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.list_transactions",
		Parameters:  json.RawMessage(`{"access_token":"access-sandbox-abc123","start_date":"2026-01-01","end_date":"2026-03-01"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	txns, ok := data["transactions"].([]any)
	if !ok || len(txns) != 1 {
		t.Errorf("expected 1 transaction, got %v", data["transactions"])
	}
}

func TestListTransactions_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.list_transactions"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing access_token", params: `{"start_date":"2026-01-01","end_date":"2026-03-01"}`},
		{name: "missing start_date", params: `{"access_token":"tok","end_date":"2026-03-01"}`},
		{name: "missing end_date", params: `{"access_token":"tok","start_date":"2026-01-01"}`},
		{name: "invalid start_date format", params: `{"access_token":"tok","start_date":"01-01-2026","end_date":"2026-03-01"}`},
		{name: "invalid end_date format", params: `{"access_token":"tok","start_date":"2026-01-01","end_date":"03-01-2026"}`},
		{name: "start after end", params: `{"access_token":"tok","start_date":"2026-06-01","end_date":"2026-01-01"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "plaid.list_transactions",
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

func TestListTransactions_WithOptions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		opts, ok := body["options"].(map[string]any)
		if !ok {
			t.Error("expected options in request body")
			return
		}
		if count, ok := opts["count"].(float64); !ok || count != 10 {
			t.Errorf("count = %v, want 10", opts["count"])
		}
		if offset, ok := opts["offset"].(float64); !ok || offset != 5 {
			t.Errorf("offset = %v, want 5", opts["offset"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"transactions": []any{}, "total_transactions": 0, "request_id": "req"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.list_transactions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.list_transactions",
		Parameters:  json.RawMessage(`{"access_token":"tok","start_date":"2026-01-01","end_date":"2026-03-01","count":10,"offset":5}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
