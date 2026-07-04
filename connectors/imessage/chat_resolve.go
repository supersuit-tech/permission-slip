package imessage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// resolveChatForSend loads chat metadata when the send targets an existing thread.
func resolveChatForSend(ctx context.Context, client *imsgClient, creds connectors.Credentials, p sendMessageParams) (*chat, error) {
	if p.ChatID <= 0 && p.ChatGUID == "" && p.ChatIdentifier == "" {
		return nil, nil
	}

	if p.ChatID > 0 {
		lines, err := client.runCLI(ctx, creds, "group", "--chat-id", fmt.Sprintf("%d", p.ChatID), "--json")
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			return nil, fmt.Errorf("chat %d not found", p.ChatID)
		}
		var chatObj chat
		if err := json.Unmarshal(lines[0], &chatObj); err != nil {
			return nil, err
		}
		return &chatObj, nil
	}

	chats, err := listChatsByRef(ctx, client, creds, p.ChatGUID, p.ChatIdentifier)
	if err != nil {
		return nil, err
	}
	if len(chats) == 0 {
		return nil, fmt.Errorf("chat not found")
	}
	return &chats[0], nil
}

const chatListLookupLimit = 200

// lookupChatByID finds a chat via chats.list (rich metadata including contact names).
func lookupChatByID(ctx context.Context, client *imsgClient, creds connectors.Credentials, chatID int) (*chat, error) {
	if chatID <= 0 {
		return nil, fmt.Errorf("invalid chat_id")
	}

	var result chatsListResult
	if err := client.rpcCall(ctx, creds, "chats.list", map[string]any{"limit": chatListLookupLimit}, &result); err != nil {
		return nil, err
	}
	for _, ch := range result.Chats {
		if ch.ID == chatID {
			copy := ch
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("chat %d not found", chatID)
}

func listChatsByRef(ctx context.Context, client *imsgClient, creds connectors.Credentials, guid, identifier string) ([]chat, error) {
	var result chatsListResult
	if err := client.rpcCall(ctx, creds, "chats.list", map[string]any{"limit": 200}, &result); err != nil {
		return nil, err
	}
	matches := make([]chat, 0, 1)
	for _, ch := range result.Chats {
		if guid != "" && ch.GUID == guid {
			matches = append(matches, ch)
		}
		if identifier != "" && ch.Identifier == identifier {
			matches = append(matches, ch)
		}
	}
	return matches, nil
}
