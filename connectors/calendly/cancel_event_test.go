package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCancelEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/scheduled_events/ev-uuid-123/cancellation" {
			t.Errorf("expected path /scheduled_events/ev-uuid-123/cancellation, got %s", r.URL.Path)
		}

		var body calendlyCancelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Reason != "Meeting no longer needed" {
			t.Errorf("expected reason 'Meeting no longer needed', got %q", body.Reason)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(calendlyCancelResponse{
			Resource: struct {
				CanceledBy string `json:"canceled_by"`
				Reason     string `json:"reason"`
			}{
				CanceledBy: "https://api.calendly.com/users/abc123",
				Reason:     "Meeting no longer needed",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &cancelEventAction{conn: conn}

	params, _ := json.Marshal(cancelEventParams{
		EventUUID: "ev-uuid-123",
		Reason:    "Meeting no longer needed",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.cancel_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["reason"] != "Meeting no longer needed" {
		t.Errorf("unexpected reason: %v", data["reason"])
	}
}

func TestCancelEvent_MissingEventUUID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &cancelEventAction{conn: conn}

	params, _ := json.Marshal(cancelEventParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.cancel_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing event_uuid")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCancelEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &cancelEventAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.cancel_event",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
