package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

	// Actions that only resolve the channel name via conversations.info (no slack_context).
	channelActions := []string{
		"slack.read_channel_messages", "slack.read_thread",
		"slack.set_topic",
		"slack.upload_file",
		"slack.rename_channel",
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

	now := time.Now().UTC()
	ts := fmt.Sprintf("%d.%06d", now.Add(-1*time.Hour).Unix(), 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://x.slack.com/", "user_id": "U1"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":          "G0AMRGKRTA4",
					"name":        "secret-plans",
					"is_private":  true,
					"topic":       map[string]any{"value": ""},
					"purpose":     map[string]any{"value": ""},
					"num_members": 2,
				},
			})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "x", "ts": ts},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U1", "name": "a", "profile": map[string]any{}},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)

	params, _ := json.Marshal(map[string]string{"channel": "G0AMRGKRTA4", "message": "hi"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_message", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Private channels should not get # prefix on legacy channel_name.
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

func TestResolveResourceDetails_AddReaction_IncludesSlackContext(t *testing.T) {
	t.Parallel()

	tsHi := fmt.Sprintf("%d.%06d", time.Now().UTC().Unix()-60, 0)
	tsBefore := fmt.Sprintf("%d.%06d", time.Now().UTC().Unix()-120, 0)

	srv, conn := testSlackResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U_SELF"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":          "C01234567",
					"name":        "general",
					"num_members": 5,
				},
			})
		case "/conversations.history":
			var body struct {
				Latest    string `json:"latest"`
				Inclusive bool   `json:"inclusive"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Latest != "" && body.Inclusive {
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"messages": []map[string]any{
						{"user": "U1", "text": "hello", "ts": tsHi},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U0", "text": "before", "ts": tsBefore},
					{"user": "U1", "text": "hello", "ts": tsHi},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"user": map[string]any{
					"id":   "U1",
					"name": "alice",
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"channel":   "C01234567",
		"timestamp": tsHi,
		"name":      "thumbsup",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.add_reaction", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["channel_name"] != "#general" {
		t.Errorf("channel_name = %v, want #general", details["channel_name"])
	}
	sc, ok := details["slack_context"].(map[string]any)
	if !ok {
		t.Fatalf("expected slack_context map, got %T", details["slack_context"])
	}
	if sc["context_scope"] != "recent_channel" {
		t.Errorf("context_scope = %v", sc["context_scope"])
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
