package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/scheduled-events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Name != "Game Night" {
			t.Errorf("expected name 'Game Night', got %q", body.Name)
		}
		if body.PrivacyLevel != 2 {
			t.Errorf("expected privacy level 2, got %d", body.PrivacyLevel)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "555",
			"name": "Game Night",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEventAction{conn: conn}

	params, _ := json.Marshal(createEventParams{
		GuildID:            "111",
		Name:               "Game Night",
		ScheduledStartTime: "2026-04-01T20:00:00Z",
		EntityType:         2,
		ChannelID:          "222",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_event",
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
	if data["id"] != "555" {
		t.Errorf("expected id '555', got %q", data["id"])
	}
}

func TestCreateEvent_MissingName(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createEventAction{conn: conn}

	params, _ := json.Marshal(createEventParams{
		GuildID:            "111",
		ScheduledStartTime: "2026-04-01T20:00:00Z",
		EntityType:         2,
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_event",
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

func TestCreateEvent_InvalidEntityType(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createEventAction{conn: conn}

	params, _ := json.Marshal(createEventParams{
		GuildID:            "111",
		Name:               "Test",
		ScheduledStartTime: "2026-04-01T20:00:00Z",
		EntityType:         5,
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid entity_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
