package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetSuppressions_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/suppression/unsubscribes" {
			t.Errorf("got %s %s, want GET /suppression/unsubscribes", r.Method, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"email": "unsub@example.com", "created": 1700000000},
			{"email": "unsub2@example.com", "created": 1700001000},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_suppressions"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_suppressions",
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
	if data["count"] != float64(2) {
		t.Errorf("count = %v, want 2", data["count"])
	}

	// Timestamps should be ISO-8601 strings, not raw unix integers.
	sups := data["suppressions"].([]any)
	first := sups[0].(map[string]any)
	if _, ok := first["created_at"].(string); !ok {
		t.Errorf("suppressions[0].created_at should be a string, got %T", first["created_at"])
	}
	if _, exists := first["created"]; exists {
		t.Errorf("suppressions[0] should not expose raw 'created' unix timestamp — use 'created_at'")
	}
}

func TestGetSuppressions_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", q.Get("limit"))
		}
		if q.Get("offset") != "20" {
			t.Errorf("offset = %q, want 20", q.Get("offset"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_suppressions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_suppressions",
		Parameters:  json.RawMessage(`{"limit":10,"offset":20}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetSuppressions_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.get_suppressions"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "negative limit", params: `{"limit":-1}`},
		{name: "negative offset", params: `{"offset":-1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.get_suppressions",
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
