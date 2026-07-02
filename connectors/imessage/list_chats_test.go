package imessage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestFilterUnreadChats(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, Name: "Alice", UnreadCount: 3},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Carol", UnreadCount: 1},
		{ID: 4, Name: "Dave", UnreadCount: 0},
	}
	got := filterUnreadChats(chats, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("got = %#v", got)
	}
}

func TestListChatsAction_UnreadCountsPassthrough(t *testing.T) {
	mock := newMockIMsgWithChatsList(t, `{"chats":[{"id":1,"name":"Alice","unread_count":5},{"id":2,"name":"Bob","unread_count":0}]}`)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"limit":10}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Chats []chat `json:"chats"`
	}
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Chats) != 2 {
		t.Fatalf("chats len = %d", len(payload.Chats))
	}
	if payload.Chats[0].UnreadCount != 5 {
		t.Fatalf("first unread_count = %d", payload.Chats[0].UnreadCount)
	}
}

func TestListChatsAction_UnreadOnlyFilters(t *testing.T) {
	mock := newMockIMsgWithChatsList(t, `{"chats":[{"id":1,"name":"Alice","unread_count":2},{"id":2,"name":"Bob","unread_count":0},{"id":3,"name":"Carol","unread_count":1}]}`)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"limit":10,"unread_only":true}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Chats []chat `json:"chats"`
	}
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Chats) != 2 {
		t.Fatalf("chats = %#v", payload.Chats)
	}
	if payload.Chats[0].ID != 1 || payload.Chats[1].ID != 3 {
		t.Fatalf("chats = %#v", payload.Chats)
	}
}

func TestListChatsAction_UnreadOnlyTreatsMissingFieldAsZero(t *testing.T) {
	// Older imsg builds omit unread_count entirely.
	mock := newMockIMsgWithChatsList(t, `{"chats":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob","unread_count":1}]}`)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"unread_only":true}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Chats []chat `json:"chats"`
	}
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Chats) != 1 || payload.Chats[0].ID != 2 {
		t.Fatalf("chats = %#v", payload.Chats)
	}
}

func newMockIMsgWithChatsList(t *testing.T, chatsListResult string) *mockIMsg {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "imsg")
	script := `#!/bin/bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  rpc)
    while IFS= read -r line; do
      id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
      method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
      case "$method" in
        chats.list)
          result='` + chatsListResult + `'
          ;;
        *)
          result='{}'
          ;;
      esac
      printf '{"jsonrpc":"2.0","id":%s,"result":%s}\n' "$id" "$result"
    done
    ;;
  *)
    echo "unknown command" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return &mockIMsg{path: scriptPath, client: newIMsgClient()}
}
