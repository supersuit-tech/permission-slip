package imessage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const maxDisplayedGroupParticipants = 4

type nicknameResult struct {
	Address             string `json:"address"`
	LocalContactName    string `json:"local_contact_name"`
	Found               bool   `json:"found"`
	Source              string `json:"source"`
	ContactsUnavailable bool   `json:"contacts_unavailable"`
}

// chatDisplayLabel builds a human-readable chat label for approval summaries.
//
// Priority:
//  1. Named group → group name
//  2. Direct message → "with <contact name>" or formatted handle
//  3. Unnamed group → "with <participants>" (contact names where resolvable)
func chatDisplayLabel(ctx context.Context, client *imsgClient, creds connectors.Credentials, ch *chat) string {
	if ch == nil {
		return ""
	}

	if ch.IsGroup {
		if name := strings.TrimSpace(ch.Name); name != "" {
			return name
		}
		if name := strings.TrimSpace(ch.DisplayName); name != "" {
			return name
		}
		return formatUnnamedGroupParticipants(ctx, client, creds, ch.Participants)
	}

	if name := strings.TrimSpace(ch.ContactName); name != "" {
		return "with " + name
	}
	if name := strings.TrimSpace(ch.DisplayName); name != "" {
		return "with " + name
	}
	if name := strings.TrimSpace(ch.Name); name != "" {
		return "with " + name
	}
	if len(ch.Participants) > 0 {
		handle := strings.TrimSpace(ch.Participants[0])
		if handle != "" {
			return "with " + handle
		}
	}
	return ""
}

func formatUnnamedGroupParticipants(ctx context.Context, client *imsgClient, creds connectors.Credentials, participants []string) string {
	if len(participants) == 0 {
		return ""
	}

	displayCount := len(participants)
	if displayCount > maxDisplayedGroupParticipants {
		displayCount = maxDisplayedGroupParticipants
	}

	names := make([]string, 0, displayCount)
	for i := 0; i < displayCount; i++ {
		names = append(names, resolveParticipantName(ctx, client, creds, participants[i]))
	}

	label := "with " + strings.Join(names, ", ")
	if remaining := len(participants) - displayCount; remaining > 0 {
		label += fmt.Sprintf(", +%d more", remaining)
	}
	return label
}

func resolveParticipantName(ctx context.Context, client *imsgClient, creds connectors.Credentials, handle string) string {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return handle
	}

	lines, err := client.runCLI(ctx, creds, "nickname", "--address", handle, "--local")
	if err != nil || len(lines) == 0 {
		return handle
	}

	var result nicknameResult
	if err := json.Unmarshal(lines[0], &result); err != nil {
		return handle
	}
	if result.ContactsUnavailable || !result.Found {
		return handle
	}
	if name := strings.TrimSpace(result.LocalContactName); name != "" {
		return name
	}
	return handle
}
