package imessage

import (
	"bufio"
	"context"
	"strings"
	"testing"
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

func TestSessionPool_ReusesSession(t *testing.T) {
	t.Parallel()
	// session pool logic is exercised via integration tests with mock imsg script.
	pool := newSessionPool()
	if pool.sessionKey(commandConfig{CLIPath: "imsg"}) != "imsg|" {
		t.Fatalf("unexpected session key")
	}
}
