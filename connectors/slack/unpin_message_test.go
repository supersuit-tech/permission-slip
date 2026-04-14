package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUnpinMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pins.remove" {
			t.Errorf("path = %s, want /pins.remove", r.URL.Path)
		}
		var body unpinMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("channel = %q, want C01234567", body.Channel)
		}
		if body.Timestamp != "1234567890.123456" {
			t.Errorf("timestamp = %q, want 1234567890.123456", body.Timestamp)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &unpinMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.unpin_message",
		Parameters:  json.RawMessage(`{"channel":"C01234567","ts":"1234567890.123456"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnpinMessage_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &unpinMessageAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing ts", `{"channel":"C01234567"}`},
		{"missing channel", `{"ts":"1.2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.unpin_message",
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
