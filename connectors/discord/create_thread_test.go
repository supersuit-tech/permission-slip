package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateThread_WithMessage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/channels/111/messages/222/threads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "333",
			"name": "discussion",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createThreadAction{conn: conn}

	params, _ := json.Marshal(createThreadParams{
		ChannelID: "111",
		Name:      "discussion",
		MessageID: "222",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if data["id"] != "333" {
		t.Errorf("expected id '333', got %q", data["id"])
	}
	if data["name"] != "discussion" {
		t.Errorf("expected name 'discussion', got %q", data["name"])
	}
}

func TestCreateThread_WithoutMessage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/channels/111/threads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body createThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Type != 11 {
			t.Errorf("expected type 11 for standalone thread, got %d", body.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "444",
			"name": "standalone",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createThreadAction{conn: conn}

	params, _ := json.Marshal(createThreadParams{
		ChannelID: "111",
		Name:      "standalone",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateThread_MissingName(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createThreadAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"channel_id": "111"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateThread_InvalidArchiveDuration(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createThreadAction{conn: conn}

	params, _ := json.Marshal(createThreadParams{
		ChannelID:           "111",
		Name:                "test",
		AutoArchiveDuration: 999,
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_thread",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid archive duration")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
