package imessage

import (
	"context"
	"encoding/json"
	"log"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails enriches iMessage approvals with human-readable chat names.
func (c *IMessageConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	actionCtx, cancel := c.actionTimeout(ctx)
	defer cancel()

	switch actionType {
	case "imessage.send_message":
		return c.resolveSendMessageDetails(actionCtx, params, creds)
	case "imessage.read_history", "imessage.get_chat":
		return c.resolveChatIDDetails(actionCtx, params, creds)
	default:
		return nil, nil
	}
}

func (c *IMessageConnector) resolveChatIDDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		ChatID int `json:"chat_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		log.Printf("imessage: resolve chat details: parse params: %v", err)
		return nil, nil
	}
	if p.ChatID <= 0 {
		log.Printf("imessage: resolve chat details: missing chat_id")
		return nil, nil
	}

	chatObj, err := lookupChatByID(ctx, c.client, creds, p.ChatID)
	if err != nil {
		log.Printf("imessage: resolve chat details: chat lookup: %v", err)
		return nil, nil
	}

	label := chatDisplayLabel(ctx, c.client, creds, chatObj)
	if label == "" {
		return nil, nil
	}
	details := map[string]any{"chat_name": label}
	if participants := participantsFromChat(chatObj); len(participants) > 0 {
		details["participants"] = participants
	}
	return details, nil
}

func (c *IMessageConnector) resolveSendMessageDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var sendParams sendMessageParams
	if err := json.Unmarshal(params, &sendParams); err != nil {
		log.Printf("imessage: resolve send_message details: parse params: %v", err)
		return nil, nil
	}
	if err := sendParams.validate(); err != nil {
		log.Printf("imessage: resolve send_message details: invalid params: %v", err)
		return nil, nil
	}

	chatObj, err := resolveChatForSend(ctx, c.client, creds, sendParams)
	if err != nil {
		log.Printf("imessage: resolve send_message details: chat lookup: %v", err)
	}

	service, disclosure := resolveDeliveryDisclosure(sendParams, chatObj)
	details := map[string]any{
		"to":                  sendPreviewTarget(sendParams),
		"delivery_service":    service,
		"delivery_disclosure": disclosure,
	}
	if chatObj != nil && chatObj.Service != "" {
		details["chat_service"] = chatObj.Service
	}

	displayChat := chatForDisplayLabel(ctx, c.client, creds, sendParams, chatObj)
	if label := chatDisplayLabel(ctx, c.client, creds, displayChat); label != "" {
		details["chat_name"] = label
	}
	participants := participantsFromChat(displayChat)
	if len(participants) == 0 {
		participants = participantsFromToHandles(sendParams.To)
	}
	if len(participants) > 0 {
		details["participants"] = participants
	}
	return details, nil
}

func participantsFromChat(ch *chat) []string {
	if ch == nil || len(ch.Participants) == 0 {
		return nil
	}
	return ch.Participants
}

func participantsFromToHandles(handles []Handle) []string {
	if len(handles) == 0 {
		return nil
	}
	out := make([]string, 0, len(handles))
	for _, h := range handles {
		if h.Value != "" {
			out = append(out, h.Value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// chatForDisplayLabel prefers chats.list metadata (contact names) over group CLI output.
func chatForDisplayLabel(ctx context.Context, client *imsgClient, creds connectors.Credentials, p sendMessageParams, chatObj *chat) *chat {
	if p.ChatID > 0 {
		if enriched, err := lookupChatByID(ctx, client, creds, p.ChatID); err == nil {
			return enriched
		}
	}
	return chatObj
}

var _ connectors.ResourceDetailResolver = (*IMessageConnector)(nil)
