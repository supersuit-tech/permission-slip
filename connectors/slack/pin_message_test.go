package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPinMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pins.add" {
			t.Errorf("path = %s, want /pins.add", r.URL.Path)
		}
		var body pinMessageRequest
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
	action := &pinMessageAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.pin_message",
		Parameters:  json.RawMessage(`{"channel":"C01234567","ts":"1234567890.123456"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["ts"] != "1234567890.123456" {
		t.Errorf("ts = %q, want 1234567890.123456", data["ts"])
	}
}

func TestPinMessage_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &pinMessageAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing channel", `{"ts":"1.2"}`},
		{"missing ts", `{"channel":"C01234567"}`},
		{"malformed ts", `{"channel":"C01234567","ts":"not-a-ts"}`},
		{"name instead of ID", `{"channel":"#general","ts":"1.2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.pin_message",
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
