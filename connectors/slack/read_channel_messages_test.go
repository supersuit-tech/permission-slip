package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReadChannelMessages_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/conversations.info":
			// Return public channel info for the access check.
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C01234567", "is_private": false},
			})
			return
		case "/conversations.history":
			// Continue with existing logic.
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body readChannelMessagesRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel 'C01234567', got %q", body.Channel)
		}
		if body.Limit != 20 {
			t.Errorf("expected limit 20, got %d", body.Limit)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": []map[string]any{
				{
					"type":        "message",
					"user":        "U001",
					"text":        "Hello everyone!",
					"ts":          "1678900000.000100",
					"thread_ts":   "",
					"reply_count": 0,
				},
				{
					"type":        "message",
					"user":        "U002",
					"text":        "Check out this thread",
					"ts":          "1678900001.000200",
					"thread_ts":   "1678900001.000200",
					"reply_count": 5,
				},
			},
			"has_more": true,
			"response_metadata": map[string]string{
				"next_cursor": "bmV4dA==",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "C01234567",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
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
	if len(data.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(data.Messages))
	}
	if data.Messages[0].User != "U001" {
		t.Errorf("expected user 'U001', got %q", data.Messages[0].User)
	}
	if data.Messages[0].Text != "Hello everyone!" {
		t.Errorf("expected text 'Hello everyone!', got %q", data.Messages[0].Text)
	}
	if data.Messages[1].ReplyCount != 5 {
		t.Errorf("expected reply_count 5, got %d", data.Messages[1].ReplyCount)
	}
	if !data.HasMore {
		t.Error("expected has_more to be true")
	}
	if data.NextCursor != "bmV4dA==" {
		t.Errorf("expected next_cursor 'bmV4dA==', got %q", data.NextCursor)
	}
}

func TestReadChannelMessages_WithTimeRange(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/conversations.info" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C01234567", "is_private": false},
			})
			return
		}

		var body readChannelMessagesRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Oldest != "1678800000.000000" {
			t.Errorf("expected oldest '1678800000.000000', got %q", body.Oldest)
		}
		if body.Latest != "1678900000.000000" {
			t.Errorf("expected latest '1678900000.000000', got %q", body.Latest)
		}
		if body.Limit != 50 {
			t.Errorf("expected limit 50, got %d", body.Limit)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"messages": []map[string]any{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "C01234567",
		Limit:   50,
		Oldest:  "1678800000.000000",
		Latest:  "1678900000.000000",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
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
	if data.HasMore {
		t.Error("expected has_more to be false")
	}
}

func TestReadChannelMessages_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(map[string]int{"limit": 10})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
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

func TestReadChannelMessages_ChannelNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Both conversations.info and conversations.history return channel_not_found.
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "C99999999",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for channel_not_found")
	}
	// The error now comes from verifyChannelAccess (ValidationError wrapping the Slack error)
	// rather than directly from conversations.history.
}

func TestReadChannelMessages_InvalidChannelName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "#general",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
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

func TestReadChannelMessages_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "C01234567",
		Limit:   5000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for limit out of range")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadChannelMessages_MissingScope(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if r.URL.Path == "/conversations.info" {
			// Return public channel so access check passes.
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C01234567", "is_private": false},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "missing_scope",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "C01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing_scope")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for missing_scope, got: %T", err)
	}
}

func TestReadChannelMessages_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readChannelMessagesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  []byte(`not json`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
