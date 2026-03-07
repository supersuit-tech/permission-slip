package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListScheduledEvents_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(usersmeResponse{
				Resource: struct {
					URI  string `json:"uri"`
					Name string `json:"name"`
				}{URI: "https://api.calendly.com/users/abc123", Name: "Test User"},
			})
			return
		}

		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/scheduled_events" {
			t.Errorf("expected path /scheduled_events, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("status") != "active" {
			t.Errorf("expected status=active, got %s", r.URL.Query().Get("status"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendlyScheduledEventsResponse{
			Collection: []calendlyScheduledEvent{
				{
					URI:       "https://api.calendly.com/scheduled_events/ev1",
					Name:      "30 Minute Meeting",
					Status:    "active",
					StartTime: "2024-01-15T09:00:00.000000Z",
					EndTime:   "2024-01-15T09:30:00.000000Z",
					EventType: "https://api.calendly.com/event_types/et1",
					EventGuests: []calendlyScheduledGuest{
						{Email: "guest@example.com"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listScheduledEventsAction{conn: conn}

	params, _ := json.Marshal(listScheduledEventsParams{Status: "active"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_scheduled_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalEvents int                  `json:"total_events"`
		Events      []scheduledEventItem `json:"events"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(data.Events))
	}
	if data.Events[0].Name != "30 Minute Meeting" {
		t.Errorf("expected name '30 Minute Meeting', got %q", data.Events[0].Name)
	}
	if data.Events[0].Status != "active" {
		t.Errorf("expected status 'active', got %q", data.Events[0].Status)
	}
	if data.Events[0].GuestCount != 1 {
		t.Errorf("expected guest_count 1, got %d", data.Events[0].GuestCount)
	}
	if len(data.Events[0].Guests) != 1 || data.Events[0].Guests[0] != "guest@example.com" {
		t.Errorf("expected guests [guest@example.com], got %v", data.Events[0].Guests)
	}
}

func TestListScheduledEvents_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listScheduledEventsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"status": "invalid"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_scheduled_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListScheduledEvents_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listScheduledEventsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_scheduled_events",
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
