package imessage

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestReadRPCResponse_SkipsNotifications(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"message\",\"params\":{}}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"ok\":true}}\n")
	reader := bufio.NewReader(input)
	resp, err := readRPCResponse(context.Background(), reader, 7)
	if err != nil {
		t.Fatalf("readRPCResponse: %v", err)
	}
	if resp.ID != 7 {
		t.Fatalf("id = %d", resp.ID)
	}
}

func TestReadLineLimited_RejectsOversizedLine(t *testing.T) {
	t.Parallel()
	reader := bufio.NewReader(strings.NewReader(strings.Repeat("a", 20) + "\n"))
	_, err := readLineLimited(reader, 10)
	if err == nil || !strings.Contains(err.Error(), "line exceeds") {
		t.Fatalf("got %v", err)
	}
}

func TestBoundedStderr_KeepsTail(t *testing.T) {
	t.Parallel()
	buf := newBoundedStderr(8)
	_, _ = buf.Write([]byte("1234567890"))
	if got := buf.String(); got != "34567890" {
		t.Fatalf("tail = %q", got)
	}
}

func TestMapRPCError_IncludesCodeAndData(t *testing.T) {
	t.Parallel()
	err := mapRPCError(&rpcError{
		Code:    -32603,
		Message: "Internal error",
		Data:    json.RawMessage(`"db locked"`),
	}, "trace: permission denied")
	ext, ok := err.(*connectors.ExternalError)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if !strings.Contains(ext.Message, "imsg rpc error -32603") {
		t.Fatalf("msg = %q", ext.Message)
	}
	if !strings.Contains(ext.Message, "data: \"db locked\"") {
		t.Fatalf("msg = %q", ext.Message)
	}
	if !strings.Contains(ext.Message, "stderr: trace: permission denied") {
		t.Fatalf("msg = %q", ext.Message)
	}
}

func TestMapRPCError_ClassifiesDataField(t *testing.T) {
	t.Parallel()
	err := mapRPCError(&rpcError{
		Code:    -32603,
		Message: "Internal error",
		Data:    json.RawMessage(`"unable to open database file"`),
	}, "")
	if !connectors.IsAuthError(err) {
		t.Fatalf("got %T: %v", err, err)
	}
}

func TestValidateCommandConfig_RejectsSSHInjection(t *testing.T) {
	t.Parallel()
	cases := []commandConfig{
		{RemoteHost: "-oProxyCommand=id"},
		{RemoteHost: "host;rm -rf /"},
		{CLIPath: "/usr/bin/imsg;id"},
		{CLIPath: "-version"},
	}
	for _, cfg := range cases {
		if err := validateCommandConfig(cfg); err == nil {
			t.Fatalf("expected validation error for %#v", cfg)
		}
	}
}

func TestValidateCommandConfig_AllowsValid(t *testing.T) {
	t.Parallel()
	if err := validateCommandConfig(commandConfig{
		RemoteHost: "messages-mac",
		CLIPath:    "/opt/homebrew/bin/imsg",
	}); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestSessionPool_ReusesSession(t *testing.T) {
	t.Parallel()
	pool := newSessionPool()
	if pool.sessionKey(commandConfig{CLIPath: "imsg"}) != "imsg|" {
		t.Fatalf("unexpected session key")
	}
}
