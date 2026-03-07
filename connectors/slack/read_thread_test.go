package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReadThread_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/conversations.replies" {
			t.Errorf("expected path /conversations.replies, got %s", r.URL.Path)
		}

		var body readThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel 'C01234567', got %q", body.Channel)
		}
		if body.TS != "1678900001.000200" {
			t.Errorf("expected ts '1678900001.000200', got %q", body.TS)
		}
		if body.Limit != 50 {
			t.Errorf("expected limit 50, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": []map[string]any{
				{
					"type":        "message",
					"user":        "U002",
					"text":        "Check out this thread",
					"ts":          "1678900001.000200",
					"thread_ts":   "1678900001.000200",
					"reply_count": 2,
				},
				{
					"type":      "message",
					"user":      "U003",
					"text":      "Great idea!",
					"ts":        "1678900002.000300",
					"thread_ts": "1678900001.000200",
				},
				{
					"type":      "message",
					"user":      "U001",
					"text":      "Thanks!",
					"ts":        "1678900003.000400",
					"thread_ts": "1678900001.000200",
				},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readThreadAction{conn: conn}

	params, _ := json.Marshal(readThreadParams{
		Channel:  "C01234567",
		ThreadTS: "1678900001.000200",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data messagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Messages) != 3 {
		t.Fatalf("expected 3 messages (parent + 2 replies), got %d", len(data.Messages))
	}
	// First message is the parent.
	if data.Messages[0].Text != "Check out this thread" {
		t.Errorf("expected parent text 'Check out this thread', got %q", data.Messages[0].Text)
	}
	if data.Messages[1].Text != "Great idea!" {
		t.Errorf("expected first reply 'Great idea!', got %q", data.Messages[1].Text)
	}
	if data.Messages[2].User != "U001" {
		t.Errorf("expected last reply user 'U001', got %q", data.Messages[2].User)
	}
	if data.HasMore {
		t.Error("expected has_more to be false")
	}
}

func TestReadThread_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readThreadAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"thread_ts": "1678900001.000200",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
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

func TestReadThread_InvalidChannelName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readThreadAction{conn: conn}

	params, _ := json.Marshal(readThreadParams{
		Channel:  "general",
		ThreadTS: "1678900001.000200",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for channel name instead of ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadThread_MissingThreadTS(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readThreadAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing thread_ts")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadThread_ThreadNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "thread_not_found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readThreadAction{conn: conn}

	params, _ := json.Marshal(readThreadParams{
		Channel:  "C01234567",
		ThreadTS: "0000000000.000000",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for thread_not_found")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestReadThread_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readThreadAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_thread",
		Parameters:  []byte(`{bad`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
