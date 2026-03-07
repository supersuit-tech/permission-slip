package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddReaction_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reactions.add" {
			t.Errorf("expected path /reactions.add, got %s", r.URL.Path)
		}

		var body addReactionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel C01234567, got %q", body.Channel)
		}
		if body.Timestamp != "1234567890.123456" {
			t.Errorf("expected timestamp '1234567890.123456', got %q", body.Timestamp)
		}
		if body.Name != "thumbsup" {
			t.Errorf("expected name 'thumbsup', got %q", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(addReactionParams{
		Channel:   "C01234567",
		Timestamp: "1234567890.123456",
		Name:      "thumbsup",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
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
	if data["reaction"] != "thumbsup" {
		t.Errorf("expected reaction 'thumbsup', got %q", data["reaction"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}
	if data["timestamp"] != "1234567890.123456" {
		t.Errorf("expected timestamp '1234567890.123456', got %q", data["timestamp"])
	}
}

func TestAddReaction_StripsColonsFromEmoji(t *testing.T) {
	t.Parallel()

	var receivedName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body addReactionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		receivedName = body.Name

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addReactionAction{conn: conn}

	// Users often pass :thumbsup: with colons — we should strip them.
	params, _ := json.Marshal(addReactionParams{
		Channel:   "C01234567",
		Timestamp: "1234567890.123456",
		Name:      ":thumbsup:",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedName != "thumbsup" {
		t.Errorf("expected colons to be stripped, got %q", receivedName)
	}
}

func TestAddReaction_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"timestamp": "1234567890.123456",
		"name":      "thumbsup",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
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

func TestAddReaction_InvalidChannelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(addReactionParams{
		Channel:   "#general",
		Timestamp: "1234567890.123456",
		Name:      "thumbsup",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
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

func TestAddReaction_MissingTimestamp(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
		"name":    "thumbsup",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing timestamp")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddReaction_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel":   "C01234567",
		"timestamp": "1234567890.123456",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddReaction_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "already_reacted",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addReactionAction{conn: conn}

	params, _ := json.Marshal(addReactionParams{
		Channel:   "C01234567",
		Timestamp: "1234567890.123456",
		Name:      "thumbsup",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.add_reaction",
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
