package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// handleIMMessage processes message.im events from Slack DMs.
// It parses the typed payload and logs structured event data.
// This handler is the integration point for future DM automation workflows.
func handleIMMessage(_ context.Context, event *connectors.Event) error {
	var msg struct {
		User      string `json:"user"`
		Channel   string `json:"channel"`
		Text      string `json:"text"`
		Timestamp string `json:"ts"`
	}
	if err := json.Unmarshal(event.Payload, &msg); err != nil {
		log.Printf("SlackEvents: failed to parse message.im payload: %v", err)
		return nil // don't retry on parse errors
	}

	log.Printf("SlackEvents: DM received: team=%s channel=%s user=%s ts=%s text_length=%d",
		event.TeamID, msg.Channel, msg.User, msg.Timestamp, len(msg.Text))

	return nil
}
