package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testSlackResolveServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *SlackConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
}

func TestResolveResourceDetails_Channel(t *testing.T) {
	t.Parallel()

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.info" {
			t.Errorf("expected /conversations.info, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"name":       "general",
				"is_private": false,
			},
		})
	}))
	defer srv.Close()

	channelActions := []string{
		"slack.send_message", "slack.read_channel_messages", "slack.read_thread",
		"slack.schedule_message", "slack.set_topic", "slack.invite_to_channel",
		"slack.upload_file", "slack.add_reaction", "slack.update_message",
		"slack.delete_message",
		"slack.remove_from_channel", "slack.remove_reaction", "slack.pin_message",
		"slack.unpin_message", "slack.archive_channel", "slack.rename_channel",
	}

	params, _ := json.Marshal(map[string]string{"channel": "C0AMRGKRTA4"})

	for _, actionType := range channelActions {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["channel_name"] != "#general" {
			t.Errorf("%s: expected channel_name '#general', got %v", actionType, details["channel_name"])
		}
	}
}

func TestResolveResourceDetails_PrivateChannel(t *testing.T) {
	t.Parallel()

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"name":       "secret-plans",
				"is_private": true,
			},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"channel": "G0AMRGKRTA4"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_message", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Private channels should not get # prefix.
	if details["channel_name"] != "secret-plans" {
		t.Errorf("expected channel_name 'secret-plans', got %v", details["channel_name"])
	}
}

func TestResolveResourceDetails_User(t *testing.T) {
	t.Parallel()

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.info" {
			t.Errorf("expected /users.info, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"user": map[string]any{
				"name":      "jsmith",
				"real_name": "John Smith",
				"profile": map[string]any{
					"display_name": "Johnny",
					"real_name":    "John Smith",
				},
			},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"user_id": "U12345678"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_dm", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should prefer display_name.
	if details["user_name"] != "Johnny" {
		t.Errorf("expected user_name 'Johnny', got %v", details["user_name"])
	}
}

func TestResolveResourceDetails_UserFallbackToRealName(t *testing.T) {
	t.Parallel()

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"user": map[string]any{
				"name":      "jsmith",
				"real_name": "John Smith",
				"profile": map[string]any{
					"display_name": "",
					"real_name":    "John Smith",
				},
			},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"user_id": "U12345678"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_dm", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["user_name"] != "John Smith" {
		t.Errorf("expected user_name 'John Smith', got %v", details["user_name"])
	}
}

func TestResolveResourceDetails_SearchMessages_DefaultChannelName(t *testing.T) {
	t.Parallel()

	conn := New()
	params, _ := json.Marshal(map[string]string{"query": "hello"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.search_messages", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["channel_name"] != "Slack" {
		t.Errorf("expected channel_name 'Slack', got %v", details["channel_name"])
	}
}

func TestResolveResourceDetails_SearchMessages_WithChannelID(t *testing.T) {
	t.Parallel()

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.info" {
			t.Errorf("expected /conversations.info, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"name":       "engineering",
				"is_private": false,
			},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"query":   "deploy",
		"channel": "C0AMRGKRTA4",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.search_messages", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["channel_name"] != "#engineering" {
		t.Errorf("expected channel_name '#engineering', got %v", details["channel_name"])
	}
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	t.Parallel()

	conn := New()
	params, _ := json.Marshal(map[string]string{"foo": "bar"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.unknown_action", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details for unknown action, got %v", details)
	}
}
