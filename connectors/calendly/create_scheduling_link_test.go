package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSchedulingLink_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/scheduling_links" {
			t.Errorf("expected path /scheduling_links, got %s", r.URL.Path)
		}

		var body calendlySchedulingLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Owner != "https://api.calendly.com/event_types/et1" {
			t.Errorf("expected owner to be event type URI, got %q", body.Owner)
		}
		if body.OwnerType != "EventType" {
			t.Errorf("expected owner_type 'EventType', got %q", body.OwnerType)
		}
		if body.MaxEventCount != 1 {
			t.Errorf("expected max_event_count 1, got %d", body.MaxEventCount)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(calendlySchedulingLinkResponse{
			Resource: struct {
				BookingURL string `json:"booking_url"`
				Owner      string `json:"owner"`
				OwnerType  string `json:"owner_type"`
			}{
				BookingURL: "https://calendly.com/d/abc-123-xyz/30min",
				Owner:      "https://api.calendly.com/event_types/et1",
				OwnerType:  "EventType",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSchedulingLinkAction{conn: conn}

	params, _ := json.Marshal(createSchedulingLinkParams{
		EventTypeURI: "https://api.calendly.com/event_types/et1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.create_scheduling_link",
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
	if data["booking_url"] != "https://calendly.com/d/abc-123-xyz/30min" {
		t.Errorf("unexpected booking_url: %v", data["booking_url"])
	}
}

func TestCreateSchedulingLink_MissingEventTypeURI(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSchedulingLinkAction{conn: conn}

	params, _ := json.Marshal(createSchedulingLinkParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.create_scheduling_link",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing event_type_uri")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSchedulingLink_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSchedulingLinkAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.create_scheduling_link",
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
