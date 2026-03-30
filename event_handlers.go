package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// handleIMMessage processes message.im events from Slack DMs.
// It parses the typed payload and logs structured event data.
// This handler is the integration point for future DM automation workflows.
func handleIMMessage(ctx context.Context, logger *slog.Logger, event *connectors.Event) error {
	var msg struct {
		User      string `json:"user"`
		Channel   string `json:"channel"`
		Text      string `json:"text"`
		Timestamp string `json:"ts"`
	}
	if err := json.Unmarshal(event.Payload, &msg); err != nil {
		if logger != nil {
			logger.WarnContext(ctx, "SlackEvents: failed to parse message.im payload", "error", err)
		}
		return nil // don't retry on parse errors
	}

	if logger != nil {
		logger.InfoContext(ctx, "SlackEvents: DM received",
			"team_id", event.TeamID,
			"channel", msg.Channel,
			"user", msg.User,
			"ts", msg.Timestamp,
			"text_length", len(msg.Text))
	}

	return nil
}
