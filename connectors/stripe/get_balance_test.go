package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/balance" {
			t.Errorf("path = %s, want /v1/balance", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_abc123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk_test_abc123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"available": []map[string]any{
				{"amount": 50000, "currency": "usd"},
			},
			"pending": []map[string]any{
				{"amount": 10000, "currency": "usd"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	available, ok := data["available"].([]any)
	if !ok {
		t.Fatalf("available is not an array: %T", data["available"])
	}
	if len(available) != 1 {
		t.Errorf("len(available) = %d, want 1", len(available))
	}

	pending, ok := data["pending"].([]any)
	if !ok {
		t.Fatalf("pending is not an array: %T", data["pending"])
	}
	if len(pending) != 1 {
		t.Errorf("len(pending) = %d, want 1", len(pending))
	}
}

func TestGetBalance_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "Invalid API Key",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestGetBalance_ExternalError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "api_error",
				"message": "Internal server error",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
