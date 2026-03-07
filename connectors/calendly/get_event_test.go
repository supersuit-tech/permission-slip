package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/scheduled_events/ev-uuid-123" {
			t.Errorf("expected path /scheduled_events/ev-uuid-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{
			"resource": {
				"uri": "https://api.calendly.com/scheduled_events/ev-uuid-123",
				"name": "30 Minute Meeting",
				"status": "active",
				"start_time": "2024-01-15T09:00:00.000000Z",
				"end_time": "2024-01-15T09:30:00.000000Z",
				"event_type": "https://api.calendly.com/event_types/et1",
				"location": {
					"type": "zoom",
					"location": "https://zoom.us/j/123",
					"join_url": "https://zoom.us/j/123"
				},
				"created_at": "2024-01-10T10:00:00.000000Z",
				"updated_at": "2024-01-10T10:00:00.000000Z",
				"event_guests": [
					{"email": "guest@example.com", "created_at": "2024-01-10T10:00:00.000000Z", "updated_at": "2024-01-10T10:00:00.000000Z"}
				]
			}
		}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getEventAction{conn: conn}

	params, _ := json.Marshal(getEventParams{EventUUID: "ev-uuid-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.get_event",
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
	if data["name"] != "30 Minute Meeting" {
		t.Errorf("unexpected name: %v", data["name"])
	}
	if data["status"] != "active" {
		t.Errorf("unexpected status: %v", data["status"])
	}
	if data["join_url"] != "https://zoom.us/j/123" {
		t.Errorf("unexpected join_url: %v", data["join_url"])
	}
	if data["location_type"] != "zoom" {
		t.Errorf("unexpected location_type: %v", data["location_type"])
	}

	guests, ok := data["guests"].([]any)
	if !ok {
		t.Fatalf("expected guests to be array, got %T", data["guests"])
	}
	if len(guests) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(guests))
	}
}

func TestGetEvent_MissingEventUUID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getEventAction{conn: conn}

	params, _ := json.Marshal(getEventParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.get_event",
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

func TestGetEvent_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getEventAction{conn: conn}

	params, _ := json.Marshal(getEventParams{EventUUID: "../../../users/me"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.get_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal UUID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(calendlyAPIError{Title: "Resource Not Found", Message: "The event does not exist."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getEventAction{conn: conn}

	params, _ := json.Marshal(getEventParams{EventUUID: "nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.get_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got: %T", err)
	}
}
