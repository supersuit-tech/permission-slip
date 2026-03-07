package pagerduty

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListOnCall_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/oncalls" {
			t.Errorf("path = %s, want /oncalls", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"oncalls": []map[string]any{
				{
					"user": map[string]any{
						"id":   "PUSER1",
						"name": "Jane Doe",
					},
					"schedule": map[string]any{
						"id":   "PSCHED1",
						"name": "Primary",
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.list_on_call"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.list_on_call",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestListOnCall_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("schedule_ids[]"); got != "PSCHED1" {
			t.Errorf("schedule_ids[] = %q, want PSCHED1", got)
		}
		if got := r.URL.Query().Get("since"); got != "2024-01-01T00:00:00Z" {
			t.Errorf("since = %q, want 2024-01-01T00:00:00Z", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"oncalls": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.list_on_call"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.list_on_call",
		Parameters:  json.RawMessage(`{"schedule_ids":["PSCHED1"],"since":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListOnCall_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.list_on_call"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.list_on_call",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
