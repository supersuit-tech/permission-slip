package plaid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetIdentity_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/identity/get" {
			t.Errorf("path = %s, want /identity/get", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"accounts": []map[string]any{
				{
					"account_id": "acc123",
					"owners": []map[string]any{
						{
							"names":          []string{"John Doe"},
							"phone_numbers":  []map[string]any{{"data": "+15551234567", "type": "mobile"}},
							"emails":         []map[string]any{{"data": "john@example.com", "type": "primary"}},
							"addresses":      []map[string]any{{"data": map[string]any{"street": "123 Main St"}}},
						},
					},
				},
			},
			"request_id": "req123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_identity"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_identity",
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

func TestGetIdentity_MissingAccessToken(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.get_identity"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_identity",
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
