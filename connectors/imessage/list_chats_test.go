package imessage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestFilterChatsByActivity_Since(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, LastMessageAt: "2026-07-01T12:00:00Z"},
		{ID: 2, LastMessageAt: "2026-06-01T12:00:00Z"},
		{ID: 3, LastMessageAt: "2026-07-05T12:00:00Z"},
		{ID: 4}, // missing timestamp excluded
	}
	got := filterChatsByActivity(chats, "2026-07-01T00:00:00Z", "", 10)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("got = %#v", got)
	}
}

func TestFilterChatsByActivity_Before(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, LastMessageAt: "2026-07-01T12:00:00Z"},
		{ID: 2, LastMessageAt: "2026-06-01T12:00:00Z"},
		{ID: 3, LastMessageAt: "2026-07-05T12:00:00Z"},
	}
	got := filterChatsByActivity(chats, "", "2026-07-01T12:00:00Z", 10)
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("got = %#v", got)
	}
}

func TestFilterChatsByActivity_RespectsLimit(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, LastMessageAt: "2026-07-03T12:00:00Z"},
		{ID: 2, LastMessageAt: "2026-07-02T12:00:00Z"},
		{ID: 3, LastMessageAt: "2026-07-01T12:00:00Z"},
	}
	got := filterChatsByActivity(chats, "2026-07-01T00:00:00Z", "", 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestListChatsParams_ValidateRejectsInvalidSince(t *testing.T) {
	t.Parallel()
	p := listChatsParams{Since: "not-a-date"}
	if err := p.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestListChatsParams_ValidateRejectsBeforeBeforeSince(t *testing.T) {
	t.Parallel()
	p := listChatsParams{
		Since:  "2026-07-07T00:00:00Z",
		Before: "2026-07-01T00:00:00Z",
	}
	if err := p.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestListChatsParams_ValidateDefaultsOrderAndSort(t *testing.T) {
	t.Parallel()
	p := listChatsParams{}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if p.OrderBy != "last_activity" {
		t.Fatalf("order_by = %q, want last_activity", p.OrderBy)
	}
	if p.Sort != "desc" {
		t.Fatalf("sort = %q, want desc", p.Sort)
	}
}

func TestListChatsParams_ValidateRejectsInvalidOrderBy(t *testing.T) {
	t.Parallel()
	p := listChatsParams{OrderBy: "created_at"}
	if err := p.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestListChatsParams_ValidateRejectsInvalidSort(t *testing.T) {
	t.Parallel()
	p := listChatsParams{Sort: "newest"}
	if err := p.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSortChats_LastActivityDesc(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, LastMessageAt: "2026-06-01T12:00:00Z"},
		{ID: 2, LastMessageAt: "2026-07-05T12:00:00Z"},
		{ID: 3, LastMessageAt: "2026-07-01T12:00:00Z"},
		{ID: 4},
	}
	sortChats(chats, "last_activity", "desc")
	if chats[0].ID != 2 || chats[1].ID != 3 || chats[2].ID != 1 || chats[3].ID != 4 {
		t.Fatalf("got = %#v", chats)
	}
}

func TestSortChats_LastActivityAsc(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, LastMessageAt: "2026-07-05T12:00:00Z"},
		{ID: 2, LastMessageAt: "2026-06-01T12:00:00Z"},
		{ID: 3},
	}
	sortChats(chats, "last_activity", "asc")
	if chats[0].ID != 3 || chats[1].ID != 2 || chats[2].ID != 1 {
		t.Fatalf("got = %#v", chats)
	}
}

func TestSortChats_ContactNameAsc(t *testing.T) {
	t.Parallel()
	chats := []chat{
		{ID: 1, ContactName: "Zoe"},
		{ID: 2, ContactName: "Alice"},
		{ID: 3, DisplayName: "Bob"},
	}
	sortChats(chats, "contact_name", "asc")
	if chats[0].ID != 2 || chats[1].ID != 3 || chats[2].ID != 1 {
		t.Fatalf("got = %#v", chats)
	}
}

func TestListChatsAction_SortsByLastActivityDesc(t *testing.T) {
	mock := newMockIMsgWithChatsList(t, `{"chats":[{"id":1,"name":"Old","last_message_at":"2026-06-01T10:00:00Z"},{"id":2,"name":"Newest","last_message_at":"2026-07-07T10:00:00Z"},{"id":3,"name":"Middle","last_message_at":"2026-07-03T10:00:00Z"}]}`)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"limit":2,"order_by":"last_activity","sort":"desc"}`),
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
	if payload.Chats[0].ID != 2 || payload.Chats[1].ID != 3 {
		t.Fatalf("chats = %#v", payload.Chats)
	}
}

func TestListChatsAction_SinceFilters(t *testing.T) {
	mock := newMockIMsgWithChatsList(t, `{"chats":[{"id":1,"name":"Recent","last_message_at":"2026-07-07T10:00:00Z"},{"id":2,"name":"Old","last_message_at":"2026-06-01T10:00:00Z"},{"id":3,"name":"AlsoRecent","last_message_at":"2026-07-06T10:00:00Z"}]}`)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"limit":10,"since":"2026-07-06T00:00:00Z"}`),
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
