package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListSenders_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/verified_senders" {
			t.Errorf("path = %s, want /verified_senders", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":         123,
					"nickname":   "Marketing",
					"from_email": "marketing@example.com",
					"from_name":  "Example Co",
					"reply_to":   "reply@example.com",
					"verified":   true,
				},
				{
					"id":         456,
					"nickname":   "Support",
					"from_email": "support@example.com",
					"from_name":  "Example Support",
					"reply_to":   "support@example.com",
					"verified":   true,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_senders"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_senders",
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
	senders, ok := data["senders"].([]any)
	if !ok {
		t.Fatal("senders not present in result")
	}
	if len(senders) != 2 {
		t.Errorf("senders length = %d, want 2", len(senders))
	}
	first := senders[0].(map[string]any)
	if first["from_email"] != "marketing@example.com" {
		t.Errorf("first sender email = %v, want marketing@example.com", first["from_email"])
	}
}

func TestListSenders_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_senders"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_senders",
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
	if data["count"] != float64(0) {
		t.Errorf("count = %v, want 0", data["count"])
	}
}
