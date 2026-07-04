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

func TestResolveResourceDetails_ReadHistory_NamedGroup(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"name":"Family 🏠","is_group":true,"participants":["+15551111111","+15552222222"]}]}`,
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	if details["chat_name"] != "Family 🏠" {
		t.Fatalf("chat_name = %#v", details["chat_name"])
	}
	assertParticipants(t, details, []string{"+15551111111", "+15552222222"})
}

func TestResolveResourceDetails_GetChat_NamedGroup(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"name":"Family 🏠","is_group":true}]}`,
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.get_chat", `{"chat_id":30}`)
	if details["chat_name"] != "Family 🏠" {
		t.Fatalf("chat_name = %#v", details["chat_name"])
	}
}

func TestResolveResourceDetails_ReadHistory_DMWithContactName(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"display_name":"Jane Appleseed","contact_name":"Jane Appleseed","participants":["+15551234567"]}]}`,
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	if details["chat_name"] != "with Jane Appleseed" {
		t.Fatalf("chat_name = %#v", details["chat_name"])
	}
	assertParticipants(t, details, []string{"+15551234567"})
}

func TestResolveResourceDetails_ReadHistory_DMWithoutContactName(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"display_name":"+15551234567","participants":["+15551234567"]}]}`,
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	if details["chat_name"] != "with +15551234567" {
		t.Fatalf("chat_name = %#v", details["chat_name"])
	}
}

func TestResolveResourceDetails_ReadHistory_UnnamedGroupWithNicknames(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"is_group":true,"participants":["+15551111111","+15552222222","+15553333333"]}]}`,
		nicknames: map[string]string{
			"+15551111111": "Jane",
			"+15552222222": "Bob",
		},
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	want := "with Jane, Bob, +15553333333"
	if details["chat_name"] != want {
		t.Fatalf("chat_name = %q, want %q", details["chat_name"], want)
	}
}

func TestResolveResourceDetails_ReadHistory_UnnamedGroupTruncated(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"is_group":true,"participants":["+15551111111","+15552222222","+15553333333","+15554444444","+15555555555","+15556666666"]}]}`,
		nicknames: map[string]string{
			"+15551111111": "Jane",
			"+15552222222": "Bob",
			"+15553333333": "Carol",
			"+15554444444": "Dave",
		},
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	want := "with Jane, Bob, Carol, Dave, +2 more"
	if details["chat_name"] != want {
		t.Fatalf("chat_name = %q, want %q", details["chat_name"], want)
	}
}

func TestResolveResourceDetails_ReadHistory_NicknameFallback(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":30,"is_group":true,"participants":["+15551111111","+15552222222"]}]}`,
		nicknames: map[string]string{
			"+15551111111": "",
		},
		nicknameUnavailable: map[string]bool{
			"+15551111111": true,
		},
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.read_history", `{"chat_id":30}`)
	want := "with +15551111111, +15552222222"
	if details["chat_name"] != want {
		t.Fatalf("chat_name = %q, want %q", details["chat_name"], want)
	}
}

func TestResolveResourceDetails_ReadHistory_LookupFailure(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":99,"name":"Other"}]}`,
	})
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	details, err := c.ResolveResourceDetails(context.Background(), "imessage.read_history",
		json.RawMessage(`{"chat_id":30}`), creds)
	if err != nil {
		t.Fatalf("ResolveResourceDetails: %v", err)
	}
	if details != nil {
		t.Fatalf("details = %#v, want nil", details)
	}
}

func TestResolveResourceDetails_SendMessage_UsesChatDisplayLabel(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		chatsList: `{"chats":[{"id":42,"display_name":"Jane Appleseed","contact_name":"Jane Appleseed","participants":["+15551234567"]}]}`,
	})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.send_message", `{"chat_id":42,"text":"hi"}`)
	if details["chat_name"] != "with Jane Appleseed" {
		t.Fatalf("chat_name = %#v", details["chat_name"])
	}
	if details["delivery_disclosure"] == "" {
		t.Fatalf("details = %#v", details)
	}
	assertParticipants(t, details, []string{"+15551234567"})
}

func TestResolveResourceDetails_SendMessage_ToAddressedFallback(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{})
	defer mock.Close()

	details := resolveChatDetails(t, mock, "imessage.send_message",
		`{"to":[{"type":"phone","value":"+15559876543"}],"text":"hi"}`)
	assertParticipants(t, details, []string{"+15559876543"})
}

func assertParticipants(t *testing.T, details map[string]any, want []string) {
	t.Helper()
	got := stringSliceFromAny(details["participants"])
	if len(got) != len(want) {
		t.Fatalf("participants = %#v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("participants[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func stringSliceFromAny(v any) []string {
	switch raw := v.(type) {
	case []string:
		return raw
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func resolveChatDetails(t *testing.T, mock *mockIMsg, actionType, params string) map[string]any {
	t.Helper()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	details, err := c.ResolveResourceDetails(context.Background(), actionType, json.RawMessage(params), creds)
	if err != nil {
		t.Fatalf("ResolveResourceDetails: %v", err)
	}
	if details == nil {
		t.Fatal("details is nil")
	}
	return details
}

type resolverMockConfig struct {
	chatsList           string
	nicknames           map[string]string
	nicknameUnavailable map[string]bool
	nicknameUnsupported bool
}

func newMockIMsgForResolver(t *testing.T, cfg resolverMockConfig) *mockIMsg {
	t.Helper()

	if cfg.chatsList == "" {
		cfg.chatsList = `{"chats":[]}`
	}

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "imsg")

	script := fmt.Sprintf(`#!/bin/bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  rpc)
    while IFS= read -r line; do
      id=$(printf '%%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
      method=$(printf '%%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
      case "$method" in
        chats.list)
          result='%s'
          ;;
        *)
          result='{}'
          ;;
      esac
      printf '{"jsonrpc":"2.0","id":%%s,"result":%%s}\n' "$id" "$result"
    done
    ;;
  group)
    printf '{"id":42,"guid":"iMessage;+;chat1","participants":["+15551234567"],"is_group":false}\n'
    ;;
  nickname)
    if [ "%t" = true ]; then
      echo "unknown command" >&2
      exit 1
    fi
    address=""
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --address) address="$2"; shift 2 ;;
        --local) shift ;;
        *) shift ;;
      esac
    done
%s
    ;;
  *)
    echo "unknown command" >&2
    exit 1
    ;;
esac
`, cfg.chatsList, cfg.nicknameUnsupported, buildNicknameCase(cfg))

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return &mockIMsg{path: scriptPath, client: newIMsgClient()}
}

func buildNicknameCase(cfg resolverMockConfig) string {
	if len(cfg.nicknames) == 0 && len(cfg.nicknameUnavailable) == 0 {
		return `    printf '{"address":"%s","found":false}\n' "$address"`
	}

	var b strings.Builder
	b.WriteString("    case \"$address\" in\n")
	seen := make(map[string]struct{})
	for addr := range cfg.nicknames {
		seen[addr] = struct{}{}
	}
	for addr := range cfg.nicknameUnavailable {
		seen[addr] = struct{}{}
	}
	for addr := range seen {
		if cfg.nicknameUnavailable != nil && cfg.nicknameUnavailable[addr] {
			fmt.Fprintf(&b, "      %q) printf '{\"address\":%q,\"found\":false,\"contacts_unavailable\":true}\\n' ;;\n", addr, addr)
			continue
		}
		name := ""
		if cfg.nicknames != nil {
			name = cfg.nicknames[addr]
		}
		if name != "" {
			fmt.Fprintf(&b, "      %q) printf '{\"address\":%q,\"local_contact_name\":%q,\"found\":true,\"source\":\"local\"}\\n' ;;\n", addr, addr, name)
		} else {
			fmt.Fprintf(&b, "      %q) printf '{\"address\":%q,\"found\":false}\\n' ;;\n", addr, addr)
		}
	}
	b.WriteString("      *) printf '{\"address\":\"%s\",\"found\":false}\\n' \"$address\" ;;\n")
	b.WriteString("    esac\n")
	return b.String()
}

func TestChatDisplayLabel_NamedGroupUsesName(t *testing.T) {
	label := chatDisplayLabel(context.Background(), nil, connectors.Credentials{}, &chat{
		IsGroup: true,
		Name:    "Family",
	})
	if label != "Family" {
		t.Fatalf("label = %q", label)
	}
}

func TestChatDisplayLabel_DMUsesContactName(t *testing.T) {
	label := chatDisplayLabel(context.Background(), nil, connectors.Credentials{}, &chat{
		ContactName: "Jane Appleseed",
	})
	if label != "with Jane Appleseed" {
		t.Fatalf("label = %q", label)
	}
}

func TestResolveParticipantName_FallsBackToHandle(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{
		nicknameUnsupported: true,
	})
	defer mock.Close()

	name := resolveParticipantName(context.Background(), mock.client,
		connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path}), "+15551234567")
	if name != "+15551234567" {
		t.Fatalf("name = %q", name)
	}
}

func TestFormatUnnamedGroupParticipants_Empty(t *testing.T) {
	label := formatUnnamedGroupParticipants(context.Background(), nil, connectors.Credentials{}, nil)
	if label != "" {
		t.Fatalf("label = %q", label)
	}
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	mock := newMockIMsgForResolver(t, resolverMockConfig{})
	defer mock.Close()

	c := New()
	c.client = mock.client
	creds := connectors.NewCredentials(map[string]string{credKeyCLIPath: mock.path})

	details, err := c.ResolveResourceDetails(context.Background(), "imessage.search", json.RawMessage(`{"query":"hi"}`), creds)
	if err != nil {
		t.Fatalf("ResolveResourceDetails: %v", err)
	}
	if details != nil {
		t.Fatalf("details = %#v", details)
	}
}

// Ensure nickname mock script stays valid bash when chats list contains quotes.
func TestResolverMockConfig_EscapesQuotes(t *testing.T) {
	cfg := resolverMockConfig{
		chatsList: strings.ReplaceAll(`{"chats":[{"id":1,"name":"O'Brien"}]}`, "'", "'\\''"),
	}
	mock := newMockIMsgForResolver(t, cfg)
	defer mock.Close()
}
