package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat.delete" {
			t.Errorf("expected path /chat.delete, got %s", r.URL.Path)
		}

		var body deleteMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel 'C01234567', got %q", body.Channel)
		}
		if body.TS != "1234567890.123456" {
			t.Errorf("expected ts '1234567890.123456', got %q", body.TS)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMessageAction{conn: conn}

	params, _ := json.Marshal(deleteMessageParams{
		Channel: "C01234567",
		TS:      "1234567890.123456",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
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
	if data["ts"] != "1234567890.123456" {
		t.Errorf("expected ts '1234567890.123456', got %q", data["ts"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}
}

func TestDeleteMessage_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ts": "1234567890.123456",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
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

func TestDeleteMessage_MissingTS(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing ts")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteMessage_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "cant_delete_message",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMessageAction{conn: conn}

	params, _ := json.Marshal(deleteMessageParams{
		Channel: "C01234567",
		TS:      "1234567890.123456",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for cant_delete_message")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestDeleteMessage_InvalidTSFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
		"ts":      "abc123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid ts format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
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
