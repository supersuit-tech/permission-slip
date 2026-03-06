package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListCalendarEvents_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":      "event-1",
					"subject": "Team Standup",
					"start":   map[string]string{"dateTime": "2024-01-15T09:00:00", "timeZone": "UTC"},
					"end":     map[string]string{"dateTime": "2024-01-15T09:30:00", "timeZone": "UTC"},
					"location": map[string]string{
						"displayName": "Zoom",
					},
					"organizer": map[string]any{
						"emailAddress": map[string]string{
							"name":    "Manager",
							"address": "manager@example.com",
						},
					},
					"webLink": "https://outlook.office365.com/calendar/item/event-1",
					"isAllDay": false,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(listCalendarEventsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_calendar_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []eventSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 event, got %d", len(summaries))
	}
	if summaries[0].ID != "event-1" {
		t.Errorf("expected id 'event-1', got %q", summaries[0].ID)
	}
	if summaries[0].Subject != "Team Standup" {
		t.Errorf("expected subject 'Team Standup', got %q", summaries[0].Subject)
	}
	if summaries[0].Location != "Zoom" {
		t.Errorf("expected location 'Zoom', got %q", summaries[0].Location)
	}
	if summaries[0].Organizer != "manager@example.com" {
		t.Errorf("expected organizer 'manager@example.com', got %q", summaries[0].Organizer)
	}
}

func TestListCalendarEvents_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listCalendarEventsParams
	params.defaults()
	if params.Top != 10 {
		t.Errorf("expected default top 10, got %d", params.Top)
	}
}

func TestListCalendarEvents_TopClamped(t *testing.T) {
	t.Parallel()

	params := listCalendarEventsParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListCalendarEvents_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCalendarEventsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_calendar_events",
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
