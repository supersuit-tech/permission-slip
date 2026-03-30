package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendChatMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/spaces/AAAA1234/messages" {
			t.Errorf("expected path /v1/spaces/AAAA1234/messages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body chatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Text != "Hello, team!" {
			t.Errorf("expected text 'Hello, team!', got %q", body.Text)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatMessageResponse{
			Name:       "spaces/AAAA1234/messages/msg-001",
			CreateTime: "2024-01-15T09:00:00Z",
			Space:      struct{ Name string `json:"name"` }{Name: "spaces/AAAA1234"},
			Thread:     struct{ Name string `json:"name"` }{Name: "spaces/AAAA1234/threads/thread-001"},
		})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(sendChatMessageParams{
		SpaceName: "spaces/AAAA1234",
		Text:      "Hello, team!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
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
	if data["name"] != "spaces/AAAA1234/messages/msg-001" {
		t.Errorf("expected name 'spaces/AAAA1234/messages/msg-001', got %q", data["name"])
	}
	if data["space"] != "spaces/AAAA1234" {
		t.Errorf("expected space 'spaces/AAAA1234', got %q", data["space"])
	}
}

func TestSendChatMessage_MissingSpaceName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"text": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing space_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChatMessage_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"space_name": "spaces/AAAA1234"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing text")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChatMessage_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    401,
				"message": "Invalid Credentials",
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(sendChatMessageParams{
		SpaceName: "spaces/AAAA1234",
		Text:      "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestSendChatMessage_InvalidSpaceNamePrefix(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(sendChatMessageParams{
		SpaceName: "invalid/AAAA1234",
		Text:      "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid space_name prefix")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChatMessage_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	cases := []struct {
		name      string
		spaceName string
	}{
		{"dot-dot traversal", "spaces/.."},
		{"slash in space ID", "spaces/foo/bar"},
		{"query string injection", "spaces/foo?admin=true"},
		{"fragment injection", "spaces/foo#admin"},
		{"empty space ID", "spaces/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, _ := json.Marshal(sendChatMessageParams{
				SpaceName: tc.spaceName,
				Text:      "Hello",
			})

			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "google.send_chat_message",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatalf("expected error for space_name %q", tc.spaceName)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T (%v)", err, err)
			}
		})
	}
}

func TestSendChatMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_chat_message",
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
