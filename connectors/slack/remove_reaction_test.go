package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveReaction_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reactions.remove" {
			t.Errorf("path = %s, want /reactions.remove", r.URL.Path)
		}
		var body removeReactionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if body.Name != "thumbsup" {
			t.Errorf("name = %q, want thumbsup", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &removeReactionAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "slack.remove_reaction",
		Parameters: json.RawMessage(
			`{"channel":"C01234567","timestamp":"1234567890.123456","name":"thumbsup"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["reaction"] != "thumbsup" {
		t.Errorf("reaction = %q, want thumbsup", data["reaction"])
	}
}

func TestRemoveReaction_StripsColonsFromEmoji(t *testing.T) {
	t.Parallel()

	var receivedName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body removeReactionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		receivedName = body.Name
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &removeReactionAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "slack.remove_reaction",
		Parameters: json.RawMessage(
			`{"channel":"C01234567","timestamp":"1234567890.123456","name":":heart:"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedName != "heart" {
		t.Errorf("expected colons stripped, got %q", receivedName)
	}
}

func TestRemoveReaction_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &removeReactionAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing channel", `{"timestamp":"1.2","name":"heart"}`},
		{"missing timestamp", `{"channel":"C01234567","name":"heart"}`},
		{"missing name", `{"channel":"C01234567","timestamp":"1.2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.remove_reaction",
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
