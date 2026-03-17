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
		if body.PostAt != 1893369600 {
			t.Errorf("expected post_at 1893369600, got %d", body.PostAt)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// 1893369600 = 2029-12-31T00:00:00Z
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  "2029-12-31T00:00:00Z",
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

func TestScheduleMessage_LegacyIntegerPostAt(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.PostAt != 1893369600 {
			t.Errorf("expected post_at 1893369600, got %d", body.PostAt)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// Simulate legacy stored approval with integer post_at
	params := []byte(`{"channel":"#general","message":"Hello later!","post_at":1893369600}`)

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleMessage_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"message": "Hello",
		"post_at": "2029-12-31T00:00:00Z",
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
		PostAt:  "2001-09-08T00:00:00Z", // well in the past
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

func TestScheduleMessage_DatetimeLocalFormat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		// 2029-12-31T00:00 interpreted as UTC = 1893369600
		if body.PostAt != 1893369600 {
			t.Errorf("expected post_at 1893369600, got %d", body.PostAt)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// HTML datetime-local format: no seconds, no timezone
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  "2029-12-31T00:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleMessage_DatetimeLocalWithSeconds(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		// 2029-12-31T00:00:05 interpreted as UTC = 1893369605
		if body.PostAt != 1893369605 {
			t.Errorf("expected post_at 1893369605, got %d", body.PostAt)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369605,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// datetime-local with seconds but no timezone
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  "2029-12-31T00:00:05",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleMessage_FractionalSeconds(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.PostAt != 1893369600 {
			t.Errorf("expected post_at 1893369600, got %d", body.PostAt)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// JavaScript Date.toISOString() format with fractional seconds
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  "2029-12-31T00:00:00.000Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleMessage_InvalidPostAtFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello",
		PostAt:  "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid post_at format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestScheduleMessage_NullPostAt(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &scheduleMessageAction{conn: conn}

	// Simulate JSON with null post_at — should be caught by validate() as missing.
	params := []byte(`{"channel":"#general","message":"Hello","post_at":null}`)

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for null post_at")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestScheduleMessage_NumericStringPostAt(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.PostAt != 1893369600 {
			t.Errorf("expected post_at 1893369600, got %d", body.PostAt)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1234ABCD",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}

	// Numeric string (e.g. from flexDateTime converting int → RFC 3339 string,
	// or from a user passing a Unix timestamp as a string).
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "Hello later!",
		PostAt:  "1893369600",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		PostAt:  "2029-12-31T00:00:00Z",
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
