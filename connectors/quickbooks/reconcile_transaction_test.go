package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestReconcileTransaction_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/deposit" {
			t.Errorf("path = %s, want /v3/company/1234567890/deposit", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Deposit": map[string]any{
				"Id":       "7001",
				"TotalAmt": 1000.0,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.reconcile_transaction"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.reconcile_transaction",
		Parameters:  json.RawMessage(`{"account_id":"35","amount":1000,"txn_date":"2025-06-15","description":"Wire transfer"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["Id"] != "7001" {
		t.Errorf("Id = %v, want 7001", data["Id"])
	}
}

func TestReconcileTransaction_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.reconcile_transaction"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.reconcile_transaction",
		Parameters:  json.RawMessage(`{"amount":1000}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestReconcileTransaction_ZeroAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.reconcile_transaction"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.reconcile_transaction",
		Parameters:  json.RawMessage(`{"account_id":"35","amount":0}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
