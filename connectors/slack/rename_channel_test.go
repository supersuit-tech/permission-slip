package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRenameChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.rename" {
			t.Errorf("path = %s, want /conversations.rename", r.URL.Path)
		}
		var body renameChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("channel = %q, want C01234567", body.Channel)
		}
		if body.Name != "project-updates" {
			t.Errorf("name = %q, want project-updates", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id": "C01234567", "name": "project-updates",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &renameChannelAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.rename_channel",
		Parameters:  json.RawMessage(`{"channel":"C01234567","name":"project-updates"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["new_name"] != "project-updates" {
		t.Errorf("new_name = %q, want project-updates", data["new_name"])
	}
}

func TestRenameChannel_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &renameChannelAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing channel", `{"name":"new-name"}`},
		{"missing name", `{"channel":"C01234567"}`},
		{"empty name", `{"channel":"C01234567","name":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.rename_channel",
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
