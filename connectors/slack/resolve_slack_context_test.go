package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	slackctx "github.com/supersuit-tech/permission-slip/connectors/slack/context"
)

func testAuthAndChannelServer(t *testing.T, historyMsgs []map[string]any) (*httptest.Server, *SlackConnector) {
	t.Helper()
	now := time.Now().UTC()
	ts1 := fmt.Sprintf("%d.%06d", now.Add(-2*time.Hour).Unix(), 0)
	ts2 := fmt.Sprintf("%d.%06d", now.Add(-1*time.Hour).Unix(), 0)
	if len(historyMsgs) == 0 {
		historyMsgs = []map[string]any{
			{"user": "U1", "text": "hello <@U2>", "ts": ts1},
			{"user": "U1", "text": "world", "ts": ts2},
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"url":     "https://acme.slack.com/",
				"user_id": "U9",
			})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":          "C1",
					"name":        "general",
					"is_private":  false,
					"is_im":       false,
					"is_mpim":     false,
					"num_members": 3,
					"topic":       map[string]any{"value": "t"},
					"purpose":     map[string]any{"value": "p"},
				},
			})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"messages": historyMsgs,
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"user": map[string]any{
					"id":        "U1",
					"name":      "alice",
					"real_name": "Alice",
					"profile": map[string]any{
						"title":          "",
						"image_original": "",
						"image_512":      "",
					},
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, newForTest(srv.Client(), srv.URL)
}

func TestResolveSendMessage_RecentChannel(t *testing.T) {
	t.Parallel()
	srv, conn := testAuthAndChannelServer(t, nil)
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"channel": "C1",
		"message": "hi",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc, ok := details["slack_context"].(map[string]any)
	if !ok {
		t.Fatalf("missing slack_context: %#v", details)
	}
	if sc["context_scope"] != string(slackctx.ScopeRecentChannel) {
		t.Fatalf("scope %v", sc["context_scope"])
	}
	if details["channel_name"] != "#general" {
		t.Fatalf("legacy channel_name: %v", details["channel_name"])
	}
}

func TestResolveSendMessage_Thread(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	parentTS := "100.000000"
	replyTS := fmt.Sprintf("%d.%06d", now.Add(-30*time.Minute).Unix(), 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U9"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": "C1", "name": "general", "is_private": false,
					"topic": map[string]any{"value": ""}, "purpose": map[string]any{"value": ""},
				},
			})
		case "/conversations.replies":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "parent", "ts": parentTS},
					{"user": "U1", "text": "reply", "ts": replyTS},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U1", "name": "a", "profile": map[string]any{}},
			})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)

	params, _ := json.Marshal(map[string]string{
		"channel":   "C1",
		"message":   "hi",
		"thread_ts": parentTS,
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeThread) {
		t.Fatalf("got %v", sc["context_scope"])
	}
}

func TestResolveSendMessage_RateLimitDegrades(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth.test" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://x.slack.com/", "user_id": "U1"})
			return
		}
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"channel": "C1", "message": "m"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeMetadataOnly) {
		t.Fatalf("got %#v", sc)
	}
}

func TestResolveScheduleMessage_PostAtInResourceDetails(t *testing.T) {
	t.Parallel()
	srv, conn := testAuthAndChannelServer(t, nil)
	defer srv.Close()
	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	params, _ := json.Marshal(map[string]string{
		"channel": "C1",
		"message": "later",
		"post_at": future,
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.schedule_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := details["post_at"]; !ok {
		t.Fatalf("expected post_at in details: %#v", details)
	}
}

func TestResolveUpdateMessage_ThreadedTarget(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	rootTS := fmt.Sprintf("%d.%06d", now.Add(-2*time.Hour).Unix(), 0)
	targetTS := fmt.Sprintf("%d.%06d", now.Add(-1*time.Hour).Unix(), 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U9"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": "C1", "name": "general", "is_private": false,
					"topic": map[string]any{"value": ""}, "purpose": map[string]any{"value": ""},
				},
			})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "root", "ts": rootTS},
					{"user": "U1", "text": "in thread", "ts": targetTS, "thread_ts": rootTS},
				},
			})
		case "/conversations.replies":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "root", "ts": rootTS},
					{"user": "U1", "text": "in thread", "ts": targetTS},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U1", "name": "a", "profile": map[string]any{}},
			})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{
		"channel": "C1",
		"ts":      targetTS,
		"message": "edit",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.update_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeThread) {
		t.Fatalf("got %v", sc["context_scope"])
	}
}

func TestResolveSendDM_FirstContact(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U9"})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"user": map[string]any{
					"id": "UNEW", "name": "newuser", "profile": map[string]any{},
				},
			})
		case "/conversations.open":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "channel": map[string]any{"id": "DNEW"}})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "messages": []any{}})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": "DNEW", "name": "dm", "is_im": true,
					"topic": map[string]any{"value": ""}, "purpose": map[string]any{"value": ""},
				},
			})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"user_id": "UNEW", "message": "hi"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_dm", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeFirstContactDM) {
		t.Fatalf("got %v", sc["context_scope"])
	}
}

func TestResolveDeleteMessage_MissingTarget(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	tsOther := fmt.Sprintf("%d.%06d", now.Add(-1*time.Hour).Unix(), 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U9"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": "C1", "name": "general", "is_private": false,
					"topic": map[string]any{"value": ""}, "purpose": map[string]any{"value": ""},
				},
			})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "other", "ts": tsOther},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U1", "name": "a", "profile": map[string]any{}},
			})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{
		"channel": "C1",
		"ts":      "99.999999",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.delete_message", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeMetadataOnly) {
		t.Fatalf("got %v", sc["context_scope"])
	}
}

func TestResolveSendDM_SelfDM(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "USELF"})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"user": map[string]any{
					"id": "USELF", "name": "me", "profile": map[string]any{},
				},
			})
		case "/conversations.open":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "channel": map[string]any{"id": "D1"}})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": "D1", "name": "self", "is_im": true,
					"topic": map[string]any{"value": ""}, "purpose": map[string]any{"value": ""},
				},
			})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	conn := newForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"user_id": "USELF", "message": "note"})
	details, err := conn.ResolveResourceDetails(context.Background(), "slack.send_dm", params, validCreds())
	if err != nil {
		t.Fatal(err)
	}
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != string(slackctx.ScopeSelfDM) {
		t.Fatalf("got %v", sc["context_scope"])
	}
}
