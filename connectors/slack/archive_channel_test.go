package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestArchiveChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.archive" {
			t.Errorf("path = %s, want /conversations.archive", r.URL.Path)
		}
		var body archiveChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("channel = %q, want C01234567", body.Channel)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &archiveChannelAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.archive_channel",
		Parameters:  json.RawMessage(`{"channel":"C01234567"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["channel"] != "C01234567" {
		t.Errorf("channel = %q, want C01234567", data["channel"])
	}
}

func TestArchiveChannel_InvalidChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveChannelAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing channel", `{}`},
		{"name instead of ID", `{"channel":"#general"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.archive_channel",
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

func TestArchiveChannel_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "already_archived"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &archiveChannelAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.archive_channel",
		Parameters:  json.RawMessage(`{"channel":"C01234567"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}
