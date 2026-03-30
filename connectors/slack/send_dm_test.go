package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendDM_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/conversations.open":
			callCount++
			var body conversationsOpenRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode conversations.open body: %v", err)
			}
			if body.Users != "U01234567" {
				t.Errorf("expected users 'U01234567', got %q", body.Users)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]string{"id": "D09876543"},
			})

		case "/chat.postMessage":
			callCount++
			var body sendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode chat.postMessage body: %v", err)
			}
			if body.Channel != "D09876543" {
				t.Errorf("expected channel 'D09876543', got %q", body.Channel)
			}
			if body.Text != "Hello via DM!" {
				t.Errorf("expected text 'Hello via DM!', got %q", body.Text)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"ts":      "1234567890.123456",
				"channel": "D09876543",
			})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(sendDMParams{
		UserID:  "U01234567",
		Message: "Hello via DM!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["ts"] != "1234567890.123456" {
		t.Errorf("expected ts '1234567890.123456', got %q", data["ts"])
	}
	if data["channel"] != "D09876543" {
		t.Errorf("expected channel 'D09876543', got %q", data["channel"])
	}
}

func TestSendDM_SelfDM(t *testing.T) {
	t.Parallel()

	// Slack supports DM-ing yourself by passing your own user ID to
	// conversations.open. The code path is identical — no special-casing needed.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/conversations.open":
			callCount++
			var body conversationsOpenRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode conversations.open body: %v", err)
			}
			// The user ID is the caller's own — Slack returns a self-DM channel.
			if body.Users != "U_MYSELF" {
				t.Errorf("expected users 'U_MYSELF', got %q", body.Users)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]string{"id": "D_SELF"},
			})

		case "/chat.postMessage":
			callCount++
			var body sendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode chat.postMessage body: %v", err)
			}
			if body.Channel != "D_SELF" {
				t.Errorf("expected channel 'D_SELF', got %q", body.Channel)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"ts":      "1700000000.000001",
				"channel": "D_SELF",
			})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(sendDMParams{
		UserID:  "U_MYSELF",
		Message: "Note to self",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["channel"] != "D_SELF" {
		t.Errorf("expected channel 'D_SELF', got %q", data["channel"])
	}
}

func TestSendDM_MissingUserID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"message": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendDM_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"user_id": "U01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendDM_ConversationsOpenError(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "user_not_found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(sendDMParams{
		UserID:  "U_INVALID",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for user_not_found")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (conversations.open only), got %d", callCount)
	}
}

func TestSendDM_InvalidUserIDFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(sendDMParams{
		UserID:  "john.doe",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid user_id format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendDM_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendDMAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
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
