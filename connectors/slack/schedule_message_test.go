package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestScheduleMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.scheduleMessage" {
			t.Errorf("expected path /chat.scheduleMessage, got %s", r.URL.Path)
		}

		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "#general" {
			t.Errorf("expected channel #general, got %q", body.Channel)
		}
		if body.Text != "Hello later!" {
			t.Errorf("expected text 'Hello later!', got %q", body.Text)
		}
		if body.PostAt != 1893456000 {
			t.Errorf("expected post_at 1893456000, got %d", body.PostAt)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893456000,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  1893456000,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
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
	if data["scheduled_message_id"] != "Q1234ABCD" {
		t.Errorf("expected scheduled_message_id 'Q1234ABCD', got %v", data["scheduled_message_id"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %v", data["channel"])
	}
}

func TestScheduleMessage_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"message": "Hello",
		"post_at": 1893456000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestScheduleMessage_MissingPostAt(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"channel": "#general",
		"message": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing post_at")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestScheduleMessage_PostAtInPast(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello",
		PostAt:  1000000000, // 2001-09-08, well in the past
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for post_at in the past")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestScheduleMessage_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "time_in_past",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello",
		PostAt:  1893456000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for Slack API error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
