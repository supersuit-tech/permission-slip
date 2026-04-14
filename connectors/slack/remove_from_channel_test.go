package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveFromChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.kick" {
			t.Errorf("path = %s, want /conversations.kick", r.URL.Path)
		}
		var body removeFromChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("channel = %q, want C01234567", body.Channel)
		}
		if body.User != "U01234567" {
			t.Errorf("user = %q, want U01234567", body.User)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &removeFromChannelAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.remove_from_channel",
		Parameters:  json.RawMessage(`{"channel":"C01234567","user":"U01234567"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveFromChannel_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &removeFromChannelAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing channel", `{"user":"U01234567"}`},
		{"missing user", `{"channel":"C01234567"}`},
		{"username instead of user ID", `{"channel":"C01234567","user":"octocat"}`},
		{"channel name instead of ID", `{"channel":"#general","user":"U01234567"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.remove_from_channel",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}
