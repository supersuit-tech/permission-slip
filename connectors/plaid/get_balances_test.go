package plaid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetBalances_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/balance/get" {
			t.Errorf("path = %s, want /accounts/balance/get", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["access_token"] != "access-sandbox-abc123" {
			t.Errorf("access_token = %v, want access-sandbox-abc123", body["access_token"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"accounts": []map[string]any{
				{
					"account_id": "acc123",
					"balances": map[string]any{
						"available": 100.50,
						"current":   110.00,
					},
					"name": "Checking",
					"type": "depository",
				},
			},
			"request_id": "req456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_balances"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_balances",
		Parameters:  json.RawMessage(`{"access_token":"access-sandbox-abc123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	accounts, ok := data["accounts"].([]any)
	if !ok || len(accounts) != 1 {
		t.Errorf("expected 1 account, got %v", data["accounts"])
	}
}

func TestGetBalances_WithAccountIDs(t *testing.T) {
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
		ids, ok := opts["account_ids"].([]any)
		if !ok || len(ids) != 1 {
			t.Errorf("expected 1 account_id in options, got %v", opts["account_ids"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"accounts": []any{}, "request_id": "req789"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_balances"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_balances",
		Parameters:  json.RawMessage(`{"access_token":"access-sandbox-abc123","account_ids":["acc1"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetBalances_MissingAccessToken(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.get_balances"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_balances",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
