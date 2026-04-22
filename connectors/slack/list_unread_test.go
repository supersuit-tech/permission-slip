package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListUnread_NoEmail(t *testing.T) {
	t.Parallel()

	conn := newForTest(http.DefaultClient, "http://unused.example")
	action := &listUnreadAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "",
	})
	if err == nil {
		t.Fatal("expected error for missing user email")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListUnread_NoUnreads(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U111"},
			})
		case "/users.conversations":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "C1", "name": "general"},
				},
			})
		case "/conversations.info":
			q := r.URL.Query().Get("channel")
			if q != "C1" {
				t.Errorf("expected channel C1, got %q", q)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":                   "C1",
					"name":                 "general",
					"is_private":           false,
					"unread_count_display": 0,
					"last_read":            "1.0",
					"latest":               map[string]any{"text": "hi", "user": "U2", "ts": "2.0"},
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.UnreadChannels) != 0 {
		t.Fatalf("expected no unreads, got %+v", out.UnreadChannels)
	}
}

func TestListUnread_MixedTypes_WithPreview(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U111"},
			})
		case "/users.conversations":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "Cpub", "name": "announce"},
					{"id": "Ddm", "is_im": true, "user": "U222"},
					{"id": "Ggrp", "is_mpim": true, "name": "mpim-name"},
				},
			})
		case "/conversations.info":
			ch := r.URL.Query().Get("channel")
			switch ch {
			case "Cpub":
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"channel": map[string]any{
						"id":                   "Cpub",
						"name":                 "announce",
						"is_private":           false,
						"unread_count_display": 2,
						"last_read":            "100.0",
						"latest": map[string]any{
							"text": strings.Repeat("a", 250),
							"user": "U9",
							"ts":   "200.0",
						},
					},
				})
			case "Ddm":
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"channel": map[string]any{
						"id":                   "Ddm",
						"is_im":                true,
						"user":                 "U222",
						"unread_count_display": 1,
						"last_read":            "1.0",
						"latest": map[string]any{
							"text": "dm hello",
							"user": "U222",
							"ts":   "2.0",
						},
					},
				})
			case "Ggrp":
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"channel": map[string]any{
						"id":                   "Ggrp",
						"is_mpim":              true,
						"name":                 "mpim-name",
						"unread_count_display": 3,
						"last_read":            "3.0",
						"latest": map[string]any{
							"text": "group ping",
							"user": "U3",
							"ts":   "4.0",
						},
					},
				})
			default:
				t.Fatalf("unexpected channel %q", ch)
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.UnreadChannels) != 3 {
		t.Fatalf("expected 3 unreads, got %d", len(out.UnreadChannels))
	}
	// Find Cpub and check truncation
	var pub *unreadChannelEntry
	for i := range out.UnreadChannels {
		if out.UnreadChannels[i].ChannelID == "Cpub" {
			pub = &out.UnreadChannels[i]
			break
		}
	}
	if pub == nil {
		t.Fatal("missing Cpub entry")
	}
	if pub.ChannelType != "public_channel" {
		t.Errorf("Cpub type: got %q", pub.ChannelType)
	}
	if pub.LatestMessagePreview == nil {
		t.Fatal("missing preview")
	}
	if got := len([]rune(pub.LatestMessagePreview.Text)); got != 201 { // 200 runes + ellipsis
		t.Errorf("expected truncated text length 201 runes, got %d", got)
	}
}

func TestListUnread_EmptyChannelList(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": map[string]any{"id": "U1"}})
		case "/users.conversations":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "channels": []any{}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "a@b.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	_ = json.Unmarshal(res.Data, &out)
	if len(out.UnreadChannels) != 0 {
		t.Fatalf("expected empty, got %d", len(out.UnreadChannels))
	}
}

func TestListUnread_MissingLatestOmitsPreview(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": map[string]any{"id": "U1"}})
		case "/users.conversations":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"channels": []map[string]any{{"id": "C1", "name": "x"}},
			})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":                   "C1",
					"name":                 "x",
					"unread_count_display": 1,
					"last_read":            "1.0",
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "a@b.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	_ = json.Unmarshal(res.Data, &out)
	if len(out.UnreadChannels) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(out.UnreadChannels))
	}
	if out.UnreadChannels[0].LatestMessagePreview != nil {
		t.Fatal("expected nil latest_message_preview")
	}
}

func TestListUnread_RateLimitOnInfo(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.lookupByEmail":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": map[string]any{"id": "U1"}})
		case "/users.conversations":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"channels": []map[string]any{{"id": "C1"}},
			})
		case "/conversations.info":
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Retry-After", "2")
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "ratelimited"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "a@b.com",
	})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestListUnread_PaginatesUsersConversations(t *testing.T) {
	t.Parallel()

	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": map[string]any{"id": "U1"}})
		case "/users.conversations":
			var body usersConversationsRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			// User must be omitted on user-token calls (#1031).
			if body.User != "" {
				t.Errorf("expected empty user param on users.conversations, got %q", body.User)
			}
			page++
			if page == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"ok":                true,
					"channels":          []map[string]any{{"id": "C1"}},
					"response_metadata": map[string]any{"next_cursor": "c2"},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"channels": []map[string]any{{"id": "C2"}},
			})
		case "/conversations.info":
			ch := r.URL.Query().Get("channel")
			unread := 0
			if ch == "C2" {
				unread = 1
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id": ch, "name": ch,
					"unread_count_display": unread,
					"last_read":            "1.0",
					"latest":               map[string]any{"text": "x", "user": "U1", "ts": "2.0"},
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUnreadAction{conn: conn}

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_unread",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
		UserEmail:   "a@b.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	_ = json.Unmarshal(res.Data, &out)
	if len(out.UnreadChannels) != 1 || out.UnreadChannels[0].ChannelID != "C2" {
		t.Fatalf("expected single unread on C2, got %+v", out.UnreadChannels)
	}
}
