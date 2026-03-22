package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendMessage_PrefersUserAccessToken(t *testing.T) {
	t.Parallel()

	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			auth = r.Header.Get("Authorization")
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
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{Channel: "#general", Message: "hi"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != "Bearer xoxp-user" {
		t.Errorf("expected user token in Authorization, got %q", auth)
	}
}

func TestSendDM_PrefersUserAccessTokenOnBothCalls(t *testing.T) {
	t.Parallel()

	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.open":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]string{"id": "D1"},
			})
		case "/chat.postMessage":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"ts":      "1.0",
				"channel": "D1",
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendDMAction{conn: conn}

	params, _ := json.Marshal(sendDMParams{UserID: "U01234567", Message: "hi"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_dm",
		Parameters:  params,
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auths) != 2 {
		t.Fatalf("expected 2 API calls, got %d", len(auths))
	}
	for i, a := range auths {
		if a != "Bearer xoxp-user" {
			t.Errorf("call %d: expected Bearer xoxp-user, got %q", i, a)
		}
	}
}

func TestUpdateMessage_PrefersUserAccessToken(t *testing.T) {
	t.Parallel()

	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1.0", "channel": "C01234567"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateMessageAction{conn: conn}
	params, _ := json.Marshal(updateMessageParams{Channel: "C01234567", TS: "1.0", Message: "x"})
	creds := connectors.NewCredentials(map[string]string{
		"bot_token":         "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.update_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != "Bearer xoxp-user" {
		t.Errorf("expected user token, got %q", auth)
	}
}

func TestDeleteMessage_PrefersUserAccessToken(t *testing.T) {
	t.Parallel()

	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1.0", "channel": "C01234567"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMessageAction{conn: conn}
	params, _ := json.Marshal(deleteMessageParams{Channel: "C01234567", TS: "1.0"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.delete_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != "Bearer xoxp-user" {
		t.Errorf("expected user token, got %q", auth)
	}
}

func TestScheduleMessage_PrefersUserAccessToken(t *testing.T) {
	t.Parallel()

	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":                   true,
			"scheduled_message_id": "Q1",
			"post_at":              1893369600,
			"channel":              "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &scheduleMessageAction{conn: conn}
	params, _ := json.Marshal(scheduleMessageParams{
		Channel: "#general",
		Message: "later",
		PostAt:  "2029-12-31T00:00:00Z",
	})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.schedule_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != "Bearer xoxp-user" {
		t.Errorf("expected user token, got %q", auth)
	}
}

func TestSendMessage_UserTokenMissingScopeReturnsClearAuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "missing_scope"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}
	params, _ := json.Marshal(sendMessageParams{Channel: "C01234567", Message: "hi"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-stale-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Fatalf("expected AuthError, got %T", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "chat:write") || !strings.Contains(msg, "re-authorize") {
		t.Errorf("expected message to mention chat:write and re-authorize, got: %s", msg)
	}
}

func TestSendMessage_NotAllowedTokenTypeReturnsAuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "not_allowed_token_type"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}
	params, _ := json.Marshal(sendMessageParams{Channel: "C01234567", Message: "hi"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot",
		"user_access_token": "xoxp-user",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: creds,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Fatalf("expected AuthError, got %T", err)
	}
}
