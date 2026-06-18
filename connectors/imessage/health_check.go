package imessage

import (
	"context"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// chatsListResult is the RPC response for chats.list.
type chatsListResult struct {
	Chats []chat `json:"chats"`
}

// ProbeIMsg verifies imsg is reachable and can read chat.db with a real query.
func ProbeIMsg(ctx context.Context, client *imsgClient, creds connectors.Credentials, timeout time.Duration) error {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var result chatsListResult
	if err := client.rpcCall(probeCtx, creds, "chats.list", map[string]any{"limit": 1}, &result); err != nil {
		return err
	}
	return nil
}
