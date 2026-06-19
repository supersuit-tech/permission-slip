// Package imessage implements the built-in iMessage connector for Permission
// Slip. It wraps openclaw/imsg (https://github.com/openclaw/imsg) to read and
// send iMessages via a persistent JSON-RPC session over stdio.
//
// imsg must be installed on a Mac with Messages.app signed in:
//
//	brew install steipete/tap/imsg
//
// Permission Slip can run on the same Mac, or on Linux with imsg reached over
// SSH (set remote_host in connector credentials).
package imessage

import (
	"context"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultTimeout = 30 * time.Second

	credKeyCLIPath    = "cli_path"
	credKeyRemoteHost = "remote_host"
)

// IMessageConnector owns shared configuration for all iMessage actions.
type IMessageConnector struct {
	timeout time.Duration
	client  *imsgClient
}

// New creates an IMessageConnector with sensible defaults.
func New() *IMessageConnector {
	return &IMessageConnector{
		timeout: defaultTimeout,
		client:  newIMsgClient(),
	}
}

// ID returns "imessage".
func (c *IMessageConnector) ID() string { return "imessage" }

// Actions returns the registered action handlers keyed by action_type.
func (c *IMessageConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"imessage.list_chats":   &listChatsAction{conn: c},
		"imessage.get_chat":     &getChatAction{conn: c},
		"imessage.read_history": &readHistoryAction{conn: c},
		"imessage.search":       &searchAction{conn: c},
		"imessage.send_message": &sendMessageAction{conn: c},
	}
}

// ValidateCredentials probes imsg with a real read (chats.list limit 1).
func (c *IMessageConnector) ValidateCredentials(ctx context.Context, creds connectors.Credentials) error {
	timeout := c.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	return ProbeIMsg(ctx, c.client, creds, timeout)
}

func (c *IMessageConnector) actionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(ctx, timeout)
}
