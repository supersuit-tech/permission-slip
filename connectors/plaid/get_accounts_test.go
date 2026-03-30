package plaid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetAccounts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/get" {
			t.Errorf("path = %s, want /accounts/get", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"accounts": []map[string]any{
				{
					"account_id": "acc123",
					"name":       "Checking",
					"type":       "depository",
					"subtype":    "checking",
					"mask":       "1234",
				},
			},
			"request_id": "req123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_accounts"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_accounts",
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

func TestGetAccounts_MissingAccessToken(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.get_accounts"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_accounts",
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
