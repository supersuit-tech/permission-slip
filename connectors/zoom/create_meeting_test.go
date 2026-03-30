package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateMeeting_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/me/meetings" {
			t.Errorf("expected path /users/me/meetings, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body zoomCreateMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Topic != "Sprint Planning" {
			t.Errorf("expected topic 'Sprint Planning', got %q", body.Topic)
		}
		if body.Type != 2 {
			t.Errorf("expected type 2, got %d", body.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(zoomCreateMeetingResponse{
			ID:        789012,
			UUID:      "uuid-new",
			Topic:     "Sprint Planning",
			Type:      2,
			StartTime: "2024-01-20T14:00:00Z",
			Duration:  60,
			Timezone:  "America/New_York",
			JoinURL:   "https://zoom.us/j/789012",
			StartURL:  "https://zoom.us/s/789012",
			Password:  "abc123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(createMeetingParams{
		Topic:     "Sprint Planning",
		Type:      2,
		StartTime: "2024-01-20T14:00:00Z",
		Duration:  60,
		Timezone:  "America/New_York",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.create_meeting",
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
	if data["join_url"] != "https://zoom.us/j/789012" {
		t.Errorf("expected join_url 'https://zoom.us/j/789012', got %v", data["join_url"])
	}
	if data["password"] != "abc123" {
		t.Errorf("expected password 'abc123', got %v", data["password"])
	}
}

func TestCreateMeeting_DefaultType(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body zoomCreateMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Type != 2 {
			t.Errorf("expected default type 2, got %d", body.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(zoomCreateMeetingResponse{
			ID:      111,
			Topic:   "Quick Chat",
			Type:    2,
			JoinURL: "https://zoom.us/j/111",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"topic": "Quick Chat"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateMeeting_MissingTopic(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"type": 2})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing topic")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateMeeting_InvalidType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"topic": "Test", "type": 99})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateMeeting_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.create_meeting",
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
