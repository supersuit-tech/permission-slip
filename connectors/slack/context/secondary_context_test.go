package context

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestBuildReactionContext_SurroundingMessages(t *testing.T) {
	t.Parallel()

	api := newMockAPI()
	api.postHandlers["conversations.info"] = func(body json.RawMessage) (any, error) {
		return map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id":          "C1",
				"name":        "general",
				"num_members": 3,
			},
		}, nil
	}
	tsTarget := "100.000001"
	tsOlder := "99.000001"
	tsNewer := "101.000001"
	api.postHandlers["conversations.history"] = func(body json.RawMessage) (any, error) {
		var req readChannelHistoryRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		// Single-message fetch for reaction target (latest + inclusive).
		if req.Latest != "" && req.Inclusive {
			return map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U1", "text": "target", "ts": tsTarget},
				},
			}, nil
		}
		// 24h window fetch (oldest set) — surrounding messages for ±3 slice.
		if req.Oldest != "" {
			return map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U0", "text": "old", "ts": tsOlder},
					{"user": "U1", "text": "target", "ts": tsTarget},
					{"user": "U2", "text": "new", "ts": tsNewer},
				},
			}, nil
		}
		return map[string]any{"ok": true, "messages": []map[string]any{}}, nil
	}
	api.getHandlers["users.info"] = func(params map[string]string) (any, error) {
		uid := params["user"]
		return map[string]any{
			"ok": true,
			"user": map[string]any{
				"id":        uid,
				"name":      "u" + uid,
				"real_name": "User " + uid,
			},
		}, nil
	}

	sc, err := BuildReactionContext(context.Background(), api, "C1", tsTarget, testSlackCredentials(), nil, &MentionCache{})
	if err != nil {
		t.Fatalf("BuildReactionContext: %v", err)
	}
	if sc.TargetMessage == nil || sc.TargetMessage.Text != "target" {
		t.Fatalf("target_message: %+v", sc.TargetMessage)
	}
	if len(sc.RecentMessages) != 2 {
		t.Fatalf("expected 2 surrounding messages, got %d", len(sc.RecentMessages))
	}
	if sc.Channel == nil || sc.Channel.Name != "general" {
		t.Fatalf("channel: %+v", sc.Channel)
	}
}

func TestBuildArchiveContext_RateLimitDegrades(t *testing.T) {
	t.Parallel()

	api := newMockAPI()
	api.postHandlers["conversations.info"] = func(body json.RawMessage) (any, error) {
		return nil, &connectors.RateLimitError{Message: "slow down"}
	}

	sc, err := BuildArchiveContext(context.Background(), api, "C1", testSlackCredentials(), nil, &MentionCache{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.ContextScope != ScopeMetadataOnly {
		t.Fatalf("expected metadata_only, got %s", sc.ContextScope)
	}
}

func TestBuildInviteContext_Recipient(t *testing.T) {
	t.Parallel()

	api := newMockAPI()
	api.postHandlers["conversations.info"] = func(body json.RawMessage) (any, error) {
		return map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id":          "C1",
				"name":        "eng",
				"num_members": 10,
			},
		}, nil
	}
	api.getHandlers["users.info"] = func(params map[string]string) (any, error) {
		return map[string]any{
			"ok": true,
			"user": map[string]any{
				"id":        "U9",
				"name":      "alice",
				"real_name": "Alice",
				"profile":   map[string]any{},
			},
		}, nil
	}

	sc, err := BuildInviteContext(context.Background(), api, "C1", "U9, U8", testSlackCredentials(), nil)
	if err != nil {
		t.Fatalf("BuildInviteContext: %v", err)
	}
	if sc.Recipient == nil || sc.Recipient.Name != "alice" {
		t.Fatalf("recipient: %+v", sc.Recipient)
	}
}
