package imessage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestIMessageConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "imessage" {
		t.Fatalf("ID() = %q", c.ID())
	}
}

func TestIMessageConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	want := []string{
		"imessage.list_chats",
		"imessage.get_chat",
		"imessage.read_history",
		"imessage.search",
		"imessage.send_message",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("missing action %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("got %d actions, want %d", len(actions), len(want))
	}
}

func TestIMessageConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if m.ID != "imessage" {
		t.Fatalf("manifest id = %q", m.ID)
	}
	if len(m.Actions) != 5 {
		t.Fatalf("got %d actions in manifest", len(m.Actions))
	}
}

func TestListChatsAction_Execute(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.list_chats"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.list_chats",
		Parameters:  json.RawMessage(`{"limit":5}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	chats, ok := payload["chats"].([]any)
	if !ok || len(chats) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSendMessageAction_Execute(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.send_message"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.send_message",
		Parameters:  json.RawMessage(`{"to":[{"type":"phone","value":"+15551234567"}],"text":"hello","service":"imessage"}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload sendResult
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !payload.OK || payload.GUID != "ABC" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSendMessageAction_IdempotentRetry(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.send_message"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.send_message",
		Parameters:  json.RawMessage(`{"to":[{"type":"phone","value":"+15551234567"}],"text":"hello","retry_guid":"PRIOR-GUID"}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["idempotent"] != true {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestResolveResourceDetails_SendMessage(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	details, err := c.ResolveResourceDetails(context.Background(), "imessage.send_message",
		json.RawMessage(`{"chat_id":42,"text":"hi"}`), creds)
	if err != nil {
		t.Fatalf("ResolveResourceDetails: %v", err)
	}
	if details["delivery_disclosure"] == "" {
		t.Fatalf("details = %#v", details)
	}
}

func TestGetChatAction_Execute(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.get_chat"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.get_chat",
		Parameters:  json.RawMessage(`{"chat_id":42}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload chat
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ID != 42 {
		t.Fatalf("id = %d", payload.ID)
	}
}

func TestSearchAction_Execute(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	result, err := c.Actions()["imessage.search"].Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "imessage.search",
		Parameters:  json.RawMessage(`{"query":"pizza"}`),
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["count"].(float64) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestProbeIMsg(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})
	if err := ProbeIMsg(context.Background(), mock.client, creds, defaultTimeout); err != nil {
		t.Fatalf("ProbeIMsg: %v", err)
	}
}

type mockIMsg struct {
	path   string
	client *imsgClient
}

func newMockIMsg(t *testing.T) *mockIMsg {
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
          result='{"chats":[{"id":1,"name":"Alice","participants":["+15551234567"]}]}'
          ;;
        messages.history)
          result='{"messages":[{"id":10,"guid":"g1","text":"hi","chat_id":1}]}'
          ;;
        send)
          result='{"ok":true,"guid":"ABC"}'
          ;;
        message.send_status)
          result='{"ok":true,"guid":"ABC","send_state":"delivered","service":"iMessage"}'
          ;;
        *)
          result='{}'
          ;;
      esac
      printf '{"jsonrpc":"2.0","id":%s,"result":%s}\n' "$id" "$result"
    done
    ;;
  group)
    printf '{"id":42,"guid":"iMessage;+;chat1","participants":["+15551234567"],"is_group":false}\n'
    ;;
  search)
    printf '{"id":9,"guid":"g9","text":"pizza"}\n'
    ;;
  account)
    printf '{"account_login":"me@icloud.com","account_id":"acc1"}\n'
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

func (m *mockIMsg) Close() {
	m.client.pool.closeAll()
}

func TestResolveSendConstraintMetadata(t *testing.T) {
	mock := newMockIMsg(t)
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	meta, err := c.ResolveConstraintMetadata(context.Background(), "imessage.send_message",
		json.RawMessage(`{"to":[{"type":"phone","value":"+15551234567"}],"text":"hi"}`), creds)
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["from_handle"] != "me@icloud.com" {
		t.Fatalf("from_handle = %v", meta["from_handle"])
	}
	to, ok := meta["to"].([]map[string]string)
	if !ok {
		// JSON round-trip via map[string]any uses []interface{}
		raw, ok := meta["to"].([]any)
		if !ok || len(raw) != 1 {
			t.Fatalf("to = %#v", meta["to"])
		}
		return
	}
	if len(to) != 1 {
		t.Fatalf("to = %#v", to)
	}
}

func TestMapRPCError_FullDiskAccess(t *testing.T) {
	err := mapRPCError(&rpcError{Message: "authorization denied: unable to open database file"}, "")
	if !connectors.IsAuthError(err) {
		t.Fatalf("got %T", err)
	}
}

func TestMapStartError_NotFound(t *testing.T) {
	err := mapStartError(fmt.Errorf("executable file not found"))
	if !connectors.IsExternalError(err) {
		t.Fatalf("got %T", err)
	}
	if !strings.Contains(err.Error(), "brew install") {
		t.Fatalf("msg = %q", err.Error())
	}
}
