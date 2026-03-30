package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestScheduleCampaign_Success(t *testing.T) {
	t.Parallel()

	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := int(callCount.Add(1))
		switch call {
		case 1:
			// Create single send
			if r.Method != http.MethodPost || r.URL.Path != "/marketing/singlesends" {
				t.Errorf("call 1: %s %s, want POST /marketing/singlesends", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "ss_456"})
		case 2:
			// Schedule
			if r.Method != http.MethodPut || r.URL.Path != "/marketing/singlesends/ss_456/schedule" {
				t.Errorf("call 2: %s %s, want PUT /marketing/singlesends/ss_456/schedule", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "scheduled",
				"send_at": futureTime,
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.schedule_campaign"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.schedule_campaign",
		Parameters: json.RawMessage(`{
			"name": "April Campaign",
			"subject": "Coming Soon",
			"html_content": "<h1>Preview</h1>",
			"list_ids": ["list_xyz"],
			"sender_id": 10,
			"send_at": "` + futureTime + `"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["singlesend_id"] != "ss_456" {
		t.Errorf("singlesend_id = %v, want ss_456", data["singlesend_id"])
	}
	if data["status"] != "scheduled" {
		t.Errorf("status = %v, want scheduled", data["status"])
	}
}

func TestScheduleCampaign_PastDate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.schedule_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.schedule_campaign",
		Parameters: json.RawMessage(`{
			"name": "Test",
			"subject": "Hi",
			"html_content": "<p>test</p>",
			"list_ids": ["a"],
			"sender_id": 1,
			"send_at": "2020-01-01T00:00:00Z"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error for past date, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestScheduleCampaign_InvalidDate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.schedule_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.schedule_campaign",
		Parameters: json.RawMessage(`{
			"name": "Test",
			"subject": "Hi",
			"html_content": "<p>test</p>",
			"list_ids": ["a"],
			"sender_id": 1,
			"send_at": "not-a-date"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error for invalid date, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestScheduleCampaign_MissingSendAt(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.schedule_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.schedule_campaign",
		Parameters: json.RawMessage(`{
			"name": "Test",
			"subject": "Hi",
			"html_content": "<p>test</p>",
			"list_ids": ["a"],
			"sender_id": 1
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error for missing send_at, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
