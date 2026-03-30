package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSetTopic_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.setTopic" {
			t.Errorf("expected path /conversations.setTopic, got %s", r.URL.Path)
		}

		var body setTopicRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel C01234567, got %q", body.Channel)
		}
		if body.Topic != "New topic" {
			t.Errorf("expected topic 'New topic', got %q", body.Topic)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id": "C01234567",
				"topic": map[string]string{
					"value": "New topic",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &setTopicAction{conn: conn}

	params, _ := json.Marshal(setTopicParams{
		Channel: "C01234567",
		Topic:   "New topic",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["topic"] != "New topic" {
		t.Errorf("expected topic 'New topic', got %q", data["topic"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}
}

func TestSetTopic_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setTopicAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"topic": "New topic",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
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

func TestSetTopic_InvalidChannelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setTopicAction{conn: conn}

	params, _ := json.Marshal(setTopicParams{
		Channel: "#general",
		Topic:   "New topic",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid channel ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSetTopic_MissingTopic(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setTopicAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
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

func TestSetTopic_TopicTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setTopicAction{conn: conn}

	// Create a topic that exceeds 250 characters.
	longTopic := ""
	for i := 0; i < 251; i++ {
		longTopic += "x"
	}

	params, _ := json.Marshal(setTopicParams{
		Channel: "C01234567",
		Topic:   longTopic,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for topic too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSetTopic_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &setTopicAction{conn: conn}

	params, _ := json.Marshal(setTopicParams{
		Channel: "C01234567",
		Topic:   "New topic",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.set_topic",
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
