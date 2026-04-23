package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListUnread_NoEmail_Succeeds(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
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
		UserEmail:   "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out listUnreadResult
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
	}
	if len(out.UnreadChannels) != 0 {
		t.Fatalf("expected no unreads, got %+v", out.UnreadChannels)
	}
}

func TestListUnread_NoUnreads(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
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
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
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
				// Slack does not surface unread_count_display for public channels; still listed in users.conversations.
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"channel": map[string]any{
						"id":                   "Cpub",
						"name":                 "announce",
						"is_private":           false,
						"unread_count_display": 0,
						"last_read":            "100.0",
						"latest": map[string]any{
							"text": "would not be treated as unread",
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
							"text": strings.Repeat("a", 250),
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
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
	}
	if len(out.UnreadChannels) != 2 {
		t.Fatalf("expected 2 unreads (DM + mpim; public channel omitted by Slack), got %d", len(out.UnreadChannels))
	}
	// Find Ddm and check truncation on preview text
	var dm *unreadChannelEntry
	for i := range out.UnreadChannels {
		if out.UnreadChannels[i].ChannelID == "Ddm" {
			dm = &out.UnreadChannels[i]
			break
		}
	}
	if dm == nil {
		t.Fatal("missing Ddm entry")
	}
	if dm.ChannelType != "im" {
		t.Errorf("Ddm type: got %q", dm.ChannelType)
	}
	var mpim *unreadChannelEntry
	for i := range out.UnreadChannels {
		if out.UnreadChannels[i].ChannelID == "Ggrp" {
			mpim = &out.UnreadChannels[i]
			break
		}
	}
	if mpim == nil {
		t.Fatal("missing Ggrp entry")
	}
	if mpim.ChannelType != "mpim" {
		t.Errorf("Ggrp type: got %q", mpim.ChannelType)
	}
	if dm.LatestMessagePreview == nil {
		t.Fatal("missing preview")
	}
	if got := len([]rune(dm.LatestMessagePreview.Text)); got != 201 { // 200 runes + ellipsis
		t.Errorf("expected truncated text length 201 runes, got %d", got)
	}
}

func TestListUnread_EmptyChannelList(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
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
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
	}
	if len(out.UnreadChannels) != 0 {
		t.Fatalf("expected empty, got %d", len(out.UnreadChannels))
	}
}

func TestListUnread_MissingLatestOmitsPreview(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
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
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
	}
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
		case "/users.conversations":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"channels": []map[string]any{{"id": "C1"}},
			})
		case "/conversations.info":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
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
	if out.Notes != listUnreadResponseNotes {
		t.Fatalf("notes mismatch")
	}
	if len(out.UnreadChannels) != 1 || out.UnreadChannels[0].ChannelID != "C2" {
		t.Fatalf("expected single unread on C2, got %+v", out.UnreadChannels)
	}
}
